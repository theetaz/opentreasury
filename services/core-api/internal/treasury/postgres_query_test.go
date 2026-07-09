package treasury

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func transactionColumns() []string {
	return []string{"id", "institution_id", "fiscal_year", "amount_minor", "currency", "description", "transaction_date", "total_count"}
}

func TestListTransactionsAppliesDateRangeAndAmountFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows(transactionColumns()).
		AddRow("txn-1", "minfin", 2026, int64(125000), "USD", "Road maintenance", "2026-06-28", 1)

	mock.ExpectQuery(regexp.QuoteMeta("WHERE institution_id = $1 AND transaction_date >= $2 AND transaction_date <= $3 AND amount_minor >= $4 AND amount_minor <= $5")).
		WithArgs("minfin", "2026-01-01", "2026-12-31", int64(100000), int64(200000), 25, 0).
		WillReturnRows(rows)

	page, err := NewPostgresTransactionRepository(db).List(context.Background(), ListTransactionsFilter{
		InstitutionID:  "minfin",
		DateFrom:       "2026-01-01",
		DateTo:         "2026-12-31",
		AmountMinorGte: 100000,
		AmountMinorLte: 200000,
		Pagination:     Pagination{Page: 1, PageSize: 25},
	})

	require.NoError(t, err)
	require.Len(t, page.Transactions, 1)
	require.Equal(t, 1, page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListTransactionsPaginatesWithOffsetAndReportsTotal(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows(transactionColumns()).
		AddRow("txn-26", "minfin", 2026, int64(1000), "USD", "Row 26", "2026-06-01", 51)

	mock.ExpectQuery(regexp.QuoteMeta("LIMIT $1 OFFSET $2")).
		WithArgs(25, 25).
		WillReturnRows(rows)

	page, err := NewPostgresTransactionRepository(db).List(context.Background(), ListTransactionsFilter{
		Pagination: Pagination{Page: 2, PageSize: 25},
	})

	require.NoError(t, err)
	require.Equal(t, 51, page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListInstitutionsFiltersByStatusAndPaginates(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"id", "name", "type", "country_code", "status", "total_count"}).
		AddRow("minfin", "Ministry of Finance", "MINISTRY", "KE", "ACTIVE", 3)

	mock.ExpectQuery(regexp.QuoteMeta("WHERE status = $1")).
		WithArgs("ACTIVE", 25, 0).
		WillReturnRows(rows)

	page, err := NewPostgresInstitutionRepository(db).ListInstitutions(context.Background(), ListInstitutionsFilter{
		Status:     "ACTIVE",
		Pagination: Pagination{Page: 1, PageSize: 25},
	})

	require.NoError(t, err)
	require.Len(t, page.Institutions, 1)
	require.Equal(t, 3, page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListAuditEventsFiltersByOccurredRange(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"id", "event_type", "transaction_id", "institution_id", "occurred_at", "summary", "total_count"}).
		AddRow("audit-txn-1-created", "TRANSACTION_CREATED", "txn-1", "minfin", "2026-06-28T10:00:00Z", "Transaction txn-1 was created.", 1)

	mock.ExpectQuery(regexp.QuoteMeta("created_at >= $")).
		WithArgs("2026-06-01", "2026-06-30", 25, 0).
		WillReturnRows(rows)

	page, err := NewPostgresTransactionRepository(db).ListAuditEvents(context.Background(), ListAuditEventsFilter{
		DateFrom:   "2026-06-01",
		DateTo:     "2026-06-30",
		Pagination: Pagination{Page: 1, PageSize: 25},
	})

	require.NoError(t, err)
	require.Len(t, page.Events, 1)
	require.Equal(t, 1, page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}
