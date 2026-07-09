//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

func TestIngestionPipelinePostsQuarantinesAndDeduplicates(t *testing.T) {
	dsn := startPostgres(t)
	migrator := newMigrator(t, dsn)
	db := openDB(t, dsn)

	require.NoError(t, migrator.Up())
	applySeeds(t, db)

	journal := treasury.NewPostgresJournalRepository(db)
	staging := treasury.NewPostgresStagingRepository(db, journal)
	ctx := treasury.ContextWithAuditMetadata(context.Background(), treasury.AuditMetadata{Actor: "connector", RequestID: "req"})

	valid := treasury.StagingRecord{
		SourceSystem: "itmis", SourceRef: "PAY-1", SourceHash: "hash-1", Profile: "itmis@1",
		InstitutionID: "minfin", OccurredAt: "2026-07-01",
		Lines: []treasury.JournalLine{
			{AccountCode: "22", Direction: "DEBIT", AmountMinor: 4000, Currency: "USD"},
			{AccountCode: "6202", Direction: "CREDIT", AmountMinor: 4000, Currency: "USD"},
		},
	}

	// 1. Valid record posts and moves stocks.
	outcome, err := staging.Ingest(ctx, valid)
	require.NoError(t, err)
	require.Equal(t, "POSTED", outcome.Status)
	require.Equal(t, "itmis-PAY-1", outcome.EntryID)

	balances, err := journal.ListBalances(ctx, treasury.ListBalancesFilter{
		InstitutionID: "minfin", AccountCode: "6202",
		Pagination: treasury.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	require.Equal(t, int64(-4000), balances.Balances[0].BalanceMinor)

	// 2. Redelivery of the same hash is a no-op.
	redelivered, err := staging.Ingest(ctx, valid)
	require.NoError(t, err)
	require.Equal(t, "DUPLICATE", redelivered.Status)

	rebalanced, err := journal.ListBalances(ctx, treasury.ListBalancesFilter{
		InstitutionID: "minfin", AccountCode: "6202",
		Pagination: treasury.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	require.Equal(t, int64(-4000), rebalanced.Balances[0].BalanceMinor, "redelivery must not double-post")

	// 3. Unknown institution quarantines with a reason; no journal entry.
	badInstitution := valid
	badInstitution.SourceRef = "PAY-2"
	badInstitution.SourceHash = "hash-2"
	badInstitution.InstitutionID = "ghost-ministry"
	quarantined, err := staging.Ingest(ctx, badInstitution)
	require.NoError(t, err)
	require.Equal(t, "QUARANTINED", quarantined.Status)
	require.Contains(t, quarantined.Reason, "institution does not exist")

	// 4. Quarantined records are listable with their reasons.
	page, err := staging.ListStagingRecords(ctx, treasury.ListStagingRecordsFilter{
		Status:     "QUARANTINED",
		Pagination: treasury.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	require.Len(t, page.Records, 1)
	require.Equal(t, "PAY-2", page.Records[0].SourceRef)
	require.Empty(t, page.Records[0].EntryID)
}
