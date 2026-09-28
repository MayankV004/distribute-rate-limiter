package limiter

import (
	"context"
	"errors"
	"time"
)

type Algorithm string

const (
	AlgorithmTokenBucket          Algorithm = "token_bucket"
	AlgorithmSlidingWindowLog     Algorithm = "sliding_window_log"
	AlgorithmSlidingWindowCounter Algorithm = "sliding_window_counter"
	AlgorithmLeakyBucket          Algorithm = "leaky_bucket"
)

type Store string

const (
	StoreLocal Store = "local"
	StoreRedis Store = "redis"
)

type Spec struct {
	Limit  int64
	Window time.Duration
	Burst  int64
}

func (s Spec) Rate() float64 {
	return float64(s.Limit) / s.Window.Seconds()
}

func (s Spec) Capacity() int64 {
	if s.Burst > 0 {
		return s.Burst
	}
	return s.Limit
}

type Decision struct {
	Allowed    bool
	Limit      int64
	Remaining  int64
	ResetAfter time.Duration
	RetryAfter time.Duration
}

func Allowed(limit, remaining int64, resetAfter time.Duration) Decision {
	if remaining < 0 {
		remaining = 0
	}
	return Decision{
		Allowed:    true,
		Limit:      limit,
		Remaining:  remaining,
		ResetAfter: resetAfter,
	}
}

func Denied(limit int64, retryAfter, resetAfter time.Duration) Decision {
	return Decision{
		Limit:      limit,
		ResetAfter: resetAfter,
		RetryAfter: retryAfter,
	}
}

// ErrUnavailable indicates the backing store is unreachable.
// Callers must apply their configured fail-open/fail-closed policy.
var ErrUnavailable = errors.New("rate limiter store unavailable")

type Limiter interface {
	Allow(ctx context.Context, key string, cost int64) (Decision, error)
}