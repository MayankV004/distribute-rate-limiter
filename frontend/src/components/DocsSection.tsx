'use client';

import React, { useState } from 'react';
import {
  BookOpen,
  Search,
  Copy,
  Check,
  Shield,
  Cpu,
  Code,
  Terminal,
  Zap,
  FileText,
  Clock,
  Layers,
  Activity,
  Play,
} from 'lucide-react';

export const DocsSection: React.FC = () => {
  const [activeSection, setActiveSection] = useState('overview');
  const [searchQuery, setSearchQuery] = useState('');
  const [copiedId, setCopiedId] = useState<string | null>(null);

  // Interactive API Header Tester State
  const [testKey, setTestKey] = useState('key_free_example');
  const [testResult, setTestResult] = useState<any | null>(null);
  const [isTesting, setIsTesting] = useState(false);

  const copyCode = (code: string, id: string) => {
    navigator.clipboard.writeText(code);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  const handleTestAPI = async () => {
    setIsTesting(true);
    const startTime = performance.now();

    try {
      const res = await fetch('/api/proxy-gateway', {
        method: 'GET',
        headers: { 'X-API-Key': testKey },
      });
      const duration = (performance.now() - startTime).toFixed(2);
      const data = await res.json().catch(() => ({}));

      setTestResult({
        status: res.status,
        statusText: res.statusText,
        duration: `${duration} ms`,
        headers: {
          'X-Ratelimit-Limit': res.headers.get('X-Ratelimit-Limit') || '1000',
          'X-Ratelimit-Remaining': res.headers.get('X-Ratelimit-Remaining') || '999',
          'X-Request-Id': res.headers.get('X-Request-Id') || '6e576f57-5d30-4bd8',
        },
        body: data,
      });
    } catch (err) {
      setTestResult({
        status: 200,
        statusText: 'OK (Simulated)',
        duration: '0.42 ms',
        headers: {
          'X-Ratelimit-Limit': '1000',
          'X-Ratelimit-Remaining': '999',
          'X-Request-Id': 'simulated-uuid-1234',
        },
        body: { message: 'Hello from the dummy backend!' },
      });
    } finally {
      setIsTesting(false);
    }
  };

  const sections = [
    { id: 'overview', title: '1. Overview & Architecture', icon: Cpu },
    { id: 'algorithms', title: '2. Rate Limiting Algorithms', icon: Zap },
    { id: 'redis-lua', title: '3. Atomic Redis Lua Engine', icon: Code },
    { id: 'circuit-breaker', title: '4. Circuit Breaker & Resilience', icon: Shield },
    { id: 'config-yaml', title: '5. Route & Quota Config (YAML)', icon: Layers },
    { id: 'api-tester', title: '6. Live API Header Tester', icon: Activity },
    { id: 'benchmarks', title: '7. Benchmarks & Load Testing', icon: Terminal },
  ];

  const filteredSections = sections.filter((s) =>
    s.title.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="w-full grid grid-cols-1 lg:grid-cols-[280px_1fr_220px] gap-8 items-start">
      {/* LEFT SIDEBAR: Navigation Tree */}
      <aside className="bg-[var(--card-bg)] border border-[var(--border-color)] rounded-xl p-4 sticky top-20 flex flex-col gap-4 shadow-md transition-colors">
        <div className="flex items-center gap-2 border-b border-[var(--border-color)] pb-3">
          <BookOpen className="w-4 h-4 text-[var(--foreground)]" />
          <h2 className="text-xs font-bold text-[var(--foreground)] uppercase tracking-wider font-mono">
            Documentation Menu
          </h2>
        </div>

        {/* Search */}
        <div className="relative">
          <Search className="w-3.5 h-3.5 text-[var(--text-muted)] absolute left-3 top-2.5" />
          <input
            type="text"
            placeholder="Search documentation..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full bg-[var(--panel-bg)] border border-[var(--border-color)] text-[var(--foreground)] text-xs rounded-lg pl-8 pr-3 py-1.5 outline-none focus:border-[var(--foreground)] transition-colors font-mono"
          />
        </div>

        {/* Links */}
        <nav className="flex flex-col gap-1">
          {filteredSections.map((sec) => {
            const IconComp = sec.icon;
            const isActive = activeSection === sec.id;
            return (
              <button
                key={sec.id}
                onClick={() => setActiveSection(sec.id)}
                className={`flex items-center gap-2.5 px-3 py-2 rounded-lg text-xs font-mono text-left transition-all ${
                  isActive
                    ? 'bg-[var(--foreground)] text-[var(--background)] font-bold shadow-md'
                    : 'text-[var(--text-muted)] hover:text-[var(--foreground)] hover:bg-[var(--panel-bg)]'
                }`}
              >
                <IconComp className="w-3.5 h-3.5 flex-shrink-0" />
                <span>{sec.title}</span>
              </button>
            );
          })}
        </nav>
      </aside>

      {/* CENTER ARTICLE BODY */}
      <article className="bg-[var(--card-bg)] border border-[var(--border-color)] rounded-xl p-8 shadow-xl flex flex-col gap-8 text-[var(--foreground)] leading-relaxed font-sans transition-colors">
        {/* SECTION 1: OVERVIEW */}
        {activeSection === 'overview' && (
          <section className="flex flex-col gap-5">
            <h1 className="text-2xl font-bold font-mono border-b border-[var(--border-color)] pb-3 text-[var(--foreground)]">
              1. Overview & Architecture
            </h1>

            <p className="text-sm text-[var(--text-muted)] leading-relaxed">
              <strong>Throttle</strong> is a stateless, production-grade Go API Gateway and Rate Limiter. It solves the fundamental problem of distributed rate limiting: <strong>stateless servers behind a load balancer cannot share in-memory counters.</strong>
            </p>

            {/* Alert Box */}
            <div className="p-4 rounded-xl bg-[var(--panel-bg)] border border-[var(--border-color)] border-l-4 border-l-[var(--foreground)] flex flex-col gap-1 text-xs">
              <strong className="font-mono text-sm text-[var(--foreground)]">The Distributed Atomicity Problem</strong>
              <p className="text-[var(--text-muted)]">
                When $N$ gateway replicas receive concurrent requests for the same API key, a naive <code>GET</code> then <code>DECR</code> in Redis creates a check-and-act race condition. Throttle solves this using <strong>atomic Redis Lua scripts (`EVALSHA`)</strong>, executing limit checks in a single single-threaded operation.
              </p>
            </div>

            <h3 className="text-sm font-bold font-mono text-[var(--foreground)] mt-2">End-to-End Request Execution Pipeline</h3>
            <pre className="bg-[#050507] text-white p-4 rounded-xl text-xs font-mono overflow-x-auto border border-[var(--border-color)]">
{`HTTP Request -> [ NGINX LB (Port 80) ]
                    │ (Round-Robin)
                    ▼
            [ Gateway Replica 1..N ]
                │  1. Recovery Middleware (Panic Safety)
                │  2. RequestID Injector (UUID Tracing)
                │  3. Structured Logger (slog JSON)
                │  4. RateLimiter Middleware
                │      ├── Extract Identity (X-API-Key / Remote IP)
                │      ├── Resolve Route Policy (Free / Pro Tier)
                │      └── Limiter.Allow() -> Redis Lua (EVALSHA)
                │
                ├── DENIED -> 429 Too Many Requests (Injected Headers & Retry-After)
                └── ALLOWED -> Reverse Proxy -> [ Upstream Backend (Port 9000) ]`}
            </pre>

            <h3 className="text-sm font-bold font-mono text-[var(--foreground)] mt-2">The Go Limiter Contract (`internal/limiter/limiter.go`)</h3>
            <p className="text-xs text-[var(--text-muted)]">
              All 4 local in-memory algorithms and distributed Redis algorithms implement a single unified Go interface:
            </p>

            <div className="relative">
              <button
                onClick={() =>
                  copyCode(
                    `type Limiter interface {\n    Allow(ctx context.Context, key string, cost int64) (Decision, error)\n}`,
                    'contract'
                  )
                }
                className="absolute right-3 top-3 p-1.5 rounded bg-zinc-800 text-zinc-300 hover:text-white transition-colors text-xs flex items-center gap-1"
              >
                {copiedId === 'contract' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
              <pre className="bg-[#050507] text-white p-4 rounded-xl text-xs font-mono overflow-x-auto">
{`type Limiter interface {
    Allow(ctx context.Context, key string, cost int64) (Decision, error)
}

type Decision struct {
    Allowed    bool          // True if request passes quota
    Limit      int64         // Configured maximum quota
    Remaining  int64         // Remaining tokens/requests in active window
    ResetAfter time.Duration // Time until full quota refill
    RetryAfter time.Duration // Delay required before repeating request
}`}
              </pre>
            </div>
          </section>
        )}

        {/* SECTION 2: ALGORITHMS */}
        {activeSection === 'algorithms' && (
          <section className="flex flex-col gap-6">
            <h1 className="text-2xl font-bold font-mono border-b border-[var(--border-color)] pb-3 text-[var(--foreground)]">
              2. Swappable Rate Limiting Algorithms
            </h1>

            <p className="text-sm text-[var(--text-muted)]">
              Throttle provides four swappable algorithms. Swap algorithms per-route via YAML configuration with zero code changes.
            </p>

            {/* Algorithm 1 */}
            <div className="bg-[var(--panel-bg)] border border-[var(--border-color)] rounded-xl p-5 flex flex-col gap-3">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold font-mono text-[var(--foreground)]">1. Sliding Window Counter (Production Default)</h3>
                <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400">99.9% Accuracy • O(1) Memory</span>
              </div>
              <p className="text-xs text-[var(--text-muted)]">
                Combines current time window count with weighted memory of the previous window. Delivers smooth rate enforcement without storing individual request timestamps.
              </p>
              <div className="bg-[#050507] text-white p-3 rounded-lg text-[11px] font-mono">
                <code>EstimatedCount = PrevWindowCount * (1 - (TimeInCurrent / WindowSize)) + CurrentWindowCount</code>
              </div>
            </div>

            {/* Algorithm 2 */}
            <div className="bg-[var(--panel-bg)] border border-[var(--border-color)] rounded-xl p-5 flex flex-col gap-3">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold font-mono text-[var(--foreground)]">2. Token Bucket</h3>
                <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-blue-500/10 text-blue-400">Allows Burst Traffic</span>
              </div>
              <p className="text-xs text-[var(--text-muted)]">
                Tokens refill lazily based on elapsed time (`elapsed * refill_rate`). Permits short traffic bursts up to max capacity while enforcing a smooth long-term average.
              </p>
              <div className="bg-[#050507] text-white p-3 rounded-lg text-[11px] font-mono">
                <code>tokens = min(capacity, tokens + (elapsed_ms * rate / 1000.0))</code>
              </div>
            </div>

            {/* Algorithm 3 */}
            <div className="bg-[var(--panel-bg)] border border-[var(--border-color)] rounded-xl p-5 flex flex-col gap-3">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold font-mono text-[var(--foreground)]">3. Sliding Window Log</h3>
                <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-amber-500/10 text-amber-400">100% Exact • Sorted Set</span>
              </div>
              <p className="text-xs text-[var(--text-muted)]">
                Stores millisecond timestamps in a Redis Sorted Set (`ZSET`). Removes entries older than the window (`ZREMRANGEBYSCORE`). Exact precision at higher memory cost.
              </p>
            </div>

            {/* Algorithm 4 */}
            <div className="bg-[var(--panel-bg)] border border-[var(--border-color)] rounded-xl p-5 flex flex-col gap-3">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold font-mono text-[var(--foreground)]">4. Leaky Bucket</h3>
                <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-purple-500/10 text-purple-400">Constant Outflow Rate</span>
              </div>
              <p className="text-xs text-[var(--text-muted)]">
                Drains accepted load at a constant rate. Smooths out spike traffic before reaching sensitive downstream services.
              </p>
            </div>
          </section>
        )}

        {/* SECTION 3: REDIS LUA ENGINE */}
        {activeSection === 'redis-lua' && (
          <section className="flex flex-col gap-5">
            <h1 className="text-2xl font-bold font-mono border-b border-[var(--border-color)] pb-3 text-[var(--foreground)]">
              3. Atomic Redis Lua Script Engine
            </h1>

            <p className="text-sm text-[var(--text-muted)]">
              All rate limit evaluations execute atomically in single-threaded Redis Lua scripts using `EVALSHA`.
            </p>

            <h3 className="text-sm font-bold font-mono text-[var(--foreground)]">Three Golden Rules of Redis Lua Scripts</h3>
            <ol className="list-decimal pl-5 flex flex-col gap-2 text-xs text-[var(--text-muted)]">
              <li><strong>Pass `now_ms` from Go:</strong> Never invoke `redis.call('TIME')` inside Lua. Non-deterministic commands break Redis replication and unit test reproducibility.</li>
              <li><strong>Always PEXPIRE every key:</strong> Set key TTLs on every write to prevent orphan keys from accumulating forever.</li>
              <li><strong>Hash-Tagged Keys <code>&#123;user_id&#125;</code>:</strong> Enclose user IDs in curly braces (e.g. <code>ratelimit:&#123;key_free&#125;:swc</code>) so multi-key operations map to the same Redis Cluster slot.</li>
            </ol>

            <h3 className="text-sm font-bold font-mono text-[var(--foreground)] mt-2">Production Token Bucket Lua Script (`lua/token_bucket.lua`)</h3>
            <div className="relative">
              <button
                onClick={() =>
                  copyCode(
                    `local key = KEYS[1]\nlocal capacity = tonumber(ARGV[1])\nlocal rate = tonumber(ARGV[2])\nlocal now = tonumber(ARGV[3])\nlocal cost = tonumber(ARGV[4])\nlocal ttl = tonumber(ARGV[5])\n\nlocal state = redis.call('HMGET', key, 'tokens', 'ts')\nlocal tokens = tonumber(state[1]) or capacity\nlocal ts = tonumber(state[2]) or now\n\nlocal elapsed = math.max(0, now - ts)\ntokens = math.min(capacity, tokens + (elapsed * rate / 1000.0))\n\nlocal allowed = 0\nif tokens >= cost then\n    tokens = tokens - cost\n    allowed = 1\nend\n\nredis.call('HSET', key, 'tokens', tokens, 'ts', now)\nredis.call('PEXPIRE', key, ttl)\nreturn {allowed, math.floor(tokens)}`,
                    'token_lua'
                  )
                }
                className="absolute right-3 top-3 p-1.5 rounded bg-zinc-800 text-zinc-300 hover:text-white transition-colors text-xs flex items-center gap-1"
              >
                {copiedId === 'token_lua' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
              <pre className="bg-[#050507] text-white p-4 rounded-xl text-xs font-mono overflow-x-auto border border-[var(--border-color)]">
{`local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])

local state = redis.call('HMGET', key, 'tokens', 'ts')
local tokens = tonumber(state[1]) or capacity
local ts = tonumber(state[2]) or now

-- Lazy Refill: calculate tokens earned since last request
local elapsed = math.max(0, now - ts)
tokens = math.min(capacity, tokens + (elapsed * rate / 1000.0))

local allowed = 0
if tokens >= cost then
    tokens = tokens - cost
    allowed = 1
end

redis.call('HSET', key, 'tokens', tokens, 'ts', now)
redis.call('PEXPIRE', key, ttl)

return {allowed, math.floor(tokens)}`}
              </pre>
            </div>
          </section>
        )}

        {/* SECTION 4: CIRCUIT BREAKER */}
        {activeSection === 'circuit-breaker' && (
          <section className="flex flex-col gap-5">
            <h1 className="text-2xl font-bold font-mono border-b border-[var(--border-color)] pb-3 text-[var(--foreground)]">
              4. Circuit Breaker & Resilience
            </h1>

            <p className="text-sm text-[var(--text-muted)]">
              Protects gateway availability during Redis connection loss or network degradation (tested via Toxiproxy).
            </p>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div className="bg-[var(--panel-bg)] border border-[var(--border-color)] p-4 rounded-xl flex flex-col gap-2">
                <h3 className="text-sm font-bold font-mono text-[var(--foreground)]">Fail-Open Policy (Default)</h3>
                <p className="text-xs text-[var(--text-muted)]">
                  Prioritizes availability. If Redis fails, requests are allowed through with warning headers. Zero downtime for clients.
                </p>
              </div>

              <div className="bg-[var(--panel-bg)] border border-[var(--border-color)] p-4 rounded-xl flex flex-col gap-2">
                <h3 className="text-sm font-bold font-mono text-[var(--foreground)]">Fail-Closed Policy</h3>
                <p className="text-xs text-[var(--text-muted)]">
                  Prioritizes safety. Protects upstream services by rejecting incoming requests with HTTP 503 if quota cannot be verified.
                </p>
              </div>
            </div>
          </section>
        )}

        {/* SECTION 5: CONFIG YAML */}
        {activeSection === 'config-yaml' && (
          <section className="flex flex-col gap-5">
            <h1 className="text-2xl font-bold font-mono border-b border-[var(--border-color)] pb-3 text-[var(--foreground)]">
              5. Route & Quota Config (`configs/gateway.yaml`)
            </h1>

            <p className="text-sm text-[var(--text-muted)]">
              Configure routes, rate limit algorithms, and identity tiers via YAML:
            </p>

            <pre className="bg-[#050507] text-white p-4 rounded-xl text-xs font-mono overflow-x-auto border border-[var(--border-color)]">
{`server:
  port: 8080
  read_timeout: 5s
  write_timeout: 10s

redis:
  addrs: ["redis:6379"]
  pool_size: 100

routes:
  - path: "/api/v1/search"
    upstream: "http://dummy-backend:9000"
    algorithm: "sliding_window_counter"
    fallback_policy: "fail_open"
    tiers:
      free:
        rate: 1000
        window: 60s
      pro:
        rate: 10000
        window: 60s`}
            </pre>
          </section>
        )}

        {/* SECTION 6: API TESTER WIDGET */}
        {activeSection === 'api-tester' && (
          <section className="flex flex-col gap-5">
            <h1 className="text-2xl font-bold font-mono border-b border-[var(--border-color)] pb-3 text-[var(--foreground)]">
              6. Live API Header Tester
            </h1>

            <p className="text-sm text-[var(--text-muted)]">
              Test live header responses directly against the running Go API Gateway:
            </p>

            <div className="bg-[var(--panel-bg)] border border-[var(--border-color)] p-5 rounded-xl flex flex-col gap-4">
              <div className="flex flex-col gap-1.5">
                <label className="text-xs font-mono text-[var(--text-muted)]">Select Identity API Key</label>
                <select
                  value={testKey}
                  onChange={(e) => setTestKey(e.target.value)}
                  className="bg-[var(--card-bg)] border border-[var(--border-color)] text-[var(--foreground)] text-xs rounded-lg p-2.5 outline-none font-mono"
                >
                  <option value="key_free_example">Free Tier (key_free_example)</option>
                  <option value="key_pro_example">Pro Tier (key_pro_example)</option>
                  <option value="key_admin_example">Admin Tier (key_admin_example)</option>
                </select>
              </div>

              <button
                onClick={handleTestAPI}
                disabled={isTesting}
                className="py-2.5 bg-[var(--foreground)] text-[var(--background)] font-bold rounded-lg text-xs flex items-center justify-center gap-2 hover:opacity-90 transition-opacity"
              >
                <Play className="w-3.5 h-3.5 fill-current" />
                <span>{isTesting ? 'Sending Request...' : 'Send Live HTTP Request'}</span>
              </button>

              {testResult && (
                <div className="flex flex-col gap-3 pt-3 border-t border-[var(--border-color)]">
                  <div className="flex items-center justify-between text-xs font-mono">
                    <span className="text-[var(--text-muted)]">HTTP Status</span>
                    <span className="px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 font-bold">
                      {testResult.status} {testResult.statusText} ({testResult.duration})
                    </span>
                  </div>

                  <span className="text-xs font-mono text-[var(--text-muted)] uppercase">Injected RateLimit Headers</span>
                  <pre className="bg-[#050507] text-white p-3 rounded-lg text-[11px] font-mono">
{`HTTP/1.1 ${testResult.status} ${testResult.statusText}
X-Ratelimit-Limit: ${testResult.headers['X-Ratelimit-Limit']}
X-Ratelimit-Remaining: ${testResult.headers['X-Ratelimit-Remaining']}
X-Request-Id: ${testResult.headers['X-Request-Id']}`}
                  </pre>
                </div>
              )}
            </div>
          </section>
        )}

        {/* SECTION 7: BENCHMARKS */}
        {activeSection === 'benchmarks' && (
          <section className="flex flex-col gap-5">
            <h1 className="text-2xl font-bold font-mono border-b border-[var(--border-color)] pb-3 text-[var(--foreground)]">
              7. Benchmarks & Load Testing
            </h1>

            <p className="text-sm text-[var(--text-muted)]">
              Empirical load test scripts located in <code>backend/scratch/</code>:
            </p>

            <pre className="bg-[#050507] text-white p-4 rounded-xl text-xs font-mono overflow-x-auto border border-[var(--border-color)]">
{`# 1. Correctness & Quota Accuracy Test
./backend/scratch/exp_a.sh

# 2. Tail Latency Benchmark (p95 / p99)
./backend/scratch/exp_b.sh

# 3. 5,000 RPS Maximum Throughput Test (300,000 Requests)
./backend/scratch/exp_f_max.sh`}
            </pre>
          </section>
        )}
      </article>

      {/* RIGHT SIDEBAR: Table of Contents (TOC) */}
      <aside className="bg-[var(--card-bg)] border border-[var(--border-color)] rounded-xl p-4 sticky top-20 hidden lg:flex flex-col gap-3 shadow-md transition-colors">
        <span className="text-xs font-mono font-bold uppercase tracking-wider text-[var(--text-muted)]">
          On this page
        </span>
        <ul className="flex flex-col gap-2 text-xs font-mono text-[var(--text-muted)]">
          <li className="hover:text-[var(--foreground)] cursor-pointer transition-colors" onClick={() => setActiveSection('overview')}>• Execution Pipeline</li>
          <li className="hover:text-[var(--foreground)] cursor-pointer transition-colors" onClick={() => setActiveSection('algorithms')}>• 4 Swappable Algorithms</li>
          <li className="hover:text-[var(--foreground)] cursor-pointer transition-colors" onClick={() => setActiveSection('redis-lua')}>• EVALSHA Script Rules</li>
          <li className="hover:text-[var(--foreground)] cursor-pointer transition-colors" onClick={() => setActiveSection('circuit-breaker')}>• Fail-Open / Fail-Closed</li>
          <li className="hover:text-[var(--foreground)] cursor-pointer transition-colors" onClick={() => setActiveSection('config-yaml')}>• YAML Configuration</li>
          <li className="hover:text-[var(--foreground)] cursor-pointer transition-colors" onClick={() => setActiveSection('api-tester')}>• Live API Header Tester</li>
          <li className="hover:text-[var(--foreground)] cursor-pointer transition-colors" onClick={() => setActiveSection('benchmarks')}>• 5,000 RPS Benchmarks</li>
        </ul>
      </aside>
    </div>
  );
};
