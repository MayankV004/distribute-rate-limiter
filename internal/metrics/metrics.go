package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// DecisionsTotal tracks whether requests were allowed, denied, or handled by fallback.
	DecisionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ratelimit_decisions_total",
			Help: "Total rate limit decisions made",
		},
		[]string{"route", "tier", "algorithm", "decision"},
	)

	// StoreLatency tracks the latency of operations against the state store (Redis/Local).
	StoreLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ratelimit_store_latency_seconds",
			Help:    "Latency of store operations",
			Buckets: []float64{0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1},
		},
		[]string{"algorithm"},
	)

	// StoreErrorsTotal tracks errors returned by the state store.
	StoreErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ratelimit_store_errors_total",
			Help: "Total store errors",
		},
		[]string{"kind"},
	)

	// FallbackTotal tracks fallback actions taken when the store is unavailable.
	FallbackTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ratelimit_fallback_total",
			Help: "Total fallback actions taken",
		},
		[]string{"route", "policy"},
	)

	// UpstreamLatency tracks latency of requests forwarded to the upstream proxy.
	UpstreamLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_upstream_latency_seconds",
			Help:    "Latency of requests sent to upstream",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"route", "code"},
	)

	// BreakerState exposes the current state of circuit breakers.
	BreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ratelimit_breaker_state",
			Help: "Current state of the circuit breaker (0=closed, 1=half-open, 2=open)",
		},
		[]string{"name"},
	)
)

// Handler returns the HTTP handler for exposing Prometheus metrics.
func Handler() http.Handler {
	return promhttp.Handler()
}

// ObserveStoreLatency is a helper to record store latency.
func ObserveStoreLatency(algorithm string, start time.Time) {
	StoreLatency.WithLabelValues(algorithm).Observe(time.Since(start).Seconds())
}
