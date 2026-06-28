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

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db, mock
}
