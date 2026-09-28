-- token_bucket.lua
-- Atomically refills and consumes tokens from a token bucket stored as a Redis HASH.
--
-- KEYS[1]  = bucket key (hash-tagged, e.g. ratelimit:{user}:tb)
-- ARGV[1]  = capacity       (max tokens, float)
-- ARGV[2]  = refill_per_sec (tokens per second, float)
-- ARGV[3]  = now_ms         (unix milliseconds from Go — never call TIME here)
-- ARGV[4]  = cost           (tokens to consume, int)
-- ARGV[5]  = ttl_ms         (key TTL in milliseconds — every key must have one)
--
-- Returns: {allowed (0|1), remaining_tokens (int), reset_after_ms (int)}

local capacity = tonumber(ARGV[1])
local rate     = tonumber(ARGV[2])
local now      = tonumber(ARGV[3])
local cost     = tonumber(ARGV[4])
local ttl      = tonumber(ARGV[5])

local state  = redis.call('HMGET', KEYS[1], 'tokens', 'ts')
local tokens = tonumber(state[1]) or capacity
local ts     = tonumber(state[2]) or now

-- Lazily refill: compute tokens earned since the last request rather than
-- running a background ticker. Capped at capacity to prevent unbounded accumulation.
local elapsed_ms = math.max(0, now - ts)
tokens = math.min(capacity, tokens + (elapsed_ms * rate / 1000.0))

local allowed = 0
local reset_after_ms = math.ceil((capacity - tokens) / rate * 1000)

if tokens >= cost then
    tokens         = tokens - cost
    allowed        = 1
    reset_after_ms = math.ceil((capacity - tokens) / rate * 1000)
end

redis.call('HSET', KEYS[1], 'tokens', tokens, 'ts', now)
redis.call('PEXPIRE', KEYS[1], ttl)

return {allowed, math.floor(tokens), reset_after_ms}
