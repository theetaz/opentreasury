package treasury

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func stagedRecord() StagingRecord {
	return StagingRecord{
		SourceSystem:  "itmis",
		SourceRef:     "PAY-001",
		SourceHash:    "hash-pay-001",
		Profile:       "itmis-payments@1",
		InstitutionID: "minfin",
		OccurredAt:    "2026-07-01",
		Lines: []JournalLine{
			{AccountCode: "22", Direction: "DEBIT", AmountMinor: 4000, Currency: "USD"},
			{AccountCode: "6202", Direction: "CREDIT", AmountMinor: 4000, Currency: "USD"},
		},
	}
}

func TestIngestPostsAValidRecord(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresStagingRepository(db, NewPostgresJournalRepository(db))

	// Duplicate probe finds nothing.
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status, reason, COALESCE(entry_id, '') FROM staging_records WHERE source_hash = $1")).
		WithArgs("hash-pay-001").
		WillReturnRows(sqlmock.NewRows([]string{"status", "reason", "entry_id"}))
	// Journal posting succeeds (entry + lines + balances + audit in a tx).
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO journal_entries")).WillReturnResult(sqlmock.NewResult(0, 1))
	for range 2 {
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO journal_lines")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO account_balances")).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO audit_events")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	// Staging record persisted as POSTED.
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO staging_records")).
		WillReturnResult(sqlmock.NewResult(0, 1))

	outcome, err := repository.Ingest(context.Background(), stagedRecord())

	require.NoError(t, err)
	require.Equal(t, "POSTED", outcome.Status)
	require.Equal(t, "itmis-PAY-001", outcome.EntryID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIngestQuarantinesAnInvalidRecord(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresStagingRepository(db, NewPostgresJournalRepository(db))

	record := stagedRecord()
	record.Lines[1].AmountMinor = 999 // unbalanced

	mock.ExpectQuery(regexp.QuoteMeta("FROM staging_records WHERE source_hash")).
		WithArgs("hash-pay-001").
		WillReturnRows(sqlmock.NewRows([]string{"status", "reason", "entry_id"}))
	// No journal writes happen; the record lands quarantined with the reason.
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO staging_records")).
		WillReturnResult(sqlmock.NewResult(0, 1))

	outcome, err := repository.Ingest(context.Background(), record)

	require.NoError(t, err)
	require.Equal(t, "QUARANTINED", outcome.Status)
	require.Contains(t, outcome.Reason, "balanced")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIngestIsIdempotentOnRedelivery(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresStagingRepository(db, NewPostgresJournalRepository(db))

	mock.ExpectQuery(regexp.QuoteMeta("FROM staging_records WHERE source_hash")).
		WithArgs("hash-pay-001").
		WillReturnRows(sqlmock.NewRows([]string{"status", "reason", "entry_id"}).
			AddRow("POSTED", "", "itmis-PAY-001"))

	outcome, err := repository.Ingest(context.Background(), stagedRecord())

	require.NoError(t, err)
	require.Equal(t, "DUPLICATE", outcome.Status)
	require.Equal(t, "itmis-PAY-001", outcome.EntryID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListStagingRecordsFiltersByStatus(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresStagingRepository(db, NewPostgresJournalRepository(db))

	rows := sqlmock.NewRows([]string{
		"id", "source_system", "source_ref", "source_hash", "profile",
		"institution_id", "occurred_at", "lines", "status", "reason", "entry_id", "created_at", "total_count",
	}).AddRow(
		"sr-1", "itmis", "PAY-002", "hash-2", "itmis-payments@1",
		"minfin", "2026-07-01", []byte(`[]`), "QUARANTINED", "institution does not exist", "", "2026-07-09T12:00:00Z", 1,
	)

	mock.ExpectQuery(regexp.QuoteMeta("WHERE status = $1")).
		WithArgs("QUARANTINED", 15, 0).
		WillReturnRows(rows)

	page, err := repository.ListStagingRecords(context.Background(), ListStagingRecordsFilter{
		Status:     "QUARANTINED",
		Pagination: Pagination{Page: 1, PageSize: 15},
	})

	require.NoError(t, err)
	require.Len(t, page.Records, 1)
	require.Equal(t, "QUARANTINED", page.Records[0].Status)
	require.NoError(t, mock.ExpectationsWereMet())
}
