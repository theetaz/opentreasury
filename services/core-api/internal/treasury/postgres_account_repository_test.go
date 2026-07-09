package treasury

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func accountColumns() []string {
	return []string{"code", "name", "account_type", "parent_code", "gfsm_code", "cofog_code", "active", "total_count"}
}

func TestListAccountsReadsTheActiveChartVersion(t *testing.T) {
	db, mock := newMockDB(t)

	rows := sqlmock.NewRows(accountColumns()).
		AddRow("1", "Revenue", "REVENUE", "", "1", "", true, 2).
		AddRow("11", "Taxes", "REVENUE", "1", "11", "", true, 2)

	mock.ExpectQuery(regexp.QuoteMeta("WHERE c.status = 'ACTIVE'")).
		WithArgs(25, 0).
		WillReturnRows(rows)

	page, err := NewPostgresAccountRepository(db).ListAccounts(context.Background(), ListAccountsFilter{
		Pagination: Pagination{Page: 1, PageSize: 25},
	})

	require.NoError(t, err)
	require.Len(t, page.Accounts, 2)
	require.Equal(t, 2, page.Total)
	require.Equal(t, "1", page.Accounts[0].Code)
	require.Equal(t, "Taxes", page.Accounts[1].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListAccountsFiltersByAccountType(t *testing.T) {
	db, mock := newMockDB(t)

	rows := sqlmock.NewRows(accountColumns()).
		AddRow("63", "Liabilities", "LIABILITY", "", "63", "", true, 1)

	mock.ExpectQuery(regexp.QuoteMeta("AND a.account_type = $1")).
		WithArgs("LIABILITY", 25, 0).
		WillReturnRows(rows)

	page, err := NewPostgresAccountRepository(db).ListAccounts(context.Background(), ListAccountsFilter{
		AccountType: "LIABILITY",
		Pagination:  Pagination{Page: 1, PageSize: 25},
	})

	require.NoError(t, err)
	require.Len(t, page.Accounts, 1)
	require.Equal(t, "LIABILITY", page.Accounts[0].AccountType)
	require.NoError(t, mock.ExpectationsWereMet())
}
