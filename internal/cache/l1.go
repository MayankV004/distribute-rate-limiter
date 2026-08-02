// Package cache provides an optional L1 in-process allow-cache.
//
// TODO (Phase 7): implement the local allow-cache.
//
//   type L1Cache struct { entries sync.Map }
//
//   type entry struct {
//       decision  limiter.Decision
//       expiresAt time.Time
//   }
//
//   func New() *L1Cache
//
//   func (c *L1Cache) Get(key string, now time.Time) (limiter.Decision, bool)
//     - Returns cached decision if it exists and has not expired.
//     - Never returns a cached deny. Stale denies would lock out a client past reset.
//
//   func (c *L1Cache) Set(key string, d limiter.Decision, ttl time.Duration)
//     - Only cache allow decisions where Remaining is comfortably above zero
//       (e.g. > limit*0.1). Never cache near-limit decisions; accuracy matters most
//       right at the boundary.
//
// IMPORTANT: measure both the Redis-call reduction AND the over-admission
// introduced by caching. Report both numbers in docs/BENCHMARKS.md.
// Reporting speedup without over-admission would be dishonest.
package cache
