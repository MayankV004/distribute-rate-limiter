-- sliding_window_log.lua
-- Exact sliding window using a Redis sorted set of request timestamps.
--
-- KEYS[1]  = log key (ZSET, hash-tagged)
-- ARGV[1]  = limit       (int, max requests per window)
-- ARGV[2]  = window_ms   (int, window size in milliseconds)
-- ARGV[3]  = now_ms      (int, from Go — never call TIME here)
-- ARGV[4]  = cost        (int, slots to consume)
-- ARGV[5]  = ttl_ms      (int, key TTL)
-- ARGV[6]  = member_id   (string, unique per request — prevents ZADD collision at same ms)
--
-- Returns: {allowed (0|1), remaining (int), oldest_entry_ms (int)}
--
-- Memory grows with the number of requests in the window, not with key cardinality.
-- A hostile client that sustains exactly `limit` RPS can force O(limit) memory per key.

local limit      = tonumber(ARGV[1])
local window_ms  = tonumber(ARGV[2])
local now        = tonumber(ARGV[3])
local cost       = tonumber(ARGV[4])
local ttl        = tonumber(ARGV[5])
local member_id  = ARGV[6]

local window_start = now - window_ms

-- Evict entries that have fallen outside the sliding window.
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', window_start)

local count = redis.call('ZCARD', KEYS[1])

if count + cost <= limit then
    -- Add cost entries, each with a unique member to avoid ZADD score collisions.
    for i = 1, cost do
        redis.call('ZADD', KEYS[1], now, member_id .. ':' .. i)
    end
    redis.call('PEXPIRE', KEYS[1], ttl)

    local remaining = limit - count - cost
    local oldest_ms = 0
    local oldest = redis.call('ZRANGE', KEYS[1], 0, 0, 'WITHSCORES')
    if #oldest > 0 then
        oldest_ms = tonumber(oldest[2])
    end
    return {1, remaining, oldest_ms}
end

-- Denied: return when the oldest entry expires so the caller can set Retry-After.
local oldest_ms = 0
local oldest = redis.call('ZRANGE', KEYS[1], 0, 0, 'WITHSCORES')
if #oldest > 0 then
    oldest_ms = tonumber(oldest[2])
end
redis.call('PEXPIRE', KEYS[1], ttl)

return {0, 0, oldest_ms}

