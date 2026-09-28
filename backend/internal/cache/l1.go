package cache

import (
	"context"
	"sync"
	"time"

	"github.com/streamliner/rate-limiter/internal/config"
	"github.com/streamliner/rate-limiter/internal/limiter"
)

type entry struct {
	decision  limiter.Decision
	expiresAt time.Time
}

// L1Cache provides a short-TTL in-process allow-cache.
type L1Cache struct {
	entries sync.Map
}

func New() *L1Cache {
	return &L1Cache{}
}

// Get returns the cached decision if it exists, is allowed, and hasn't expired.
func (c *L1Cache) Get(key string, now time.Time) (limiter.Decision, bool) {
	val, ok := c.entries.Load(key)
	if !ok {
		return limiter.Decision{}, false
	}
	
	e := val.(entry)
	if now.After(e.expiresAt) {
		c.entries.Delete(key)
		return limiter.Decision{}, false
	}
	
	// We only cache 'Allow' decisions, but double-check to be safe.
	if !e.decision.Allowed {
		return limiter.Decision{}, false
	}
	
	return e.decision, true
}

// Set caches an allow decision only if Remaining is comfortably above zero.
func (c *L1Cache) Set(key string, d limiter.Decision, ttl time.Duration, threshold float64, now time.Time) {
	if !d.Allowed {
		return
	}
	
	// Check threshold: only cache if remaining > limit * threshold
	if float64(d.Remaining) <= float64(d.Limit)*threshold {
		return
	}
	
	c.entries.Store(key, entry{
		decision:  d,
		expiresAt: now.Add(ttl),
	})
}

type l1Wrapper struct {
	cache  *L1Cache
	inner  limiter.Limiter
	config config.L1CacheConfig
	clock  limiter.Clock
}

// Wrap returns a new Limiter that caches allow decisions from the inner Limiter.
func Wrap(inner limiter.Limiter, cfg config.L1CacheConfig, clock limiter.Clock) limiter.Limiter {
	if !cfg.Enabled {
		return inner
	}
	return &l1Wrapper{
		cache:  New(),
		inner:  inner,
		config: cfg,
		clock:  clock,
	}
}

func (w *l1Wrapper) Allow(ctx context.Context, key string, cost int64) (limiter.Decision, error) {
	now := w.clock.Now()
	
	// 1. Check L1 Cache
	if dec, ok := w.cache.Get(key, now); ok {
		// Note: we can't reliably decrement the cached "Remaining" for concurrent requests 
		// without locking, which defeats the point. Returning the cached Remaining is an 
		// accepted trade-off that leads to over-admission.
		return dec, nil
	}
	
	// 2. Cache miss, call inner limiter
	dec, err := w.inner.Allow(ctx, key, cost)
	if err != nil {
		return dec, err
	}
	
	// 3. Cache the result
	w.cache.Set(key, dec, w.config.TTL, w.config.RemainingThreshold, now)
	
	return dec, nil
}
