'use client';

import React from 'react';
import { Sliders, Play, Pause, Flame, RefreshCw, AlertTriangle, Radio, Zap } from 'lucide-react';

interface TrafficControllerProps {
  algorithm: string;
  setAlgorithm: (algo: string) => void;
  tier: string;
  setTier: (tier: string) => void;
  rps: number;
  setRps: (rps: number) => void;
  isStreaming: boolean;
  toggleStream: () => void;
  fireBurst: (count: number) => void;
  isChaosActive: boolean;
  toggleChaos: () => void;
  resetStats: () => void;
  isLiveMode: boolean;
  setIsLiveMode: (live: boolean) => void;
}

export const TrafficController: React.FC<TrafficControllerProps> = ({
  algorithm,
  setAlgorithm,
  tier,
  setTier,
  rps,
  setRps,
  isStreaming,
  toggleStream,
  fireBurst,
  isChaosActive,
  toggleChaos,
  resetStats,
  isLiveMode,
  setIsLiveMode,
}) => {
  const rpsPresets = [10, 50, 200, 1000, 5000];

  return (
    <aside className="w-full bg-[#09090b] border border-[#27272a] rounded-xl p-5 shadow-xl flex flex-col gap-4">
      {/* Header & Reset */}
      <div className="flex items-center justify-between border-b border-[#27272a] pb-3">
        <div className="flex items-center gap-2">
          <Sliders className="w-4 h-4 text-white" />
          <h2 className="text-xs font-bold text-white uppercase tracking-wider font-mono">
            Traffic & Policy Controller
          </h2>
        </div>
        <button
          onClick={resetStats}
          className="text-zinc-400 hover:text-white transition-colors p-1"
          title="Reset Counters"
        >
          <RefreshCw className="w-3.5 h-3.5" />
        </button>
      </div>

      {/* Execution Mode Switcher */}
      <div className="bg-[#121215] p-2 rounded-lg border border-[#27272a] flex flex-col gap-1.5">
        <span className="text-[11px] font-mono text-zinc-400 flex items-center gap-1.5">
          <Radio className="w-3.5 h-3.5 text-white" /> Execution Mode
        </span>
        <div className="grid grid-cols-2 gap-1 bg-[#09090b] p-1 rounded-md border border-[#27272a]">
          <button
            onClick={() => setIsLiveMode(true)}
            className={`py-1.5 px-2 rounded text-[11px] font-mono font-semibold transition-all ${
              isLiveMode
                ? 'bg-white text-black shadow-sm'
                : 'text-zinc-400 hover:text-white'
            }`}
          >
            Live Gateway API
          </button>
          <button
            onClick={() => setIsLiveMode(false)}
            className={`py-1.5 px-2 rounded text-[11px] font-mono font-semibold transition-all ${
              !isLiveMode
                ? 'bg-white text-black shadow-sm'
                : 'text-zinc-400 hover:text-white'
            }`}
          >
            Offline Demo
          </button>
        </div>
      </div>

      {/* Algorithm Selector */}
      <div className="flex flex-col gap-1">
        <label className="text-[11px] font-mono text-zinc-400">Rate Limiter Algorithm</label>
        <select
          value={algorithm}
          onChange={(e) => setAlgorithm(e.target.value)}
          className="bg-[#121215] border border-[#27272a] text-white text-xs rounded-lg px-3 py-2 outline-none focus:border-white transition-colors font-sans"
        >
          <option value="sliding_window_counter">Sliding Window Counter (Default)</option>
          <option value="token_bucket">Token Bucket</option>
          <option value="sliding_window_log">Sliding Window Log</option>
          <option value="leaky_bucket">Leaky Bucket</option>
        </select>
      </div>

      {/* Identity Quota Tier */}
      <div className="flex flex-col gap-1">
        <label className="text-[11px] font-mono text-zinc-400">Identity Tier & API Key</label>
        <select
          value={tier}
          onChange={(e) => setTier(e.target.value)}
          className="bg-[#121215] border border-[#27272a] text-white text-xs rounded-lg px-3 py-2 outline-none focus:border-white transition-colors font-sans"
        >
          <option value="free">Free Tier (1,000 req/min quota - key_free_example)</option>
          <option value="pro">Pro Tier (10,000 req/min quota - key_pro_example)</option>
          <option value="admin">Admin Tier (Unlimited quota)</option>
        </select>
      </div>

      {/* Extended Traffic RPS Control (Range up to 5,000 RPS) */}
      <div className="flex flex-col gap-2">
        <div className="flex justify-between items-center text-[11px] font-mono">
          <span className="text-zinc-400">Traffic Arrival Rate</span>
          <span className="text-white font-bold font-mono text-xs px-2 py-0.5 rounded bg-[#18181b] border border-[#27272a]">
            {rps.toLocaleString()} RPS
          </span>
        </div>

        {/* Range Slider 1 to 5,000 */}
        <input
          type="range"
          min="1"
          max="5000"
          step={rps <= 100 ? 5 : rps <= 1000 ? 50 : 250}
          value={rps}
          onChange={(e) => setRps(Number(e.target.value))}
          className="w-full accent-white bg-[#18181b] h-1.5 rounded-lg cursor-pointer"
        />

        {/* Preset Quick Buttons */}
        <div className="flex items-center justify-between gap-1 mt-1">
          {rpsPresets.map((val) => (
            <button
              key={val}
              onClick={() => setRps(val)}
              className={`flex-1 py-1 text-[10px] font-mono rounded border transition-all ${
                rps === val
                  ? 'bg-white text-black border-white font-bold'
                  : 'bg-[#121215] text-zinc-400 border-[#27272a] hover:text-white hover:border-[#3f3f46]'
              }`}
            >
              {val >= 1000 ? `${val / 1000}k` : val}
            </button>
          ))}
        </div>
      </div>

      {/* Chaos Mode Toggle */}
      <div className="flex items-center justify-between bg-[#121215] p-2.5 rounded-lg border border-[#27272a]">
        <div className="flex items-center gap-2">
          <AlertTriangle className={`w-4 h-4 ${isChaosActive ? 'text-white' : 'text-zinc-500'}`} />
          <div className="flex flex-col">
            <span className="text-xs font-semibold text-white">Simulate Redis Outage</span>
            <span className="text-[10px] text-zinc-400">Circuit Breaker Fail-Open</span>
          </div>
        </div>
        <button
          onClick={toggleChaos}
          className={`w-9 h-5 rounded-full p-0.5 transition-colors ${
            isChaosActive ? 'bg-white' : 'bg-[#27272a]'
          }`}
        >
          <div
            className={`w-4 h-4 rounded-full bg-black transition-transform ${
              isChaosActive ? 'translate-x-4' : 'translate-x-0'
            }`}
          />
        </button>
      </div>

      {/* Burst Actions */}
      <div className="flex flex-col gap-2 pt-1">
        <div className="grid grid-cols-3 gap-1.5">
          <button
            onClick={() => fireBurst(20)}
            className="py-2 bg-white text-black font-bold rounded-lg text-[11px] flex items-center justify-center gap-1 hover:bg-zinc-200 transition-colors shadow-md"
          >
            <Flame className="w-3.5 h-3.5 fill-black" />
            <span>20 Burst</span>
          </button>
          <button
            onClick={() => fireBurst(100)}
            className="py-2 bg-white text-black font-bold rounded-lg text-[11px] flex items-center justify-center gap-1 hover:bg-zinc-200 transition-colors shadow-md"
          >
            <Zap className="w-3.5 h-3.5 fill-black" />
            <span>100 Burst</span>
          </button>
          <button
            onClick={() => fireBurst(500)}
            className="py-2 bg-white text-black font-bold rounded-lg text-[11px] flex items-center justify-center gap-1 hover:bg-zinc-200 transition-colors shadow-md"
          >
            <Zap className="w-3.5 h-3.5 fill-black" />
            <span>500 Stress</span>
          </button>
        </div>

        <button
          onClick={toggleStream}
          className={`w-full py-2.5 border rounded-lg text-xs font-semibold flex items-center justify-center gap-2 transition-colors ${
            isStreaming
              ? 'bg-[#27272a] text-white border-[#3f3f46]'
              : 'bg-transparent text-white border-[#27272a] hover:bg-[#121215]'
          }`}
        >
          {isStreaming ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
          <span>{isStreaming ? 'Pause Auto-Stream' : 'Start Continuous Stream'}</span>
        </button>
      </div>
    </aside>
  );
};
