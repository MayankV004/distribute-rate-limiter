package registry

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	
	"github.com/streamliner/rate-limiter/internal/breaker"
	"github.com/streamliner/rate-limiter/internal/cache"
	"github.com/streamliner/rate-limiter/internal/config"
	"github.com/streamliner/rate-limiter/internal/limiter"
	"github.com/streamliner/rate-limiter/internal/metrics"
	"github.com/streamliner/rate-limiter/internal/limiter/distributed"
	"github.com/streamliner/rate-limiter/internal/limiter/local"
)

// New is the factory function that instantiates a Limiter based on strings
// parsed from the configuration. This makes swapping between local and Redis
// a pure configuration change (Phase 8 requirement).
func New(routePattern string, algorithm string, store string, spec limiter.Spec, rdb *redis.Client, clock limiter.Clock, breakerCfg config.BreakerConfig, cmdTimeout time.Duration, l1Cfg config.L1CacheConfig) (limiter.Limiter, error) {
	switch store {
	case "local":
		switch algorithm {
		case "token_bucket":
			return local.NewTokenBucket(spec, clock), nil
		case "sliding_window_log":
			return local.NewSlidingWindowLog(spec, clock), nil
		case "sliding_window_counter":
			return local.NewSlidingWindowCounter(spec, clock), nil
		case "leaky_bucket":
			return local.NewLeakyBucket(spec, clock), nil
		default:
			return nil, fmt.Errorf("unknown algorithm %q for store %q", algorithm, store)
		}

	case "redis":
		// A Redis client is mandatory for the distributed store.
		if rdb == nil {
			return nil, fmt.Errorf("redis store requested but rdb is nil")
		}

		var l limiter.Limiter
		switch algorithm {
		case "token_bucket":
			l = distributed.NewTokenBucket(spec, rdb, clock)
		case "sliding_window_log":
			l = distributed.NewSlidingWindowLog(spec, rdb, clock)
		case "sliding_window_counter":
			l = distributed.NewSlidingWindowCounter(spec, rdb, clock)
		case "leaky_bucket":
			l = distributed.NewLeakyBucket(spec, rdb, clock)
		default:
			return nil, fmt.Errorf("unknown algorithm %q for store %q", algorithm, store)
		}
		
		wrapped := &breakerWrapper{
			b:         breaker.New(routePattern, breakerCfg),
			inner:     l,
			timeout:   cmdTimeout,
			algorithm: algorithm,
		}
		
		return cache.Wrap(wrapped, l1Cfg, clock), nil

	default:
		return nil, fmt.Errorf("unknown store %q", store)
	}
}

type breakerWrapper struct {
	b         *breaker.Breaker
	inner     limiter.Limiter
	timeout   time.Duration
	algorithm string
}

func (bw *breakerWrapper) Allow(ctx context.Context, key string, cost int64) (limiter.Decision, error) {
	var dec limiter.Decision
	
	start := time.Now()
	
	err := bw.b.Do(ctx, func(innerCtx context.Context) error {
		// Enforce hard per-call deadline on the store call
		timeoutCtx, cancel := context.WithTimeout(innerCtx, bw.timeout)
		defer cancel()

		var e error
		dec, e = bw.inner.Allow(timeoutCtx, key, cost)
		return e
	})
	
	metrics.ObserveStoreLatency(bw.algorithm, start)
	if err != nil {
		metrics.StoreErrorsTotal.WithLabelValues("unavailable").Inc()
	}
	
	return dec, err
}
