package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/opentreasury/opentreasury/services/core-api/internal/treasury"
)

type InsightsRepository interface {
	MonthlyFlows(ctx context.Context, institutionID string) ([]treasury.MonthlyFlow, error)
	FlowObservations(ctx context.Context, institutionID string) ([]treasury.FlowObservation, error)
}

func WithInsightsRepository(repository InsightsRepository) RouterOption {
	return func(config *routerConfig) {
		config.insightsRepository = repository
	}
}

const (
	defaultForecastHorizon = 3
	maxForecastHorizon     = 12
	// The disclosed method strings: numbers a government publishes must say
	// how they were made.
	forecastMethod = "3-month moving average with a ±1 MAD band widening by √distance"
	anomalyMethod  = "modified z-score per account (median/MAD), threshold 3.5, minimum 5 observations"
)

func (config routerConfig) forecastInsights(response http.ResponseWriter, request *http.Request) {
	if !config.authorized(response, request, request.URL.Query().Get("institutionId")) {
		return
	}
	if config.insightsRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "insights repository is not configured")
		return
	}

	query := request.URL.Query()
	horizon := defaultForecastHorizon
	if raw := query.Get("horizon"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 || value > maxForecastHorizon {
			writeError(response, http.StatusBadRequest, "horizon must be between 1 and 12")
			return
		}
		horizon = value
	}

	history, err := config.insightsRepository.MonthlyFlows(request.Context(), query.Get("institutionId"))
	if err != nil {
		config.internalError(response, request, err)
		return
	}

	forecast := treasury.ForecastFlows(history, horizon)

	historyRows := make([]monthlyFlowResponse, 0, len(history))
	for _, month := range history {
		historyRows = append(historyRows, monthlyFlowResponse{Period: month.Period, TotalMinor: month.TotalMinor})
	}
	forecastRows := make([]forecastPointResponse, 0, len(forecast))
	for _, point := range forecast {
		forecastRows = append(forecastRows, forecastPointResponse{
			Period:         point.Period,
			ProjectedMinor: point.ProjectedMinor,
			LowMinor:       point.LowMinor,
			HighMinor:      point.HighMinor,
		})
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(forecastResponse{
		History:  historyRows,
		Forecast: forecastRows,
		Method:   forecastMethod,
	})
}

func (config routerConfig) anomalyInsights(response http.ResponseWriter, request *http.Request) {
	if !config.authorized(response, request, request.URL.Query().Get("institutionId")) {
		return
	}
	if config.insightsRepository == nil {
		writeError(response, http.StatusServiceUnavailable, "insights repository is not configured")
		return
	}

	query := request.URL.Query()
	pagination, err := parsePagination(query)
	if err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}

	observations, err := config.insightsRepository.FlowObservations(request.Context(), query.Get("institutionId"))
	if err != nil {
		config.internalError(response, request, err)
		return
	}

	anomalies := treasury.ScoreAnomalies(observations)
	total := len(anomalies)

	start := (pagination.Page - 1) * pagination.PageSize
	if start > total {
		start = total
	}
	end := start + pagination.PageSize
	if end > total {
		end = total
	}

	rows := make([]anomalyResponse, 0, end-start)
	for _, anomaly := range anomalies[start:end] {
		rows = append(rows, anomalyResponse{
			EntryID:       anomaly.EntryID,
			InstitutionID: anomaly.InstitutionID,
			AccountCode:   anomaly.AccountCode,
			AmountMinor:   anomaly.AmountMinor,
			EffectiveDate: anomaly.EffectiveDate,
			TypicalMinor:  anomaly.TypicalMinor,
			Score:         anomaly.Score,
		})
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(anomaliesResponse{
		Anomalies:  rows,
		Method:     anomalyMethod,
		Pagination: toPaginationResponse(pagination, total),
	})
}

type monthlyFlowResponse struct {
	Period     string `json:"period"`
	TotalMinor int64  `json:"totalMinor"`
}

type forecastPointResponse struct {
	Period         string `json:"period"`
	ProjectedMinor int64  `json:"projectedMinor"`
	LowMinor       int64  `json:"lowMinor"`
	HighMinor      int64  `json:"highMinor"`
}

type forecastResponse struct {
	History  []monthlyFlowResponse   `json:"history"`
	Forecast []forecastPointResponse `json:"forecast"`
	Method   string                  `json:"method"`
}

type anomalyResponse struct {
	EntryID       string  `json:"entryId"`
	InstitutionID string  `json:"institutionId"`
	AccountCode   string  `json:"accountCode"`
	AmountMinor   int64   `json:"amountMinor"`
	EffectiveDate string  `json:"effectiveDate"`
	TypicalMinor  int64   `json:"typicalMinor"`
	Score         float64 `json:"score"`
}

type anomaliesResponse struct {
	Anomalies  []anomalyResponse  `json:"anomalies"`
	Method     string             `json:"method"`
	Pagination paginationResponse `json:"pagination"`
}
