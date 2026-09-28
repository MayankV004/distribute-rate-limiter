package distributed

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/streamliner/rate-limiter/internal/limiter"
)

// SlidingWindowLog is a Redis-backed exact sliding window using a ZSET of timestamps.
// Accurate but O(requests-in-window) memory — not suitable for high-volume endpoints.
type SlidingWindowLog struct {
	spec  limiter.Spec
	rdb   *redis.Client
	clock limiter.Clock
}

func NewSlidingWindowLog(spec limiter.Spec, rdb *redis.Client, clock limiter.Clock) *SlidingWindowLog {
	return &SlidingWindowLog{spec: spec, rdb: rdb, clock: clock}
}

func (swl *SlidingWindowLog) Allow(ctx context.Context, key string, cost int64) (limiter.Decision, error) {
	if cost <= 0 {
		cost = 1
	}

	now := swl.clock.Now()
	nowMs := now.UnixMilli()
	windowMs := swl.spec.Window.Milliseconds()
	ttlMs := windowMs * 2

	redisKey := fmt.Sprintf("ratelimit:{%s}:swl", key)
	// A unique ID per call prevents ZADD from deduplicating two requests at the same millisecond.
	memberID := uuid.NewString()

	res, err := runScript(ctx, swl.rdb, slidingWindowLogScript,
		[]string{redisKey},
		swl.spec.Limit, windowMs, nowMs, cost, ttlMs, memberID,
	)
	if err != nil {
		return limiter.Decision{}, err
	}

	allowed := toInt64(res[0])
	remaining := toInt64(res[1])
	oldestMs := toInt64(res[2])

	var resetAfter time.Duration
	if oldestMs > 0 {
		resetAfter = time.Duration(oldestMs+windowMs-nowMs) * time.Millisecond
		if resetAfter < 0 {
			resetAfter = 0
		}
	} else {
		resetAfter = swl.spec.Window
	}

	if allowed == 1 {
		return limiter.Allowed(swl.spec.Limit, remaining, resetAfter), nil
	}
	return limiter.Denied(swl.spec.Limit, resetAfter, resetAfter), nil
}
