package local

import (
	"context"
	"sync"
	"time"

	"github.com/streamliner/rate-limiter/internal/limiter"
)

type tokenBucketState struct {
	mu sync.Mutex
	// tokens uses float64 to preserve precision for slow refill rates
	// that would otherwise round to zero on integer division.
	tokens      float64
	lastUpdated time.Time
}

type TokenBucket struct {
	spec  limiter.Spec
	clock limiter.Clock
	store *ShardedMap
}

func NewTokenBucket(spec limiter.Spec, clock limiter.Clock) *TokenBucket {
	return &TokenBucket{
		spec:  spec,
		clock: clock,
		store: NewShardedMap(5*time.Minute, time.Hour),
	}
}

func (tb *TokenBucket) Allow(ctx context.Context, key string, cost int64) (limiter.Decision, error) {
	if cost <= 0 {
		cost = 1
	}

	capacity := float64(tb.spec.Capacity())
	rate := tb.spec.Rate()

	val := tb.store.GetOrInsert(key, func() interface{} {
		return &tokenBucketState{
			tokens:      capacity,
			lastUpdated: tb.clock.Now(),
		}
	})

	state := val.(*tokenBucketState)
	state.mu.Lock()
	defer state.mu.Unlock()

	now := tb.clock.Now()

	elapsed := now.Sub(state.lastUpdated).Seconds()
	if elapsed > 0 {
		state.tokens += elapsed * rate
		if state.tokens > capacity {
			state.tokens = capacity
		}
	}
	state.lastUpdated = now

	if state.tokens >= float64(cost) {
		state.tokens -= float64(cost)

		missing := capacity - state.tokens
		resetAfterSecs := missing / rate
		resetAfter := time.Duration(resetAfterSecs * float64(time.Second))

		return limiter.Allowed(tb.spec.Limit, int64(state.tokens), resetAfter), nil
	}

	missingForCost := float64(cost) - state.tokens
	retryAfterSecs := missingForCost / rate
	retryAfter := time.Duration(retryAfterSecs * float64(time.Second))

	missingTotal := capacity - state.tokens
	resetAfterSecs := missingTotal / rate
	resetAfter := time.Duration(resetAfterSecs * float64(time.Second))

	return limiter.Denied(tb.spec.Limit, retryAfter, resetAfter), nil
}
