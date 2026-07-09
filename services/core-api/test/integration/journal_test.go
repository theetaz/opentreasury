//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

func TestPostingEntriesKeepsBalancesConsistentWithFlows(t *testing.T) {
	dsn := startPostgres(t)
	migrator := newMigrator(t, dsn)
	db := openDB(t, dsn)

	require.NoError(t, migrator.Up())
	applySeeds(t, db)

	repository := treasury.NewPostgresJournalRepository(db)
	ctx := treasury.ContextWithAuditMetadata(context.Background(), treasury.AuditMetadata{Actor: "system", RequestID: "req-je"})

	// Post: DEBIT 6202 (cash) / CREDIT 114 (tax revenue), 1,250.00 USD.
	require.NoError(t, repository.PostEntry(ctx, treasury.JournalEntry{
		ID:            "je-int-1",
		InstitutionID: "minfin",
		FiscalYear:    2026,
		EffectiveDate: "2026-06-28",
		Description:   "Tax receipt",
		Lines: []treasury.JournalLine{
			{AccountCode: "6202", Direction: "DEBIT", AmountMinor: 125000, Currency: "USD"},
			{AccountCode: "114", Direction: "CREDIT", AmountMinor: 125000, Currency: "USD"},
		},
	}))

	// Post a second entry: spend 40.00 from cash on goods (DEBIT 22 / CREDIT 6202).
	require.NoError(t, repository.PostEntry(ctx, treasury.JournalEntry{
		ID:            "je-int-2",
		InstitutionID: "minfin",
		FiscalYear:    2026,
		EffectiveDate: "2026-06-29",
		Description:   "Office supplies",
		Lines: []treasury.JournalLine{
			{AccountCode: "22", Direction: "DEBIT", AmountMinor: 4000, Currency: "USD"},
			{AccountCode: "6202", Direction: "CREDIT", AmountMinor: 4000, Currency: "USD"},
		},
	}))

	// Cash (6202) net = 125000 debit - 4000 credit = 121000.
	balances, err := repository.ListBalances(context.Background(), treasury.ListBalancesFilter{
		InstitutionID: "minfin",
		AccountCode:   "6202",
		Pagination:    treasury.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	require.Len(t, balances.Balances, 1)
	require.Equal(t, int64(121000), balances.Balances[0].BalanceMinor)
	require.Equal(t, "Currency and deposits", balances.Balances[0].AccountName)

	// The invariant: for every (institution, account, currency), the stored
	// balance equals the signed sum of its posted lines.
	rows, err := db.Query(`
		SELECT b.institution_id, b.account_code, b.currency, b.balance_minor,
		       COALESCE(SUM(CASE l.direction WHEN 'DEBIT' THEN l.amount_minor ELSE -l.amount_minor END), 0) AS flow_sum
		FROM account_balances b
		LEFT JOIN journal_lines l ON l.account_code = b.account_code AND l.currency = b.currency
		LEFT JOIN journal_entries e ON e.id = l.entry_id AND e.institution_id = b.institution_id
		GROUP BY b.institution_id, b.account_code, b.currency, b.balance_minor
	`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	checked := 0
	for rows.Next() {
		var inst, code, currency string
		var stored, flowSum int64
		require.NoError(t, rows.Scan(&inst, &code, &currency, &stored, &flowSum))
		require.Equalf(t, flowSum, stored, "stock != flows for %s/%s/%s", inst, code, currency)
		checked++
	}
	require.NoError(t, rows.Err())
	require.GreaterOrEqual(t, checked, 3, "should have checked cash, revenue, and expense accounts")

	// A duplicate entry id is rejected (append-only integrity).
	err = repository.PostEntry(ctx, treasury.JournalEntry{
		ID:            "je-int-1",
		InstitutionID: "minfin",
		FiscalYear:    2026,
		EffectiveDate: "2026-06-28",
		Description:   "Duplicate",
		Lines: []treasury.JournalLine{
			{AccountCode: "6202", Direction: "DEBIT", AmountMinor: 100, Currency: "USD"},
			{AccountCode: "114", Direction: "CREDIT", AmountMinor: 100, Currency: "USD"},
		},
	})
	require.ErrorIs(t, err, treasury.ErrDuplicateTransaction)
}

func TestReversalRestoresBalances(t *testing.T) {
	dsn := startPostgres(t)
	migrator := newMigrator(t, dsn)
	db := openDB(t, dsn)

	require.NoError(t, migrator.Up())
	applySeeds(t, db)

	repository := treasury.NewPostgresJournalRepository(db)
	ctx := context.Background()

	original := treasury.JournalEntry{
		ID:            "je-rev-orig",
		InstitutionID: "minfin",
		FiscalYear:    2026,
		EffectiveDate: "2026-06-28",
		Description:   "Original",
		Lines: []treasury.JournalLine{
			{AccountCode: "6202", Direction: "DEBIT", AmountMinor: 5000, Currency: "USD"},
			{AccountCode: "114", Direction: "CREDIT", AmountMinor: 5000, Currency: "USD"},
		},
	}
	require.NoError(t, repository.PostEntry(ctx, original))
	require.NoError(t, repository.PostEntry(ctx, original.Reversal("je-rev-rev", "2026-07-01")))

	balances, err := repository.ListBalances(ctx, treasury.ListBalancesFilter{
		InstitutionID: "minfin",
		AccountCode:   "6202",
		Pagination:    treasury.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	// Post + reversal nets to zero.
	require.Equal(t, int64(0), balances.Balances[0].BalanceMinor)
}
