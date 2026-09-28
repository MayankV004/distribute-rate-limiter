package local

import (
	"context"
	"sync"
	"time"

	"github.com/streamliner/rate-limiter/internal/limiter"
)

type counterState struct {
	mu            sync.Mutex
	currentWindow int64
	currentCount  int64
	prevCount     int64
}

type SlidingWindowCounter struct {
	spec  limiter.Spec
	clock limiter.Clock
	store *ShardedMap
}

func NewSlidingWindowCounter(spec limiter.Spec, clock limiter.Clock) *SlidingWindowCounter {
	return &SlidingWindowCounter{
		spec:  spec,
		clock: clock,
		store: NewShardedMap(5*time.Minute, time.Hour),
	}
}

func (swc *SlidingWindowCounter) Allow(ctx context.Context, key string, cost int64) (limiter.Decision, error) {
	if cost <= 0 {
		cost = 1
	}

	val := swc.store.GetOrInsert(key, func() interface{} {
		return &counterState{
			currentWindow: swc.currentWindowStart().UnixNano(),
		}
	})

	state := val.(*counterState)
	state.mu.Lock()
	defer state.mu.Unlock()

	now := swc.clock.Now()
	currentWindowStart := swc.currentWindowStart()

	stateWindowStart := time.Unix(0, state.currentWindow)
	if currentWindowStart.After(stateWindowStart) {
		if currentWindowStart.Sub(stateWindowStart) == swc.spec.Window {
			state.prevCount = state.currentCount
		} else {
			state.prevCount = 0
		}
		state.currentCount = 0
		state.currentWindow = currentWindowStart.UnixNano()
	}

	// Approximate the rolling window utilization by weighting the previous
	// window's count against the elapsed duration of the current window.
	elapsedPercentage := float64(now.Sub(currentWindowStart)) / float64(swc.spec.Window)
	weightedCount := float64(state.prevCount)*(1.0-elapsedPercentage) + float64(state.currentCount)

	if weightedCount+float64(cost) <= float64(swc.spec.Limit) {
		state.currentCount += cost

		remaining := swc.spec.Limit - int64(weightedCount) - cost
		resetAfter := swc.spec.Window - now.Sub(currentWindowStart)

		return limiter.Allowed(swc.spec.Limit, remaining, resetAfter), nil
	}

	retryAfter := swc.spec.Window - now.Sub(currentWindowStart)
	resetAfter := retryAfter

	return limiter.Denied(swc.spec.Limit, retryAfter, resetAfter), nil
}

func (swc *SlidingWindowCounter) currentWindowStart() time.Time {
	now := swc.clock.Now()
	windowNs := swc.spec.Window.Nanoseconds()
	startNs := (now.UnixNano() / windowNs) * windowNs
	return time.Unix(0, startNs)
}
