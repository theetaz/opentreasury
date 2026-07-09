package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type capturingRepository struct {
	txFilter    *treasury.ListTransactionsFilter
	auditFilter *treasury.ListAuditEventsFilter
	instFilter  *treasury.ListInstitutionsFilter
}

func (repo *capturingRepository) Save(context.Context, treasury.Transaction) error { return nil }

func (repo *capturingRepository) List(_ context.Context, filter treasury.ListTransactionsFilter) (treasury.TransactionPage, error) {
	repo.txFilter = &filter
	return treasury.TransactionPage{Transactions: []treasury.Transaction{}, Total: 42}, nil
}

func (repo *capturingRepository) ListAuditEvents(_ context.Context, filter treasury.ListAuditEventsFilter) (treasury.AuditEventPage, error) {
	repo.auditFilter = &filter
	return treasury.AuditEventPage{Events: []treasury.AuditEvent{}, Total: 7}, nil
}

func (repo *capturingRepository) ListInstitutions(_ context.Context, filter treasury.ListInstitutionsFilter) (treasury.InstitutionPage, error) {
	repo.instFilter = &filter
	return treasury.InstitutionPage{Institutions: []treasury.Institution{}, Total: 3}, nil
}

func doGet(t *testing.T, repo *capturingRepository, path string) *httptest.ResponseRecorder {
	t.Helper()

	router := NewRouter(
		WithTransactionRepository(repo),
		WithInstitutionRepository(repo),
	)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

	return recorder
}

func TestListTransactionsParsesTableQueryParameters(t *testing.T) {
	repo := &capturingRepository{}
	recorder := doGet(t, repo,
		"/v1/transactions?institutionId=minfin&fiscalYear=2026&dateFrom=2026-01-01&dateTo=2026-12-31&amountGte=100000&amountLte=200000&page=2&pageSize=10")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, repo.txFilter)
	require.Equal(t, "minfin", repo.txFilter.InstitutionID)
	require.Equal(t, 2026, repo.txFilter.FiscalYear)
	require.Equal(t, "2026-01-01", repo.txFilter.DateFrom)
	require.Equal(t, "2026-12-31", repo.txFilter.DateTo)
	require.Equal(t, int64(100000), repo.txFilter.AmountMinorGte)
	require.Equal(t, int64(200000), repo.txFilter.AmountMinorLte)
	require.Equal(t, 2, repo.txFilter.Page)
	require.Equal(t, 10, repo.txFilter.PageSize)
}

func TestListResponsesCarryPaginationEnvelope(t *testing.T) {
	repo := &capturingRepository{}
	recorder := doGet(t, repo, "/v1/transactions?page=2&pageSize=10")

	var body struct {
		Transactions []any `json:"transactions"`
		Pagination   struct {
			Page     int `json:"page"`
			PageSize int `json:"pageSize"`
			Total    int `json:"total"`
		} `json:"pagination"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, 2, body.Pagination.Page)
	require.Equal(t, 10, body.Pagination.PageSize)
	require.Equal(t, 42, body.Pagination.Total)
}

func TestLimitRemainsAsPageSizeAlias(t *testing.T) {
	repo := &capturingRepository{}
	recorder := doGet(t, repo, "/v1/transactions?limit=5")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, 5, repo.txFilter.PageSize)
	require.Equal(t, 1, repo.txFilter.Page)
}

func TestInvalidTableParametersAreRejected(t *testing.T) {
	cases := map[string]string{
		"/v1/transactions?dateFrom=junk":                         "invalid date range",
		"/v1/transactions?dateFrom=2026-12-31&dateTo=2026-01-01": "invalid date range",
		"/v1/transactions?amountGte=abc":                         "invalid amount filter",
		"/v1/transactions?amountGte=-5":                          "invalid amount filter",
		"/v1/transactions?page=0":                                "invalid page",
		"/v1/transactions?page=abc":                              "invalid page",
		"/v1/transactions?pageSize=101":                          "invalid page size",
		"/v1/institutions?status=BOGUS":                          "invalid status",
	}

	for path, wantError := range cases {
		repo := &capturingRepository{}
		recorder := doGet(t, repo, path)

		require.Equalf(t, http.StatusBadRequest, recorder.Code, "path %s", path)
		require.JSONEqf(t, `{"error":"`+wantError+`"}`, recorder.Body.String(), "path %s", path)
	}
}

func TestInstitutionStatusFilterIsForwarded(t *testing.T) {
	repo := &capturingRepository{}
	recorder := doGet(t, repo, "/v1/institutions?status=ACTIVE&page=1&pageSize=50")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "ACTIVE", repo.instFilter.Status)
	require.Equal(t, 50, repo.instFilter.PageSize)
}

func TestAuditEventDateRangeIsForwarded(t *testing.T) {
	repo := &capturingRepository{}
	recorder := doGet(t, repo, "/v1/audit-events?dateFrom=2026-06-01&dateTo=2026-06-30")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "2026-06-01", repo.auditFilter.DateFrom)
	require.Equal(t, "2026-06-30", repo.auditFilter.DateTo)
}
