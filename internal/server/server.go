// Package server starts and gracefully shuts down the gateway's HTTP listeners.
//
// TODO (Phase 0): implement two-listener server.
//
//   func Run(ctx context.Context, cfg ServerConfig, mainHandler, metricsHandler http.Handler) error
//
//   Two listeners:
//     :8080 (or cfg.Addr)         → main traffic: rate limit + proxy
//     :9090 (or cfg.MetricsAddr)  → /metrics only (NOT rate-limited, NOT public-facing)
//
//   Health endpoints on the main listener:
//     GET /healthz  → 200 OK (liveness: process is running)
//     GET /readyz   → 200 OK when Redis is reachable, 503 otherwise (readiness)
//
//   Graceful shutdown:
//     - Listen for os.Interrupt / SIGTERM
//     - Call srv.Shutdown(ctx) with cfg.ShutdownGrace timeout
//     - Drain in-flight requests before exit
//     - Log "shutdown complete" when done
package server
