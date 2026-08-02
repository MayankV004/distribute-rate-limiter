//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/streamliner/rate-limiter/internal/limiter"
	"github.com/streamliner/rate-limiter/internal/limiter/distributed"
)

// TestFailover verifies that a Redis outage returns ErrUnavailable promptly —
// the limiter must never hang, panic, or return a fabricated allow/deny decision
// when the backing store is unreachable.
func TestFailover(t *testing.T) {
	spec := limiter.Spec{Limit: 10, Window: 60 * time.Second, Burst: 10}
	clock := limiter.NewFakeClock(time.Now())

	// Point at a port that has nothing listening on it.
	deadRdb := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:19999",
		DialTimeout:  100 * time.Millisecond,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
	})
	defer deadRdb.Close()

	l := distributed.NewTokenBucket(spec, deadRdb, clock)

	_, err := l.Allow(context.Background(), "any-key", 1)
	if !errors.Is(err, limiter.ErrUnavailable) {
		t.Errorf("expected ErrUnavailable on dead Redis, got: %v", err)
	}
}

// TODO (Phase 5): test circuit breaker and failover behaviour.
//
// Test cases:
//   TestFailOpen:
//     - Configure route with fallback: open
//     - Kill/pause the Redis container mid-test
//     - Assert all subsequent requests are ALLOWED (fail-open)
//     - Assert breaker transitions to Open state
//     - Unpause Redis; assert breaker eventually re-closes
//
//   TestFailClosed:
//     - Configure route with fallback: closed
//     - Kill Redis
//     - Assert all subsequent requests are DENIED with 503 (not 429)
//     - Header Retry-After must be present
//
//   TestBreakerLatency:
//     - Slow Redis (add artificial delay via toxiproxy or sleep in Lua)
//     - Assert gateway p99 latency does NOT include the Redis timeout
//     - Once breaker opens, requests short-circuit in <1ms
