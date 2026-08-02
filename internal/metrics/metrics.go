// Package metrics registers Prometheus collectors for the gateway.
//
// TODO (Phase 6): implement metric collectors.
//
//   Counters:
//     ratelimit_decisions_total{route, tier, algorithm, decision}
//       decision ∈ {"allowed", "denied", "fail_open", "fail_closed"}
//     ratelimit_store_errors_total{kind}
//       kind ∈ {"timeout", "unavailable", "script_error"}
//     ratelimit_fallback_total{route, policy}
//
//   Histograms (custom buckets: 0.5ms, 1ms, 2ms, 5ms, 10ms, 25ms, 50ms, 100ms):
//     ratelimit_store_latency_seconds{algorithm}
//     gateway_upstream_latency_seconds{route, status_code}
//
//   Gauges:
//     ratelimit_breaker_state{name}   0=closed, 1=half-open, 2=open
//
// IMPORTANT: default Prometheus histogram buckets start at 5ms.
// At that resolution every sub-5ms measurement is invisible.
// Always declare explicit sub-millisecond buckets for store operations.
//
//   func Handler() http.Handler    → promhttp.Handler() on /metrics
//   func MustRegister()            → call once from main
package metrics
