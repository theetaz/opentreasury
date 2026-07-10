//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

func TestCommitmentLifecycleSettlesThroughTheJournal(t *testing.T) {
	dsn := startPostgres(t)
	migrator := newMigrator(t, dsn)
	db := openDB(t, dsn)

	require.NoError(t, migrator.Up())
	applySeeds(t, db)

	commitments := treasury.NewPostgresCommitmentRepository(db)
	journal := treasury.NewPostgresJournalRepository(db)
	ctx := treasury.ContextWithAuditMetadata(context.Background(), treasury.AuditMetadata{Actor: "tester", RequestID: "req"})

	require.NoError(t, commitments.CreateCommitment(ctx, treasury.Commitment{
		ID: "com-1", InstitutionID: "minfin", FiscalYear: 2026, AccountCode: "22",
		Description: "framework contract", AmountMinor: 10000, Currency: "USD",
		CommittedDate: "2026-07-01",
	}))

	entry := func(id string, amount int64) treasury.JournalEntry {
		return treasury.JournalEntry{
			ID: id, InstitutionID: "minfin", FiscalYear: 2026, EffectiveDate: "2026-07-05",
			Description: "settles com-1", CommitmentID: "com-1",
			Lines: []treasury.JournalLine{
				{AccountCode: "22", Direction: "DEBIT", AmountMinor: amount, Currency: "USD"},
				{AccountCode: "6202", Direction: "CREDIT", AmountMinor: amount, Currency: "USD"},
			},
		}
	}

	// Partial settlement leaves the commitment OPEN with the remainder.
	require.NoError(t, journal.PostEntry(ctx, entry("je-c1", 4000)))
	page, err := commitments.ListCommitments(ctx, treasury.ListCommitmentsFilter{
		InstitutionID: "minfin", Pagination: treasury.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	require.Equal(t, "OPEN", page.Commitments[0].Status)
	require.Equal(t, int64(4000), page.Commitments[0].SettledAmountMinor)

	// Over-settlement is rejected and rolls back the whole entry.
	err = journal.PostEntry(ctx, entry("je-c2", 7000))
	require.ErrorIs(t, err, treasury.ErrCommitmentExceeded)
	var entryCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM journal_entries WHERE id = 'je-c2'").Scan(&entryCount))
	require.Zero(t, entryCount, "a rejected settlement must not leave a posted entry behind")

	// Settling the exact remainder flips the commitment to SETTLED.
	require.NoError(t, journal.PostEntry(ctx, entry("je-c3", 6000)))
	page, err = commitments.ListCommitments(ctx, treasury.ListCommitmentsFilter{
		Status: "SETTLED", Pagination: treasury.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	require.Len(t, page.Commitments, 1)
	require.Equal(t, int64(10000), page.Commitments[0].SettledAmountMinor)

	// A settled commitment refuses further entries.
	err = journal.PostEntry(ctx, entry("je-c4", 1))
	require.ErrorIs(t, err, treasury.ErrCommitmentClosed)
}
