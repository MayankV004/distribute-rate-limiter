package middleware

import (
	"net/http"
	"strconv"

	"github.com/streamliner/rate-limiter/internal/limiter"
)

// SetRateLimitHeaders injects standard rate limit headers into the HTTP response.
// X-RateLimit-Reset is formatted as an absolute Unix timestamp.
func SetRateLimitHeaders(w http.ResponseWriter, d limiter.Decision) {
	w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(d.Limit, 10))
	w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(d.Remaining, 10))
	
	resetSecs := int64(d.ResetAfter.Seconds())
	if resetSecs <= 0 {
		resetSecs = 1
	}
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetSecs, 10))

	if !d.Allowed {
		retrySecs := int64(d.RetryAfter.Seconds())
		if retrySecs <= 0 {
			retrySecs = 1
		}
		w.Header().Set("Retry-After", strconv.FormatInt(retrySecs, 10))
	}
}
