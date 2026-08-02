# Benchmarks

TODO (Phase 8): fill in after running the load tests.

## Methodology

State hardware, load generator location, key cardinality, and method
*before* publishing any numbers. A defensible measured result beats
an unfalsifiable claim.

- Hardware: (fill in)
- Redis deployment: single-node, same host / separate host
- Load generator: vegeta, running in a separate container
- Key cardinality: N unique API keys, all hitting the same route
- Measurement: gateway-added latency = total latency - backend latency

## Experiment A — Correctness (local vs. redis)

**Hypothesis:** 3 gateway replicas with `store: local` admit ~3x the limit.
With `store: redis` they admit ~1x.

| Config | RPS | Limit | Admitted | Over-admission |
|---|---|---|---|---|
| store: local  | | | | |
| store: redis  | | | | |

## Experiment B — Latency

| RPS | p50 | p95 | p99 | p99.9 |
|---|---|---|---|---|
| 100   | | | | |
| 1000  | | | | |
| 5000  | | | | |

## Phase 7 — L1 Cache tradeoff

| Config | Redis calls/s | Over-admission |
|---|---|---|
| No cache     | | |
| 5ms TTL cache| | |

Report both numbers — speedup without over-admission would be dishonest.
