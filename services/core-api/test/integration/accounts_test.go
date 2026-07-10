//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

func TestReferenceChartOfAccountsLoadsAndFilters(t *testing.T) {
	dsn := startPostgres(t)
	migrator := newMigrator(t, dsn)
	db := openDB(t, dsn)

	require.NoError(t, migrator.Up())
	applySeeds(t, db)

	repository := treasury.NewPostgresAccountRepository(db)

	all, err := repository.ListAccounts(context.Background(), treasury.ListAccountsFilter{
		Pagination: treasury.Pagination{Page: 1, PageSize: 100},
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, all.Total, 20, "reference chart of accounts must seed")
	require.Equal(t, "1", all.Accounts[0].Code, "accounts are ordered by code")

	liabilities, err := repository.ListAccounts(context.Background(), treasury.ListAccountsFilter{
		AccountType: "LIABILITY",
		Pagination:  treasury.Pagination{Page: 1, PageSize: 100},
	})
	require.NoError(t, err)
	require.NotEmpty(t, liabilities.Accounts)
	for _, account := range liabilities.Accounts {
		require.Equal(t, "LIABILITY", account.AccountType)
	}

	// Referential integrity: every parent code exists in the same version.
	var orphans int
	require.NoError(t, db.QueryRow(`
		SELECT COUNT(*) FROM coa_accounts child
		WHERE parent_code IS NOT NULL
		  AND NOT EXISTS (
		    SELECT 1 FROM coa_accounts parent
		    WHERE parent.version = child.version AND parent.code = child.parent_code
		  )
	`).Scan(&orphans))
	require.Zero(t, orphans)
}
