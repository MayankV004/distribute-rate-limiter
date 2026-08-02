package middleware

// TODO (Phase 2): implement response header helpers.
//
// SetRateLimitHeaders(w http.ResponseWriter, d limiter.Decision)
//   Always sets (allowed and denied):
//     X-RateLimit-Limit:     d.Limit
//     X-RateLimit-Remaining: d.Remaining
//     X-RateLimit-Reset:     time.Now().Add(d.ResetAfter).Unix()  (Unix timestamp)
//
//   Only on denied (d.RetryAfter > 0):
//     Retry-After: ceil(d.RetryAfter.Seconds())
//
// Note: X-RateLimit-Reset must be an absolute Unix timestamp (RFC 7231 §7.1.3),
// not a relative duration, so clients can synchronize their clocks.
