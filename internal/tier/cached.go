package tier

// TODO (Phase 4): implement CachedResolver.
//
//   type CachedResolver struct {
//       inner Resolver
//       cache sync.Map      // identity → cachedEntry{tier, expiresAt}
//       ttl   time.Duration
//   }
//
//   func NewCached(inner Resolver, ttl time.Duration) *CachedResolver
//
//   func (r *CachedResolver) Resolve(ctx context.Context, identity string) (string, error)
//     1. Check cache for identity; if fresh, return cached tier
//     2. Call r.inner.Resolve(ctx, identity)
//     3. On success: cache the result with expiresAt = now + ttl
//     4. On ErrUnknown: do NOT cache — a newly-issued key should be found promptly
//
// Purpose: allows swapping in a DB-backed resolver in future without changing
// the interface. The static resolver used in this project doesn't need caching,
// but wrapping it demonstrates the pattern and keeps Phase 4 extensible.
