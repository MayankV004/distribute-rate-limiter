'use client';

import React from 'react';
import { Activity, CheckCircle, Ban, Clock } from 'lucide-react';

interface StatsGridProps {
  total: number;
  passed: number;
  denied: number;
  latencyMs: number;
}

export const StatsGrid: React.FC<StatsGridProps> = ({
  total,
  passed,
  denied,
  latencyMs,
}) => {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 w-full">
      <div className="bg-[#09090b] border border-[#27272a] rounded-xl p-4 flex flex-col justify-between shadow-md">
        <div className="flex items-center justify-between text-zinc-400">
          <span className="text-xs font-mono uppercase tracking-wider">Total Requests</span>
          <Activity className="w-4 h-4 text-white" />
        </div>
        <div className="mt-3">
          <span className="text-2xl font-bold font-mono text-white">{total.toLocaleString()}</span>
        </div>
      </div>

      <div className="bg-[#09090b] border border-[#27272a] rounded-xl p-4 flex flex-col justify-between shadow-md">
        <div className="flex items-center justify-between text-zinc-400">
          <span className="text-xs font-mono uppercase tracking-wider">Passed (200 OK)</span>
          <CheckCircle className="w-4 h-4 text-white" />
        </div>
        <div className="mt-3">
          <span className="text-2xl font-bold font-mono text-white">{passed.toLocaleString()}</span>
          <span className="text-[11px] text-zinc-400 font-mono ml-2">
            ({total > 0 ? ((passed / total) * 100).toFixed(1) : '100'}%)
          </span>
        </div>
      </div>

      <div className="bg-[#09090b] border border-[#27272a] rounded-xl p-4 flex flex-col justify-between shadow-md">
        <div className="flex items-center justify-between text-zinc-400">
          <span className="text-xs font-mono uppercase tracking-wider">Denied (429 Rate Limit)</span>
          <Ban className="w-4 h-4 text-zinc-400" />
        </div>
        <div className="mt-3">
          <span className="text-2xl font-bold font-mono text-zinc-300">{denied.toLocaleString()}</span>
          <span className="text-[11px] text-zinc-500 font-mono ml-2">
            ({total > 0 ? ((denied / total) * 100).toFixed(1) : '0'}%)
          </span>
        </div>
      </div>

      <div className="bg-[#09090b] border border-[#27272a] rounded-xl p-4 flex flex-col justify-between shadow-md">
        <div className="flex items-center justify-between text-zinc-400">
          <span className="text-xs font-mono uppercase tracking-wider">p95 Latency Overhead</span>
          <Clock className="w-4 h-4 text-white" />
        </div>
        <div className="mt-3">
          <span className="text-2xl font-bold font-mono text-white">{latencyMs.toFixed(2)} ms</span>
          <span className="text-[11px] text-zinc-400 font-mono ml-2">Sub-millisecond</span>
        </div>
      </div>
    </div>
  );
};
