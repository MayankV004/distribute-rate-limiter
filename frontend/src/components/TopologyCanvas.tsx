'use client';

import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Server, Database, AlertOctagon, Cpu, Globe, Check, Info } from 'lucide-react';
import { NodeDetailModal, NodeDetailData } from '@/components/NodeDetailModal';

export interface RequestParticle {
  id: string;
  status: 'passed' | 'denied';
  gatewayIndex: number; // 0, 1, or 2
}

interface TopologyCanvasProps {
  particles: RequestParticle[];
  isChaosActive: boolean;
  algorithm: string;
  isLiveMode: boolean;
  totalRequests?: number;
  passedCount?: number;
  deniedCount?: number;
}

export const TopologyCanvas: React.FC<TopologyCanvasProps> = ({
  particles,
  isChaosActive,
  algorithm,
  isLiveMode,
  totalRequests = 0,
  passedCount = 0,
  deniedCount = 0,
}) => {
  const [selectedNode, setSelectedNode] = useState<NodeDetailData | null>(null);

  const handleNodeClick = (nodeType: string) => {
    switch (nodeType) {
      case 'client':
        setSelectedNode({
          id: 'client',
          name: 'HTTP Load Generator (Client)',
          type: 'client',
          status: 'Active Load Stream',
          port: 'Dynamic',
          description:
            'Generates REST API calls with identity headers (X-API-Key) to simulate single or multi-tenant burst traffic.',
          responsibilities: [
            'Attaches API key authentication headers',
            'Simulates configurable RPS traffic flows',
            'Receives 200 OK payloads and 429 Too Many Requests responses',
          ],
          metrics: [
            { label: 'Total Fired', value: totalRequests.toLocaleString() },
            { label: 'Mode', value: isLiveMode ? 'Live Gateway API' : 'Demo Simulation' },
          ],
          logs: [
            `[CLIENT] GET /api/v1/search X-API-Key: key_free_example`,
            `[CLIENT] Received 200 OK (duration: 0.78ms)`,
            `[CLIENT] Rate limit headers extracted: X-Ratelimit-Remaining: 999`,
          ],
        });
        break;

      case 'nginx':
        setSelectedNode({
          id: 'nginx',
          name: 'NGINX Load Balancer',
          type: 'nginx',
          status: '100% Healthy (Round Robin)',
          port: '80',
          description:
            'Primary ingress load balancer distributing HTTP requests evenly across stateless API gateway replicas.',
          responsibilities: [
            'Terminates external client TCP connections on port 80',
            'Applies round-robin proxy pass across gateway1, gateway2, and gateway3',
            'Manages keep-alive connections and upstream pool health',
          ],
          metrics: [
            { label: 'Upstream Pool', value: '3 Gateway Replicas' },
            { label: 'Algorithm', value: 'Round Robin' },
          ],
          logs: [
            `[NGINX] proxy_pass http://gateway_upstream;`,
            `[NGINX] Round-robin dispatching request to gateway2:8080`,
            `[NGINX] 200 OK relayed to client in 0.8ms`,
          ],
        });
        break;

      case 'gateway':
        setSelectedNode({
          id: 'gateway',
          name: 'Stateless API Gateway Replicas',
          type: 'gateway',
          status: '3/3 Replicas Live',
          port: '8080 / 8081 / 8082',
          description:
            'High-concurrency Go API Gateways executing middleware pipelines, rate limit quota evaluation, and reverse proxying.',
          responsibilities: [
            'Panic recovery & request ID tracing middleware',
            'Structured JSON logging with slog',
            'Identity extraction (X-API-Key) and quota evaluation via Redis EVALSHA',
          ],
          metrics: [
            { label: 'Active Replicas', value: '3 Replicas' },
            { label: 'Rate Limiter', value: algorithm.replace(/_/g, ' ') },
          ],
          logs: [
            `[GATEWAY] RequestID: 6e576f57-5d30-4bd8-a4c1-10dfb19b4475`,
            `[GATEWAY] Rate limit check for key_free_example -> ALLOWED`,
            `[GATEWAY] Reverse proxying to http://dummy-backend:9000`,
          ],
        });
        break;

      case 'redis':
        setSelectedNode({
          id: 'redis',
          name: 'Redis Cluster (Atomic Lua Engine)',
          type: 'redis',
          status: isChaosActive ? 'CIRCUIT BREAKER OPEN' : 'Cluster Healthy (EVALSHA)',
          port: '6379 / 26379',
          description:
            'Distributed in-memory store executing single-threaded atomic Lua scripts to maintain global quota consistency.',
          responsibilities: [
            'Atomic execution of EVALSHA script without check-and-act race conditions',
            'TTL key eviction and sliding window timestamp calculations',
            'Integrates with Toxiproxy for chaos resilience testing',
          ],
          metrics: [
            { label: 'EVALSHA Latency', value: '< 0.4 ms' },
            { label: 'Circuit Breaker', value: isChaosActive ? 'OPEN (Fail-Open)' : 'CLOSED' },
          ],
          logs: [
            `[REDIS] EVALSHA c8f4b109... 1 rate:key_free_example 1000 60`,
            `[REDIS] Return: 1 (ALLOWED) - Remaining: 999`,
            `[REDIS] Counter key TTL set to 60s`,
          ],
        });
        break;

      case 'backend':
        setSelectedNode({
          id: 'backend',
          name: 'Upstream Backend Server',
          type: 'backend',
          status: '200 OK (Healthy)',
          port: '9000',
          description:
            'Upstream microservice simulating REST endpoint responses when requests pass through rate limit checks.',
          responsibilities: [
            'Responds to proxies requests with JSON payload',
            'Reflects request headers and client IP trace',
          ],
          metrics: [
            { label: 'Passed 200 OK', value: passedCount.toLocaleString() },
            { label: 'Denied 429', value: deniedCount.toLocaleString() },
          ],
          logs: [
            `[BACKEND] GET /api/v1/search 200 OK`,
            `[BACKEND] Response payload: {"message":"Hello from dummy backend"}`,
          ],
        });
        break;
    }
  };

  return (
    <div className="relative w-full h-[580px] bg-[var(--panel-bg)] border border-[var(--border-color)] rounded-xl overflow-hidden flex flex-col shadow-2xl transition-colors">
      {/* Topology Header */}
      <div className="flex flex-wrap justify-between items-center px-6 py-3.5 border-b border-[var(--border-color)] bg-[var(--card-bg)]">
        <div className="flex items-center gap-3">
          <Cpu className="w-4 h-4 text-[var(--foreground)]" />
          <h2 className="text-xs font-bold text-[var(--foreground)] tracking-widest uppercase font-mono">
            Interactive System Topology (Click Any Node)
          </h2>
          <span className="text-[10px] font-mono px-2.5 py-0.5 rounded-full bg-[var(--panel-bg)] border border-[var(--border-color)] text-[var(--foreground)] flex items-center gap-1.5">
            <span className={`w-1.5 h-1.5 rounded-full ${isLiveMode ? 'bg-emerald-400 animate-pulse' : 'bg-zinc-400'}`} />
            Mode: {isLiveMode ? 'LIVE GATEWAY API' : 'SIMULATED DEMO'}
          </span>
        </div>

        <div className="flex items-center gap-6 text-xs font-mono">
          <span className="flex items-center gap-2 text-[var(--foreground)] font-semibold">
            <span className="w-3 h-3 rounded-full bg-emerald-400 shadow-[0_0_10px_rgba(52,211,153,0.8)]" />
            200 OK (Allowed)
          </span>
          <span className="flex items-center gap-2 text-[var(--text-muted)]">
            <span className="w-3 h-3 rounded-full bg-zinc-600 border border-zinc-400" />
            429 Denied (Rate Limited)
          </span>
        </div>
      </div>

      {/* SVG Stage */}
      <div className="relative flex-1 w-full h-full p-4 overflow-hidden">
        <svg
          viewBox="0 0 1000 500"
          preserveAspectRatio="xMidYMid meet"
          className="absolute inset-0 w-full h-full pointer-events-none"
        >
          {/* Paths */}
          <path d="M 130 250 L 200 250" stroke="var(--svg-stroke)" strokeWidth="2.5" fill="none" />

          <path d="M 320 250 C 390 250, 390 110, 415 110" stroke="var(--svg-stroke)" strokeWidth="2" fill="none" />
          <path d="M 320 250 L 415 250" stroke="var(--svg-stroke)" strokeWidth="2" fill="none" />
          <path d="M 320 250 C 390 250, 390 390, 415 390" stroke="var(--svg-stroke)" strokeWidth="2" fill="none" />

          <path
            d="M 545 110 C 620 110, 620 250, 655 250"
            stroke={isChaosActive ? '#ef4444' : 'var(--svg-stroke)'}
            strokeWidth="2"
            strokeDasharray={isChaosActive ? '4 4' : 'none'}
            fill="none"
          />
          <path
            d="M 545 250 L 655 250"
            stroke={isChaosActive ? '#ef4444' : 'var(--svg-stroke)'}
            strokeWidth="2"
            strokeDasharray={isChaosActive ? '4 4' : 'none'}
            fill="none"
          />
          <path
            d="M 545 390 C 620 390, 620 250, 655 250"
            stroke={isChaosActive ? '#ef4444' : 'var(--svg-stroke)'}
            strokeWidth="2"
            strokeDasharray={isChaosActive ? '4 4' : 'none'}
            fill="none"
          />

          <path d="M 545 110 C 720 110, 800 250, 850 250" stroke="var(--svg-stroke)" strokeWidth="1.5" strokeDasharray="3 3" fill="none" />
          <path d="M 545 250 L 850 250" stroke="var(--svg-stroke)" strokeWidth="1.5" strokeDasharray="3 3" fill="none" />
          <path d="M 545 390 C 720 390, 800 250, 850 250" stroke="var(--svg-stroke)" strokeWidth="1.5" fill="none" />
        </svg>

        {/* Clickable Nodes Layer */}
        <div className="relative w-full h-full">
          {/* Node 1: Client */}
          <button
            onClick={() => handleNodeClick('client')}
            className="absolute -translate-x-1/2 -translate-y-1/2 w-32 h-28 bg-[var(--card-bg)] border border-[var(--border-color)] rounded-xl flex flex-col items-center justify-center p-3 shadow-lg hover:scale-105 hover:border-[var(--foreground)] transition-all group z-10"
            style={{ left: '8%', top: '50%' }}
          >
            <Globe className="w-6 h-6 text-[var(--foreground)] mb-1 group-hover:rotate-12 transition-transform" />
            <span className="text-[10px] font-mono text-[var(--text-muted)] uppercase">CLIENT</span>
            <span className="text-xs font-bold text-[var(--foreground)]">HTTP Load</span>
            <span className="text-[9px] font-mono text-emerald-400 mt-1 px-1.5 py-0.5 rounded bg-emerald-500/10">
              Active Stream
            </span>
          </button>

          {/* Node 2: NGINX LB */}
          <button
            onClick={() => handleNodeClick('nginx')}
            className="absolute -translate-x-1/2 -translate-y-1/2 w-36 h-32 bg-[var(--card-bg)] border border-[var(--border-bright)] rounded-xl flex flex-col items-center justify-center p-3 shadow-xl hover:scale-105 hover:border-[var(--foreground)] transition-all group z-10"
            style={{ left: '26%', top: '50%' }}
          >
            <Server className="w-7 h-7 text-[var(--foreground)] mb-1" />
            <span className="text-[10px] font-mono text-[var(--text-muted)] uppercase">NGINX LB</span>
            <span className="text-xs font-bold text-[var(--foreground)]">Round-Robin</span>
            <span className="text-[9px] font-mono text-zinc-400 mt-1">Port :80</span>
            <span className="text-[9px] font-mono text-emerald-400 mt-1 px-1.5 py-0.5 rounded bg-emerald-500/10">
              100% Healthy
            </span>
          </button>

          {/* Node 3: Gateway Replicas */}
          <button
            onClick={() => handleNodeClick('gateway')}
            className="absolute -translate-x-1/2 -translate-y-1/2 flex flex-col gap-6 z-10 group"
            style={{ left: '48%', top: '50%' }}
          >
            {['Gateway 1 (:8080)', 'Gateway 2 (:8081)', 'Gateway 3 (:8082)'].map((gw) => (
              <div
                key={gw}
                className="w-38 h-14 bg-[var(--card-bg)] border border-[var(--border-color)] rounded-lg flex items-center justify-between px-3 shadow-md group-hover:border-[var(--foreground)] transition-all hover:scale-105"
              >
                <div className="flex flex-col text-left">
                  <span className="text-xs font-bold text-[var(--foreground)]">{gw.split(' ')[0]}</span>
                  <span className="text-[9px] font-mono text-[var(--text-muted)]">{gw.split(' ')[1]}</span>
                </div>
                <span className="w-2.5 h-2.5 rounded-full bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.8)]" />
              </div>
            ))}
          </button>

          {/* Node 4: Redis Store */}
          <button
            onClick={() => handleNodeClick('redis')}
            className={`absolute -translate-x-1/2 -translate-y-1/2 w-40 h-40 rounded-xl border p-3 flex flex-col items-center justify-center z-10 transition-all hover:scale-105 ${
              isChaosActive
                ? 'bg-red-950/20 border-red-500 text-red-400 shadow-[0_0_20px_rgba(239,68,68,0.3)]'
                : 'bg-[var(--card-bg)] border-[var(--border-bright)] text-[var(--foreground)] shadow-xl hover:border-[var(--foreground)]'
            }`}
            style={{ left: '72%', top: '50%' }}
          >
            {isChaosActive ? (
              <AlertOctagon className="w-8 h-8 text-red-400 animate-pulse mb-1" />
            ) : (
              <Database className="w-8 h-8 text-[var(--foreground)] mb-1" />
            )}
            <span className="text-[10px] font-mono text-[var(--text-muted)] uppercase">REDIS CLUSTER</span>
            <span className="text-xs font-bold text-center mt-0.5">Atomic Lua EVALSHA</span>
            <span className={`text-[9px] font-mono mt-1 px-1.5 py-0.5 rounded ${isChaosActive ? 'bg-red-500/20 text-red-400' : 'bg-emerald-500/10 text-emerald-400'}`}>
              {isChaosActive ? '[CIRCUIT OPEN]' : 'Cluster Healthy'}
            </span>
          </button>

          {/* Node 5: Upstream Backend */}
          <button
            onClick={() => handleNodeClick('backend')}
            className="absolute -translate-x-1/2 -translate-y-1/2 w-32 h-28 bg-[var(--card-bg)] border border-[var(--border-color)] rounded-xl flex flex-col items-center justify-center p-3 shadow-lg hover:scale-105 hover:border-[var(--foreground)] transition-all group z-10"
            style={{ left: '91%', top: '50%' }}
          >
            <Check className="w-6 h-6 text-[var(--foreground)] mb-1" />
            <span className="text-[10px] font-mono text-[var(--text-muted)] uppercase">UPSTREAM</span>
            <span className="text-xs font-bold text-[var(--foreground)]">Dummy Backend</span>
            <span className="text-[9px] font-mono text-emerald-400 mt-1 px-1.5 py-0.5 rounded bg-emerald-500/10">
              200 OK
            </span>
          </button>
        </div>

        {/* Dynamic Framer Motion Particles */}
        <AnimatePresence>
          {particles.map((p) => {
            const gwY = p.gatewayIndex === 0 ? '22%' : p.gatewayIndex === 1 ? '50%' : '78%';
            const endX = p.status === 'passed' ? '91%' : '48%';
            const endY = p.status === 'passed' ? '50%' : gwY;

            return (
              <motion.div
                key={p.id}
                initial={{ left: '8%', top: '50%', opacity: 1, scale: 1 }}
                animate={{
                  left: ['8%', '26%', '48%', endX],
                  top: ['50%', '50%', gwY, endY],
                  opacity: p.status === 'denied' ? [1, 1, 1, 0] : [1, 1, 1, 1],
                  scale: p.status === 'denied' ? [1, 1.3, 1.6, 0] : 1,
                }}
                transition={{ duration: 1.1, ease: 'easeInOut' }}
                className={`absolute w-4 h-4 rounded-full pointer-events-none z-30 -translate-x-1/2 -translate-y-1/2 flex items-center justify-center ${
                  p.status === 'passed'
                    ? 'bg-emerald-400 shadow-[0_0_14px_rgba(52,211,153,1)] border border-emerald-300'
                    : 'bg-zinc-600 border border-zinc-400 shadow-[0_0_8px_rgba(113,113,122,0.8)]'
                }`}
              >
                <div className={`w-1.5 h-1.5 rounded-full ${p.status === 'passed' ? 'bg-black' : 'bg-white'}`} />
              </motion.div>
            );
          })}
        </AnimatePresence>
      </div>

      {/* Node Detail Slide-Over Drawer */}
      <NodeDetailModal node={selectedNode} onClose={() => setSelectedNode(null)} />
    </div>
  );
};
