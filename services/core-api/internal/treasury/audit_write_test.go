package treasury

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestSaveWritesTransactionAndAuditEventAtomically(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresTransactionRepository(db)
	tx := validTransaction()
	ctx := ContextWithAuditMetadata(context.Background(), AuditMetadata{
		Actor:     "system",
		RequestID: "req-abc-123",
	})

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO treasury_transactions")).
		WithArgs(tx.ID, tx.InstitutionID, tx.FiscalYear, tx.AmountMinor, tx.Currency, tx.Description, tx.TransactionDate, "POSTED").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO audit_events")).
		WithArgs("TRANSACTION_CREATED", tx.ID, tx.InstitutionID, "Transaction "+tx.ID+" was created.", "system", "req-abc-123").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repository.Save(ctx, tx))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveRollsBackWhenAuditWriteFails(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresTransactionRepository(db)
	tx := validTransaction()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO treasury_transactions")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO audit_events")).
		WillReturnError(errors.New("disk full"))
	mock.ExpectRollback()

	require.Error(t, repository.Save(context.Background(), tx))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListAuditEventsReadsTheAuditTable(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresTransactionRepository(db)

	rows := sqlmock.NewRows([]string{"id", "event_type", "transaction_id", "institution_id", "occurred_at", "summary", "total_count"}).
		AddRow("evt-1", "TRANSACTION_CREATED", "txn-1", "minfin", "2026-07-09T12:00:00Z", "Transaction txn-1 was created.", 1)

	mock.ExpectQuery(regexp.QuoteMeta("FROM audit_events")).
		WithArgs("minfin", 25, 0).
		WillReturnRows(rows)

	page, err := repository.ListAuditEvents(context.Background(), ListAuditEventsFilter{
		InstitutionID: "minfin",
		Pagination:    Pagination{Page: 1, PageSize: 25},
	})

	require.NoError(t, err)
	require.Len(t, page.Events, 1)
	require.Equal(t, "evt-1", page.Events[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}
