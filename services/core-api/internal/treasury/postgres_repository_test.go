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
		WithArgs(
			tx.ID,
			tx.InstitutionID,
			tx.FiscalYear,
			tx.AmountMinor,
			tx.Currency,
			tx.Description,
			tx.TransactionDate,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

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

	rows := sqlmock.NewRows([]string{
		"id",
		"institution_id",
		"fiscal_year",
		"amount_minor",
		"currency",
		"description",
		"transaction_date",
	}).
		AddRow("txn-2026-0002", "minfin", 2026, int64(450075), "USD", "Quarterly grant release", "2026-06-29").
		AddRow("txn-2026-0001", "minfin", 2026, int64(125000), "USD", "Road maintenance payment", "2026-06-28")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			id,
			institution_id,
			fiscal_year,
			amount_minor,
			currency,
			description,
			transaction_date::text
		FROM treasury_transactions
		WHERE institution_id = $1 AND fiscal_year = $2
		ORDER BY transaction_date DESC, created_at DESC, id DESC
		LIMIT $3
	`)).
		WithArgs("minfin", 2026, 25).
		WillReturnRows(rows)

	transactions, err := repository.List(context.Background(), ListTransactionsFilter{
		InstitutionID: "minfin",
		FiscalYear:    2026,
		Limit:         25,
	})

	require.NoError(t, err)
	require.Len(t, transactions, 2)
	require.Equal(t, "txn-2026-0002", transactions[0].ID)
	require.Equal(t, int64(125000), transactions[1].AmountMinor)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresTransactionRepository_ListReturnsRecentTransactions(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresTransactionRepository(db)

	rows := sqlmock.NewRows([]string{
		"id",
		"institution_id",
		"fiscal_year",
		"amount_minor",
		"currency",
		"description",
		"transaction_date",
	}).
		AddRow("txn-2026-0002", "minfin", 2026, int64(450075), "USD", "Quarterly grant release", "2026-06-29")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			id,
			institution_id,
			fiscal_year,
			amount_minor,
			currency,
			description,
			transaction_date::text
		FROM treasury_transactions
		ORDER BY transaction_date DESC, created_at DESC, id DESC
		LIMIT $1
	`)).
		WithArgs(50).
		WillReturnRows(rows)

	transactions, err := repository.List(context.Background(), ListTransactionsFilter{
		Limit: 50,
	})

	require.NoError(t, err)
	require.Len(t, transactions, 1)
	require.Equal(t, "txn-2026-0002", transactions[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresTransactionRepository_ListAuditEventsReturnsTransactionCreationEvents(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresTransactionRepository(db)

	rows := sqlmock.NewRows([]string{
		"id",
		"event_type",
		"transaction_id",
		"institution_id",
		"occurred_at",
		"summary",
	}).
		AddRow(
			"audit-txn-2026-0001-created",
			"TRANSACTION_CREATED",
			"txn-2026-0001",
			"minfin",
			"2026-06-28T10:24:28Z",
			"Transaction txn-2026-0001 was created.",
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			'audit-' || id || '-created' AS id,
			'TRANSACTION_CREATED' AS event_type,
			id AS transaction_id,
			institution_id,
			created_at::text AS occurred_at,
			'Transaction ' || id || ' was created.' AS summary
		FROM treasury_transactions
		WHERE institution_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`)).
		WithArgs("minfin", 25).
		WillReturnRows(rows)

	events, err := repository.ListAuditEvents(context.Background(), ListAuditEventsFilter{
		InstitutionID: "minfin",
		Limit:         25,
	})

	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "audit-txn-2026-0001-created", events[0].ID)
	require.Equal(t, "TRANSACTION_CREATED", events[0].EventType)
	require.Equal(t, "txn-2026-0001", events[0].TransactionID)
	require.Equal(t, "minfin", events[0].InstitutionID)
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
