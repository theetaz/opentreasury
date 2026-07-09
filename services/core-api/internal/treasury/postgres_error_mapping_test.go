package treasury

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func validSaveTransaction() Transaction {
	return Transaction{
		ID:              "txn-map-1",
		InstitutionID:   "minfin",
		FiscalYear:      2026,
		AmountMinor:     1000,
		Currency:        "USD",
		Description:     "error mapping test",
		TransactionDate: "2026-07-09",
	}
}

func expectInsert(mock sqlmock.Sqlmock) *sqlmock.ExpectedExec {
	return mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO treasury_transactions (
			id,
			institution_id,
			fiscal_year,
			amount_minor,
			currency,
			description,
			transaction_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`))
}

func TestSaveMapsUniqueViolationToDuplicateTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectInsert(mock).WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "treasury_transactions_pkey"})

	saveErr := NewPostgresTransactionRepository(db).Save(context.Background(), validSaveTransaction())

	require.ErrorIs(t, saveErr, ErrDuplicateTransaction)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveMapsForeignKeyViolationToUnknownInstitution(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectInsert(mock).WillReturnError(&pgconn.PgError{Code: "23503", ConstraintName: "treasury_transactions_institution_id_fkey"})

	saveErr := NewPostgresTransactionRepository(db).Save(context.Background(), validSaveTransaction())

	require.ErrorIs(t, saveErr, ErrUnknownInstitution)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSavePassesThroughUnrecognizedDatabaseErrors(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	expectInsert(mock).WillReturnError(&pgconn.PgError{Code: "57P01"})

	saveErr := NewPostgresTransactionRepository(db).Save(context.Background(), validSaveTransaction())

	require.Error(t, saveErr)
	require.NotErrorIs(t, saveErr, ErrDuplicateTransaction)
	require.NotErrorIs(t, saveErr, ErrUnknownInstitution)
	require.NoError(t, mock.ExpectationsWereMet())
}
