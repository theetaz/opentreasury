package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// requestMetrics instruments HTTP traffic for Prometheus. Each router owns its
// own registry so tests (and multiple routers in one process) never collide on
// duplicate collector registration.
type requestMetrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func newRequestMetrics() *requestMetrics {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "opentreasury_http_requests_total",
		Help: "HTTP requests served, by matched route pattern and status code.",
	}, []string{"method", "route", "status"})
	registry.MustRegister(requests)

	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "opentreasury_http_request_duration_seconds",
		Help:    "HTTP request latency, by matched route pattern.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	}, []string{"method", "route"})
	registry.MustRegister(duration)

	return &requestMetrics{registry: registry, requests: requests, duration: duration}
}

func (m *requestMetrics) handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// withMetrics records a counter and latency observation per request. The
// route label is the mux's matched pattern (e.g. "GET /v1/entries/{id}/proof"),
// never the raw path — raw paths carry identifiers and would explode label
// cardinality.
func withMetrics(metrics *requestMetrics, mux *http.ServeMux, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		route := "unmatched"
		if _, pattern := mux.Handler(request); pattern != "" {
			route = pattern
		}

		recorder := &statusRecorder{ResponseWriter: response, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(recorder, request)

		metrics.requests.WithLabelValues(request.Method, route, strconv.Itoa(recorder.status)).Inc()
		metrics.duration.WithLabelValues(request.Method, route).Observe(time.Since(start).Seconds())
	})
}
