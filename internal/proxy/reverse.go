// Package proxy wraps httputil.ReverseProxy with gateway-specific behaviour.
//
// TODO (Phase 2): implement the reverse proxy.
//
//   func New(upstream string, timeout time.Duration) (http.Handler, error)
//     - Parse and validate upstream URL
//     - Create httputil.ReverseProxy targeting upstream
//     - Set Transport with per-request timeout (via context)
//     - Custom ErrorHandler:
//         dial/timeout errors    → 504 Gateway Timeout
//         connection refused     → 502 Bad Gateway
//         context cancelled      → 499 (log only, client already gone)
//
//   Important: rate-limit quota is NOT refunded on upstream errors.
//   This is documented (plan § 12.2) and deliberate — refunding creates
//   retry amplification and is complex to make atomic.
package proxy
