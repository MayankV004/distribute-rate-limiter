package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recover is a middleware that intercepts panics in downstream handlers.
// It logs the stack trace and returns a 500 Internal Server Error,
// ensuring the gateway process does not crash on a single bad request.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				slog.Error("panic recovered",
					"id", RequestIDFromContext(r.Context()),
					"error", err,
					"stack", string(debug.Stack()),
				)
				
				// Return a generic 500 without leaking internal details
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
