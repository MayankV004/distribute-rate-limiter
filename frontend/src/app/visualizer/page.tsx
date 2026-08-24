'use client';

import React, { useState, useEffect, useRef, useCallback } from 'react';
import { TopologyCanvas, RequestParticle } from '@/components/TopologyCanvas';
import { TrafficController } from '@/components/TrafficController';
import { StatsGrid } from '@/components/StatsGrid';

export default function VisualizerPage() {
  const [algorithm, setAlgorithm] = useState('sliding_window_counter');
  const [tier, setTier] = useState('free'); // free, pro, admin
  const [rps, setRps] = useState(50);
  const [isStreaming, setIsStreaming] = useState(false);
  const [isChaosActive, setIsChaosActive] = useState(false);
  const [isLiveMode, setIsLiveMode] = useState(true);

  // Telemetry Counters
  const [total, setTotal] = useState(0);
  const [passed, setPassed] = useState(0);
  const [denied, setDenied] = useState(0);
  const [latencyMs, setLatencyMs] = useState(0.78);

  // Animation Particles
  const [particles, setParticles] = useState<RequestParticle[]>([]);

  // Sliding Window Mock Counter
  const mockHistoryRef = useRef<number[]>([]);

  // Process a requested number of requests
  const processRequests = useCallback(
    async (count: number) => {
      const apiKey =
        tier === 'free' ? 'key_free_example' : tier === 'pro' ? 'key_pro_example' : 'key_admin_example';
      const quotaLimit = tier === 'free' ? 1000 : tier === 'pro' ? 10000 : 999999;
      const now = Date.now();

      const newParticles: RequestParticle[] = [];
      let newPassed = 0;
      let newDenied = 0;
      let totalLatencyAcc = 0;

      if (isLiveMode) {
        // Process exact 'count' requests in sub-batches of 50
        const subBatchSize = 50;
        let processedCount = 0;

        for (let chunk = 0; chunk < count; chunk += subBatchSize) {
          const currentChunkSize = Math.min(subBatchSize, count - chunk);
          const startTime = performance.now();

          const fetchPromises = Array.from({ length: currentChunkSize }).map(() =>
            fetch('/api/proxy-gateway', {
              method: 'GET',
              headers: { 'X-API-Key': apiKey },
            }).catch(() => null)
          );

          const responses = await Promise.all(fetchPromises);
          const elapsed = performance.now() - startTime;
          totalLatencyAcc += elapsed / currentChunkSize;

          responses.forEach((res) => {
            const isAllowed = res ? res.status === 200 : true;
            if (isAllowed) newPassed++;
            else newDenied++;

            newParticles.push({
              id: `${now}-${Math.random()}`,
              status: isAllowed ? 'passed' : 'denied',
              gatewayIndex: Math.floor(Math.random() * 3),
            });
          });

          processedCount += currentChunkSize;
        }

        const avgChunkLatency = totalLatencyAcc / Math.ceil(count / subBatchSize);
        setLatencyMs((prev) => prev * 0.5 + avgChunkLatency * 0.5);
      } else {
        // Offline Simulation Mode
        mockHistoryRef.current = mockHistoryRef.current.filter((t) => now - t < 1000);

        for (let i = 0; i < count; i++) {
          const currentCount = mockHistoryRef.current.length;
          const isAllowed = isChaosActive || currentCount < quotaLimit / 60;
          if (isAllowed) mockHistoryRef.current.push(now);

          if (isAllowed) newPassed++;
          else newDenied++;

          newParticles.push({
            id: `${now}-${Math.random()}`,
            status: isAllowed ? 'passed' : 'denied',
            gatewayIndex: Math.floor(Math.random() * 3),
          });
        }
        totalLatencyAcc = isChaosActive ? 0.95 : 0.4 + Math.random() * 0.2;
        setLatencyMs((prev) => prev * 0.7 + totalLatencyAcc * 0.3);
      }

      setTotal((prev) => prev + count);
      setPassed((prev) => prev + newPassed);
      setDenied((prev) => prev + newDenied);

      setParticles((prev) => [...prev.slice(-30), ...newParticles.slice(-15)]);
    },
    [tier, isChaosActive, isLiveMode]
  );

  // Clean old particles after animation duration
  useEffect(() => {
    const timer = setInterval(() => {
      setParticles((prev) => (prev.length > 0 ? prev.slice(1) : prev));
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  // Continuous Traffic Streamer
  useEffect(() => {
    if (!isStreaming) return;

    const tickRateMs = rps >= 1000 ? 50 : rps >= 200 ? 100 : 200;
    const batchPerTick = Math.max(1, Math.round((rps * tickRateMs) / 1000));

    const timer = setInterval(() => {
      processRequests(batchPerTick);
    }, tickRateMs);

    return () => clearInterval(timer);
  }, [isStreaming, rps, processRequests]);

  return (
    <div className="w-full flex flex-col gap-6">
      {/* Top Layout: Controller & Full-Width Widescreen Topology */}
      <div className="grid grid-cols-1 xl:grid-cols-[340px_1fr] gap-6 items-start">
        <TrafficController
          algorithm={algorithm}
          setAlgorithm={setAlgorithm}
          tier={tier}
          setTier={setTier}
          rps={rps}
          setRps={setRps}
          isStreaming={isStreaming}
          toggleStream={() => setIsStreaming((prev) => !prev)}
          fireBurst={(count) => processRequests(count)}
          isChaosActive={isChaosActive}
          toggleChaos={() => setIsChaosActive((prev) => !prev)}
          resetStats={() => {
            setTotal(0);
            setPassed(0);
            setDenied(0);
            setParticles([]);
            mockHistoryRef.current = [];
          }}
          isLiveMode={isLiveMode}
          setIsLiveMode={setIsLiveMode}
        />

        <TopologyCanvas
          particles={particles}
          isChaosActive={isChaosActive}
          algorithm={algorithm}
          isLiveMode={isLiveMode}
          totalRequests={total}
          passedCount={passed}
          deniedCount={denied}
        />
      </div>

      {/* Telemetry Stats Bar */}
      <StatsGrid
        total={total}
        passed={passed}
        denied={denied}
        latencyMs={latencyMs}
      />
    </div>
  );
}
