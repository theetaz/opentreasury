package treasury

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestPostEntryWritesEntryLinesBalancesAndAuditAtomically(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresJournalRepository(db)
	entry := balancedEntry()
	ctx := ContextWithAuditMetadata(context.Background(), AuditMetadata{Actor: "system", RequestID: "req-1"})

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO journal_entries")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Two lines, each with its own insert and balance upsert.
	for range entry.Lines {
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO journal_lines")).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO account_balances")).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO audit_events")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repository.PostEntry(ctx, entry))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostEntryRejectsUnbalancedBeforeTouchingTheDatabase(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresJournalRepository(db)
	entry := balancedEntry()
	entry.Lines[1].AmountMinor = 1

	require.ErrorIs(t, repository.PostEntry(context.Background(), entry), ErrUnbalancedEntry)
	require.NoError(t, mock.ExpectationsWereMet()) // no DB calls expected
}

func TestPostEntryRollsBackWhenBalanceUpdateFails(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresJournalRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO journal_entries")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO journal_lines")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO account_balances")).
		WillReturnError(errIntegration)
	mock.ExpectRollback()

	require.Error(t, repository.PostEntry(context.Background(), balancedEntry()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListEntriesFiltersAndPaginates(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresJournalRepository(db)

	rows := sqlmock.NewRows([]string{
		"id", "institution_id", "fiscal_year", "effective_date", "description",
		"status", "entry_type", "total_count",
	}).AddRow("je-1", "minfin", 2026, "2026-06-28", "Tax receipt", "POSTED", "STANDARD", 1)

	mock.ExpectQuery(regexp.QuoteMeta("FROM journal_entries")).
		WithArgs("minfin", 15, 0).
		WillReturnRows(rows)

	lineRows := sqlmock.NewRows([]string{"entry_id", "account_code", "direction", "amount_minor", "currency"}).
		AddRow("je-1", "6202", "DEBIT", int64(125000), "USD").
		AddRow("je-1", "114", "CREDIT", int64(125000), "USD")
	mock.ExpectQuery(regexp.QuoteMeta("FROM journal_lines")).
		WithArgs("je-1").
		WillReturnRows(lineRows)

	page, err := repository.ListEntries(context.Background(), ListJournalEntriesFilter{
		InstitutionID: "minfin",
		Pagination:    Pagination{Page: 1, PageSize: 15},
	})
	require.NoError(t, err)
	require.Len(t, page.Entries, 1)
	require.Equal(t, "je-1", page.Entries[0].ID)
	require.Len(t, page.Entries[0].Lines, 2, "entry lines must be hydrated")
	require.Equal(t, 1, page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListBalancesJoinsAccountMetadata(t *testing.T) {
	db, mock := newMockDB(t)
	repository := NewPostgresJournalRepository(db)

	rows := sqlmock.NewRows([]string{
		"institution_id", "account_code", "account_name", "account_type", "currency", "balance_minor", "total_count",
	}).AddRow("minfin", "6202", "Currency and deposits", "ASSET", "USD", int64(125000), 1)

	mock.ExpectQuery(regexp.QuoteMeta("FROM account_balances")).
		WithArgs("minfin", 15, 0).
		WillReturnRows(rows)

	page, err := repository.ListBalances(context.Background(), ListBalancesFilter{
		InstitutionID: "minfin",
		Pagination:    Pagination{Page: 1, PageSize: 15},
	})
	require.NoError(t, err)
	require.Len(t, page.Balances, 1)
	require.Equal(t, int64(125000), page.Balances[0].BalanceMinor)
	require.Equal(t, "Currency and deposits", page.Balances[0].AccountName)
	require.NoError(t, mock.ExpectationsWereMet())
}
