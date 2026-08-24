'use client';

import React from 'react';
import { X, Server, Database, Globe, Check, Cpu, Terminal, Shield, Zap } from 'lucide-react';

export interface NodeDetailData {
  id: string;
  name: string;
  type: string;
  status: string;
  port: string;
  description: string;
  responsibilities: string[];
  metrics: { label: string; value: string }[];
  logs: string[];
}

interface NodeDetailModalProps {
  node: NodeDetailData | null;
  onClose: () => void;
}

export const NodeDetailModal: React.FC<NodeDetailModalProps> = ({ node, onClose }) => {
  if (!node) return null;

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/60 backdrop-blur-sm transition-opacity">
      <div className="w-full max-w-md bg-[var(--card-bg)] border-l border-[var(--border-color)] h-full p-6 flex flex-col justify-between shadow-2xl overflow-y-auto animate-in slide-in-from-right duration-200">
        <div className="flex flex-col gap-6">
          {/* Header */}
          <div className="flex items-center justify-between border-b border-[var(--border-color)] pb-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-[var(--panel-bg)] border border-[var(--border-color)] flex items-center justify-center text-[var(--foreground)] shadow-sm">
                {node.type === 'client' && <Globe className="w-5 h-5" />}
                {node.type === 'nginx' && <Server className="w-5 h-5" />}
                {node.type === 'gateway' && <Cpu className="w-5 h-5" />}
                {node.type === 'redis' && <Database className="w-5 h-5" />}
                {node.type === 'backend' && <Check className="w-5 h-5" />}
              </div>
              <div className="flex flex-col">
                <h2 className="text-base font-bold font-mono text-[var(--foreground)]">{node.name}</h2>
                <span className="text-xs font-mono text-[var(--text-muted)]">Port: {node.port}</span>
              </div>
            </div>
            <button
              onClick={onClose}
              className="p-1.5 rounded-lg border border-[var(--border-color)] text-[var(--text-muted)] hover:text-[var(--foreground)] hover:bg-[var(--panel-bg)] transition-colors"
            >
              <X className="w-4 h-4" />
            </button>
          </div>

          {/* Status Badge & Description */}
          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-mono text-[var(--text-muted)] uppercase">Status</span>
              <span className="px-2.5 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 text-xs font-mono font-semibold flex items-center gap-1.5">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                {node.status}
              </span>
            </div>
            <p className="text-xs text-[var(--text-muted)] leading-relaxed mt-1">
              {node.description}
            </p>
          </div>

          {/* Real-Time Metrics Cards */}
          <div className="flex flex-col gap-2">
            <span className="text-xs font-mono uppercase text-[var(--text-muted)] font-bold">
              Live Component Telemetry
            </span>
            <div className="grid grid-cols-2 gap-3">
              {node.metrics.map((m, idx) => (
                <div
                  key={idx}
                  className="bg-[var(--panel-bg)] border border-[var(--border-color)] p-3 rounded-lg flex flex-col"
                >
                  <span className="text-[10px] font-mono text-[var(--text-muted)]">{m.label}</span>
                  <span className="text-sm font-bold font-mono text-[var(--foreground)] mt-0.5">
                    {m.value}
                  </span>
                </div>
              ))}
            </div>
          </div>

          {/* Responsibilities List */}
          <div className="flex flex-col gap-2">
            <span className="text-xs font-mono uppercase text-[var(--text-muted)] font-bold">
              Architectural Responsibilities
            </span>
            <ul className="flex flex-col gap-2 text-xs text-[var(--text-muted)]">
              {node.responsibilities.map((r, idx) => (
                <li key={idx} className="flex items-start gap-2">
                  <span className="text-[var(--foreground)] font-bold">•</span>
                  <span>{r}</span>
                </li>
              ))}
            </ul>
          </div>

          {/* Component Logs */}
          <div className="flex flex-col gap-2">
            <span className="text-xs font-mono uppercase text-[var(--text-muted)] font-bold flex items-center gap-1.5">
              <Terminal className="w-3.5 h-3.5" /> Component Execution Logs
            </span>
            <div className="bg-[#050507] border border-[var(--border-color)] p-3 rounded-lg text-[11px] font-mono text-zinc-300 flex flex-col gap-1 max-h-36 overflow-y-auto">
              {node.logs.map((log, idx) => (
                <div key={idx} className="text-zinc-400">
                  {log}
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Footer Close */}
        <div className="border-t border-[var(--border-color)] pt-4 mt-6">
          <button
            onClick={onClose}
            className="w-full py-2 bg-[var(--foreground)] text-[var(--background)] font-bold rounded-lg text-xs hover:opacity-90 transition-opacity"
          >
            Close Component Detail
          </button>
        </div>
      </div>
    </div>
  );
};
