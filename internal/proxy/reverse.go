package proxy

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// New creates a custom reverse proxy that forwards requests to the upstream URL.
// It sets a global timeout on the forwarded request and maps network errors
// to appropriate HTTP 5xx responses instead of crashing or serving blank pages.
func New(upstream string, timeout time.Duration) (http.Handler, error) {
	backendURL, err := url.Parse(upstream)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	// Custom Error Handler to gracefully handle upstream failures
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Warn("proxy error", "error", err, "upstream", upstream)

		if errors.Is(err, context.Canceled) {
			w.WriteHeader(499) // Client Closed Request
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "Gateway Timeout", http.StatusGatewayTimeout)
			return
		}

		// For connection refused, dial errors, etc.
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Enforce the upstream timeout using context
		var ctx context.Context
		var cancel context.CancelFunc

		if timeout > 0 {
			ctx, cancel = context.WithTimeout(r.Context(), timeout)
			defer cancel()
		} else {
			ctx = r.Context()
		}

		r = r.WithContext(ctx)
		proxy.ServeHTTP(w, r)
	}), nil
}
