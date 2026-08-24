'use client';

import React from 'react';
import Link from 'next/link';
import { Zap, Activity, BookOpen, BarChart2, ShieldCheck, Layers, Cpu, Terminal, ArrowRight, CheckCircle2, Server } from 'lucide-react';

export default function LandingPage() {
  return (
    <div className="w-full flex flex-col gap-16 py-6">
      {/* HERO SECTION */}
      <section className="flex flex-col items-center text-center gap-6 max-w-4xl mx-auto pt-6">
        <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-[var(--panel-bg)] border border-[var(--border-color)] text-xs font-mono text-[var(--text-muted)] shadow-sm">
          <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
          <span>Go API Gateway + Atomic Redis Lua Engine</span>
        </div>

        <h1 className="text-4xl sm:text-6xl font-extrabold tracking-tight font-sans text-[var(--foreground)] leading-tight">
          High-Concurrency <br />
          <span className="bg-gradient-to-r from-zinc-200 via-zinc-400 to-zinc-600 bg-clip-text text-transparent">
            Distributed Rate Limiter
          </span>
        </h1>

        <p className="text-base sm:text-lg text-[var(--text-muted)] max-w-2xl leading-relaxed">
          A production-grade, stateless API Gateway enforcing strict global quota accuracy across $N$ replicas with atomic Redis Lua execution, circuit breaking, and sub-millisecond p95 latency.
        </p>

        {/* Hero Action CTAs */}
        <div className="flex flex-wrap items-center justify-center gap-4 mt-2">
          <Link
            href="/visualizer"
            className="px-6 py-3 bg-[var(--foreground)] text-[var(--background)] font-bold rounded-xl text-sm flex items-center gap-2 hover:opacity-90 transition-all shadow-lg hover:scale-105"
          >
            <Activity className="w-4 h-4" />
            <span>Launch Live Visualizer</span>
            <ArrowRight className="w-4 h-4" />
          </Link>

          <Link
            href="/docs"
            className="px-6 py-3 bg-[var(--panel-bg)] text-[var(--foreground)] border border-[var(--border-color)] font-semibold rounded-xl text-sm flex items-center gap-2 hover:bg-[var(--border-color)] transition-all shadow-sm"
          >
            <BookOpen className="w-4 h-4 text-[var(--text-muted)]" />
            <span>Explore Documentation</span>
          </Link>
        </div>
      </section>

      {/* STATS TICKER */}
      <section className="grid grid-cols-2 md:grid-cols-4 gap-4 max-w-5xl mx-auto w-full">
        <div className="bg-[var(--card-bg)] border border-[var(--border-color)] p-5 rounded-xl flex flex-col items-center text-center shadow-md">
          <span className="text-2xl font-bold font-mono text-[var(--foreground)]">5,000 RPS</span>
          <span className="text-xs text-[var(--text-muted)] font-mono mt-1">Stress Tested (Vegeta)</span>
        </div>
        <div className="bg-[var(--card-bg)] border border-[var(--border-color)] p-5 rounded-xl flex flex-col items-center text-center shadow-md">
          <span className="text-2xl font-bold font-mono text-[var(--foreground)]">&lt; 0.4 ms</span>
          <span className="text-xs text-[var(--text-muted)] font-mono mt-1">Atomic Lua Overhead</span>
        </div>
        <div className="bg-[var(--card-bg)] border border-[var(--border-color)] p-5 rounded-xl flex flex-col items-center text-center shadow-md">
          <span className="text-2xl font-bold font-mono text-[var(--foreground)]">3 Replicas</span>
          <span className="text-xs text-[var(--text-muted)] font-mono mt-1">Stateless Scaling</span>
        </div>
        <div className="bg-[var(--card-bg)] border border-[var(--border-color)] p-5 rounded-xl flex flex-col items-center text-center shadow-md">
          <span className="text-2xl font-bold font-mono text-[var(--foreground)]">0.00 s</span>
          <span className="text-xs text-[var(--text-muted)] font-mono mt-1">Failover Downtime</span>
        </div>
      </section>

      {/* FEATURE SHOWCASE GRID */}
      <section className="max-w-6xl mx-auto w-full flex flex-col gap-8">
        <div className="flex flex-col items-center text-center gap-2">
          <h2 className="text-2xl font-bold font-mono text-[var(--foreground)] uppercase tracking-wide">
            Built for Production Reliability
          </h2>
          <p className="text-xs text-[var(--text-muted)] max-w-xl">
            Engineered to handle high-concurrency API traffic with zero check-and-act race conditions.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="bg-[var(--card-bg)] border border-[var(--border-color)] p-6 rounded-xl flex flex-col gap-3 shadow-md hover:border-[var(--foreground)] transition-colors">
            <Zap className="w-6 h-6 text-[var(--foreground)]" />
            <h3 className="text-base font-bold font-mono text-[var(--foreground)]">Swappable Algorithms</h3>
            <p className="text-xs text-[var(--text-muted)] leading-relaxed">
              Instantly toggle between <em>Sliding Window Counter</em>, <em>Token Bucket</em>, <em>Sliding Window Log</em>, and <em>Leaky Bucket</em> via YAML configuration.
            </p>
          </div>

          <div className="bg-[var(--card-bg)] border border-[var(--border-color)] p-6 rounded-xl flex flex-col gap-3 shadow-md hover:border-[var(--foreground)] transition-colors">
            <Cpu className="w-6 h-6 text-[var(--foreground)]" />
            <h3 className="text-base font-bold font-mono text-[var(--foreground)]">Atomic Redis Lua Engine</h3>
            <p className="text-xs text-[var(--text-muted)] leading-relaxed">
              Executes single-threaded Lua scripts (`EVALSHA`) directly inside Redis, guaranteeing strict atomicity across distributed replica nodes.
            </p>
          </div>

          <div className="bg-[var(--card-bg)] border border-[var(--border-color)] p-6 rounded-xl flex flex-col gap-3 shadow-md hover:border-[var(--foreground)] transition-colors">
            <ShieldCheck className="w-6 h-6 text-[var(--foreground)]" />
            <h3 className="text-base font-bold font-mono text-[var(--foreground)]">Circuit Breaker & Fallbacks</h3>
            <p className="text-xs text-[var(--text-muted)] leading-relaxed">
              Protects downstream services during Redis outages with configurable Fail-Open or Fail-Closed fallback policies.
            </p>
          </div>
        </div>
      </section>

      {/* QUICK COMMAND LAUNCHER */}
      <section className="max-w-4xl mx-auto w-full bg-[var(--card-bg)] border border-[var(--border-color)] p-8 rounded-xl flex flex-col gap-4 shadow-xl text-center">
        <h3 className="text-lg font-bold font-mono text-[var(--foreground)]">Quick Docker Startup</h3>
        <p className="text-xs text-[var(--text-muted)]">
          Run the full 10-container infrastructure (NGINX, 3 Gateways, Redis, Prometheus, Grafana, Dummy Backend):
        </p>
        <pre className="bg-[#050507] text-white p-4 rounded-xl text-xs font-mono overflow-x-auto text-left mx-auto max-w-xl border border-[var(--border-color)]">
          <code>{`docker compose -f backend/deployments/docker-compose.yaml up -d`}</code>
        </pre>
      </section>
    </div>
  );
}
