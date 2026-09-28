package breaker

import (
	"context"
	"errors"
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

func (f *fakeClock) Advance(d time.Duration) {
	f.now = f.now.Add(d)
}

func TestBreaker_StateTransitions(t *testing.T) {
	cfg := config.BreakerConfig{
		ErrorRatio:        0.5,
		MinRequests:       4,
		OpenDuration:      1 * time.Second,
		HalfOpenSuccesses: 2,
	}

	clock := &fakeClock{now: time.Now()}
	b := NewWithClock("test", cfg, clock)

	// Helper to simulate a call
	doCall := func(returnsError bool) error {
		return b.Do(context.Background(), func(ctx context.Context) error {
			if returnsError {
				return errors.New("simulated error")
			}
			return nil
		})
	}

	// 1. Initial State is Closed. Send 2 errors, 1 success (3 total).
	// Ratio is 2/3 (0.66 > 0.5), but total < MinRequests (4). Breaker remains Closed.
	doCall(true)
	doCall(true)
	doCall(false)
	if b.state != StateClosed {
		t.Fatalf("expected state Closed, got %v", b.state)
	}

	// 2. 4th call is an error. Total is 4, failures is 3. Ratio is 3/4 = 0.75 > 0.5.
	// Should transition to Open.
	doCall(true)
	if b.state != StateOpen {
		t.Fatalf("expected state Open, got %v", b.state)
	}

	// 3. In Open state, calls immediately return limiter.ErrUnavailable without executing function.
	err := doCall(false) // Should not actually execute or return nil
	if !errors.Is(err, limiter.ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}

	// 4. Advance time past OpenDuration. Next call should transition to HalfOpen and execute.
	clock.Advance(2 * time.Second)
	
	// Probe 1: Success
	err = doCall(false)
	if err != nil {
		t.Fatalf("expected successful probe, got %v", err)
	}
	if b.state != StateHalfOpen {
		t.Fatalf("expected state HalfOpen, got %v", b.state)
	}

	// 5. Probe 2: Failure. Should immediately transition back to Open.
	doCall(true)
	if b.state != StateOpen {
		t.Fatalf("expected state Open after failed probe, got %v", b.state)
	}

	// 6. Advance time again. Next call goes HalfOpen.
	clock.Advance(2 * time.Second)

	// 7. Probe 1: Success
	doCall(false)
	
	// 8. Probe 2: Success. Should transition to Closed (HalfOpenSuccesses = 2).
	doCall(false)
	if b.state != StateClosed {
		t.Fatalf("expected state Closed after %d successes, got %v", cfg.HalfOpenSuccesses, b.state)
	}

	// 9. Now we are back in Closed, counters should be reset.
	if b.total != 0 || b.failures != 0 {
		t.Fatalf("expected counters to be reset upon entering Closed, got total=%d failures=%d", b.total, b.failures)
	}
}
