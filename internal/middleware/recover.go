package middleware

// TODO (Phase 2): implement panic recovery middleware.
//
//   func Recover(next http.Handler) http.Handler
//     - defer/recover from any panic in downstream handlers
//     - Log the panic value + stack trace via slog.Error
//     - Return 500 Internal Server Error to the client
//     - Do NOT re-panic (this is a gateway; one bad request must not crash the process)
