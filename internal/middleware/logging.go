package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// responseLogger wraps http.ResponseWriter to capture the status code and bytes written.
type responseLogger struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseLogger) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseLogger) Write(b []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

// Logger is a middleware that logs each incoming request along with its duration and status code.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rl := &responseLogger{
			ResponseWriter: w,
			statusCode:     0,
		}

		next.ServeHTTP(rl, r)

		elapsed := time.Since(start)
		
		// If status code was never explicitly set, it defaults to 200 OK.
		if rl.statusCode == 0 {
			rl.statusCode = http.StatusOK
		}

		slog.Info("request",
			"id", RequestIDFromContext(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", rl.statusCode,
			"duration_ms", elapsed.Milliseconds(),
			"remote_ip", r.RemoteAddr,
		)
	})
}
