package middleware

// TODO (Phase 2): implement request ID middleware.
//
//   func RequestID(next http.Handler) http.Handler
//     - Check for incoming X-Request-ID header; use it if present
//     - Otherwise generate a new UUID
//     - Set X-Request-ID on both the request context and the response
//     - Store the ID in context so the logging middleware can pick it up
//
//   func RequestIDFromContext(ctx context.Context) string
//     - Retrieve the request ID from context
