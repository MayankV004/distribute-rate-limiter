package local

import (
	"context"
	"sync"
	"time"

	"github.com/streamliner/rate-limiter/internal/limiter"
)

type leakyState struct {
	mu       sync.Mutex
	water    float64
	lastLeak time.Time
}

type LeakyBucket struct {
	spec  limiter.Spec
	clock limiter.Clock
	store *ShardedMap
}

func NewLeakyBucket(spec limiter.Spec, clock limiter.Clock) *LeakyBucket {
	return &LeakyBucket{
		spec:  spec,
		clock: clock,
		store: NewShardedMap(5*time.Minute, time.Hour),
	}
}

func (lb *LeakyBucket) Allow(ctx context.Context, key string, cost int64) (limiter.Decision, error) {
	if cost <= 0 {
		cost = 1
	}

	capacity := float64(lb.spec.Capacity())
	rate := lb.spec.Rate()

	val := lb.store.GetOrInsert(key, func() interface{} {
		return &leakyState{
			water:    0,
			lastLeak: lb.clock.Now(),
		}
	})

	state := val.(*leakyState)
	state.mu.Lock()
	defer state.mu.Unlock()

	now := lb.clock.Now()

	elapsed := now.Sub(state.lastLeak).Seconds()
	if elapsed > 0 {
		leaked := elapsed * rate
		state.water -= leaked
		if state.water < 0 {
			state.water = 0
		}
	}
	state.lastLeak = now

	if state.water+float64(cost) <= capacity {
		state.water += float64(cost)

		remaining := capacity - state.water
		resetAfterSecs := state.water / rate
		resetAfter := time.Duration(resetAfterSecs * float64(time.Second))

		return limiter.Allowed(lb.spec.Limit, int64(remaining), resetAfter), nil
	}

	overflow := (state.water + float64(cost)) - capacity
	retryAfterSecs := overflow / rate
	retryAfter := time.Duration(retryAfterSecs * float64(time.Second))

	resetAfterSecs := state.water / rate
	resetAfter := time.Duration(resetAfterSecs * float64(time.Second))

	return limiter.Denied(lb.spec.Limit, retryAfter, resetAfter), nil
}
