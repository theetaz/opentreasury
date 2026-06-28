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

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db, mock
}
