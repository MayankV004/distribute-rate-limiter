-- sliding_window_counter.lua
-- Approximate sliding window using two fixed-slot counters with weighted blending.
-- O(1) memory per key; bounded error of at most one window's worth of requests.
--
-- KEYS[1]  = current window counter key  (STRING, hash-tagged)
-- KEYS[2]  = previous window counter key (STRING, same hash tag — same Cluster slot)
-- ARGV[1]  = limit             (int)
-- ARGV[2]  = window_ms         (int)
-- ARGV[3]  = now_ms            (int, from Go)
-- ARGV[4]  = cost              (int)
-- ARGV[5]  = window_start_ms   (int, floor(now_ms / window_ms) * window_ms)
-- ARGV[6]  = ttl_ms            (int)
--
-- Returns: {allowed (0|1), remaining (int)}
--
-- The approximation: weight the previous window by the fraction of it still
-- overlapping the current rolling window. At 30% through the current window,
-- 70% of the previous window's requests are still "in range".

local limit            = tonumber(ARGV[1])
local window_ms        = tonumber(ARGV[2])
local now              = tonumber(ARGV[3])
local cost             = tonumber(ARGV[4])
local window_start_ms  = tonumber(ARGV[5])
local ttl              = tonumber(ARGV[6])

local prev = tonumber(redis.call('GET', KEYS[2])) or 0
local curr = tonumber(redis.call('GET', KEYS[1])) or 0

local elapsed  = now - window_start_ms
local overlap  = (window_ms - elapsed) / window_ms
local estimate = prev * overlap + curr

if estimate + cost <= limit then
    redis.call('INCRBY', KEYS[1], cost)
    -- TTL is 2× window so the previous-window key stays alive while it is still needed.
    redis.call('PEXPIRE', KEYS[1], ttl)
    redis.call('PEXPIRE', KEYS[2], ttl)

    local remaining = math.floor(limit - estimate - cost)
    return {1, remaining}
end

return {0, 0}

