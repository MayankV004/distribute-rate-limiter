package distributed

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/streamliner/rate-limiter/internal/limiter"
)

// SlidingWindowCounter is a Redis-backed approximate sliding window using two STRING
// counters (current and previous window). O(1) memory; bounded over-admission error.
type SlidingWindowCounter struct {
	spec  limiter.Spec
	rdb   *redis.Client
	clock limiter.Clock
}

func NewSlidingWindowCounter(spec limiter.Spec, rdb *redis.Client, clock limiter.Clock) *SlidingWindowCounter {
	return &SlidingWindowCounter{spec: spec, rdb: rdb, clock: clock}
}

func (swc *SlidingWindowCounter) Allow(ctx context.Context, key string, cost int64) (limiter.Decision, error) {
	if cost <= 0 {
		cost = 1
	}

	now := swc.clock.Now()
	nowMs := now.UnixMilli()
	windowMs := swc.spec.Window.Milliseconds()
	// window_start_ms: floor to the nearest window boundary
	windowStartMs := (nowMs / windowMs) * windowMs
	// TTL is 2× window so the previous-window key stays alive while it is still needed.
	ttlMs := windowMs * 2

	// Both keys must share the same hash tag {key} so they land in one Redis Cluster slot.
	currKey := fmt.Sprintf("ratelimit:{%s}:swc:%d", key, windowStartMs)
	prevKey := fmt.Sprintf("ratelimit:{%s}:swc:%d", key, windowStartMs-windowMs)

	res, err := runScript(ctx, swc.rdb, slidingWindowCounterScript,
		[]string{currKey, prevKey},
		swc.spec.Limit, windowMs, nowMs, cost, windowStartMs, ttlMs,
	)
	if err != nil {
		return limiter.Decision{}, err
	}

	allowed := toInt64(res[0])
	remaining := toInt64(res[1])

	resetAfter := time.Duration(windowMs-(nowMs-windowStartMs)) * time.Millisecond

	if allowed == 1 {
		return limiter.Allowed(swc.spec.Limit, remaining, resetAfter), nil
	}
	return limiter.Denied(swc.spec.Limit, resetAfter, resetAfter), nil
}

