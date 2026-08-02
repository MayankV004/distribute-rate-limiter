package identity

// TODO (Phase 2): implement Redis key builder.
//
//   func BuildKey(identity, method, routePattern, algorithm string) string
//
// Key format (from plan § 4):
//   ratelimit:{<identity>}:<method>:<route-pattern>:<algorithm>
//   e.g. ratelimit:{key_9f3a1b}:GET:/api/v1/orders:swc
//
// Rules:
//   - Hash tag {identity} is mandatory for Redis Cluster correctness.
//     Only the braced part is used for slot assignment, so all keys for one
//     identity land in one slot, enabling future multi-key operations.
//   - Use the route pattern (e.g. /api/v1/orders/{id}), NOT the raw path,
//     so /orders/1 and /orders/2 share the same bucket.
//   - Abbreviate algorithm names in the key to avoid bloat:
//       token_bucket            → tb
//       sliding_window_log      → swl
//       sliding_window_counter  → swc
//       leaky_bucket            → lb
