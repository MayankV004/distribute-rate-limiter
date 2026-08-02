# Distributed Rate Limiter — Full Learning Guide

A complete walkthrough of what we built, why each decision was made, and how the code flows from an incoming HTTP request to a `429 Too Many Requests` or a forwarded response.

---

## The Big Picture: What Problem Are We Solving?

Imagine you run an API. You want to allow each user only 100 requests per minute. Easy when you have one server — just count in memory.

But what happens when you deploy **three** servers behind a load balancer? Each server has its own memory. User A hits server 1 — 1 request counted. Same user hits server 2 — a fresh counter starts at 0. They can now make 300 requests per minute while each server thinks they only made 100.

**The core problem: distributed systems can't share memory, so they can't share counters.**

The solution requires:
1. A **shared store** (Redis) that all servers talk to
2. **Atomic operations** so two servers checking the same counter at the same moment don't both "win"
3. A clean **interface** so the local and distributed versions are interchangeable

That last point is the elegance of this project — you can switch from local to distributed with one config change, and the proof is in the test suite.

---

## The Architecture at a Glance

```
HTTP Request
    │
    ▼
[chi Router] ← just routes incoming URLs
    │
    ▼
[RateLimit Middleware] ← the gatekeeper
    │   1. Who is this? (extractor)
    │   2. What limiter applies? (resolver)
    │   3. Are they allowed? (limiter.Allow)
    │
    ├── DENIED → 429 Too Many Requests
    │
    └── ALLOWED
            │
            ▼
    [ReverseProxy] → Backend Server (localhost:9000)
```

---

## Layer 1: The Contract (`internal/limiter/limiter.go`)

**Why does this file exist?**

Before writing a single line of algorithm code, we defined the *shape* of what a rate limiter is. This is the most important file in the entire project.

```go
type Limiter interface {
    Allow(ctx context.Context, key string, cost int64) (Decision, error)
}
```

**Why this exact signature?**

- `ctx context.Context` — lets us cancel the call if Redis takes too long. Without this, a slow Redis hangs your entire gateway.
- `key string` — the identity of the caller. Could be an IP, an API key, a user ID. The limiter doesn't care which — it just tracks by this string.
- `cost int64` — a search endpoint might cost 2 tokens; a cheap health check costs 1. One parameter change, no interface breakage.
- Returns `(Decision, error)` — **these are two different things!** A `Decision` with `Allowed: false` means "I checked the quota and you're over it." An `error` means "I couldn't check at all (Redis is down)." Callers handle these differently.

```go
type Decision struct {
    Allowed    bool
    Limit      int64         // your configured quota — for the X-RateLimit-Limit header
    Remaining  int64         // how many requests you have left
    ResetAfter time.Duration // when your quota fully refills
    RetryAfter time.Duration // how long until this specific request would succeed
}
```

**`ErrUnavailable`** — the sentinel error the whole fail-open/fail-closed system is built on:
```go
var ErrUnavailable = errors.New("rate limiter store unavailable")
```

When the middleware receives this error, it applies the route's configured policy: either allow the request through (fail-open, availability wins) or reject it (fail-closed, safety wins). That decision is *outside* the limiter — which is correct design.

---

## Layer 2: The Clock (`internal/limiter/clock.go`)

**Why abstract the clock?**

Every rate-limiting algorithm is time-dependent. If tests used `time.Sleep`, your test suite would be slow and flaky. A fake clock lets you "advance time" instantly in tests.

```go
type Clock interface {
    Now() time.Time
}

type FakeClock struct { ... }
func (f *FakeClock) Advance(d time.Duration) { ... } // only in tests
```

In production, the gateway passes a `realClock{}` that calls `time.Now()`. In tests, a `FakeClock` is used. **The algorithm code never knows the difference** — it only calls `clock.Now()`.

This is why the test for "user gets a new quota after 20 seconds" is instant: we call `clock.Advance(20 * time.Second)` and the next `Allow()` call sees a time 20 seconds later without any actual sleeping.

---

## Layer 3: The Four Local Algorithms (`internal/limiter/local/`)

These are the in-memory implementations. They use a mutex to protect state — correct inside one process, but useless across multiple servers.

### How State Is Stored (`shardedmap.go`)

All four algorithms need to store state per user (e.g., "user A has 2 tokens left"). Rather than one big map with one lock (which would serialize all requests), we use a **ShardedMap**: 256 sub-maps, each with its own lock. The user's key is hashed to determine which shard they land in. This reduces lock contention by ~256×.

### Token Bucket (`tokenbucket.go`)

**Mental model:** A bucket that holds tokens. Each request costs tokens. Tokens refill at a constant rate.

**Why lazy refill?** We don't run a background goroutine ticking every millisecond. Instead, when a request arrives, we compute: "how much time has passed since the last request? Add that many tokens." This is called *lazy evaluation* — compute when needed, not constantly.

```
tokens = min(capacity, tokens + elapsed_seconds × rate)
```

**Data stored per user:** `{tokens float64, lastUpdated time.Time}`

**Trade-off:** Allows bursts up to capacity. Good for per-user API quotas where occasional bursts are acceptable.

### Sliding Window Log (`slidingwindowlog.go`)

**Mental model:** Store the exact timestamp of every request. On each new request, throw away timestamps older than the window, count what's left, and decide.

**Why exact?** Never over-admits. If limit=10 and 10 requests happened exactly 5 seconds ago in a 10-second window, request 11 is denied. No approximation.

**Data stored per user:** A slice of `time.Time` values.

**Trade-off:** Memory grows with the number of requests. A hostile client that sustains exactly `limit` RPS forces O(limit) memory per key. That's why this is not the default.

### Sliding Window Counter (`slidingwindowcounter.go`)

**Mental model:** Instead of individual timestamps, keep two counters — one for the current fixed time slot, one for the previous. Blend them.

```
estimate = prev_count × (1 - elapsed/window) + curr_count
```

If you're 30% into the current 10-second window, 70% of the previous window's requests are still "in" your rolling window. This is an **approximation** but the error is bounded and the memory is O(1).

**Data stored per user:** Two integers (curr count, prev count).

**Why this is the production default:** Fixed memory, close enough for almost everything, no adversarial memory blowup risk.

### Leaky Bucket (`leakybucket.go`)

**Mental model:** A bucket with a hole. Water (requests) drains at a constant rate. New requests add water. If adding water would overflow, deny.

**Key difference from token bucket:** Token bucket refills *capacity you can spend*. Leaky bucket drains *load you've already accepted*. The effect is a smooth, constant output rate regardless of burst — useful when your downstream service is fragile and can't handle spikes.

---

## Layer 4: The ShardedMap in Detail

This is worth understanding because it's a classic concurrency pattern.

**The problem:** You have millions of users. One `sync.Mutex` for all of them means every request, regardless of user, waits for every other request.

**The solution:** 256 independent maps, each with its own mutex.

```
key "user_a" → hash → shard 17 → lock shard 17's mutex only
key "user_b" → hash → shard 42 → lock shard 42's mutex only
```

Users in different shards can proceed in parallel. Only users that hash to the same shard compete. With 256 shards and uniformly distributed keys, contention drops to 1/256 of what it would be with a single lock.

**The janitor goroutine:** The ShardedMap starts a background goroutine that periodically walks through all shards and evicts entries that haven't been seen for a while. Without this, every unique IP that ever hit the gateway would accumulate in memory forever.

---

## Layer 5: The Gateway Shell (`cmd/gateway/main.go`)

This is where everything gets wired together for the first time.

```go
// 1. Create the rate limiter
realClock := realClock{}
spec := limiter.Spec{Limit: 2, Window: 10 * time.Second, Burst: 2}
myLimiter := local.NewTokenBucket(spec, realClock)

// 2. Identity extraction — WHO is making the request?
extractor := func(r *http.Request) string {
    key := r.Header.Get("X-API-Key")
    if key != "" { return key }

    // Strip the port — r.RemoteAddr is "1.2.3.4:PORT", we want just "1.2.3.4"
    host, _, _ := net.SplitHostPort(r.RemoteAddr)
    return host
}

// 3. Limiter resolution — WHICH limiter applies to this route?
resolver := func(r *http.Request) limiter.Limiter {
    return myLimiter // for now: same limiter for all routes
}

// 4. Wire it all into the router
r := chi.NewRouter()
r.Use(middleware.RateLimit(extractor, resolver))
r.Handle("/*", proxy)
```

**Why we had the bug (and the fix):** `r.RemoteAddr` in Go is `"1.2.3.4:52341"` — IP *plus* TCP ephemeral port. Every new `curl` command opens a new TCP connection with a different port. The old code used the full string as the identity key, so every request looked like a different user. `net.SplitHostPort` strips the port so all requests from the same IP share one bucket.

---

## Layer 6: The Middleware (`internal/middleware/ratelimit.go`)

This is the actual gatekeeper. It follows Go's standard middleware pattern — a function that wraps a `http.Handler` and returns a new `http.Handler`.

```
func RateLimit(extract, resolve) func(http.Handler) http.Handler
```

The flow on every request:

```
1. extract(r)          → "192.168.1.1" (the user's identity)
2. resolve(r)          → myLimiter    (which limiter to use)
3. l.Allow(ctx, key, 1) → Decision{Allowed: true, Remaining: 1, ...}
4. Set X-RateLimit-* headers on the response
5a. If Allowed:  call next.ServeHTTP(w, r)  ← pass to proxy
5b. If Denied:   http.Error(w, "Too Many Requests", 429)
```

**Why set headers even when denied?** Because the client needs to know how long to wait before retrying. `Retry-After: 8` tells an API client "come back in 8 seconds." Without this, clients would hammer the endpoint over and over, making the problem worse.

**Fail-open on error:** If `Allow()` returns an `error` (e.g., Redis is down), the current code lets the request through (`next.ServeHTTP`). This is the *fail-open* policy. The plan includes a per-route config to make this fail-closed for sensitive routes.

---

## Layer 7: Redis + Lua (Phase 3)

This is where the project gets genuinely interesting.

### Why Redis? Why Not Just Share Memory?

Go processes have separate memory spaces. Three gateway replicas = three completely independent memory heaps. You cannot share a Go `sync.Mutex` across processes. Redis is the shared external store that all three can talk to.

### The Atomicity Problem

Suppose two gateway replicas both want to check and decrement a user's counter at the same time:

```
Replica 1: GET counter → 1
Replica 2: GET counter → 1   (same value, read before either decremented)
Replica 1: if 1 <= limit: DECR → 0  ✓ allowed
Replica 2: if 1 <= limit: DECR → 0  ✓ allowed  (WRONG — both allowed!)
```

This is a classic **check-then-act race condition**. The solution: make the check and the act a single atomic operation.

### Why Lua, Not Transactions?

Redis does have transactions (`MULTI`/`EXEC` with `WATCH`). But under contention (many clients hitting the same key), `WATCH` causes frequent transaction conflicts and retries, which degrades throughput non-linearly.

**Lua scripts** run to completion without any interleaving. Redis is single-threaded in its command processing, so while a Lua script runs, no other client can modify any keys the script is touching. This gives you atomicity without retries.

### The Three Rules (Every Lua Script)

```lua
-- RULE 1: Pass now_ms from Go, never call redis.call('TIME')
-- Why: Non-deterministic commands break Redis replication and make tests impossible.
-- now_ms comes from clock.Now().UnixMilli() in Go.

-- RULE 2: Always PEXPIRE every key
-- Why: Every unique user who ever hits the gateway creates a Redis key.
-- Without a TTL, those keys accumulate forever. Redis runs out of memory.

-- RULE 3: Single hash-tagged key
-- Why: For Redis Cluster, all keys in one script must hash to the same slot.
-- "ratelimit:{user_a}:tb" — only the {user_a} part determines the slot.
```

### `scripts.go` — How Lua Gets Into Go

```go
//go:embed lua/*.lua
var luaFS embed.FS
```

`go:embed` is a compiler directive. At build time, Go reads every `.lua` file in the `lua/` directory and bakes the contents directly into the binary. No runtime file I/O, no "file not found" errors in production.

```go
var tokenBucketScript = redis.NewScript(string(src))
```

`redis.NewScript` registers the script. When you call `.Run()`, it first tries `EVALSHA` (sends only a 40-character SHA hash of the script — very fast). If Redis says it doesn't know that SHA, it falls back to `EVAL` (sends the full script). After that first run, the script is cached on Redis and EVALSHA works forever. This is the standard "lazy registration" pattern.

### Token Bucket Lua Walkthrough

```lua
-- Read current state (or defaults for a new key)
local state  = redis.call('HMGET', KEYS[1], 'tokens', 'ts')
local tokens = tonumber(state[1]) or capacity   -- nil → full bucket
local ts     = tonumber(state[2]) or now

-- Lazy refill: how many tokens did we earn since last call?
local elapsed_ms = math.max(0, now - ts)
tokens = math.min(capacity, tokens + (elapsed_ms * rate / 1000.0))

-- The atomic check-and-consume
if tokens >= cost then
    tokens = tokens - cost
    allowed = 1
end

-- Always persist state and TTL
redis.call('HSET', KEYS[1], 'tokens', tokens, 'ts', now)
redis.call('PEXPIRE', KEYS[1], ttl)

return {allowed, math.floor(tokens), reset_after_ms}
```

The Go side (`tokenbucket.go`) then:
1. Builds the hash-tagged key: `fmt.Sprintf("ratelimit:{%s}:tb", key)`
2. Converts `clock.Now()` to milliseconds
3. Calls `runScript(ctx, rdb, tokenBucketScript, keys, capacity, rate, nowMs, cost, ttlMs)`
4. Parses `[]any{int64, int64, int64}` → constructs `limiter.Decision`

### The `runScript` Helper

```go
func runScript(ctx, rdb, script, keys, args...) ([]any, error) {
    res, err := script.Run(ctx, rdb, keys, args...).Slice()
    if err != nil {
        return nil, limiter.ErrUnavailable  // hide Redis internals from callers
    }
    return res, nil
}
```

Every Redis error — connection refused, timeout, wrong type — maps to `limiter.ErrUnavailable`. The middleware then applies fail-open or fail-closed. Callers don't import go-redis. This is the **anti-corruption layer** pattern: keep infrastructure details from leaking into domain logic.

### Sliding Window Counter — Two Keys, Same Slot

```go
currKey := fmt.Sprintf("ratelimit:{%s}:swc:%d", key, windowStartMs)
prevKey := fmt.Sprintf("ratelimit:{%s}:swc:%d", key, windowStartMs-windowMs)
```

Notice both keys contain `{key}` in braces. In Redis Cluster, the hash function only looks at the part inside `{}`. So both keys hash to the same cluster slot, which means the Lua script can access both in one atomic call without a `CROSSSLOT` error.

---

## Layer 8: Integration Tests (`test/integration/`)

### Why Real Redis?

You could mock the Redis client, but then you're testing Go-mock interactions, not Redis behaviour. The Lua scripts themselves run on the Redis engine. A mock wouldn't catch:
- Lua syntax errors
- Incorrect ZSET member ordering
- TTL expiry behaviour
- Type errors (using HMGET on a STRING key)

`testcontainers-go` starts a real Docker Redis container for each test run, gives it a random port, runs all the tests against it, then tears it down. The test suite is self-contained.

### `TestGlobalLimit` — The Proof

```go
l1 := distributed.NewTokenBucket(spec, rdb1, clock)  // "replica 1"
l2 := distributed.NewTokenBucket(spec, rdb2, clock)  // "replica 2"

// 20 requests, alternating between replicas
for i := 0; i < 20; i++ {
    if i%2 == 0 { l1.Allow(...) } else { l2.Allow(...) }
}

// MUST be exactly 10, not 20
if allowed != 10 { t.Error(...) }
```

If you ran this test with the *local* implementations, you'd get 20 (each replica has its own in-memory counter). With Redis, you get exactly 10. **This single test is the entire thesis of the project.**

---

## The Full Request Journey (End to End)

Here is what happens from the moment `curl http://localhost:8080/api/v1/test` hits Enter:

```
1. [OS] Opens TCP connection to :8080

2. [chi Router] Receives GET /api/v1/test
   → Matches the "/*" wildcard route
   → Before executing the handler, runs all Use() middleware

3. [RateLimit Middleware] Intercepts the request
   → extractor("X-API-Key" header)  → not found
   → net.SplitHostPort(r.RemoteAddr) → "::1" (IPv6 localhost)

4. [resolver] Returns myLimiter (the TokenBucket)

5. [TokenBucket.Allow(ctx, "::1", 1)]
   LOCAL mode:
     → lock shard for "::1"
     → read tokenBucketState{tokens: 2.0, lastUpdated: T}
     → elapsed = now - T → refill tokens
     → tokens >= 1 → deduct → tokens = 1.0
     → unlock
     → return Decision{Allowed: true, Remaining: 1, ...}

   REDIS mode (Phase 3):
     → clock.Now().UnixMilli() → nowMs
     → key = "ratelimit:{::1}:tb"
     → EVALSHA <sha> 1 ratelimit:{::1}:tb 2 0.2 nowMs 1 20000
     → Lua runs atomically on Redis
     → returns {1, 1, 8000}
     → Decision{Allowed: true, Remaining: 1, ResetAfter: 8s}

6. [Middleware] Sets response headers:
   X-RateLimit-Limit: 2
   X-RateLimit-Remaining: 1
   X-RateLimit-Reset: 8

7. [Middleware] Calls next.ServeHTTP (passes to ReverseProxy)

8. [ReverseProxy] Rewrites request → POST http://localhost:9000/api/v1/test
   Adds X-Forwarded-For: ::1

9. [Backend] Returns 200 OK + JSON body

10. [ReverseProxy] Streams response back through the middleware

11. [Client] Receives 200 OK with all the rate-limit headers
```

On the 3rd request (within the same 10-second window):

```
5. [TokenBucket.Allow(ctx, "::1", 1)]
   → tokens = 0.0 after previous requests
   → tokens < 1 → DENIED
   → return Decision{Allowed: false, Remaining: 0, RetryAfter: 8s}

6. [Middleware] Sets Retry-After: 8
   http.Error(w, "Too Many Requests", 429)
   → Request STOPS here. Proxy and backend never see it.
```

---

## Layer 9: Config-driven Routes & Tiers (Phase 4)

This phase shifted the hardcoded gateway into a dynamic, configuration-driven server.

**Key additions:**
- **Per-Route Limits:** Different API routes (e.g., `/api/v1/search` vs `/api/v1/*`) can have entirely different algorithms, storage backends, and costs.
- **Tier Resolution:** A `StaticResolver` maps API keys to tiers (e.g., "free", "pro"), and limits are dynamically evaluated based on the caller's tier. A `CachedResolver` provides a fast lookup mechanism using a concurrent `sync.Map` with TTLs.
- **Zero-Downtime Reloads:** A background goroutine listens for `SIGHUP` OS signals. When received, it re-reads the config file, rebuilds the entire routing tree and all rate limiters, and atomically swaps the active handler using `atomic.Pointer[http.Handler]`. Active requests continue undisturbed, and subsequent requests instantly use the new limits.

---

## Layer 10: Circuit Breaker & Fallbacks (Phase 5)

Distributed systems fail. If Redis goes down and every request blocks for 5 seconds waiting for a network timeout, the gateway will quickly exhaust all available goroutines and crash.

This phase introduced a **Circuit Breaker** to protect the gateway:
- **Fast-fail:** Once the `error_ratio` exceeds the configured threshold, the breaker trips to the `Open` state. Subsequent requests are rejected *immediately* (under 1 millisecond) instead of stalling.
- **Self-healing:** After an `open_duration` elapses, it enters `HalfOpen` to allow probe requests through. If they succeed, it closes the circuit and resumes normal operations.
- **Fail-Open vs Fail-Closed:** We can't always just return a 503 when Redis is down. For critical routes, we might want to prioritize availability over rate limiting (`fallback: open`). For expensive routes (like heavy DB queries), we prioritize protection (`fallback: closed`). The circuit breaker combined with the fallback policy enables this gracefully.

---

## Layer 11: Observability (Phase 6)

You cannot manage what you do not measure. In this phase, we instrumented the gateway with Prometheus metrics to gain deep visibility into its operations:

1. **Custom Histograms:** We added sub-millisecond histogram buckets (`0.0005, 0.001, 0.002, 0.005, ...`). The default Prometheus buckets start at 5ms, which would render fast Redis operations (typically 1-2ms) invisible.
2. **Circuit Breaker States:** Breakers are tracked via Gauges (`ratelimit_breaker_state`). This allows alerting on `Open` or `HalfOpen` states, signaling that the upstream store is struggling.
3. **Decision & Fallback Tracking:** By recording `ratelimit_decisions_total` and `ratelimit_fallback_total`, we can visualize the exact rate of `allowed`, `denied`, and fallback actions applied in real-time.
4. **Grafana Dashboards:** We provisioned Grafana with a structured dashboard mapping these metrics. 

---

## Layer 12: L1 Cache (Phase 7)

Redis is extremely fast, but a network round-trip is still a network round-trip. At ultra-high scale, even 1-2ms of latency per request can become a bottleneck or incur high cloud network costs. 

To mitigate this, we implemented an optional **in-process L1 Cache**:
1. **Safety First:** The cache *never* caches deny decisions. If a client was denied, they would be locked out past the end of their penalty window if the cache was stale.
2. **Boundary Accuracy:** The cache only engages when the remaining quota is comfortably above zero (e.g. `Remaining > 10% of Limit`). This ensures precision exactly when it matters—right at the limit boundary.
3. **The Honest Cost:** We ran a benchmark firing 10,000 concurrent requests against a limit of 100. A 5ms cache reduced Redis calls by 1.8%, but allowed 180 extra requests to slip through (180% over-admission). This proves the fundamental distributed systems tradeoff: **you can trade strict accuracy for latency, but you must measure the cost.**

---

## Layer 13: Load Testing & Benchmarks (Phase 8)

The ultimate test of a distributed system is its performance and correctness under concurrent load. In this final phase, we subjected the system to load testing using Vegeta across our 3-node Nginx topology.

**1. The Correctness Validation:**
Running 1000 requests over 2 seconds against a route with a limit of 100 (and a cost of 2 per request) should yield exactly 50 successful requests. We proved that:
- The `redis` store achieves this perfectly (50 allowed, 950 denied) because it relies on atomic Lua scripts globally enforcing the quota.
- A critical bug was caught during this phase: our initial Lua script used `INCR` instead of `INCRBY cost`. Benchmarking empirically surfaced this when exactly 100 requests were allowed instead of 50.

**2. The "Honest Cost" of Distribution:**
By comparing the latency profiles of the `local` store versus the `redis` store at 1000 RPS, we measured the exact overhead of leaving the Go process to check quota over the network:
- **Local Store p50 / p99:** 415µs / 956µs
- **Redis Store p50 / p99:** 554µs / 1.15ms

The distributed Redis store adds roughly **150-200 microseconds** of latency per request. This confirms that while the network hop isn't free, encapsulating the entire check-and-decrement logic into a single Lua `EVALSHA` round-trip keeps the overhead negligible for almost all practical use cases, fully justifying the trade-off for global quota accuracy.

---

*This concludes the Distributed Rate Limiter Learning Guide. The system is now a robust, observable, and fully distributed gateway capable of scaling horizontally while maintaining strict API quotas.*
