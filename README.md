# Distributed Rate Limiter & Production API Gateway

A robust, production-grade stateless Go API Gateway enforcing configurable rate limits across multiple replicas. It features four swappable rate-limiting algorithms backed by either local memory or a distributed Redis cluster, complete with observability, circuit breaking, and a full suite of production middleware.

## Features

- **Distributed & Atomic:** Uses Redis Lua scripts (`EVALSHA`) to ensure strict global quota accuracy across $N$ gateway replicas without check-and-act race conditions.
- **Config-Driven Routing:** Define routes, upstream proxies, rate limits, and authentication tiers entirely via YAML. Zero-downtime hot reloads via `SIGHUP`.
- **Swappable Algorithms:** Token Bucket, Sliding Window Log, Sliding Window Counter (default), and Leaky Bucket. Swap between `local` (in-memory) and `redis` stores per-route.
- **Production Middleware:** Built-in Panic Recovery, `log/slog` structured JSON logging, and UUID-based `X-Request-Id` tracing.
- **Circuit Breaking:** Protects the gateway from Redis outages. Configurable Fail-Open (availability wins) or Fail-Closed (protection wins) fallback policies per route.
- **L1 Caching:** Optional sub-millisecond in-process cache to reduce Redis load, with documented tradeoffs on quota accuracy.
- **Observability:** Native Prometheus metrics (`/metrics`) exposing sub-millisecond histograms, decision counters, and circuit breaker states, complete with a pre-configured Grafana dashboard.

## Architecture Overview

```text
HTTP Request
    │
    ▼
[ Nginx Load Balancer ]
    │
    ▼ (Round Robin)
[ Gateway Replica 1..N ]
    │   1. Recover (Panic Handling)
    │   2. RequestID (Inject UUID)
    │   3. Logger (slog JSON)
    │   4. RateLimit Middleware 
    │       a. Identity Extractor (API Key, JWT, IP)
    │       b. Tier Resolver (Free, Pro, etc.)
    │       c. Limiter.Allow() → Atomic Redis Lua Script
    │
    ├── DENIED → 429 Too Many Requests (with X-RateLimit & Retry-After headers)
    │
    └── ALLOWED
            │
            ▼
    [ Reverse Proxy ] → Upstream Backend Server
```

## Running the Project

The entire system is orchestrated via Docker Compose.

### 1. Start the Infrastructure
Start the Nginx load balancer, 3 Gateway replicas, Redis, Prometheus, Grafana, and the Dummy Backend:

```bash
docker compose -f deployments/docker-compose.yaml up -d --build
```

### 2. Verify it works
Send a request using an API key mapped to the `free` tier:
```bash
curl -i -H "X-API-Key: key_free_example" http://localhost/api/v1/search
```
You should see `200 OK` along with injected tracing and rate-limiting headers:
```http
X-Ratelimit-Limit: 1000
X-Ratelimit-Remaining: 999
X-Request-Id: <uuid>
```

### 3. View the Grafana Dashboard
1. Open [http://localhost:3000](http://localhost:3000) in your browser.
2. Log in with `admin` / `admin`.
3. Navigate to **Dashboards > Rate Limiter Dashboard** to see live traffic, latency histograms, and circuit breaker health.

### Load Testing & Verification

The project includes an advanced dual-testing strategy demonstrating both microbenchmarking and real-world simulation:

#### 1. Vegeta: Microbenchmarking & Correctness
Vegeta is used for constant-arrival-rate testing. It is perfect for proving algorithmic exactness and measuring raw latency overhead without masking slow server responses.
- `scratch/exp_a.sh`: Tests raw local algorithms (Correctness & Quota Accuracy).
- `scratch/exp_b.sh`: Measures true p95 tail latency through the NGINX → Gateway → Redis stack.
- `scratch/exp_c_tiers.sh`: Tests multi-tier quota limits (e.g., Free vs. Pro tiers) and sliding window burst logic at `1:1` costs.
- `scratch/exp_d_chaos.sh`: Kills the Redis cluster mid-flight to prove instant Circuit Breaker fail-open capabilities (0 seconds of downtime).
- `scratch/exp_f_max.sh`: The ultimate stress test. Slams the gateway with 5,000 Requests Per Second for 60 seconds (300,000 requests) to test TCP port exhaustion and NGINX worker connection limits.

#### 2. k6: Real-World Scenario Simulation
k6 is used for closed-model virtual user testing. It simulates messy, real-world traffic flows where users have "think time", mix different API keys, and hit various endpoints.
- `scratch/k6_scenario.js`: A complex scenario ramping up to 1,000 concurrent virtual users. 80% use Free tier keys, 20% use Pro tier keys. 
- Run it via Docker using the wrapper script: `./scratch/exp_e_k6.sh`

### How to Run the Tests
```bash
# Run the automated correctness test
./scratch/exp_a.sh

# Run the automated latency benchmark
./scratch/exp_b.sh

# Run the 5,000 RPS Maximum Throughput test!
./scratch/exp_f_max.sh
```

## Project Status
**COMPLETE.** All 8 educational phases and the final Production API Gateway upgrades have been successfully implemented, tested, and benchmarked. 

For a deep dive into the architecture and decisions, read the [Learning Guide](LearningGuide.md) and the [Benchmarks Report](docs/BENCHMARKS.md).

## License
MIT
