package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
	"github.com/stretchr/testify/require"
)

type recordingReconciliationRepository struct {
	filter treasury.ListReconciliationFilter
	page   treasury.ReconciliationPage
}

func (r *recordingReconciliationRepository) ListReconciliation(_ context.Context, filter treasury.ListReconciliationFilter) (treasury.ReconciliationPage, error) {
	r.filter = filter
	return r.page, nil
}

func TestReconciliationEndpointReturnsPeriodRows(t *testing.T) {
	repository := &recordingReconciliationRepository{
		page: treasury.ReconciliationPage{
			Total: 1,
			Rows: []treasury.ReconciliationRow{{
				SourceSystem:      "itmis",
				Period:            "2026-07",
				StagedCount:       6,
				PostedCount:       5,
				QuarantinedCount:  1,
				StagedAmountMinor: 500000,
				PostedAmountMinor: 500000,
			}},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/reconciliation?sourceSystem=itmis&fiscalYear=2026&pageSize=25", nil)
	response := httptest.NewRecorder()

	NewRouter(WithReconciliationRepository(repository)).ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, treasury.ListReconciliationFilter{
		SourceSystem: "itmis",
		FiscalYear:   2026,
		Pagination:   treasury.Pagination{Page: 1, PageSize: 25},
	}, repository.filter)

	var body struct {
		Rows []struct {
			SourceSystem      string `json:"sourceSystem"`
			Period            string `json:"period"`
			StagedCount       int    `json:"stagedCount"`
			PostedCount       int    `json:"postedCount"`
			QuarantinedCount  int    `json:"quarantinedCount"`
			StagedAmountMinor int64  `json:"stagedAmountMinor"`
			PostedAmountMinor int64  `json:"postedAmountMinor"`
			DiscrepancyMinor  int64  `json:"discrepancyMinor"`
			Status            string `json:"status"`
		} `json:"rows"`
		Pagination paginationResponse `json:"pagination"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	require.Len(t, body.Rows, 1)
	require.Equal(t, "itmis", body.Rows[0].SourceSystem)
	require.Equal(t, "2026-07", body.Rows[0].Period)
	require.Equal(t, int64(0), body.Rows[0].DiscrepancyMinor)
	// Amounts match but a quarantined record needs human attention.
	require.Equal(t, "ATTENTION", body.Rows[0].Status)
	require.Equal(t, 1, body.Pagination.Total)
}

func TestReconciliationStatusIsMatchedWhenCleanAndDiscrepantOnMismatch(t *testing.T) {
	repository := &recordingReconciliationRepository{
		page: treasury.ReconciliationPage{
			Total: 2,
			Rows: []treasury.ReconciliationRow{
				{SourceSystem: "itmis", Period: "2026-06", StagedCount: 3, PostedCount: 3,
					StagedAmountMinor: 100, PostedAmountMinor: 100},
				{SourceSystem: "itmis", Period: "2026-05", StagedCount: 2, PostedCount: 2,
					StagedAmountMinor: 300, PostedAmountMinor: 200},
			},
		},
	}
	response := httptest.NewRecorder()
	NewRouter(WithReconciliationRepository(repository)).
		ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/reconciliation", nil))

	var body struct {
		Rows []struct {
			DiscrepancyMinor int64  `json:"discrepancyMinor"`
			Status           string `json:"status"`
		} `json:"rows"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	require.Equal(t, "MATCHED", body.Rows[0].Status)
	require.Equal(t, "DISCREPANCY", body.Rows[1].Status)
	require.Equal(t, int64(100), body.Rows[1].DiscrepancyMinor)
}

func TestReconciliationEndpointRejectsInvalidFiscalYear(t *testing.T) {
	response := httptest.NewRecorder()
	NewRouter(WithReconciliationRepository(&recordingReconciliationRepository{})).
		ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/reconciliation?fiscalYear=zero", nil))
	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestReconciliationEndpointRequiresRepository(t *testing.T) {
	response := httptest.NewRecorder()
	NewRouter().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/reconciliation", nil))
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
}
