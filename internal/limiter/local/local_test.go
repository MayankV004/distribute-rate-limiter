package local

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/streamliner/rate-limiter/internal/limiter"
)

func TestLocalLimiters(t *testing.T) {
	runTests := func(t *testing.T, name string, factory func(spec limiter.Spec, clock limiter.Clock) limiter.Limiter) {
		t.Run(name, func(t *testing.T) {
			clock := limiter.NewFakeClock(time.Now())
			spec := limiter.Spec{Limit: 2, Window: 10 * time.Second, Burst: 2}
			l := factory(spec, clock)

			d1, _ := l.Allow(context.Background(), "user_1", 1)
			if !d1.Allowed {
				t.Errorf("expected request 1 to be allowed")
			}

			d2, _ := l.Allow(context.Background(), "user_1", 1)
			if !d2.Allowed {
				t.Errorf("expected request 2 to be allowed")
			}

			d3, _ := l.Allow(context.Background(), "user_1", 1)
			if d3.Allowed {
				t.Errorf("expected request 3 to be DENIED")
			}

			clock.Advance(20 * time.Second)

			d4, _ := l.Allow(context.Background(), "user_1", 1)
			if !d4.Allowed {
				t.Errorf("expected request 4 to be allowed after 20 seconds")
			}
		})
	}

	runTests(t, "TokenBucket", func(spec limiter.Spec, clock limiter.Clock) limiter.Limiter {
		return NewTokenBucket(spec, clock)
	})

	runTests(t, "SlidingWindowLog", func(spec limiter.Spec, clock limiter.Clock) limiter.Limiter {
		return NewSlidingWindowLog(spec, clock)
	})

	runTests(t, "SlidingWindowCounter", func(spec limiter.Spec, clock limiter.Clock) limiter.Limiter {
		return NewSlidingWindowCounter(spec, clock)
	})

	runTests(t, "LeakyBucket", func(spec limiter.Spec, clock limiter.Clock) limiter.Limiter {
		return NewLeakyBucket(spec, clock)
	})
}

// TestRaceCondition validates that the underlying sharded map and algorithm mutexes
// safely handle extreme concurrency without dropping or double-counting state.
// It is expected to be run with the -race detector flag.
func TestRaceCondition(t *testing.T) {
	clock := limiter.NewFakeClock(time.Now())
	
	// A massive limit prevents legitimate rate-limiting rejections so we can
	// assert the exact count of allowed requests.
	spec := limiter.Spec{Limit: 2000, Window: 10 * time.Minute, Burst: 2000}
	l := NewTokenBucket(spec, clock)

	const goroutines = 1000
	var wg sync.WaitGroup
	var allowedCount atomic.Int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			d, _ := l.Allow(context.Background(), "super_viral_user", 1)
			if d.Allowed {
				allowedCount.Add(1)
			}
		}()
	}

	wg.Wait()
	
	if allowedCount.Load() != int64(goroutines) {
		t.Errorf("expected %d allowed requests, got %d", goroutines, allowedCount.Load())
	}
}
