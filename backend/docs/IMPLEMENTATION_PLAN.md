# Distributed Rate Limiter + API Gateway — Implementation Plan

## 1. What we are building

A stateless Go gateway that sits in front of backend APIs. For every request it
extracts an identity (IP / API key / JWT subject), asks a shared store whether
that identity is within its quota for the matched route, then either proxies the
request upstream or returns `429 Too Many Requests`. Multiple gateway replicas
must enforce **one** shared limit, not one limit each.

The interesting part is not the HTTP plumbing. It is that the same admission
decision has to be correct under (a) thousands of concurrent goroutines inside
one process and (b) N processes racing against one Redis key. Those are two
different problems with two different solutions, and the project implements both
so they can be compared side by side.

### Success criteria

| # | Criterion | How it is proven |
|---|---|---|
| S1 | Four algorithms implemented, swappable per route by config | Config change only, no recompile |
| S2 | Exactly `limit` requests pass when 1000 goroutines hit one key | `go test -race` assertion, not eyeballing |
| S3 | 3 gateway replicas enforce the global limit, not 3x | Phase 8 experiment A |
| S4 | Measured p50/p95/p99 added latency, published honestly | Phase 8 experiment B |
| S5 | Redis outage degrades per configured policy, never hangs | `docker compose pause redis` under load |

### Non-goals (explicitly out of scope)

- Auth/authz beyond reading an API key to resolve a tier. This is a limiter, not an identity provider.
- Running a real 6-node Redis Cluster locally. Keys are designed to be Cluster-safe (hash tags) and the migration path is documented, but the local stack is single-node Redis. The key design is the insight; standing up the cluster is ops busywork.
- Persisting rate-limit state across a full Redis loss. Counters are ephemeral by definition and every key carries a TTL.
- A UI. Grafana is the UI.

---

## 2. Tech stack

| Concern | Choice | Pinned version | Rationale |
|---|---|---|---|
| Language | Go | 1.22+ | Goroutines, `sync.Map`, `-race`, and `net/http/httputil.ReverseProxy` in stdlib |
| Router | `github.com/go-chi/chi/v5` | v5.0.12 | Thin, `net/http`-compatible middleware chain, no framework lock-in |
| Redis client | `github.com/redis/go-redis/v9` | v9.5.1 | `redis.Script` does EVALSHA with automatic EVAL fallback; Cluster support is the same API |
| Config | `github.com/knadh/koanf/v2` | v2.1.1 | YAML + env overrides, small dependency tree |
| Metrics | `github.com/prometheus/client_golang` | v1.19.0 | Native histograms give real percentiles |
| JWT | `github.com/golang-jwt/jwt/v5` | v5.2.1 | G2: signature verification for JWKS (RS256/ES256) and HMAC (HS256) |
| Integration tests | `github.com/testcontainers/testcontainers-go` | v0.30.0 | Runs real Redis so Lua scripts are tested, not mocked |
| Load generation | `github.com/tsenart/vegeta/v12` | v12.11.1 | Usable as CLI and library; prints latency percentiles directly |
| Fault injection | toxiproxy | v2.7.0 (Docker) | G7: adds controlled Redis latency/jitter for breaker calibration |
| Lint | `golangci-lint` | v1.57 | Config committed so CI and local agree |

Dependencies are added in the phase that first needs them, with exact versions,
not up-front. `go.mod` starts dependency-free.

---

## 3. Folder structure

```
rate-limiter/
├── cmd/
│   ├── gateway/
│   │   └── main.go                  # flag parsing, wire config → limiter → proxy → server
│   └── backend/
│       └── main.go                  # dummy upstream: echoes request, configurable latency/error rate
│
├── internal/                        # nothing here is importable by other modules, by design
│   ├── config/
│   │   ├── config.go                # Server, Redis, Breaker, Identity, Tier, Route structs
│   │   ├── load.go                  # YAML + env, defaults, SIGHUP reload
│   │   └── validate.go              # fail fast on startup: unknown algorithm, zero limit, bad CIDR
│   │
│   ├── identity/
│   │   ├── extractor.go             # ordered strategies: api_key → jwt_sub → ip
│   │   ├── clientip.go              # X-Forwarded-For parsing, trusted-proxy CIDR allowlist
│   │   ├── jwt.go                   # G2: JWTExtractor with JWKS/HMAC verification
│   │   └── key.go                   # canonical Redis key builder (hash-tagged)
│   │
│   ├── tier/                        # G1: identity → tier resolution (separate from identity)
│   │   ├── resolver.go              # Resolver interface + ErrUnknown
│   │   ├── static.go                # StaticResolver loaded from config api_keys block
│   │   └── cached.go                # CachedResolver (sync.Map TTL wrapper)
│   │
│   ├── limiter/
│   │   ├── limiter.go               # Limiter interface + Decision + Spec  ← THE CONTRACT
│   │   ├── registry.go              # (algorithm, store) → constructor
│   │   ├── clock.go                 # Clock interface + realClock + fakeClock (tests never sleep)
│   │   ├── local/                   # single-process implementations
│   │   │   ├── shardedmap.go        # 256-shard keyed-mutex map + eviction janitor
│   │   │   ├── tokenbucket.go
│   │   │   ├── slidingwindowlog.go
│   │   │   ├── slidingwindowcounter.go
│   │   │   ├── leakybucket.go
│   │   │   └── *_test.go            # table-driven + 1000-goroutine race tests
│   │   └── distributed/             # Redis-backed implementations
│   │       ├── scripts.go           # //go:embed *.lua, redis.Script handles, arg marshalling
│   │       ├── tokenbucket.go
│   │       ├── slidingwindowlog.go
│   │       ├── slidingwindowcounter.go
│   │       ├── leakybucket.go
│   │       └── lua/
│   │           ├── token_bucket.lua
│   │           ├── sliding_window_log.lua
│   │           ├── sliding_window_counter.lua
│   │           └── leaky_bucket.lua
│   │
│   ├── breaker/
│   │   └── breaker.go               # closed → open → half-open around every store call
│   ├── cache/
│   │   └── l1.go                    # optional short-TTL local allow-cache (Phase 7)
│   │
│   ├── middleware/
│   │   ├── ratelimit.go             # the main middleware: route match → limit → allow/deny
│   │   ├── headers.go               # X-RateLimit-*, Retry-After
│   │   ├── requestid.go
│   │   ├── logging.go               # structured slog, one line per request
│   │   └── recover.go
│   │
│   ├── proxy/
│   │   └── reverse.go               # ReverseProxy per upstream, timeouts, 502/504 mapping
│   ├── metrics/
│   │   └── metrics.go               # collectors + /metrics handler (separate port)
│   └── server/
│       └── server.go                # two listeners, /healthz, /readyz, graceful shutdown
│
├── configs/
│   ├── gateway.yaml                 # committed reference config
│   └── gateway.dev.yaml             # localhost, verbose logging, low limits for manual testing
│
├── deployments/
│   ├── Dockerfile                   # multi-stage, distroless final image
│   ├── docker-compose.yaml          # nginx LB → 3x gateway → redis + backend + prom + grafana
│   ├── nginx.conf
│   ├── prometheus.yml
│   └── grafana/
│       └── dashboard.json
│
├── test/
│   ├── integration/
│   │   ├── main_test.go             # testcontainers Redis, shared across cases
│   │   ├── algorithms_test.go       # same assertions as local/, run against real Lua
│   │   └── failover_test.go         # kill Redis mid-test, assert fallback policy
│   └── load/
│       ├── vegeta.sh                # ramping RPS, writes results to docs/BENCHMARKS.md
│       └── targets.txt
│
├── docs/
│   ├── IMPLEMENTATION_PLAN.md       # this file
│   ├── ARCHITECTURE.md              # diagram, request lifecycle, failure modes, Cluster path
│   ├── ALGORITHMS.md                # tradeoff table + state layout per algorithm
│   └── BENCHMARKS.md               # methodology first, then numbers
│
├── Makefile
├── .golangci.yml
├── go.mod
└── README.md
```

**Why this shape.** `local/` and `distributed/` implement the *same* interface,
which is what makes the accuracy comparison in Phase 8 a config change rather
than a rewrite — and that comparison is the whole point of the project. Lua
lives next to the Go that embeds it so the two cannot drift. `internal/` keeps
every package unexportable, so refactoring is free.

---

## 4. The central contract

Write this first. Everything else conforms to it.

```go
// Decision is the full answer to "may this request proceed", including the
// data needed to render response headers.
type Decision struct {
    Allowed    bool
    Limit      int64         // configured quota, echoed for X-RateLimit-Limit
    Remaining  int64         // quota left after this call; never negative
    ResetAfter time.Duration // until quota is fully replenished
    RetryAfter time.Duration // hint when denied; zero when allowed
}

// Spec is the resolved limit for one (route, tier) pair.
type Spec struct {
    Limit  int64
    Window time.Duration
    Burst  int64 // token/leaky bucket capacity; defaults to Limit
}

type Limiter interface {
    // Allow atomically tests and consumes `cost` units against key.
    // A non-nil error means the decision is UNKNOWN; the caller applies
    // the configured fail-open / fail-closed policy. Implementations must
    // never return a synthesised deny on infrastructure failure.
    Allow(ctx context.Context, key string, cost int64) (Decision, error)
}
```

Two deliberate choices:

- **`cost int64`, not an implicit 1.** Expensive endpoints can charge more later without changing the signature. A search request costing 2 tokens is a one-line config change.
- **`error` is separate from `Allowed == false`.** "Denied by policy" and "I could not reach the store" are different outcomes. Keeping them distinct is what allows fail-open vs fail-closed to be a middleware policy decision instead of behaviour buried in each limiter.

### Key format

```
ratelimit:{<identity>}:<method>:<route-pattern>:<algorithm>
       e.g. ratelimit:{key_9f3a1b}:GET:/api/v1/orders:swc
```

The `{...}` is a Redis Cluster **hash tag**: only the braced part is hashed, so
every key for one identity lands in one slot. That makes future multi-key
operations for a user single-slot, and it spreads load across shards by
identity, which is the natural distribution. Route pattern (not raw path) is
used so `/orders/1` and `/orders/2` share a bucket.

---

## 5. Algorithms

| Algorithm | Memory per key | Accuracy | Burst behaviour | Redis state | Default for |
|---|---|---|---|---|---|
| Token bucket | O(1), 2 fields | Good | Allows burst to capacity, then refill rate | `HASH {tokens, ts}` | Per-user API quotas |
| Sliding window log | O(requests in window) | Exact | No burst above limit, ever | `ZSET` of timestamps | Low-volume, high-value (login, payments) |
| Sliding window counter | O(1), 2 counters | Approximate (±overlap) | Mild boundary burst | 2x `STRING` counters | **General default** |
| Leaky bucket | O(1) + queue | Exact output rate | Absorbs burst, emits steady | `HASH {level, ts}` | Fragile downstreams |

Notes that matter per algorithm:

- **Token bucket.** Refill is computed lazily from elapsed time; there is no background ticker. Store tokens as a float so slow refill rates do not round to zero.
- **Sliding window log.** `ZREMRANGEBYSCORE` to evict, `ZCARD` to count, `ZADD` only if allowed. Memory is the failure mode: cap the ZSET size and document that a hostile client can make this expensive. This is why it is not the default.
- **Sliding window counter.** Two fixed sub-windows; weight the previous one by the fraction of the window it still overlaps: `estimate = prev*overlap + curr`. Fixed memory, close enough for almost everything. This is what most production systems actually run.
- **Leaky bucket** genuinely queues — it holds the request while draining at a constant rate, which is a different execution model from the other three (pure admission checks). It is implemented as a shaper with **max queue depth** and **max hold time**, and `ALGORITHMS.md` states plainly that it is the odd one out rather than pretending it fits the same interface cleanly.

---

## 6. Redis + Lua rules

The check-and-decrement must be one atomic operation. Redis runs a script to
completion with no interleaving, which is exactly the primitive needed.
Reference sketch:

```lua
-- token_bucket.lua
-- KEYS[1] = bucket key
-- ARGV    = capacity, refill_per_sec, now_ms, cost, ttl_ms
local capacity = tonumber(ARGV[1])
local rate     = tonumber(ARGV[2])
local now      = tonumber(ARGV[3])
local cost     = tonumber(ARGV[4])
local ttl      = tonumber(ARGV[5])

local state   = redis.call('HMGET', KEYS[1], 'tokens', 'ts')
local tokens  = tonumber(state[1]) or capacity
local ts      = tonumber(state[2]) or now

local elapsed = math.max(0, now - ts)
tokens = math.min(capacity, tokens + (elapsed * rate / 1000.0))

local allowed = 0
if tokens >= cost then
  tokens  = tokens - cost
  allowed = 1
end

redis.call('HSET', KEYS[1], 'tokens', tokens, 'ts', now)
redis.call('PEXPIRE', KEYS[1], ttl)
return { allowed, math.floor(tokens), math.ceil((capacity - tokens) / rate * 1000) }
```

Three non-negotiables for every script:

1. **Pass `now` in from Go. Never call `TIME` inside the script.** Non-deterministic commands make scripts unreplicable and tests unrepeatable. It also lets a fake clock drive integration tests.
2. **Always `PEXPIRE`.** Without a TTL, every one-off client leaks a key forever and Redis memory grows without bound.
3. **Single key per script, hash-tagged.** Keeps it Cluster-correct with no `CROSSSLOT` errors.

Also: cap `command_timeout` at ~50ms. A hung Redis must not park goroutines; it
must trip the breaker.

---

## 7. Failure modes

| Failure | Detection | Behaviour |
|---|---|---|
| Redis slow | per-call context deadline (50ms) | Deadline exceeded → counted as breaker error → fallback policy |
| Redis down | dial error / breaker error ratio | Breaker opens, calls short-circuit immediately, fallback policy applied |
| Redis flaky | half-open probes | Limited probes decide re-close or re-open |
| Upstream down | proxy error | 502, and rate-limit quota is **not** refunded (documented, deliberate) |
| Config invalid | startup validation | Refuse to boot. A gateway with a mis-parsed limit is worse than one that is down |

**Fail-open vs fail-closed is configured per route.** `/api/v1/public/*` can
fail open (availability wins, backend takes the risk); `/api/v1/payments/*`
fails closed (protection wins). Making this a single global flag is the common
mistake — different routes have genuinely different risk profiles.

Circuit breaker: `closed → open` on error ratio above threshold with a minimum
request count (so 2 failures out of 3 does not trip it), `open → half-open`
after a cooldown, `half-open → closed` after N consecutive successes.

---

## 8. Config shape

```yaml
server:
  addr: ":8080"
  metrics_addr: ":9090"        # separate port: /metrics is not public and not rate-limited
  read_timeout: 5s
  shutdown_grace: 15s

redis:
  addrs: ["redis:6379"]
  pool_size: 100
  dial_timeout: 200ms
  command_timeout: 50ms

breaker:
  error_ratio: 0.5
  min_requests: 20
  open_duration: 5s
  half_open_successes: 3

identity:
  order: ["api_key", "jwt_sub", "ip"]
  api_key_header: "X-API-Key"
  trusted_proxy_cidrs: ["10.0.0.0/8"]   # XFF is only trusted from these

tiers:
  free: { limit: 100,   window: 1m, burst: 20 }
  pro:  { limit: 10000, window: 1m, burst: 2000 }

routes:
  - pattern: "/api/v1/search"
    methods: ["GET"]
    algorithm: sliding_window_counter
    store: redis
    cost: 2
    fallback: closed
    upstream: "http://backend:9000"

  - pattern: "/api/v1/*"
    algorithm: token_bucket
    store: redis
    cost: 1
    fallback: open
    upstream: "http://backend:9000"
```

`store: local|redis` per route is what makes the Phase 8 correctness experiment
a config flip.

---

## 9. Phased build

Each phase ends with something runnable and a stated acceptance test. No phase
depends on a later one.

### Phase 0 — Scaffold
`go mod init`, Makefile (`build run test race bench lint up down`), config
structs + loader + validation, `internal/server` with `/healthz` and graceful
shutdown, `cmd/backend` echo service.
**Done when:** `make run` serves `/healthz`, an invalid config refuses to boot.

### Phase 1 — Local limiters and the test harness
All four algorithms in-memory behind `Limiter`, on a sharded keyed-mutex map
with an eviction janitor. Fake clock throughout.
This phase builds the test harness reused for every later phase:
- table-driven scenarios: steady, burst, idle-then-burst, window boundary
- **the race test**: 1000 goroutines, one key, `limit` = 100 → assert exactly 100 allowed under `-race`
**Done when:** `make race` is green and the race test fails if you replace the mutex with a naive read-then-write. Prove the test can detect the bug it exists to catch.

### Phase 2 — Gateway shell
chi, middleware chain, route matching, `ReverseProxy` to the dummy backend.
`429` with `Retry-After`; `X-RateLimit-Limit/Remaining/Reset` on every response,
allowed or not.
**Done when:** `curl` in a loop shows Remaining counting down then a 429 with a sane `Retry-After`. Single node, local store.

### Phase 3 — Redis + Lua (the core phase)
Four Lua scripts, embedded and loaded via `redis.Script`. Redis-backed
implementations of all four algorithms. Integration tests with
testcontainers running the **real** scripts.
**Done when:** the Phase 1 assertion suite passes unchanged against the Redis limiters, and two gateway processes sharing one Redis enforce one combined limit.

### Phase 4 — Config-driven routes and tiers
Per-route algorithm/limit/cost/fallback, per-tier overrides, tier resolved from
API key, SIGHUP hot reload.
**Done when:** a free-tier and a pro-tier key hit the same route and get different limits, with no restart between config edits.

### Phase 5 — Breaker and fallback policy
Breaker wrapping every store call, per-route fail-open/fail-closed, hard
per-call deadline.
**Done when:** `docker compose pause redis` under load produces the configured behaviour per route, gateway latency stays flat (no 50ms stall per request once open), and it recovers on `unpause`.

### Phase 6 — Observability
```
ratelimit_decisions_total{route,tier,algorithm,decision}
ratelimit_store_latency_seconds{algorithm}      # histogram
ratelimit_store_errors_total{kind}
ratelimit_breaker_state{name}                   # 0 closed, 1 half-open, 2 open
ratelimit_fallback_total{route,policy}
gateway_upstream_latency_seconds{route,code}
```
Histogram buckets tuned for sub-millisecond: `0.0005, 0.001, 0.002, 0.005,
0.01, 0.025, 0.05, 0.1`. Default Prometheus buckets start at 5ms and would make
every measurement useless here.
**Done when:** a Grafana dashboard shows allow/deny rates, store p50/p95/p99, and breaker state.

### Phase 7 — L1 cache (optional)
`sync.Map` caching *allow* decisions for ~5ms, and only when `Remaining` is
comfortably above zero. Never cache denies (a client would stay blocked past
its reset). Never cache when close to the limit (that is where accuracy
matters).
**Done when:** measured Redis-call reduction **and** measured over-admission are both written to `BENCHMARKS.md`. That number is the honest cost of the optimisation; reporting the speedup without it would be dishonest.

### Phase 8 — Prove it
compose stack: nginx → 3 gateway replicas → 1 Redis → 1 backend.

- **Experiment A (correctness).** Identical load against `store: local` then `store: redis`. Expected: local admits ~3x the limit because each replica keeps its own counter; Redis admits ~1x. This is the single most convincing artifact in the project.
- **Experiment B (latency).** vegeta at ramping RPS. Report p50/p95/p99 of gateway-added latency, load generator in a separate container, numbers **including** the Redis round trip. If p99 is 4ms, publish 4ms. A defensible measured number beats an unfalsifiable claim.

**Done when:** `BENCHMARKS.md` states hardware, RPS, key cardinality, and method *before* the numbers.

---

## 10. Testing strategy

| Layer | Scope | Notes |
|---|---|---|
| Unit | algorithms, clock, key builder, identity | Fake clock; no `time.Sleep` anywhere |
| Race | one key, 1000 goroutines | `-race` in CI, non-negotiable |
| Integration | real Redis via testcontainers | Same assertions as unit, so Lua and Go cannot diverge |
| Failover | pause/kill Redis mid-flight | Asserts fallback and breaker, not just happy path |
| Load | vegeta against compose stack | Produces the published numbers |

---

## 11. Interview talking points this produces

- Why sliding window counter is the default: fixed memory, bounded error, no adversarial blowup.
- Why the atomic unit is a Lua script and not `GET` + `DECR`: check-then-act is a race across nodes, and `WATCH`/`MULTI` retries degrade badly under contention on a hot key.
- Why fail-open/fail-closed is per route: a public read endpoint and a payment endpoint do not share a risk profile.
- Why the L1 cache is a *tradeoff*, with the over-admission number to back it.
- Why hash tags matter before you need Cluster: retrofitting a key format across live traffic is painful.

---

## 12. Open decisions

1. **Distributed leaky bucket** — full queueing across nodes needs a shared queue and is genuinely hard. Plan: implement the Redis version as rate-computation only (level + last-drain timestamp) and do the actual queueing/holding locally per node. Document the limitation rather than hiding it.
2. **Quota refund on upstream 5xx** — currently no refund. Simpler and safer; revisit if it proves user-hostile.
3. **Redis Cluster** — deferred, keys designed for it, migration path in `ARCHITECTURE.md`.

---

## 13. Gap resolutions

Eight architectural gaps identified during review. Each is resolved, deferred with explicit rationale, or documented as a deliberate tradeoff. None is silently ignored.

---

### G1 — API-key → tier mapping is undefined

**Problem.** Config declares `tiers: {free, pro}` but nothing maps an incoming API key string to a tier. `identity/` resolves *who* the caller is, not *what tier they are in*. The tier lookup (DB, cache, static map?) is a missing component.

**Resolution — Phase 4.**

Add `internal/tier/` package with a `Resolver` interface:

```go
type Resolver interface {
    // Resolve returns the tier name for a given identity key.
    // Returns ("", ErrUnknown) if the key is not recognised.
    Resolve(ctx context.Context, identity string) (tier string, err error)
}
```

Two implementations shipped:

| Implementation | File | When to use |
|---|---|---|
| `StaticResolver` | `tier/static.go` | Keys declared in YAML under `api_keys:` block — sufficient for this project |
| `CachedResolver` | `tier/cached.go` | Wraps any `Resolver` with a TTL cache; swap in a DB-backed resolver later |

Config addition:
```yaml
api_keys:
  "key_free_abc123": free
  "key_pro_xyz789":  pro
  # Keys not listed here → default tier (configurable, default: "free")

identity:
  default_tier: free      # fallback when key is unknown
  unknown_key_policy: deny  # "deny" | "default_tier"
```

`unknown_key_policy: deny` is the safe default — an unrecognised key gets no quota rather than a free pass. Configurable per deployment.

Middleware pipeline update:
```
identity extraction → tier resolution → key building → limiter.Allow
```

Tier name is injected into the Redis key: `ratelimit:{identity}:GET:/api/v1/orders:tb:free` — so free and pro keys never share a bucket.

**Folder addition:**
```
internal/tier/
├── resolver.go    # Resolver interface + ErrUnknown
├── static.go      # StaticResolver (loaded from config at startup)
└── cached.go      # CachedResolver (sync.Map TTL wrapper)
```

---

### G2 — JWT trusted without signature verification

**Problem.** `jwt_sub` extraction reads the `sub` claim from a Bearer token and uses it as the rate-limit identity key. Without verifying the JWT signature, any caller can forge an arbitrary `sub` to get a fresh, independent quota bucket — a trivial bypass.

**Decision: verify or reject, never trust blindly.**

Two modes, selected per deployment:

| Mode | Config | Behaviour |
|---|---|---|
| `jwks_uri` | URL to a JWKS endpoint | Fetch public keys, verify RS256/ES256 signature + `exp` + `aud` |
| `hmac_secret` | Static secret (env var) | Verify HS256 signature — simpler, suitable for internal services |
| `passthrough` | (absent) | `jwt_sub` extractor is disabled; falls back to next identity strategy |

`passthrough` mode allows the project to run without a real IdP, but it is explicit in config rather than the current implicit behaviour of trusting unsigned tokens.

**Implementation — Phase 2 (alongside identity package).**

```go
// internal/identity/jwt.go
type JWTExtractor struct {
    keyFunc jwt.Keyfunc   // populated from jwks_uri or hmac_secret
    audience string        // validated against aud claim
}

func (e *JWTExtractor) Extract(r *http.Request) (string, bool) {
    raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
    tok, err := jwt.Parse(raw, e.keyFunc,
        jwt.WithValidMethods([]string{"RS256", "ES256", "HS256"}),
        jwt.WithAudience(e.audience),
        jwt.WithExpirationRequired(),
    )
    if err != nil { return "", false }     // malformed or expired → fall through
    sub, ok := tok.Claims.(jwt.MapClaims)["sub"].(string)
    return sub, ok && sub != ""
}
```

Key points:
- Invalid or expired tokens fall through to the next identity strategy (IP), they do not block the request — rate-limiting is not the auth layer.
- JWKS keys are cached with a refresh interval; a `kid` mismatch triggers a single re-fetch.
- Add `github.com/golang-jwt/jwt/v5` to tech-stack table (it was listed as a dependency but its use was underspecified).

Config:
```yaml
identity:
  jwt:
    mode: jwks_uri                             # or hmac_secret | passthrough
    jwks_uri: "https://auth.example.com/.well-known/jwks.json"
    audience: "api.example.com"
    jwks_refresh_interval: 1h
```

---

### G3 — Clock skew across replicas

**Problem.** Each gateway process calls `clock.Now()` and passes `now_ms` into the Lua script. If replica wall clocks drift (common without NTP discipline), token-bucket refill math becomes replica-dependent: a request landing on a replica with a "fast" clock gets tokens that haven't been earned yet.

**Resolution — two layers, implemented in Phase 3.**

**Layer 1 — Use Redis `TIME` for the authoritative timestamp inside Lua.**

Wait — this contradicts the existing rule "never call `TIME` inside Lua." The original rationale was testability and replication safety. The resolution:

- **In production:** pass `now_ms` from Go as before, but add a **clock-drift guard** in the Lua script:

```lua
-- Reject timestamps that are unreasonably far from Redis server time.
-- This catches replicas whose wall clock has drifted > drift_tolerance_ms.
local redis_now = redis.call('TIME')
local redis_now_ms = redis_now[1] * 1000 + math.floor(redis_now[2] / 1000)
local drift = math.abs(now_ms - redis_now_ms)
if drift > tonumber(ARGV[drift_tolerance_idx]) then
    return {-1, 0, 0}   -- special sentinel: caller treats as ErrClockSkew
end
```

- **In tests:** inject a fake `redis_now` by also passing it as an ARGV — the guard is disabled when `drift_tolerance_ms` is set to `math.maxinteger` (a test-only config).

**Layer 2 — Operational requirement, documented in `ARCHITECTURE.md`.**

> All gateway replicas MUST be NTP-synchronised with tolerance ≤ 100ms. Deploy with `chronyd` or a cloud provider's time-sync agent. The 50ms `command_timeout` already bounds Redis latency; clock drift should be an order of magnitude smaller than the token-bucket window (minimum recommended window: 10s).

**ErrClockSkew handling:**
```go
// Gateway treats ErrClockSkew as ErrUnavailable.
// It is logged as a WARNING (not an error, not silently ignored).
// The route's fail-open/closed policy applies.
```

This makes clock-skew a monitorable, configurable condition rather than a silent accuracy bug.

Config addition:
```yaml
redis:
  clock_drift_tolerance_ms: 500   # 0 to disable the guard (not recommended)
```

---

### G4 — L1 cache undermines S3 (global limit correctness)

**Problem.** Phase 8's Experiment A is the project's headline proof: Redis enforces one global limit across N replicas. Phase 7's L1 cache reintroduces per-replica state and causes exactly the over-admission S3 exists to eliminate. The current plan flags this honestly but leaves the architectural tension unresolved.

**Resolution — explicit shipping policy.**

| State | L1 cache | Behaviour |
|---|---|---|
| Default (prod) | **Disabled** | S3 holds exactly. Experiment A produces clean numbers. |
| Opt-in | Enabled via config | S3 degrades proportionally to cache TTL and `Remaining` threshold. Measured and published. |

Config:
```yaml
routes:
  - pattern: "/api/v1/search"
    l1_cache:
      enabled: false          # default
      ttl: 5ms                # only meaningful when enabled
      remaining_threshold: 0.1  # only cache when Remaining > limit * threshold
```

**Experiment A** runs with `l1_cache.enabled: false` (the default). This is the proof.

**Phase 7** explicitly re-runs Experiment A with `l1_cache.enabled: true` and publishes:
1. Redis call reduction (the benefit)
2. Observed over-admission vs. Redis-only baseline (the cost)
3. The break-even: at what RPS does the latency saving justify the accuracy loss?

Framing: the L1 cache is documented as a **latency optimisation with a quantified accuracy cost**, not a default feature. Shipping it disabled by default means S3 is true by default.

---

### G5 — `Spec.Burst` is meaningless for two algorithms

**Problem.** `Spec{Limit, Window, Burst}` is shared by all four algorithms. Token bucket and leaky bucket use `Burst` (= bucket capacity). Sliding-window-log and sliding-window-counter have no burst concept — `Burst` is silently ignored, an interface smell.

**Resolution — document and enforce at validation time.**

```go
// internal/config/validate.go

// Burst is only meaningful for token_bucket and leaky_bucket.
// For sliding_window_log and sliding_window_counter it must be zero
// (unset). A non-zero Burst on these algorithms is a config error,
// not a silent ignore — the gateway refuses to boot.
if route.Algorithm == "sliding_window_log" || route.Algorithm == "sliding_window_counter" {
    if route.Burst != 0 {
        return fmt.Errorf("route %q: algorithm %q does not support burst; remove the burst field",
            route.Pattern, route.Algorithm)
    }
}
```

`Spec.Burst` is annotated in code:
```go
// Burst is the bucket capacity for token_bucket and leaky_bucket only.
// It is ignored and must be zero for sliding_window_log and
// sliding_window_counter. Config validation enforces this.
Burst int64
```

This is the lowest-cost fix: no interface change, validation catches misuse at startup, documentation makes the constraint clear.

---

### G6 — Reverse-proxy retries not reconciled with quota

**Problem.** If `httputil.ReverseProxy` retries a request on a transient upstream error, each retry is a new HTTP round-trip. If the middleware chain re-runs `Allow()` per retry, quota is double-charged. If it does not re-run, retries bypass rate limiting entirely.

**Resolution — no automatic retries (by design), explicit policy.**

`httputil.ReverseProxy` does **not** retry by default. The custom `ErrorHandler` in `internal/proxy/reverse.go` maps errors to 502/504 and returns — it does not retry. This is the correct behaviour for a rate-limiting gateway.

Document this explicitly:

> The gateway does not retry upstream failures. A failed request consumes quota once. Retry logic belongs in the client (with exponential backoff) or in a separate retry layer upstream of the gateway, not inside the gateway where it would interact with rate limiting in non-obvious ways.

If retry is added in future:
- The `Allow()` call must happen **once per client request**, not per upstream attempt.
- The retry loop must be inside the proxy handler, after the `Allow()` decision is already recorded.
- This is enforced by architecture (Allow is called in middleware, proxy is called in the next handler) — no code change needed, just documented invariant.

Config addition to make the no-retry policy explicit:
```yaml
routes:
  - upstream_retries: 0   # default and only valid value for now
                          # non-zero is rejected by validation until retry logic ships
```

---

### G7 — Breaker thresholds are arbitrary, not experimentally derived

**Problem.** `error_ratio: 0.5, min_requests: 20, open_duration: 5s` are picked numbers, not tuned. Under realistic conditions (GC pause, Redis latency spike, network jitter) these may cause the breaker to flap — tripping on normal variance rather than real outages. Unlike every other number in this plan, these are not backed by a measured experiment.

**Resolution — add a breaker-tuning experiment to Phase 5.**

**Phase 5 acceptance test** (addition):

```
Experiment: Breaker sensitivity calibration
  1. Run gateway under moderate load (500 RPS) against real Redis
  2. Introduce controlled Redis latency via toxiproxy: 10ms, 30ms, 60ms, 100ms jitter
  3. Record: breaker trips at what latency? How many false trips per hour?
  4. Tune min_requests and open_duration so breaker does NOT trip under <100ms jitter
  5. Record: breaker trips reliably when Redis is killed?
  6. Publish the calibrated defaults in BENCHMARKS.md with the methodology
```

**Revised default rationale** (to be filled in after the experiment):
```yaml
breaker:
  error_ratio: 0.5          # TODO: replace with experimentally derived value
  min_requests: 20          # must be high enough to survive a GC pause burst
  open_duration: 5s         # must be > Redis restart time in compose stack
  half_open_successes: 3    # number of probes before re-closing
```

Until Phase 5 numbers are in, defaults are explicitly labelled as starting points, not tuned values. The Grafana dashboard includes a `ratelimit_breaker_state` panel specifically so flapping is visible in Phase 5 load tests.

---

### G8 — `/metrics` endpoint has no access control

**Problem.** `/metrics` is on a separate port (`:9090`) and explicitly not rate-limited. This is correct for Prometheus scraping. It is also correct for anyone who can reach that port to scrape internal route cardinality, tier distribution, and traffic patterns — operational data that may be sensitive.

**Resolution — two controls, implemented in Phase 6.**

**Control 1 — network-level (deploy config).**

`deployments/docker-compose.yaml`: the metrics port is **not published** to the host network. Only the Prometheus container (on the same Docker network) can reach it.

```yaml
gateway1:
  expose: ["9090"]      # visible on the docker network only
  ports: ["8080:8080"]  # only the proxy port is published externally
```

`ARCHITECTURE.md` documents: "Do not publish the metrics port to the public internet. Prometheus scrapes it from within the same network. If exposing via a scraping agent, restrict access at the network/firewall layer."

**Control 2 — optional bearer token auth on `/metrics` (Phase 6, off by default).**

```yaml
server:
  metrics_token: ""     # empty = no auth (default, relies on network controls)
                        # set to a secret → require "Authorization: Bearer <token>"
```

```go
// internal/server/server.go
func metricsHandler(token string, h http.Handler) http.Handler {
    if token == "" {
        return h   // network controls are sufficient
    }
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("Authorization") != "Bearer "+token {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        h.ServeHTTP(w, r)
    })
}
```

This is not a substitute for network controls — it is defence-in-depth. Neither is mandatory; both are documented.

---

### Gap summary table

| # | Gap | Severity | Resolution | Phase |
|---|---|---|---|---|
| G1 | API-key → tier mapping missing | **High** | `internal/tier/` package + config `api_keys:` block | 4 |
| G2 | JWT trusted without verification | **Critical** | `JWTExtractor` with JWKS/HMAC; `passthrough` mode is explicit | 2 |
| G3 | Clock skew across replicas | **Medium** | Lua drift guard + `ARCHITECTURE.md` NTP requirement | 3 |
| G4 | L1 cache undermines S3 | **Medium** | Cache disabled by default; Experiment A run without it | 7 |
| G5 | `Spec.Burst` ignored silently | **Low** | Config validation rejects non-zero Burst on window algorithms | 0 |
| G6 | Proxy retries vs. quota | **Low** | No retries by design; `upstream_retries: 0` validated | 2 |
| G7 | Breaker thresholds arbitrary | **Medium** | Calibration experiment added to Phase 5 acceptance criteria | 5 |
| G8 | `/metrics` no access control | **Low** | Network isolation in compose + optional bearer token | 6 |
