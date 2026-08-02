# Architecture — Distributed Rate Limiter + API Gateway

> Living document. Diagrams reflect the Phase 8 compose stack. Single-node dev
> omits nginx and runs one gateway against a local Redis.

---

## 1. System overview

```
                             ┌─────────────────────────────┐
                             │      Observability Stack     │
                             │  Prometheus  ◄──── :9090     │
                             │       │                      │
                             │   Grafana                    │
                             └─────────────────────────────┘
                                        ▲  (scrape)
                                        │
                 ┌──────────────────────┼────────────────────────┐
                 │                      │                        │
  Client         ▼                      ▼                        ▼
  ──────► ┌─────────────┐   ┌──────────────────┐   ┌──────────────────┐
          │    nginx    │   │   Gateway :8080  │   │   Gateway :8080  │
          │  (LB :80)  │──►│   (replica 1)   │   │   (replica N)   │
          └─────────────┘   └────────┬─────────┘   └────────┬─────────┘
                                     │                       │
                    round-robin       └──────────┬────────────┘
                    (Phase 8 stack)              │
                                                 ▼
                                    ┌────────────────────────┐
                                    │         Redis          │  ← single source
                                    │  (shared rate-limit    │    of truth for
                                    │    state, all keys)    │    all counters
                                    └────────────────────────┘
                                                 │
                    (if allowed)                 │ (Lua script result)
                                    ┌────────────────────────┐
                                    │    Backend Service     │
                                    │     (cmd/backend)      │
                                    └────────────────────────┘
```

**Key property:** The gateway replicas are **stateless** — they hold no rate-limit
counters themselves (L1 cache is disabled by default, see § 6). Redis is the
single source of truth. All N replicas read and write the same keys under the
same Lua scripts, so the global limit is enforced once, not N times.

---

## 2. Request lifecycle (per gateway replica)

```
  Incoming HTTP request
        │
        ▼
┌──────────────────────────────────────────────────────────────────────┐
│  Middleware chain (internal/middleware)                              │
│                                                                      │
│  1. Recover         — catch panics, return 500, never crash          │
│  2. RequestID       — generate / echo X-Request-ID                  │
│  3. Logger          — structured slog, one line per request          │
│  4. RateLimit  ─────────────────────────────────────────────────┐   │
│     │                                                           │   │
│     a. Identity extraction (internal/identity)                  │   │
│        order: api_key → jwt_sub → ip                            │   │
│        JWT: signature verified (JWKS or HMAC), not trusted      │   │
│        IP:  X-Forwarded-For parsed with trusted-proxy CIDR list │   │
│     │                                                           │   │
│     b. Tier resolution (internal/tier)                          │   │
│        StaticResolver: identity → "free" | "pro"                │   │
│        unknown_key_policy: deny | default_tier                  │   │
│     │                                                           │   │
│     c. Spec lookup                                              │   │
│        Route pattern match → (algorithm, limit, window, cost)   │   │
│        Tier override applied                                     │   │
│     │                                                           │   │
│     d. Redis key build (internal/identity/key.go)               │   │
│        ratelimit:{identity}:METHOD:/pattern:algo:tier           │   │
│     │                                                           │   │
│     e. [Optional] L1 cache check (disabled by default)          │   │
│        Never cache denies. Never cache when Remaining is low.   │   │
│     │                                                           │   │
│     f. Circuit breaker check (internal/breaker)                 │   │
│        OPEN  → short-circuit; apply fail-open|closed policy     │   │
│        CLOSED|HALF-OPEN → proceed                               │   │
│     │                                                           │   │
│     g. Lua script (EVALSHA) → Redis                             │   │
│        Atomically: read state → compute → update → PEXPIRE      │   │
│        Returns: {allowed, remaining, reset_after_ms}            │   │
│     │                                                           │   │
│     h. Error path: ErrUnavailable → fail-open|closed per route  │   │
│        ErrClockSkew    → WARNING log, treat as ErrUnavailable   │   │
│     │                                                           │   │
│     i. Set response headers (always, allowed or denied)         │   │
│        X-RateLimit-Limit / Remaining / Reset                    │   │
│        Retry-After (only on 429)                                │   │
│     │                                                           │   │
│     j. Emit Prometheus metrics                                  │   │
│        ratelimit_decisions_total{route,tier,algorithm,decision} │   │
│        ratelimit_store_latency_seconds{algorithm}               │   │
└─────┼───────────────────────────────────────────────────────────┘   │
      │                                                               │
      ▼ (if allowed)                        ▼ (if denied)            │
┌──────────────┐                    ┌───────────────┐                │
│ ReverseProxy │                    │  429 response │                │
│  (proxy.go)  │                    │  Retry-After  │                │
│              │                    └───────────────┘                │
│  upstream    │
│  → backend   │
│              │
│  502 on err  │  ← quota NOT refunded on upstream failure (by design)
└──────────────┘
```

---

## 3. Concurrency domains

There are two separate races to eliminate, requiring different solutions:

```
Domain A — Single process (local store)
─────────────────────────────────────────────────────────────────
  goroutine 1 ──┐
  goroutine 2 ──┤──► ShardedMap.Lock(key)  ──► token bucket state
  goroutine 3 ──┤         │
  ...           │     mutex held for the
  goroutine N ──┘     full read-modify-write
                        cycle
                        │
                    Release mutex
                        │
                    Next goroutine

  Tool: 256-shard keyed mutex map (internal/limiter/local/shardedmap.go)
  Proof: 1000-goroutine race test with go test -race (Phase 1)
  Cost: zero network I/O, no serialisation

─────────────────────────────────────────────────────────────────
Domain B — Multiple processes (Redis / distributed store)
─────────────────────────────────────────────────────────────────

  Gateway 1 ──┐
  Gateway 2 ──┤──► Redis EVALSHA ──► Lua script runs atomically
  Gateway 3 ──┘         │            (Redis is single-threaded;
                         │             no interleaving possible)
                         │
                    One Lua script:
                    1. HMGET / GET  state
                    2. compute new state
                    3. HSET / SET   new state
                    4. PEXPIRE      TTL reset
                    5. return result

  Why NOT GET + DECR:
    Gateway 1 reads "1 token left"
    Gateway 2 reads "1 token left"   ← race! both see 1
    Gateway 1 writes "0 tokens"
    Gateway 2 writes "0 tokens"      ← both pass; limit violated

  Why NOT WATCH/MULTI (optimistic locking):
    Under contention on a hot key, retries degrade O(goroutines).
    A hot key is exactly the worst case for this design.
    Lua script has no retry loop: one round trip, guaranteed correct.

─────────────────────────────────────────────────────────────────
```

---

## 4. Redis key design

```
Key format:
  ratelimit:{<identity>}:<METHOD>:<route-pattern>:<algo>:<tier>

Example:
  ratelimit:{key_9f3a1b}:GET:/api/v1/orders:swc:pro

Components:
  ┌──────────┬────────────────────────────────────────────────────┐
  │ Segment  │ Value / notes                                      │
  ├──────────┼────────────────────────────────────────────────────┤
  │ prefix   │ ratelimit: — namespaces keys in a shared Redis     │
  │ {identity│ Hash tag — only the braced part is hashed for      │
  │          │ Redis Cluster slot assignment. All keys for one    │
  │          │ identity land in one slot → single-slot Lua ops.  │
  │ METHOD   │ GET / POST / etc. — different methods get different│
  │          │ buckets even on the same path                     │
  │ pattern  │ Route pattern, NOT raw path. /orders/{id} means   │
  │          │ /orders/1 and /orders/2 share one bucket.         │
  │ algo     │ tb | swl | swc | lb — short form                  │
  │ tier     │ free | pro — so tiers never share a bucket        │
  └──────────┴────────────────────────────────────────────────────┘

Cluster slot assignment:
  CLUSTER KEYSLOT "ratelimit:{key_9f3a1b}:GET:/api/v1/orders:swc:pro"
  → hashes only "key_9f3a1b" (the braced content)
  → all routes for that identity → same slot
  → future MULTI-key operations (e.g. burst + sustained quota) are single-slot
```

---

## 5. Circuit breaker state machine

```
                ┌──────────────────────────────┐
                │                              │
   requests ──► │          CLOSED              │ ◄── normal operation
                │  errors counted per window   │
                │                              │
                └──────────────┬───────────────┘
                               │
                   error_ratio > threshold
                   AND requests >= min_requests
                               │
                               ▼
                ┌──────────────────────────────┐
                │                              │
   all calls ──►│           OPEN               │ ──► ErrUnavailable immediately
    fail fast   │  no calls to Redis           │     no goroutines parked
                │  open_duration timer running │
                │                              │
                └──────────────┬───────────────┘
                               │
                     open_duration elapsed
                               │
                               ▼
                ┌──────────────────────────────┐
                │                              │
  limited  ───► │         HALF-OPEN            │ ──► probe calls allowed
  probes        │  counting successes          │
                │                              │
                └───────┬──────────────────────┘
                        │
           ┌────────────┴────────────┐
           │                         │
   N consecutive              any failure
    successes                        │
           │                         │
           ▼                         ▼
        CLOSED                     OPEN
     (re-close)                 (re-open)

Tuning notes (Phase 5 experiment, see BENCHMARKS.md):
  min_requests: must survive a GC pause burst without false trips
  open_duration: must be > Redis container restart time (~2-3s in compose)
  error_ratio:   validated under toxiproxy jitter at 10/30/60/100ms
```

---

## 6. L1 cache (disabled by default)

```
         ┌─────────────────────────────────────────┐
         │          Per-replica sync.Map            │
         │                                          │
         │  key → {Decision, expiresAt}             │
         │                                          │
         │  Rules:                                  │
         │   • Only cache ALLOW decisions           │
         │   • Only when Remaining > limit * 0.1   │ ← accuracy matters near limit
         │   • TTL: ~5ms                            │
         │   • Never cache DENY (stale lock-out)    │
         └──────────────┬──────────────────────────┘
                        │  hit (within TTL)
                        ▼
                   allow immediately
                   (no Redis call)

         ┌──────────── WARNING ─────────────────────┐
         │                                          │
         │  L1 cache DISABLED by default.           │
         │                                          │
         │  With cache enabled, each replica may    │
         │  admit up to (limit * TTL / window)      │
         │  extra requests before Redis is checked. │
         │                                          │
         │  Phase 8 Experiment A runs WITHOUT the   │
         │  cache to prove S3 (global limit holds). │
         │  Phase 7 measures and publishes the      │
         │  over-admission cost honestly.           │
         └──────────────────────────────────────────┘

  Per-route config:
    l1_cache:
      enabled: false   ← default; flip to true only after Phase 7 measurement
      ttl: 5ms
      remaining_threshold: 0.1
```

---

## 7. Failure modes

| Failure | Detection | Response | Configured by |
|---|---|---|---|
| Redis slow (>50ms) | `command_timeout` context deadline | Deadline exceeded → breaker error counter | `redis.command_timeout` |
| Redis down | dial error / error ratio crosses threshold | Breaker opens → `ErrUnavailable` → fail-open or fail-closed | `breaker.*` |
| Redis flaky | breaker half-open probes | Re-close on N successes, re-open on any failure | `breaker.half_open_successes` |
| Clock skew | Lua drift guard (|now_ms − redis_ms| > tolerance) | Returns `ErrClockSkew` → treated as `ErrUnavailable`, WARNING logged | `redis.clock_drift_tolerance_ms` |
| Upstream down | proxy `ErrorHandler` | 502 (timeout → 504); quota **not** refunded | — |
| Config invalid | startup `validate.go` | Gateway refuses to boot | — |
| Unknown API key | tier `StaticResolver` | `ErrUnknown` → honour `unknown_key_policy` | `identity.unknown_key_policy` |
| JWT invalid/expired | `JWTExtractor.Extract` returns false | Falls through to next identity strategy (IP) | `identity.jwt.mode` |

### Fail-open vs fail-closed (per route)

```
  route.fallback = "open"                route.fallback = "closed"
  ─────────────────────────────────      ──────────────────────────────
  Redis down?  → ALLOW request           Redis down?  → DENY (503)
                 backend sees full load                 backend protected
  Risk: backend unprotected              Risk: client denied despite quota

  Good for: public read APIs             Good for: payments, auth, writes
            "availability > protection"             "protection > availability"
```

---

## 8. Redis Cluster migration path

The current dev stack runs a single Redis node. The key design is already
Cluster-ready; no application code changes are needed to add nodes.

```
  Current (single-node Redis):
  ─────────────────────────────────────────────────────
  All keys → one Redis instance
  Slot assignment: irrelevant (all slots on one node)

  Future (Redis Cluster, 6 nodes):
  ─────────────────────────────────────────────────────
  ratelimit:{key_9f3a1b}:...  → hash("key_9f3a1b") → slot 4311 → Node 2
  ratelimit:{key_a1b2c3}:...  → hash("key_a1b2c3") → slot 9482 → Node 4
  ratelimit:{key_9f3a1b}:... (different route) → same slot 4311 → Node 2
                                 ▲
                                 same hash tag → same node → no CROSSSLOT

  go-redis/v9 client:
    Single node:  redis.NewClient(&redis.Options{Addr: "redis:6379"})
    Cluster:      redis.NewClusterClient(&redis.ClusterOptions{
                      Addrs: []string{"redis1:6379", "redis2:6379", ...},
                  })
    API is identical. The application code does not change.

  Migration steps:
    1. Provision a 6-node Redis Cluster (3 primary + 3 replica)
    2. Change configs/gateway.yaml: addrs: [list of 6 nodes]
    3. Restart gateways. Done.
    4. Lua scripts already use single KEYS[1] — no CROSSSLOT errors.
```

> [!NOTE]
> The Sliding Window Counter uses two keys (`curr` + `prev` window buckets).
> Both share the same hash tag `{identity}`, so they land in the same Cluster
> slot. No special handling required.

---

## 9. Observability wiring

```
  ┌──────────────────┐
  │  Gateway :8080   │  (proxy port — public, rate-limited)
  └──────────────────┘

  ┌──────────────────┐
  │  Gateway :9090   │  (metrics port — docker-network only, NOT published)
  │  GET /metrics    │
  └────────┬─────────┘
           │ scrape every 5s
           ▼
  ┌──────────────────┐
  │   Prometheus     │
  └────────┬─────────┘
           │ query
           ▼
  ┌──────────────────┐
  │    Grafana       │  → dashboard: allow/deny rates, p50/p95/p99,
  └──────────────────┘               breaker state, fallback count

  Key metrics:
    ratelimit_decisions_total{route,tier,algorithm,decision}
    ratelimit_store_latency_seconds{algorithm}    ← histogram, sub-ms buckets
    ratelimit_store_errors_total{kind}
    ratelimit_breaker_state{name}                 ← 0=closed,1=half,2=open
    ratelimit_fallback_total{route,policy}
    gateway_upstream_latency_seconds{route,code}

  Histogram buckets (explicitly sub-millisecond):
    [0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1]
    Default Prometheus buckets start at 5ms — useless at this resolution.

  Access control (G8):
    Metrics port uses docker expose: not ports: → not reachable from host.
    Optional: server.metrics_token in config → Bearer auth on /metrics.
```

---

## 10. Phase 8 compose stack (full multi-node)

```
  ┌──────────────────────────────────────────────────────────────────────┐
  │                       Docker network                                 │
  │                                                                      │
  │  ┌───────────┐    ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │
  │  │  vegeta   │    │  gateway 1  │  │  gateway 2  │  │ gateway 3  │ │
  │  │(load gen) │    │   :8080     │  │   :8080     │  │  :8080     │ │
  │  └─────┬─────┘    └──────┬──────┘  └──────┬──────┘  └─────┬──────┘ │
  │        │                 │                 │                │        │
  │        ▼                 └─────────────────┴────────────────┘        │
  │  ┌───────────┐                             │                         │
  │  │   nginx   │ ◄───────────────────────────┘                        │
  │  │ (LB :80)  │                                                       │
  │  └───────────┘                                                       │
  │                           ┌─────────────────────────┐               │
  │  All gateways share ────► │         Redis            │               │
  │  one Redis instance       │  (rate-limit state)     │               │
  │                           └─────────────────────────┘               │
  │                                                                      │
  │  ┌──────────────────────────────────────────┐                       │
  │  │  toxiproxy (Phase 5 only)                │                       │
  │  │  Sits between gateways and Redis.        │                       │
  │  │  Adds controlled latency/jitter/errors   │                       │
  │  │  for breaker calibration experiment.     │                       │
  │  └──────────────────────────────────────────┘                       │
  │                                                                      │
  │  ┌──────────┐   ┌───────────┐   ┌───────────┐                      │
  │  │ backend  │   │Prometheus │   │  Grafana  │                      │
  │  │ :9000    │   │ (scrapes  │   │ :3000     │                      │
  │  │(echo svc)│   │  :9090)   │   │(dashbrd)  │                      │
  │  └──────────┘   └───────────┘   └───────────┘                      │
  └──────────────────────────────────────────────────────────────────────┘

  Phase 8 Experiment A (correctness proof):
    Run identical load with store: local  → observe ~3x over-admission
    Run identical load with store: redis  → observe ~1x (global limit holds)
    This is the single most important diagram in the project.

  Phase 8 Experiment B (latency):
    vegeta in a separate container → nginx → gateways → Redis → backend
    Measure: total latency - backend latency = gateway-added latency
    Report: p50 / p95 / p99 including the Redis round trip
```
