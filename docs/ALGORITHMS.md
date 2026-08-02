# Algorithms

TODO (Phase 1): document each algorithm here after implementing it.

Suggested sections per algorithm:

## Token Bucket
- State: tokens (float), last_refill_ms (int)
- Allow/deny logic
- Burst behaviour
- Memory: O(1) per key
- When to use

## Sliding Window Log
- State: sorted set of request timestamps
- Allow/deny logic
- Memory: O(requests in window) per key — and why that matters
- Adversarial case: hostile client at limit holds limit entries forever

## Sliding Window Counter
- State: 2 counters (curr, prev)
- Weighted overlap formula
- Max approximation error: ±(limit/window) at exact boundary
- Why this is the production default

## Leaky Bucket
- State: level (float), last_drain_ms (int)
- Constant output rate vs. burst tolerance
- Why it is the odd one out (shapes traffic, not just admits it)
- Distributed limitation: queueing is per-node

## Tradeoff table
| Algorithm | Memory | Accuracy | Burst | Redis state |
|---|---|---|---|---|
