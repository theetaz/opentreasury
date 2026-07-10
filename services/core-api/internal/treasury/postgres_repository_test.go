package treasury

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestPostgresTransactionRepository_Save(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresTransactionRepository(db)
	tx := validTransaction()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO treasury_transactions (
			id,
			institution_id,
			fiscal_year,
			amount_minor,
			currency,
			description,
			transaction_date,
			status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`)).
		WithArgs(
			tx.ID,
			tx.InstitutionID,
			tx.FiscalYear,
			tx.AmountMinor,
			tx.Currency,
			tx.Description,
			tx.TransactionDate,
			"POSTED",
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO audit_events")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repository.Save(context.Background(), tx)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresTransactionRepository_Save_WithInvalidTransaction(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresTransactionRepository(db)
	tx := validTransaction()
	tx.AmountMinor = 0

	err := repository.Save(context.Background(), tx)

	require.ErrorIs(t, err, ErrInvalidAmount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresTransactionRepository_ListReturnsFilteredTransactions(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresTransactionRepository(db)

	rows := sqlmock.NewRows(transactionColumns()).
		AddRow("txn-2026-0002", "minfin", 2026, int64(450075), "USD", "Quarterly grant release", "2026-06-29", "POSTED", 2).
		AddRow("txn-2026-0001", "minfin", 2026, int64(125000), "USD", "Road maintenance payment", "2026-06-28", "POSTED", 2)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			id,
			institution_id,
			fiscal_year,
			amount_minor,
			currency,
			description,
			transaction_date::text,
			status,
			COUNT(*) OVER() AS total_count
		FROM treasury_transactions
		WHERE institution_id = $1 AND fiscal_year = $2
		ORDER BY transaction_date DESC, created_at DESC, id DESC
		LIMIT $3 OFFSET $4
	`)).
		WithArgs("minfin", 2026, 25, 0).
		WillReturnRows(rows)

	page, err := repository.List(context.Background(), ListTransactionsFilter{
		InstitutionID: "minfin",
		FiscalYear:    2026,
		Pagination:    Pagination{Page: 1, PageSize: 25},
	})

	require.NoError(t, err)
	require.Len(t, page.Transactions, 2)
	require.Equal(t, 2, page.Total)
	require.Equal(t, "txn-2026-0002", page.Transactions[0].ID)
	require.Equal(t, int64(125000), page.Transactions[1].AmountMinor)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresTransactionRepository_ListReturnsRecentTransactions(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresTransactionRepository(db)

	rows := sqlmock.NewRows(transactionColumns()).
		AddRow("txn-2026-0002", "minfin", 2026, int64(450075), "USD", "Quarterly grant release", "2026-06-29", "POSTED", 1)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			id,
			institution_id,
			fiscal_year,
			amount_minor,
			currency,
			description,
			transaction_date::text,
			status,
			COUNT(*) OVER() AS total_count
		FROM treasury_transactions
		ORDER BY transaction_date DESC, created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`)).
		WithArgs(50, 0).
		WillReturnRows(rows)

	page, err := repository.List(context.Background(), ListTransactionsFilter{
		Pagination: Pagination{Page: 1, PageSize: 50},
	})

	require.NoError(t, err)
	require.Len(t, page.Transactions, 1)
	require.Equal(t, "txn-2026-0002", page.Transactions[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInstitutionRepository_ListReturnsInstitutions(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresInstitutionRepository(db)

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

	page, err := repository.ListInstitutions(context.Background(), ListInstitutionsFilter{
		Pagination: Pagination{Page: 1, PageSize: 25},
	})

	require.NoError(t, err)
	require.Len(t, page.Institutions, 1)
	require.Equal(t, "minfin", page.Institutions[0].ID)
	require.Equal(t, "Ministry of Finance", page.Institutions[0].Name)
	require.Equal(t, "MINISTRY", page.Institutions[0].Type)
	require.Equal(t, "KE", page.Institutions[0].CountryCode)
	require.Equal(t, "ACTIVE", page.Institutions[0].Status)
	require.NoError(t, mock.ExpectationsWereMet())
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
