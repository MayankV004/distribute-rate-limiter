# Gateway Configuration

The API Gateway is entirely driven by the `configs/gateway.yaml` file. Changes to this file can be hot-reloaded without downtime by sending a `SIGHUP` to the gateway process.

## Core Sections

### `server`
Controls the physical HTTP listeners.
- `addr`: The public-facing port for API traffic (e.g., `:8080`).
- `metrics_addr`: The internal port for Prometheus scraping (e.g., `:9090`).
- `read_timeout`: Defends against slowloris attacks.
- `shutdown_grace`: How long the server waits for active connections to finish before exiting.

### `redis`
Controls the distributed storage pool.
- `addrs`: A list of Redis nodes (e.g., `["redis:6379"]`).
- `pool_size`: Size of the connection pool. Maximize this based on your expected RPS to avoid connection bottlenecks.

### `breaker`
Controls the Circuit Breaker that protects the gateway from Redis outages.
- `error_ratio`: The percentage of failed Redis calls (e.g., `0.5` for 50%) required to trip the breaker.
- `min_requests`: Minimum traffic volume before the error ratio is calculated.
- `open_duration`: How long the breaker stays open (protecting Redis) before attempting a half-open trial.

### `identity`
Controls how the gateway figures out *who* is making the request.
- `order`: The cascade array. E.g., `["api_key", "ip"]`. The gateway will first look for an API key. If none is found, it falls back to the client's IP address.
- `default_tier`: The tier assigned if the identity is unknown or doesn't map to a specific key in `api_keys`.

### `tiers`
Defines the quotas available in the system.
```yaml
tiers:
  free:
    limit: 1000
    window: 1m
    burst: 200
```
- `limit`: The absolute mathematical limit per `window`.
- `burst`: For Token Bucket, the max burst capacity.

### `routes`
The most powerful section. Defines exactly how traffic is routed and protected.
```yaml
routes:
  - pattern: "/api/v1/search"
    methods: ["GET"]
    algorithm: sliding_window_counter
    store: redis
    cost: 1
    fallback: closed
    upstream: "http://dummy-backend:9000"
```
- `algorithm`: `token_bucket`, `sliding_window_log`, `sliding_window_counter`, or `leaky_bucket`.
- `store`: `local` (in-memory) or `redis` (distributed).
- `cost`: How many tokens are deducted per request (e.g., expensive endpoints can cost 5 tokens).
- `fallback`: `open` (allow traffic if Redis fails) or `closed` (block traffic if Redis fails).
- `upstream`: Where the traffic is proxied if it passes the rate limiter.
