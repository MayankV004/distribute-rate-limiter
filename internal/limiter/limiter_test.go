package limiter

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSpecRateAndCapacity(t *testing.T) {
	s := Spec{Limit: 100, Window: time.Minute}
	if s.Rate() != 100.0/60.0 {
		t.Errorf("expected %v, got %v", 100.0/60.0, s.Rate())
	}
	if s.Capacity() != 100 {
		t.Errorf("expected 100 capacity fallback, got %v", s.Capacity())
	}

	sWithBurst := Spec{Limit: 100, Window: time.Minute, Burst: 50}
	if sWithBurst.Capacity() != 50 {
		t.Errorf("expected 50 capacity, got %v", sWithBurst.Capacity())
	}
}

func TestAllowedClampsRemaining(t *testing.T) {
	d := Allowed(100, -5, time.Second)
	if d.Remaining != 0 {
		t.Errorf("expected negative remaining to clamp to 0, got %v", d.Remaining)
	}
}

func TestDenied(t *testing.T) {
	d := Denied(50, 2*time.Second, 10*time.Second)
	if d.Allowed {
		t.Error("Denied should not be Allowed")
	}
	if d.Limit != 50 {
		t.Errorf("expected limit 50, got %v", d.Limit)
	}
	if d.Remaining != 0 {
		t.Errorf("expected remaining 0, got %v", d.Remaining)
	}
}

func TestFakeClockAdvances(t *testing.T) {
	start := time.Unix(1000, 0)
	c := NewFakeClock(start)

	if c.Now() != start {
		t.Errorf("expected %v, got %v", start, c.Now())
	}

	c.Advance(time.Second)
	if c.Now() != start.Add(time.Second) {
		t.Errorf("expected %v, got %v", start.Add(time.Second), c.Now())
	}

	millis := UnixMillis(c.Now())
	if millis != 1001000 {
		t.Errorf("expected 1001000 ms, got %v", millis)
	}
}

func TestFakeClockConcurrent(t *testing.T) {
	c := NewFakeClock(time.Now())
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = c.Now()
		}()
		go func() {
			defer wg.Done()
			c.Advance(time.Millisecond)
		}()
	}

	wg.Wait()
}

type noopLimiter struct{}

func (noopLimiter) Allow(ctx context.Context, key string, cost int64) (Decision, error) {
	return Decision{}, nil
}

// Compile-time assertion that noopLimiter implements Limiter.
var _ Limiter = (*noopLimiter)(nil)
