# Distributed Rate Limiter & Production API Gateway Platform

A high-performance, fault-tolerant Go API Gateway enforcing atomic, distributed rate limiting at scale, accompanied by an interactive Next.js topology & live traffic simulation dashboard.

---

## 🏗️ Repository Architecture

```text
rate-limiter/
├── backend/          # Go API Gateway, Redis Lua scripts, Docker Compose, Benchmarks
│   ├── cmd/          # Gateway entry point & server initialization
│   ├── internal/     # Core algorithms, middlewares, stores, circuit breaker
│   ├── deployments/  # Docker Compose (Nginx, Gateway x3, Redis, Prometheus, Grafana)
│   ├── docs/         # Architecture & Benchmark reports
│   └── scratch/      # Vegeta & k6 load testing scripts (up to 5,000 RPS)
└── frontend/         # Next.js Dashboard & Topology Simulator
    └── src/          # Interactive canvas, traffic controller, stats dashboard
```

---

## ✨ Features

### Backend (Go Gateway & Infrastructure)
- **Atomic Distributed Quotas**: Uses Redis Lua scripts (`EVALSHA`) to prevent check-and-act race conditions across $N$ gateway replicas.
- **4 Swappable Rate Limiting Algorithms**:
  - Sliding Window Counter (Default)
  - Token Bucket
  - Sliding Window Log
  - Leaky Bucket
- **Dual-Tier Storage**: Instant toggling between `local` (in-memory) and `redis` cluster storage per-route.
- **Resiliency & Circuit Breaking**: Protects against Redis cluster latency and outages with configurable `Fail-Open` or `Fail-Closed` fallback policies.
- **Production Middlewares**: Panic recovery, `log/slog` structured JSON logs, and `X-Request-Id` tracing.
- **Full Observability Stack**: Native Prometheus metrics (`/metrics`) and custom Grafana dashboard.

### Frontend (Next.js Visualization Dashboard)
- **Interactive Topology Canvas**: Real-time visualization of system nodes (Load Balancer, Gateway Replicas, Redis Cluster, Upstreams).
- **Traffic Simulator**: Interactive control panel to generate multi-tier traffic streams (Free vs. Pro tier keys).
- **Real-Time System Metrics**: Displays live throughput (RPS), rejection rates, p95 tail latencies, and circuit breaker status.

---

## 🚀 Quick Start

### 1. Prerequisites
- [Docker Engine & Docker Compose](https://docs.docker.com/get-docker/)
- [Node.js 18+](https://nodejs.org/) (for running the frontend dashboard locally)
- [Go 1.22+](https://go.dev/) (optional, if running backend binaries outside Docker)

---

### 2. Start the Backend Infrastructure

Navigate to the root directory and start all services via Docker Compose:

```bash
docker compose -f backend/deployments/docker-compose.yaml up -d
```

This starts:
- **Nginx Load Balancer**: `http://localhost:80`
- **Gateway Replicas**: 3 stateless nodes running behind Nginx
- **Redis Cluster**: Rate limiter state store (`port 6379`)
- **Prometheus**: `http://localhost:9090`
- **Grafana**: `http://localhost:3030` (Credentials: `admin` / `admin`)
- **Dummy Backend**: Target upstream service (`port 8081`)

---

### 3. Verify Backend API Gateway

Send a test request through Nginx using a `free` tier API key:

```bash
curl -i -H "X-API-Key: key_free_example" http://localhost/api/v1/search
```

**Expected Response (`200 OK`):**
```http
HTTP/1.1 200 OK
Content-Type: application/json
X-Ratelimit-Limit: 1000
X-Ratelimit-Remaining: 999
X-Request-Id: 8c1c4e97-4b72-4d2b-98b3-3a83fbc7d9a1

{"status":"success","message":"Upstream response OK"}
```

---

### 4. Start the Frontend Dashboard

Navigate to `frontend/` and launch the Next.js development server:

```bash
cd frontend
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) in your browser to interact with the topology simulator and traffic controls.

---

## 📊 Observability & Monitoring

### Grafana Dashboard (`http://localhost:3030`)
1. Log in using `admin` / `admin`.
2. Navigate to **Dashboards > Rate Limiter Dashboard**.
3. View real-time graphs for:
   - Total Traffic vs. Allowed vs. Rate-Limited Requests (429s).
   - p50, p95, and p99 Gateway Latencies.
   - Redis Operation Latency & Lua Execution Durations.
   - Circuit Breaker State & Failures.

---

## ⚡ Load Testing & Microbenchmarks

The repository includes pre-configured Vegeta and k6 benchmark suites inside `backend/scratch/`:

```bash
# Test local algorithm correctness & accuracy
./backend/scratch/exp_a.sh

# Measure tail latency (p95 / p99) under load
./backend/scratch/exp_b.sh

# Simulate Chaos Mode: Kills Redis mid-traffic to test Circuit Breaker fail-open policy
./backend/scratch/exp_d_chaos.sh

# High-Throughput Stress Test: 5,000 RPS for 60s
./backend/scratch/exp_f_max.sh
```

---

## 📚 Documentation & Deep Dives

- 📖 **[Backend Architecture & Guide](file:///home/streamliner/rate-liimter/backend/README.md)**: Details on rate-limiting algorithms, middleware chains, and Go package structure.
- 🎓 **[Educational Learning Guide](file:///home/streamliner/rate-liimter/backend/LearningGuide.md)**: Deep dive into step-by-step phase implementations.
- 📈 **[Performance Benchmarks Report](file:///home/streamliner/rate-liimter/backend/docs/BENCHMARKS.md)**: Detailed report on latency, concurrency limits, and Redis Lua performance.
- 💻 **[Frontend README](file:///home/streamliner/rate-liimter/frontend/README.md)**: Frontend architecture details and Next.js configuration.

---

## 📜 License

[MIT](LICENSE)
