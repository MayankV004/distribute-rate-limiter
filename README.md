# Distributed Rate Limiter + API Gateway

A production-grade stateless Go gateway enforcing configurable rate limits across multiple replicas, with four swappable algorithms (token bucket, sliding window log, sliding window counter, leaky bucket) backed by either local memory or Redis.

## What makes this interesting

This isn't just HTTP middleware. The core problem is that **the same admission decision has to be correct** under:

1. Thousands of concurrent goroutines inside one process (mutexes, atomic ops)
2. N processes racing against one Redis key (Lua scripts for atomicity)

Two different concurrency domains, two different solutions, one interface. The local and distributed implementations share the `Limiter` contract so they can be A/B tested by config alone — that comparison is the punchline.

## Project structure

```
rate-limiter/
├── cmd/
│   ├── gateway/        # Wire config → limiter → proxy → server
│   └── backend/        # Dummy upstream for testing
├── internal/
│   ├── limiter/        # ← The contract: Limiter interface + Decision + Spec
│   │   ├── local/      # Single-process implementations (mutex-based)
│   │   └── distributed/# Redis-backed implementations (Lua scripts)
│   ├── config/         # YAML + validation
│   ├── identity/       # Extract client ID from IP/API-key/JWT
│   ├── breaker/        # Circuit breaker around Redis
│   ├── middleware/     # Rate limit + headers + logging + recovery
│   ├── proxy/          # ReverseProxy to upstream
│   ├── metrics/        # Prometheus collectors
│   └── server/         # Graceful shutdown, /healthz
├── configs/            # Committed reference configs
├── deployments/        # docker-compose: nginx LB → 3 gateways → redis
├── docs/               # IMPLEMENTATION_PLAN.md, ARCHITECTURE.md, ALGORITHMS.md
└── test/
    ├── integration/    # Real Redis via testcontainers
    └── load/           # vegeta scripts for Phase 8 experiments
```

## Quickstart

```bash
# Phase 0 check
make help

# Once Phase 1 is done:
make test       # Unit tests, fake clock
make race       # The 1000-goroutine correctness test

# Once Phase 2 is done:
make run        # Single gateway on :8080, dummy backend on :9000

# Once Phase 8 is done:
make up         # Full stack: 3 gateways + redis + prometheus + grafana
curl -H "X-API-Key: free_tier_key" http://localhost/api/v1/search
make down
```

## The contract (write this first, everything else conforms)

```go
type Decision struct {
    Allowed    bool
    Limit      int64
    Remaining  int64
    ResetAfter time.Duration
    RetryAfter time.Duration
}

type Limiter interface {
    Allow(ctx context.Context, key string, cost int64) (Decision, error)
}
```

Two deliberate choices:

- `cost int64` lets expensive endpoints charge more without changing the signature.
- `error` separate from `Allowed == false` is what allows fail-open/fail-closed to be a **middleware policy** rather than behaviour buried in each limiter.

## Success criteria

| # | What | How it's proven |
|---|---|---|
| S1 | Four algorithms swappable by config | No recompile between token-bucket and sliding-window-counter |
| S2 | Exactly `limit` pass when 1000 goroutines hit one key | `go test -race` assertion, not eyeballing |
| S3 | 3 replicas enforce one global limit, not 3x | Phase 8 experiment A: local vs redis under identical load |
| S4 | Measured p50/p95/p99 added latency | Phase 8 experiment B: vegeta, published honestly |
| S5 | Redis outage degrades per policy, never hangs | `docker compose pause redis` under load |

## Phased build

Each phase delivers something runnable and testable. No phase depends on a later one.

- **Phase 0:** Scaffold (go.mod, Makefile, config loader, /healthz) — *you are here*
- **Phase 1:** Local limiters + the test harness (fake clock, race test)
- **Phase 2:** Gateway shell (chi, ReverseProxy, 429 + headers)
- **Phase 3:** Redis + Lua (the core: atomic scripts, integration tests)
- **Phase 4:** Config-driven routes and tiers
- **Phase 5:** Circuit breaker + fail-open/closed per route
- **Phase 6:** Prometheus metrics + Grafana dashboard
- **Phase 7:** Optional L1 cache (with honest over-admission measurement)
- **Phase 8:** Prove it (nginx → 3 gateways, vegeta load test, write up the numbers)

Full plan in [`docs/IMPLEMENTATION_PLAN.md`](docs/IMPLEMENTATION_PLAN.md).

## Tech stack

- **Go 1.22+** — goroutines, `sync.Map`, `-race`, `httputil.ReverseProxy` in stdlib
- **chi** — thin router, middleware-friendly
- **go-redis/v9** — `redis.Script` handles EVALSHA fallback automatically
- **koanf** — YAML + env config
- **Prometheus + Grafana** — observability
- **testcontainers** — integration tests against real Redis
- **vegeta** — load generation

## The algorithms (tradeoff table)

| Algorithm | Memory | Accuracy | Burst | Redis state | Default for |
|---|---|---|---|---|---|
| Token bucket | O(1) | Good | Allows burst to capacity | `HASH` | Per-user quotas |
| Sliding window log | O(requests) | Exact | None | `ZSET` | Low-volume, high-value |
| Sliding window counter | O(1) | Approximate | Mild | 2x `STRING` | **General default** |
| Leaky bucket | O(1) + queue | Exact rate | Absorbs | `HASH` | Fragile downstreams |

See [`docs/ALGORITHMS.md`](docs/ALGORITHMS.md) for state diagrams and why sliding-window-counter is the production choice.

## Key design decisions

1. **Lua scripts for atomicity.** Check-and-decrement is a race across nodes. `WATCH`/`MULTI` retries degrade under contention on hot keys; Lua runs atomically with no interleaving.

2. **Fail-open vs fail-closed is per route.** `/api/v1/public/*` can fail open (availability wins); `/api/v1/payments/*` fails closed (protection wins). A single global flag is the common mistake.

3. **Hash-tagged keys before Cluster.** `ratelimit:{user123}:GET:/api/v1/orders` — only the braced part is hashed, so every key for one identity lands in one Redis Cluster slot. Retrofitting a key format across live traffic is painful; design for it now.

4. **Fake clock for tests.** Every algorithm is time-dependent. Testing with `time.Sleep` produces suites that are slow and flaky. A fake clock makes a "wait one minute" test instant and deterministic.

5. **L1 cache is a tradeoff.** Short-TTL local allow-cache cuts Redis calls but over-admits. Phase 7 measures both the speedup **and** the over-admission — reporting the first without the second would be dishonest.

## Interview talking points this produces

- Why sliding window counter is the default: fixed memory, bounded error, no adversarial blowup.
- Why the atomic unit is a Lua script, not `GET` + `DECR`.
- Why fail-open/fail-closed is per route: different risk profiles.
- Why hash tags matter before you need Cluster.
- The L1 cache tradeoff, with measured numbers.

## License

MIT

## Status

**Phase 0 scaffold in progress.** The `Limiter` interface, `Clock`, and `Decision` helpers are written; `go test` is green. Next: Phase 1 local implementations + the race test.
