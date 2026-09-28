package distributed

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/streamliner/rate-limiter/internal/limiter"
)

// TokenBucket is a Redis-backed token bucket. Multiple gateway replicas sharing
// the same Redis instance will see the same token count and enforce one global limit.
type TokenBucket struct {
	spec  limiter.Spec
	rdb   *redis.Client
	clock limiter.Clock
}

func NewTokenBucket(spec limiter.Spec, rdb *redis.Client, clock limiter.Clock) *TokenBucket {
	return &TokenBucket{spec: spec, rdb: rdb, clock: clock}
}

func (tb *TokenBucket) Allow(ctx context.Context, key string, cost int64) (limiter.Decision, error) {
	if cost <= 0 {
		cost = 1
	}

	now := tb.clock.Now()
	nowMs := now.UnixMilli()
	capacity := float64(tb.spec.Capacity())
	rate := tb.spec.Rate()
	// TTL is 2× window so the key survives a full quiet period and refills correctly on return.
	ttlMs := tb.spec.Window.Milliseconds() * 2

	redisKey := fmt.Sprintf("ratelimit:{%s}:tb", key)

	res, err := runScript(ctx, tb.rdb, tokenBucketScript,
		[]string{redisKey},
		capacity, rate, nowMs, cost, ttlMs,
	)
	if err != nil {
		return limiter.Decision{}, err
	}

	allowed := toInt64(res[0])
	remaining := toInt64(res[1])
	resetMs := toInt64(res[2])
	resetAfter := time.Duration(resetMs) * time.Millisecond

	if allowed == 1 {
		return limiter.Allowed(tb.spec.Limit, remaining, resetAfter), nil
	}
	return limiter.Denied(tb.spec.Limit, resetAfter, resetAfter), nil
}

// toInt64 safely converts the interface{} values returned by go-redis Lua scripts.
func toInt64(v any) int64 {
	if n, ok := v.(int64); ok {
		return n
	}
	return 0
}
