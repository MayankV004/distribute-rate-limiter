-- leaky_bucket.lua
-- Leaky bucket: constant drain rate, bursts absorbed up to capacity.
-- Rate computation only — request queuing/holding is the Go layer's responsibility.
--
-- KEYS[1]  = bucket key (HASH: fields "level" and "last_drain_ms")
-- ARGV[1]  = capacity       (float, max water level)
-- ARGV[2]  = drain_per_sec  (float, = limit / window_seconds)
-- ARGV[3]  = now_ms         (int, from Go — never call TIME here)
-- ARGV[4]  = cost           (float, water added by this request)
-- ARGV[5]  = ttl_ms         (int)
--
-- Returns: {allowed (0|1), remaining (int), retry_after_ms (int)}

local capacity      = tonumber(ARGV[1])
local drain_per_sec = tonumber(ARGV[2])
local now           = tonumber(ARGV[3])
local cost          = tonumber(ARGV[4])
local ttl           = tonumber(ARGV[5])

local state         = redis.call('HMGET', KEYS[1], 'level', 'last_drain_ms')
local level         = tonumber(state[1]) or 0
local last_drain_ms = tonumber(state[2]) or now

-- Drain the bucket based on elapsed time since last call.
local elapsed_s = math.max(0, (now - last_drain_ms) / 1000.0)
level = math.max(0, level - elapsed_s * drain_per_sec)

local retry_after_ms = 0

if level + cost <= capacity then
    level = level + cost
    redis.call('HSET', KEYS[1], 'level', level, 'last_drain_ms', now)
    redis.call('PEXPIRE', KEYS[1], ttl)

    local remaining = math.floor(capacity - level)
    return {1, remaining, 0}
end

-- Time until enough water drains to accept `cost` more units.
local overflow   = (level + cost) - capacity
retry_after_ms   = math.ceil(overflow / drain_per_sec * 1000)
local remaining  = math.floor(capacity - level)

redis.call('HSET', KEYS[1], 'level', level, 'last_drain_ms', now)
redis.call('PEXPIRE', KEYS[1], ttl)

return {0, remaining, retry_after_ms}

