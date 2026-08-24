'use client';

import React from 'react';
import { BarChart3, Cpu, Zap, ShieldAlert, CheckCircle2, TrendingUp, Layers } from 'lucide-react';

export const ScalingSection: React.FC = () => {
  return (
    <div className="w-full flex flex-col gap-8">
      {/* Header Banner */}
      <div className="bg-[#09090b] border border-[#27272a] rounded-xl p-6 shadow-xl flex flex-col gap-2">
        <div className="flex items-center gap-3">
          <TrendingUp className="w-6 h-6 text-white" />
          <h1 className="text-xl font-bold text-white tracking-wide uppercase font-mono">
            Multi-Node Scaling & Performance Telemetry
          </h1>
        </div>
        <p className="text-sm text-zinc-400 max-w-3xl">
          Empirical benchmark results measured across NGINX load balancer, 3 gateway replicas, and atomic Redis Lua script evaluation under Vegeta and k6 load generators.
        </p>
      </div>

      {/* Metrics Cards Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-[#09090b] border border-[#27272a] rounded-xl p-6 flex flex-col gap-4 shadow-lg">
          <div className="flex items-center justify-between">
            <span className="text-xs font-mono text-zinc-400 uppercase">Throughput Stress Test</span>
            <Zap className="w-5 h-5 text-white" />
          </div>
          <div className="flex flex-col">
            <span className="text-3xl font-extrabold font-mono text-white">5,000 RPS</span>
            <span className="text-xs text-zinc-400 mt-1">300,000 requests in 60s without TCP port exhaustion</span>
          </div>
          <div className="w-full bg-[#18181b] h-2 rounded-full overflow-hidden mt-2">
            <div className="bg-white h-full w-full" />
          </div>
          <span className="text-[11px] text-zinc-500 font-mono">Vegeta Microbenchmark (`exp_f_max.sh`)</span>
        </div>

        <div className="bg-[#09090b] border border-[#27272a] rounded-xl p-6 flex flex-col gap-4 shadow-lg">
          <div className="flex items-center justify-between">
            <span className="text-xs font-mono text-zinc-400 uppercase">p95 Tail Latency</span>
            <BarChart3 className="w-5 h-5 text-white" />
          </div>
          <div className="flex flex-col">
            <span className="text-3xl font-extrabold font-mono text-white">0.78 ms</span>
            <span className="text-xs text-zinc-400 mt-1">End-to-End latency overhead including Redis EVALSHA</span>
          </div>
          <div className="w-full bg-[#18181b] h-2 rounded-full overflow-hidden mt-2">
            <div className="bg-white h-full w-[85%]" />
          </div>
          <span className="text-[11px] text-zinc-500 font-mono">Prometheus Histogram Overhead</span>
        </div>

        <div className="bg-[#09090b] border border-[#27272a] rounded-xl p-6 flex flex-col gap-4 shadow-lg">
          <div className="flex items-center justify-between">
            <span className="text-xs font-mono text-zinc-400 uppercase">Chaos Recovery</span>
            <ShieldAlert className="w-5 h-5 text-white" />
          </div>
          <div className="flex flex-col">
            <span className="text-3xl font-extrabold font-mono text-white">0.00 s</span>
            <span className="text-xs text-zinc-400 mt-1">Downtime during mid-flight Redis cluster crash</span>
          </div>
          <div className="w-full bg-[#18181b] h-2 rounded-full overflow-hidden mt-2">
            <div className="bg-white h-full w-full" />
          </div>
          <span className="text-[11px] text-zinc-500 font-mono">Circuit Breaker Fail-Open (`exp_d_chaos.sh`)</span>
        </div>
      </div>

      {/* Algorithm Performance Matrix */}
      <div className="bg-[#09090b] border border-[#27272a] rounded-xl p-6 shadow-xl flex flex-col gap-6">
        <div className="flex items-center justify-between border-b border-[#27272a] pb-4">
          <div className="flex items-center gap-3">
            <Layers className="w-5 h-5 text-white" />
            <h2 className="text-base font-bold text-white font-mono uppercase">
              Swappable Algorithm Benchmark Matrix
            </h2>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs font-mono text-zinc-300">
            <thead className="bg-[#121215] text-zinc-400 uppercase border-b border-[#27272a]">
              <tr>
                <th className="p-3.5">Algorithm</th>
                <th className="p-3.5">Accuracy</th>
                <th className="p-3.5">Memory / Key</th>
                <th className="p-3.5">Burst Handling</th>
                <th className="p-3.5">Atomic Lua Cost</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#27272a]">
              <tr className="hover:bg-[#121215]/50 transition-colors">
                <td className="p-3.5 font-bold text-white">Sliding Window Counter</td>
                <td className="p-3.5">99.9% (Weighted)</td>
                <td className="p-3.5">O(1) - 2 Keys</td>
                <td className="p-3.5">Smooth Rate</td>
                <td className="p-3.5 text-white font-bold">&lt; 0.4ms</td>
              </tr>
              <tr className="hover:bg-[#121215]/50 transition-colors">
                <td className="p-3.5 font-bold text-white">Token Bucket</td>
                <td className="p-3.5">100% (Strict)</td>
                <td className="p-3.5">O(1) - 1 Hash</td>
                <td className="p-3.5">Permits Burst up to Capacity</td>
                <td className="p-3.5 text-white font-bold">&lt; 0.3ms</td>
              </tr>
              <tr className="hover:bg-[#121215]/50 transition-colors">
                <td className="p-3.5 font-bold text-white">Sliding Window Log</td>
                <td className="p-3.5">100% (Exact)</td>
                <td className="p-3.5">O(N) - Sorted Set</td>
                <td className="p-3.5">Exact Micro-Intervals</td>
                <td className="p-3.5 text-white font-bold">&lt; 0.9ms</td>
              </tr>
              <tr className="hover:bg-[#121215]/50 transition-colors">
                <td className="p-3.5 font-bold text-white">Leaky Bucket</td>
                <td className="p-3.5">100% (Queued)</td>
                <td className="p-3.5">O(Queue Size)</td>
                <td className="p-3.5">Constant Outflow Rate</td>
                <td className="p-3.5 text-white font-bold">&lt; 0.5ms</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
