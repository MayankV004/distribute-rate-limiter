package distributed

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/streamliner/rate-limiter/internal/limiter"
)

// LeakyBucket is a Redis-backed leaky bucket enforcing a constant output rate.
// This implementation handles rate computation atomically in Lua; the Go layer
// handles any actual request queuing/holding per node.
type LeakyBucket struct {
	spec  limiter.Spec
	rdb   *redis.Client
	clock limiter.Clock
}

func NewLeakyBucket(spec limiter.Spec, rdb *redis.Client, clock limiter.Clock) *LeakyBucket {
	return &LeakyBucket{spec: spec, rdb: rdb, clock: clock}
}

func (lb *LeakyBucket) Allow(ctx context.Context, key string, cost int64) (limiter.Decision, error) {
	if cost <= 0 {
		cost = 1
	}

	now := lb.clock.Now()
	nowMs := now.UnixMilli()
	capacity := float64(lb.spec.Capacity())
	drainPerSec := lb.spec.Rate()
	ttlMs := lb.spec.Window.Milliseconds() * 2

	redisKey := fmt.Sprintf("ratelimit:{%s}:lb", key)

	res, err := runScript(ctx, lb.rdb, leakyBucketScript,
		[]string{redisKey},
		capacity, drainPerSec, nowMs, float64(cost), ttlMs,
	)
	if err != nil {
		return limiter.Decision{}, err
	}

	allowed := toInt64(res[0])
	remaining := toInt64(res[1])
	retryMs := toInt64(res[2])

	retryAfter := time.Duration(retryMs) * time.Millisecond
	resetAfter := time.Duration(float64(remaining) / drainPerSec * float64(time.Second))

	if allowed == 1 {
		return limiter.Allowed(lb.spec.Limit, remaining, resetAfter), nil
	}
	return limiter.Denied(lb.spec.Limit, retryAfter, resetAfter), nil
}
