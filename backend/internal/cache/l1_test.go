package cache

import (
	"context"
	"testing"
	"time"

	"github.com/streamliner/rate-limiter/internal/config"
	"github.com/streamliner/rate-limiter/internal/limiter"
)

type fakeClock struct {
	now time.Time
}

func (f *fakeClock) Now() time.Time {
	return f.now
}

// fakeLimiter always returns what it was configured to return.
type fakeLimiter struct {
	decision limiter.Decision
	err      error
	calls    int
}

func (f *fakeLimiter) Allow(ctx context.Context, key string, cost int64) (limiter.Decision, error) {
	f.calls++
	return f.decision, f.err
}

func TestL1Cache_GetSet(t *testing.T) {
	c := New()
	now := time.Now()
	ttl := 10 * time.Millisecond

	// 1. Set allowed decision above threshold (should cache)
	d1 := limiter.Decision{Allowed: true, Limit: 100, Remaining: 50}
	c.Set("k1", d1, ttl, 0.1, now)

	if got, ok := c.Get("k1", now); !ok || got.Remaining != 50 {
		t.Fatalf("expected k1 to be cached, got ok=%v, %v", ok, got)
	}

	// 2. Set denied decision (should NOT cache)
	d2 := limiter.Decision{Allowed: false, Limit: 100, Remaining: 0}
	c.Set("k2", d2, ttl, 0.1, now)

	if _, ok := c.Get("k2", now); ok {
		t.Fatalf("expected k2 to not be cached")
	}

	// 3. Set allowed decision below threshold (should NOT cache)
	d3 := limiter.Decision{Allowed: true, Limit: 100, Remaining: 5} // 5 <= 10
	c.Set("k3", d3, ttl, 0.1, now)

	if _, ok := c.Get("k3", now); ok {
		t.Fatalf("expected k3 to not be cached (below threshold)")
	}

	// 4. Test expiration
	if _, ok := c.Get("k1", now.Add(20*time.Millisecond)); ok {
		t.Fatalf("expected k1 to be expired")
	}
}

func TestL1Wrapper(t *testing.T) {
	clock := &fakeClock{now: time.Now()}
	inner := &fakeLimiter{
		decision: limiter.Decision{Allowed: true, Limit: 100, Remaining: 50},
	}

	cfg := config.L1CacheConfig{
		Enabled:            true,
		TTL:                10 * time.Millisecond,
		RemainingThreshold: 0.1,
	}

	w := Wrap(inner, cfg, clock)

	// First call -> misses cache, calls inner
	dec, err := w.Allow(context.Background(), "test-key", 1)
	if err != nil || !dec.Allowed {
		t.Fatalf("expected allow, got err=%v, allowed=%v", err, dec.Allowed)
	}
	if inner.calls != 1 {
		t.Fatalf("expected 1 inner call, got %d", inner.calls)
	}

	// Second call -> hits cache
	_, err = w.Allow(context.Background(), "test-key", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("expected inner calls to stay 1, got %d", inner.calls)
	}

	// Third call after expiration -> misses cache, calls inner
	clock.now = clock.now.Add(20 * time.Millisecond)
	_, err = w.Allow(context.Background(), "test-key", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.calls != 2 {
		t.Fatalf("expected inner calls to increment to 2, got %d", inner.calls)
	}
}

func TestL1Wrapper_Disabled(t *testing.T) {
	clock := &fakeClock{now: time.Now()}
	inner := &fakeLimiter{
		decision: limiter.Decision{Allowed: true, Limit: 100, Remaining: 50},
	}

	cfg := config.L1CacheConfig{
		Enabled:            false,
		TTL:                10 * time.Millisecond,
		RemainingThreshold: 0.1,
	}

	w := Wrap(inner, cfg, clock)

	w.Allow(context.Background(), "test-key", 1)
	w.Allow(context.Background(), "test-key", 1)

	if inner.calls != 2 {
		t.Fatalf("expected 2 inner calls because cache is disabled, got %d", inner.calls)
	}
}
