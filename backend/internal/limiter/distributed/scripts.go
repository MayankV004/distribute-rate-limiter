// Package distributed provides Redis-backed rate limiter implementations.
// All four algorithms are implemented as Lua scripts that run atomically on
// the Redis server, avoiding check-then-act races across multiple gateway replicas.
package distributed

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/streamliner/rate-limiter/internal/limiter"
)

//go:embed lua/*.lua
var luaFS embed.FS

var (
	tokenBucketScript          *redis.Script
	slidingWindowLogScript     *redis.Script
	slidingWindowCounterScript *redis.Script
	leakyBucketScript          *redis.Script
)

func init() {
	tokenBucketScript = loadScript("lua/token_bucket.lua")
	slidingWindowLogScript = loadScript("lua/sliding_window_log.lua")
	slidingWindowCounterScript = loadScript("lua/sliding_window_counter.lua")
	leakyBucketScript = loadScript("lua/leaky_bucket.lua")
}

func loadScript(path string) *redis.Script {
	src, err := luaFS.ReadFile(path)
	if err != nil {
		// Lua files are embedded at compile time; a missing file is a build error, not a runtime one.
		panic(fmt.Sprintf("distributed: failed to load embedded Lua script %q: %v", path, err))
	}
	return redis.NewScript(string(src))
}

// runScript executes a pre-loaded Redis Lua script and returns the raw result slice.
// Redis client errors and deadline exceeded are both mapped to limiter.ErrUnavailable
// so callers apply fail-open/fail-closed policy without knowing the Redis client internals.
func runScript(ctx context.Context, rdb *redis.Client, script *redis.Script, keys []string, args ...any) ([]any, error) {
	res, err := script.Run(ctx, rdb, keys, args...).Slice()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return nil, limiter.ErrUnavailable
		}
		// Any other Redis error (connection refused, WRONGTYPE, etc.) is also unavailable.
		return nil, limiter.ErrUnavailable
	}
	return res, nil
}
