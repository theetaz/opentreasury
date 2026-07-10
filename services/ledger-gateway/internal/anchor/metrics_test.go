package anchor

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestMetricsCountAnchoredEntriesAndBatches(t *testing.T) {
	metrics := NewMetrics()

	metrics.RecordBatch(3)
	metrics.RecordBatch(2)
	metrics.RecordFailure()

	require.Equal(t, float64(5), testutil.ToFloat64(metrics.EntriesAnchoredTotal))
	require.Equal(t, float64(2), testutil.ToFloat64(metrics.BatchesTotal))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.FailuresTotal))
}

func TestMetricsHandlerServesPrometheusFormat(t *testing.T) {
	metrics := NewMetrics()
	metrics.RecordBatch(4)

	response := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	require.Equal(t, http.StatusOK, response.Code)
	body := response.Body.String()
	require.Contains(t, body, "opentreasury_anchor_entries_total 4")
	require.Contains(t, body, "opentreasury_anchor_batches_total 1")
	require.Contains(t, body, "go_goroutines")
}

func TestMetricsTrackAnchorLagAndLastSuccess(t *testing.T) {
	metrics := NewMetrics()

	metrics.RecordLag(7)
	require.Equal(t, float64(7), testutil.ToFloat64(metrics.AnchorLag))

	metrics.RecordLag(0)
	require.Equal(t, float64(0), testutil.ToFloat64(metrics.AnchorLag))

	require.Equal(t, float64(0), testutil.ToFloat64(metrics.LastSuccessTimestamp),
		"no success recorded yet")
	metrics.RecordBatch(1)
	require.Greater(t, testutil.ToFloat64(metrics.LastSuccessTimestamp), float64(0),
		"a committed batch stamps the last-success time")
}
