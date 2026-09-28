package tier

import (
	"context"
	"sync"
	"time"
)

type cachedEntry struct {
	tier      string
	expiresAt time.Time
}

// CachedResolver wraps an inner Resolver and caches its results for a TTL.
// It uses a sync.Map for concurrent access and caches only successful resolutions.
type CachedResolver struct {
	inner Resolver
	cache sync.Map
	ttl   time.Duration
}

// NewCached returns a new CachedResolver wrapping the given inner resolver.
func NewCached(inner Resolver, ttl time.Duration) *CachedResolver {
	return &CachedResolver{
		inner: inner,
		ttl:   ttl,
	}
}

// Resolve looks up the identity in the cache first. If found and fresh, it returns
// the cached tier. Otherwise, it delegates to the inner resolver and caches the
// successful result. It does NOT cache ErrUnknown.
func (r *CachedResolver) Resolve(ctx context.Context, identity string) (string, error) {
	// 1. Check cache
	if val, ok := r.cache.Load(identity); ok {
		entry := val.(cachedEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.tier, nil
		}
		// expired, clean up (optional, but good practice if we don't want it sitting around)
		r.cache.Delete(identity)
	}

	// 2. Call inner
	tier, err := r.inner.Resolve(ctx, identity)
	if err != nil {
		return "", err
	}

	// 3. Cache on success
	r.cache.Store(identity, cachedEntry{
		tier:      tier,
		expiresAt: time.Now().Add(r.ttl),
	})

	return tier, nil
}
