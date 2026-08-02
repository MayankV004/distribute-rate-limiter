package middleware

// TODO (Phase 2): implement structured request logging.
//
//   func Logger(next http.Handler) http.Handler
//     Log one line per request using log/slog (stdlib, no external dep):
//       slog.Info("request",
//           "id",       RequestIDFromContext(ctx),
//           "method",   r.Method,
//           "path",     r.URL.Path,
//           "status",   statusCode,
//           "duration", elapsed,
//           "decision", "allowed"|"denied",
//       )
//
//   Use a ResponseWriter wrapper that captures the status code since
//   http.ResponseWriter does not expose it after WriteHeader is called.
