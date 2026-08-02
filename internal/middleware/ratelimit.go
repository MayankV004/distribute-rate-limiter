package middleware

import (
	"context"
	"net/http"
	"strconv"

	"github.com/streamliner/rate-limiter/internal/limiter"
)

// IdentityExtractor pulls a unique string (like an IP or API key) from a request.
type IdentityExtractor func(r *http.Request) string

// LimiterResolver finds the right rate limiter for a specific request path.
type LimiterResolver func(r *http.Request) limiter.Limiter

// RateLimit is the core middleware that protects the backend.
func RateLimit(extract IdentityExtractor, resolve LimiterResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			
			// 1. Who is making the request?
			key := extract(r)
			if key == "" {
				// If we can't identify them, use their IP address as a fallback
				key = r.RemoteAddr
			}

			// 2. What limiter applies to this route?
			l := resolve(r)
			if l == nil {
				// If no rate limit applies to this route, just pass it through
				next.ServeHTTP(w, r)
				return
			}

			// 3. Ask the limiter if they are allowed (Cost is always 1 for now)
			decision, err := l.Allow(context.Background(), key, 1)

			// 4. If the Redis store is completely broken (Phase 3), we default to Allowed
			// so that a caching failure doesn't take down the entire API (Fail-Open policy)
			if err != nil {
				// Log the error in a real app, but let the request through
				next.ServeHTTP(w, r)
				return
			}

			// 5. Always set the informative headers (Allowed or Denied)
			w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(decision.Limit, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(decision.Remaining, 10))
			
			resetSecs := int64(decision.ResetAfter.Seconds())
			if resetSecs <= 0 {
				resetSecs = 1
			}
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetSecs, 10))

			// 6. Block them if they are out of tokens!
			if !decision.Allowed {
				retrySecs := int64(decision.RetryAfter.Seconds())
				if retrySecs <= 0 {
					retrySecs = 1
				}
				w.Header().Set("Retry-After", strconv.FormatInt(retrySecs, 10))
				
				// Return a strict 429 Too Many Requests
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			// 7. They are allowed! Pass the request to the real backend
			next.ServeHTTP(w, r)
		})
	}
}
