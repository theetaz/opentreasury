package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetricsEndpointExposesRequestMetrics(t *testing.T) {
	router := NewRouter()

	// Generate one observed request first.
	health := httptest.NewRecorder()
	router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, health.Code)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	require.Equal(t, http.StatusOK, response.Code)
	body := response.Body.String()
	require.Contains(t, body, `opentreasury_http_requests_total{method="GET",route="GET /healthz",status="200"} 1`,
		"request counter must be labeled by matched route pattern and status")
	require.Contains(t, body, "opentreasury_http_request_duration_seconds_bucket",
		"request latency histogram must be exposed")
	require.Contains(t, body, "go_goroutines", "runtime collectors must be registered")
}

func TestMetricsRouteLabelUsesPatternNotRawPath(t *testing.T) {
	router := NewRouter()

	// A parameterized route must not explode label cardinality: the raw id
	// must never appear as a label value.
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/entries/je-secret-id/proof", nil))

	metrics := httptest.NewRecorder()
	router.ServeHTTP(metrics, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	body := metrics.Body.String()
	require.Contains(t, body, `route="GET /v1/entries/{id}/proof"`)
	require.NotContains(t, body, "je-secret-id")
}

func TestMetricsUnmatchedPathsCollapseToOneLabel(t *testing.T) {
	router := NewRouter()

	for _, path := range []string{"/no-such-route", "/another-miss"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusNotFound, response.Code)
	}

	metrics := httptest.NewRecorder()
	router.ServeHTTP(metrics, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	body := metrics.Body.String()
	require.Contains(t, body, `opentreasury_http_requests_total{method="GET",route="unmatched",status="404"} 2`,
		"unknown paths must share a single label value, never the raw path")
	require.NotContains(t, body, "/no-such-route")
}

func TestMetricsPathIsPublic(t *testing.T) {
	require.True(t, isPublicPath(httptest.NewRequest(http.MethodGet, "/metrics", nil)),
		"/metrics must be scrapeable without a bearer token")
}
