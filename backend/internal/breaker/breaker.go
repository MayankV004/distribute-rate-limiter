// Package breaker implements a 3-state circuit breaker around Redis calls.
package breaker

import (
	"context"
	"sync"
	"time"

	"github.com/streamliner/rate-limiter/internal/config"
	"github.com/streamliner/rate-limiter/internal/limiter"
	"github.com/streamliner/rate-limiter/internal/metrics"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

// Clock defines the interface for time, allowing injection in tests.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

type Breaker struct {
	name  string
	cfg   config.BreakerConfig
	clock Clock

	mu        sync.Mutex
	state     State
	total     int
	failures  int
	openedAt  time.Time
	successes int
}

func New(name string, cfg config.BreakerConfig) *Breaker {
	return NewWithClock(name, cfg, realClock{})
}

func NewWithClock(name string, cfg config.BreakerConfig, clock Clock) *Breaker {
	b := &Breaker{
		name:  name,
		cfg:   cfg,
		clock: clock,
		state: StateClosed,
	}
	metrics.BreakerState.WithLabelValues(name).Set(0) // StateClosed = 0
	return b
}

func (b *Breaker) setState(newState State) {
	b.state = newState
	metrics.BreakerState.WithLabelValues(b.name).Set(float64(newState))
}

func (b *Breaker) Do(ctx context.Context, f func(context.Context) error) error {
	b.mu.Lock()
	switch b.state {
	case StateOpen:
		// Check if we should transition to HalfOpen
		if b.clock.Now().Sub(b.openedAt) > b.cfg.OpenDuration {
			b.setState(StateHalfOpen)
			b.successes = 0 // reset successes on entering HalfOpen
		} else {
			b.mu.Unlock()
			return limiter.ErrUnavailable
		}
	case StateHalfOpen, StateClosed:
		// allowed to proceed
	}
	// Need to unlock before executing the user-provided function so we don't block other requests
	// Note: in HalfOpen we might want to strictly limit concurrent probes, but for this simple
	// implementation we just let requests through and rely on the first failure to re-open.
	b.mu.Unlock()

	// Execute the wrapped function
	err := f(ctx)

	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		b.total++
		if err != nil {
			b.failures++
		}
		// Check transition to Open
		if b.total >= b.cfg.MinRequests {
			ratio := float64(b.failures) / float64(b.total)
			if ratio > b.cfg.ErrorRatio {
				b.setState(StateOpen)
				b.openedAt = b.clock.Now()
			}
		}
	case StateHalfOpen:
		if err != nil {
			// Any failure sends us back to Open
			b.setState(StateOpen)
			b.openedAt = b.clock.Now()
		} else {
			b.successes++
			// Consecutive successes transition back to Closed
			if b.successes >= b.cfg.HalfOpenSuccesses {
				b.setState(StateClosed)
				b.total = 0
				b.failures = 0
			}
		}
	case StateOpen:
		// The state might have concurrently changed back to Open while we were executing
		// the function (e.g. another concurrent probe failed). Do nothing.
	}

	return err
}
