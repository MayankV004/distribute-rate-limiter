package local

import (
	"context"
	"sync"
	"time"

	"github.com/streamliner/rate-limiter/internal/limiter"
)

// logState maintains an exact history of request timestamps.
// Vulnerable to memory exhaustion under continuous high-volume attacks
// up to the configured limit. Documented in ALGORITHMS.md.
type logState struct {
	mu         sync.Mutex
	timestamps []time.Time
}

type SlidingWindowLog struct {
	spec  limiter.Spec
	clock limiter.Clock
	store *ShardedMap
}

func NewSlidingWindowLog(spec limiter.Spec, clock limiter.Clock) *SlidingWindowLog {
	return &SlidingWindowLog{
		spec:  spec,
		clock: clock,
		store: NewShardedMap(5*time.Minute, time.Hour),
	}
}

func (swl *SlidingWindowLog) Allow(ctx context.Context, key string, cost int64) (limiter.Decision, error) {
	if cost <= 0 {
		cost = 1
	}

	val := swl.store.GetOrInsert(key, func() interface{} {
		return &logState{
			timestamps: make([]time.Time, 0),
		}
	})

	state := val.(*logState)
	state.mu.Lock()
	defer state.mu.Unlock()

	now := swl.clock.Now()
	windowStart := now.Add(-swl.spec.Window)

	validIdx := 0
	for i, t := range state.timestamps {
		if t.After(windowStart) {
			validIdx = i
			break
		}
		if i == len(state.timestamps)-1 {
			validIdx = len(state.timestamps)
		}
	}
	state.timestamps = state.timestamps[validIdx:]

	currentCount := int64(len(state.timestamps))

	if currentCount+cost <= swl.spec.Limit {
		for i := int64(0); i < cost; i++ {
			state.timestamps = append(state.timestamps, now)
		}

		remaining := swl.spec.Limit - (currentCount + cost)

		var resetAfter time.Duration
		if len(state.timestamps) > 0 {
			oldest := state.timestamps[0]
			resetAfter = swl.spec.Window - now.Sub(oldest)
		} else {
			resetAfter = swl.spec.Window
		}

		return limiter.Allowed(swl.spec.Limit, remaining, resetAfter), nil
	}

	var retryAfter time.Duration
	if len(state.timestamps) > 0 {
		neededIndex := (currentCount + cost) - swl.spec.Limit - 1
		if neededIndex >= 0 && neededIndex < int64(len(state.timestamps)) {
			targetTime := state.timestamps[neededIndex]
			retryAfter = swl.spec.Window - now.Sub(targetTime)
		} else {
			retryAfter = swl.spec.Window
		}
	} else {
		retryAfter = swl.spec.Window
	}

	var resetAfter time.Duration
	if len(state.timestamps) > 0 {
		resetAfter = swl.spec.Window - now.Sub(state.timestamps[0])
	} else {
		resetAfter = swl.spec.Window
	}

	return limiter.Denied(swl.spec.Limit, retryAfter, resetAfter), nil
}
