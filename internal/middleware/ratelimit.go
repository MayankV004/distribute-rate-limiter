package middleware

import (
	"log"
	"net/http"
	"strconv"

	"time"

	"github.com/streamliner/rate-limiter/internal/config"
	"github.com/streamliner/rate-limiter/internal/identity"
	"github.com/streamliner/rate-limiter/internal/limiter"
	"github.com/streamliner/rate-limiter/internal/metrics"
	"github.com/streamliner/rate-limiter/internal/tier"
)

// responseWriter wraps http.ResponseWriter to capture the HTTP status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RateLimitForRoute creates a middleware instance for a specific configured route.
// It uses pre-instantiated limiters for each tier of this route.
func RateLimitForRoute(
	routeCfg config.RouteConfig,
	limitersByTier map[string]limiter.Limiter,
	extractor identity.Extractor,
	resolver tier.Resolver,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Who is making the request?
			clientID, ok := extractor.Extract(r)
			if !ok {
				// If extractor chain completely fails, fallback to remote addr
				clientID = r.RemoteAddr
			}

			// 2. What tier are they in?
			tierName := routeCfg.Tier // Hardcoded override in route config
			if tierName == "" {
				resolvedTier, err := resolver.Resolve(r.Context(), clientID)
				if err != nil {
					if err == tier.ErrUnknown {
						http.Error(w, "Unauthorized: Unknown Identity", http.StatusUnauthorized)
						return
					}
					// Infrastructure error (e.g. database down)
					log.Printf("tier resolution error: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				tierName = resolvedTier
			}

			// 3. Find the specific limiter for this tier
			l, ok := limitersByTier[tierName]
			if !ok {
				// E.g. config defines a tier that isn't mapped to a quota
				http.Error(w, "Forbidden: No quota for tier", http.StatusForbidden)
				return
			}

			// 4. Build the canonical Redis key
			method := r.Method
			if len(routeCfg.Methods) == 0 {
				method = "*"
			}
			redisKey := identity.BuildKey(clientID, method, routeCfg.Pattern, routeCfg.Algorithm, tierName)

			// 5. Ask the limiter if they are allowed
			decision, err := l.Allow(r.Context(), redisKey, routeCfg.Cost)

			if err != nil {
				log.Printf("limiter error for key %s: %v", redisKey, err)
				if routeCfg.Fallback == "closed" {
					metrics.DecisionsTotal.WithLabelValues(routeCfg.Pattern, tierName, routeCfg.Algorithm, "fail_closed").Inc()
					metrics.FallbackTotal.WithLabelValues(routeCfg.Pattern, "closed").Inc()
					http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
					return
				}
				// Fallback == "open": allow the request to proceed (availability wins over protection)
				metrics.DecisionsTotal.WithLabelValues(routeCfg.Pattern, tierName, routeCfg.Algorithm, "fail_open").Inc()
				metrics.FallbackTotal.WithLabelValues(routeCfg.Pattern, "open").Inc()
			} else {
				if decision.Allowed {
					metrics.DecisionsTotal.WithLabelValues(routeCfg.Pattern, tierName, routeCfg.Algorithm, "allowed").Inc()
				} else {
					metrics.DecisionsTotal.WithLabelValues(routeCfg.Pattern, tierName, routeCfg.Algorithm, "denied").Inc()
				}
			}

			// 6. Always set informative headers
			SetRateLimitHeaders(w, decision)

			// 7. Block if out of tokens
			if !decision.Allowed {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			// 8. Allowed or Fail-Open! Pass to backend.
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)
			metrics.UpstreamLatency.WithLabelValues(routeCfg.Pattern, strconv.Itoa(rw.statusCode)).Observe(time.Since(start).Seconds())
		})
	}
}
