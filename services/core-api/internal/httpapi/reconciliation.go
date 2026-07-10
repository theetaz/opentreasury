package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type ReconciliationRepository interface {
	ListReconciliation(context.Context, treasury.ListReconciliationFilter) (treasury.ReconciliationPage, error)
}

// WithReconciliationRepository serves GET /v1/reconciliation: per source
// system and month, staged-vs-posted counts and debit totals.
func WithReconciliationRepository(repository ReconciliationRepository) RouterOption {
	return func(config *routerConfig) {
		config.reconciliationRepository = repository
	}
}

func (config routerConfig) listReconciliation(response http.ResponseWriter, request *http.Request) {
	if !config.authorized(response, request, "") {
		return
	}
	if config.reconciliationRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "reconciliation repository is not configured")
		return
	}

	query := request.URL.Query()
	filter := treasury.ListReconciliationFilter{
		SourceSystem: query.Get("sourceSystem"),
	}

	if fiscalYear := query.Get("fiscalYear"); fiscalYear != "" {
		value, err := strconv.Atoi(fiscalYear)
		if err != nil || value <= 0 {
			writeError(response, http.StatusBadRequest, treasury.ErrInvalidFiscalYear.Error())
			return
		}
		filter.FiscalYear = value
	}

	pagination, err := parsePagination(query)
	if err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}
	filter.Pagination = pagination

	page, err := config.reconciliationRepository.ListReconciliation(request.Context(), filter)
	if err != nil {
		config.internalError(response, request, err)
		return
	}

	rows := make([]reconciliationRowResponse, 0, len(page.Rows))
	for _, row := range page.Rows {
		rows = append(rows, toReconciliationRowResponse(row))
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(listReconciliationResponse{
		Rows:       rows,
		Pagination: toPaginationResponse(filter.Pagination, page.Total),
	})
}

type listReconciliationResponse struct {
	Rows       []reconciliationRowResponse `json:"rows"`
	Pagination paginationResponse          `json:"pagination"`
}

type reconciliationRowResponse struct {
	SourceSystem      string `json:"sourceSystem"`
	Period            string `json:"period"`
	StagedCount       int    `json:"stagedCount"`
	PostedCount       int    `json:"postedCount"`
	QuarantinedCount  int    `json:"quarantinedCount"`
	StagedAmountMinor int64  `json:"stagedAmountMinor"`
	PostedAmountMinor int64  `json:"postedAmountMinor"`
	DiscrepancyMinor  int64  `json:"discrepancyMinor"`
	Status            string `json:"status"`
}

// toReconciliationRowResponse derives the operator-facing verdict: amounts
// must match AND nothing may be quarantined for a period to read MATCHED.
func toReconciliationRowResponse(row treasury.ReconciliationRow) reconciliationRowResponse {
	discrepancy := row.StagedAmountMinor - row.PostedAmountMinor
	status := "MATCHED"
	if discrepancy != 0 {
		status = "DISCREPANCY"
	} else if row.QuarantinedCount > 0 {
		status = "ATTENTION"
	}

	return reconciliationRowResponse{
		SourceSystem:      row.SourceSystem,
		Period:            row.Period,
		StagedCount:       row.StagedCount,
		PostedCount:       row.PostedCount,
		QuarantinedCount:  row.QuarantinedCount,
		StagedAmountMinor: row.StagedAmountMinor,
		PostedAmountMinor: row.PostedAmountMinor,
		DiscrepancyMinor:  discrepancy,
		Status:            status,
	}
}
