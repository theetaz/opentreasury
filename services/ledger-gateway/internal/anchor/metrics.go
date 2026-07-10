package anchor

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics exposes the gateway's anchoring throughput and failures for
// Prometheus. The gateway serves them on its own listener so the anchoring
// loop stays a plain worker with no API surface.
type Metrics struct {
	registry             *prometheus.Registry
	EntriesAnchoredTotal prometheus.Counter
	BatchesTotal         prometheus.Counter
	FailuresTotal        prometheus.Counter
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	metrics := &Metrics{
		registry: registry,
		EntriesAnchoredTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "opentreasury_anchor_entries_total",
			Help: "Journal entries anchored into the backend ledger.",
		}),
		BatchesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "opentreasury_anchor_batches_total",
			Help: "Anchor batches committed (one Merkle root each).",
		}),
		FailuresTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "opentreasury_anchor_failures_total",
			Help: "Anchoring attempts that failed.",
		}),
	}
	registry.MustRegister(metrics.EntriesAnchoredTotal, metrics.BatchesTotal, metrics.FailuresTotal)
	return metrics
}

// RecordBatch records one committed batch of anchored entries.
func (m *Metrics) RecordBatch(entries int) {
	m.BatchesTotal.Inc()
	m.EntriesAnchoredTotal.Add(float64(entries))
}

func (m *Metrics) RecordFailure() {
	m.FailuresTotal.Inc()
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
