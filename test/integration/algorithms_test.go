//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/streamliner/rate-limiter/internal/limiter"
	"github.com/streamliner/rate-limiter/internal/limiter/distributed"
)

func newRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{Addr: redisAddr})
}

func TestDistributedLimiters(t *testing.T) {
	spec := limiter.Spec{Limit: 2, Window: 10 * time.Second, Burst: 2}
	clock := limiter.NewFakeClock(time.Now())

	type factory func(rdb *redis.Client) limiter.Limiter

	tests := []struct {
		name    string
		factory factory
	}{
		{"TokenBucket", func(rdb *redis.Client) limiter.Limiter {
			return distributed.NewTokenBucket(spec, rdb, clock)
		}},
		{"SlidingWindowLog", func(rdb *redis.Client) limiter.Limiter {
			return distributed.NewSlidingWindowLog(spec, rdb, clock)
		}},
		{"SlidingWindowCounter", func(rdb *redis.Client) limiter.Limiter {
			return distributed.NewSlidingWindowCounter(spec, rdb, clock)
		}},
		{"LeakyBucket", func(rdb *redis.Client) limiter.Limiter {
			return distributed.NewLeakyBucket(spec, rdb, clock)
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rdb := newRedisClient()
			defer rdb.Close()
			l := tc.factory(rdb)
			key := "test-user-" + tc.name
			ctx := context.Background()

			d1, err := l.Allow(ctx, key, 1)
			if err != nil || !d1.Allowed {
				t.Fatalf("request 1: expected allowed, got allowed=%v err=%v", d1.Allowed, err)
			}

			d2, err := l.Allow(ctx, key, 1)
			if err != nil || !d2.Allowed {
				t.Fatalf("request 2: expected allowed, got allowed=%v err=%v", d2.Allowed, err)
			}

			d3, err := l.Allow(ctx, key, 1)
			if err != nil || d3.Allowed {
				t.Fatalf("request 3: expected denied, got allowed=%v err=%v", d3.Allowed, err)
			}

			clock.Advance(20 * time.Second)

			d4, err := l.Allow(ctx, key, 1)
			if err != nil || !d4.Allowed {
				t.Fatalf("request 4 (after refill): expected allowed, got allowed=%v err=%v", d4.Allowed, err)
			}
		})
	}
}

// TestGlobalLimit is the core Phase 3 proof: two separate Redis clients sharing
// one Redis instance must enforce one combined limit, not two independent limits.
func TestGlobalLimit(t *testing.T) {
	spec := limiter.Spec{Limit: 10, Window: 60 * time.Second, Burst: 10}
	clock := limiter.NewFakeClock(time.Now())
	key := "shared-key-global-limit"
	ctx := context.Background()

	rdb1 := newRedisClient()
	defer rdb1.Close()
	rdb2 := newRedisClient()
	defer rdb2.Close()

	l1 := distributed.NewTokenBucket(spec, rdb1, clock)
	l2 := distributed.NewTokenBucket(spec, rdb2, clock)

	allowed := 0
	// Alternate between the two clients to simulate two gateway replicas.
	for i := 0; i < 20; i++ {
		var d limiter.Decision
		var err error
		if i%2 == 0 {
			d, err = l1.Allow(ctx, key, 1)
		} else {
			d, err = l2.Allow(ctx, key, 1)
		}
		if err != nil {
			t.Fatalf("Allow returned unexpected error: %v", err)
		}
		if d.Allowed {
			allowed++
		}
	}

	// Exactly 10 requests should be allowed across both clients combined —
	// not 20 (which would happen if each replica kept its own counter).
	if allowed != int(spec.Limit) {
		t.Errorf("global limit: expected exactly %d allowed across two clients, got %d", spec.Limit, allowed)
	}
}

