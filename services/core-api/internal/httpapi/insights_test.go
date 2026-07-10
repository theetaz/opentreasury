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

type stubInsightsRepository struct {
	flows        []treasury.MonthlyFlow
	observations []treasury.FlowObservation
	institution  string
}

func (s *stubInsightsRepository) MonthlyFlows(_ context.Context, institutionID string) ([]treasury.MonthlyFlow, error) {
	s.institution = institutionID
	return s.flows, nil
}

func (s *stubInsightsRepository) FlowObservations(_ context.Context, institutionID string) ([]treasury.FlowObservation, error) {
	s.institution = institutionID
	return s.observations, nil
}

func TestForecastEndpointReturnsHistoryAndProjection(t *testing.T) {
	repository := &stubInsightsRepository{flows: []treasury.MonthlyFlow{
		{Period: "2026-05", TotalMinor: 100},
		{Period: "2026-06", TotalMinor: 200},
		{Period: "2026-07", TotalMinor: 300},
	}}
	response := httptest.NewRecorder()
	NewRouter(WithInsightsRepository(repository)).ServeHTTP(response,
		httptest.NewRequest(http.MethodGet, "/v1/insights/forecast?institutionId=minfin&horizon=2", nil))

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "minfin", repository.institution)

	var body struct {
		History []struct {
			Period string `json:"period"`
		} `json:"history"`
		Forecast []struct {
			Period         string `json:"period"`
			ProjectedMinor int64  `json:"projectedMinor"`
			LowMinor       int64  `json:"lowMinor"`
			HighMinor      int64  `json:"highMinor"`
		} `json:"forecast"`
		Method string `json:"method"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	require.Len(t, body.History, 3)
	require.Len(t, body.Forecast, 2)
	require.Equal(t, "2026-08", body.Forecast[0].Period)
	require.NotEmpty(t, body.Method, "the method is disclosed — auditors must know how numbers were made")
}

func TestForecastEndpointRejectsBadHorizon(t *testing.T) {
	for _, horizon := range []string{"0", "13", "abc"} {
		response := httptest.NewRecorder()
		NewRouter(WithInsightsRepository(&stubInsightsRepository{})).ServeHTTP(response,
			httptest.NewRequest(http.MethodGet, "/v1/insights/forecast?horizon="+horizon, nil))
		require.Equal(t, http.StatusBadRequest, response.Code, "horizon %s", horizon)
	}
}

func TestAnomaliesEndpointScoresAndPaginates(t *testing.T) {
	observations := []treasury.FlowObservation{
		{EntryID: "n1", AccountCode: "22", AmountMinor: 100, InstitutionID: "minfin"},
		{EntryID: "n2", AccountCode: "22", AmountMinor: 101, InstitutionID: "minfin"},
		{EntryID: "n3", AccountCode: "22", AmountMinor: 99, InstitutionID: "minfin"},
		{EntryID: "n4", AccountCode: "22", AmountMinor: 102, InstitutionID: "minfin"},
		{EntryID: "n5", AccountCode: "22", AmountMinor: 100, InstitutionID: "minfin"},
		{EntryID: "spike", AccountCode: "22", AmountMinor: 90000, InstitutionID: "minfin", EffectiveDate: "2026-07-09"},
	}
	response := httptest.NewRecorder()
	NewRouter(WithInsightsRepository(&stubInsightsRepository{observations: observations})).ServeHTTP(response,
		httptest.NewRequest(http.MethodGet, "/v1/insights/anomalies?pageSize=15", nil))

	require.Equal(t, http.StatusOK, response.Code)
	var body struct {
		Anomalies []struct {
			EntryID      string  `json:"entryId"`
			Score        float64 `json:"score"`
			TypicalMinor int64   `json:"typicalMinor"`
		} `json:"anomalies"`
		Pagination paginationResponse `json:"pagination"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	require.Len(t, body.Anomalies, 1)
	require.Equal(t, "spike", body.Anomalies[0].EntryID)
	require.Equal(t, 1, body.Pagination.Total)
}

func TestInsightsEndpointsRequireRepository(t *testing.T) {
	for _, path := range []string{"/v1/insights/forecast", "/v1/insights/anomalies"} {
		response := httptest.NewRecorder()
		NewRouter().ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusServiceUnavailable, response.Code, path)
	}
}
