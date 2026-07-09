package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestNewServerUsesConfiguredAddressAndRouter(t *testing.T) {
	server := newServer(config{addr: "127.0.0.1:9090"}, nil)

	if server.Addr != "127.0.0.1:9090" {
		t.Fatalf("server.Addr = %q, want %q", server.Addr, "127.0.0.1:9090")
	}

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	server.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response.Code = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestNewServerUsesDatabaseBackedTransactionRepository(t *testing.T) {
	db, mock := newMockDB(t)
	server := newServer(config{addr: "127.0.0.1:9090"}, db)
	request := httptest.NewRequest(http.MethodPost, "/v1/transactions", strings.NewReader(`{
		"id": "txn-2026-0001",
		"institutionId": "minfin",
		"fiscalYear": 2026,
		"amountMinor": 125000,
		"currency": "USD",
		"description": "Road maintenance payment",
		"transactionDate": "2026-06-28"
	}`))
	response := httptest.NewRecorder()

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO treasury_transactions (
			id,
			institution_id,
			fiscal_year,
			amount_minor,
			currency,
			description,
			transaction_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`)).
		WithArgs("txn-2026-0001", "minfin", 2026, int64(125000), "USD", "Road maintenance payment", "2026-06-28").
		WillReturnResult(sqlmock.NewResult(0, 1))

	server.Handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusCreated, response.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNewServerUsesDatabaseBackedInstitutionRepository(t *testing.T) {
	db, mock := newMockDB(t)
	server := newServer(config{addr: "127.0.0.1:9090"}, db)
	request := httptest.NewRequest(http.MethodGet, "/v1/institutions?limit=25", nil)
	response := httptest.NewRecorder()

	rows := sqlmock.NewRows([]string{
		"id",
		"name",
		"type",
		"country_code",
		"status",
		"total_count",
	}).
		AddRow("minfin", "Ministry of Finance", "MINISTRY", "KE", "ACTIVE", 1)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			id,
			name,
			type,
			country_code,
			status,
			COUNT(*) OVER() AS total_count
		FROM treasury_institutions
		ORDER BY name ASC, id ASC
		LIMIT $1 OFFSET $2
	`)).
		WithArgs(25, 0).
		WillReturnRows(rows)

	server.Handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadConfigUsesConfiguredAddress(t *testing.T) {
	t.Setenv("OPENTREASURY_CORE_API_ADDR", "127.0.0.1:0")

	cfg := loadConfig()

	if cfg.addr != "127.0.0.1:0" {
		t.Fatalf("cfg.addr = %q, want %q", cfg.addr, "127.0.0.1:0")
	}
}

func TestLoadConfigUsesDatabaseDSN(t *testing.T) {
	t.Setenv("OPENTREASURY_DATABASE_DSN", "postgres://opentreasury:secret@localhost:5432/opentreasury")

	cfg := loadConfig()

	if cfg.databaseDSN != "postgres://opentreasury:secret@localhost:5432/opentreasury" {
		t.Fatalf("cfg.databaseDSN = %q, want configured DSN", cfg.databaseDSN)
	}
}

func TestLoadConfigUsesAllowedOrigins(t *testing.T) {
	t.Setenv("OPENTREASURY_ALLOWED_ORIGINS", "http://localhost:5173, https://treasury.example")

	cfg := loadConfig()

	require.Equal(t, []string{"http://localhost:5173", "https://treasury.example"}, cfg.allowedOrigins)
}

func TestOpenDatabaseWithoutDSN(t *testing.T) {
	db, closeDatabase, err := openDatabase(config{})

	require.NoError(t, err)
	require.Nil(t, db)
	require.NotNil(t, closeDatabase)
	require.NoError(t, closeDatabase())
}

func TestOpenDatabaseWithDSN(t *testing.T) {
	db, closeDatabase, err := openDatabase(config{databaseDSN: "postgres://opentreasury:secret@localhost:5432/opentreasury"})

	require.NoError(t, err)
	require.NotNil(t, db)
	require.NotNil(t, closeDatabase)
	require.NoError(t, closeDatabase())
}

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db, mock
}
