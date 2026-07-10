//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

func TestReconciliationAggregatesStagedAgainstPosted(t *testing.T) {
	dsn := startPostgres(t)
	migrator := newMigrator(t, dsn)
	db := openDB(t, dsn)

	require.NoError(t, migrator.Up())
	applySeeds(t, db)

	journal := treasury.NewPostgresJournalRepository(db)
	staging := treasury.NewPostgresStagingRepository(db, journal)
	ctx := treasury.ContextWithAuditMetadata(context.Background(), treasury.AuditMetadata{Actor: "connector", RequestID: "req"})

	record := func(ref, hash, institution, date string, amount int64) treasury.StagingRecord {
		return treasury.StagingRecord{
			SourceSystem: "itmis", SourceRef: ref, SourceHash: hash, Profile: "itmis@1",
			InstitutionID: institution, OccurredAt: date,
			Lines: []treasury.JournalLine{
				{AccountCode: "22", Direction: "DEBIT", AmountMinor: amount, Currency: "USD"},
				{AccountCode: "6202", Direction: "CREDIT", AmountMinor: amount, Currency: "USD"},
			},
		}
	}

	// July: two posted records and one quarantined (unknown institution).
	for _, r := range []treasury.StagingRecord{
		record("PAY-1", "h1", "minfin", "2026-07-03", 10000),
		record("PAY-2", "h2", "minfin", "2026-07-20", 2500),
		record("PAY-3", "h3", "nope", "2026-07-21", 999),
	} {
		_, err := staging.Ingest(ctx, r)
		require.NoError(t, err)
	}
	// June: one clean posted record.
	_, err := staging.Ingest(ctx, record("PAY-0", "h0", "minfin", "2026-06-15", 7777))
	require.NoError(t, err)

	reconciliation := treasury.NewPostgresReconciliationRepository(db)
	page, err := reconciliation.ListReconciliation(ctx, treasury.ListReconciliationFilter{
		SourceSystem: "itmis",
		FiscalYear:   2026,
		Pagination:   treasury.Pagination{Page: 1, PageSize: 15},
	})
	require.NoError(t, err)
	require.Equal(t, 2, page.Total, "two periods: 2026-06 and 2026-07")
	require.Len(t, page.Rows, 2)

	july := page.Rows[0] // ordered period DESC
	require.Equal(t, "2026-07", july.Period)
	require.Equal(t, 3, july.StagedCount)
	require.Equal(t, 2, july.PostedCount)
	require.Equal(t, 1, july.QuarantinedCount)
	require.Equal(t, int64(12500), july.StagedAmountMinor, "quarantined amounts are excluded")
	require.Equal(t, int64(12500), july.PostedAmountMinor, "posted journal debits equal staged debits")

	june := page.Rows[1]
	require.Equal(t, "2026-06", june.Period)
	require.Equal(t, 1, june.StagedCount)
	require.Zero(t, june.QuarantinedCount)
	require.Equal(t, int64(7777), june.StagedAmountMinor)
	require.Equal(t, int64(7777), june.PostedAmountMinor)
}
