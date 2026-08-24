'use client';

import React from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Zap, Activity, BarChart2, BookOpen, Sun, Moon, Home } from 'lucide-react';
import { useTheme } from '@/context/ThemeContext';

export const Navbar: React.FC = () => {
  const pathname = usePathname();
  const { theme, toggleTheme } = useTheme();

  return (
    <header className="w-full bg-var-card border-b border-[var(--border-color)] px-6 py-3.5 flex items-center justify-between sticky top-0 z-50 transition-colors">
      {/* Brand Logo & Name */}
      <Link href="/" className="flex items-center gap-3 group">
        <div className="w-9 h-9 rounded-lg bg-[var(--foreground)] text-[var(--background)] flex items-center justify-center font-bold shadow-md group-hover:scale-105 transition-transform">
          <Zap className="w-5 h-5 fill-current stroke-current" />
        </div>
        <div className="flex flex-col">
          <div className="flex items-center gap-2">
            <span className="font-bold text-lg tracking-wider font-mono text-[var(--foreground)]">
              THROTTLE
            </span>
            <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-[var(--panel-bg)] border border-[var(--border-color)] text-[var(--text-muted)]">
              v1.0.0
            </span>
          </div>
          <span className="text-[11px] text-[var(--text-muted)] font-sans">
            Distributed API Gateway & Rate Limiter
          </span>
        </div>
      </Link>

      {/* Navigation Links */}
      <nav className="flex items-center gap-1 bg-[var(--panel-bg)] p-1 rounded-lg border border-[var(--border-color)]">
        <Link
          href="/"
          className={`flex items-center gap-2 px-3.5 py-1.5 rounded-md text-xs font-medium transition-all ${
            pathname === '/'
              ? 'bg-[var(--border-color)] text-[var(--foreground)] font-bold shadow-sm'
              : 'text-[var(--text-muted)] hover:text-[var(--foreground)] hover:bg-[var(--card-bg)]'
          }`}
        >
          <Home className="w-3.5 h-3.5" />
          <span>Home</span>
        </Link>

        <Link
          href="/visualizer"
          className={`flex items-center gap-2 px-3.5 py-1.5 rounded-md text-xs font-medium transition-all ${
            pathname === '/visualizer'
              ? 'bg-[var(--border-color)] text-[var(--foreground)] font-bold shadow-sm'
              : 'text-[var(--text-muted)] hover:text-[var(--foreground)] hover:bg-[var(--card-bg)]'
          }`}
        >
          <Activity className="w-3.5 h-3.5" />
          <span>Visualizer</span>
        </Link>

        <Link
          href="/scaling"
          className={`flex items-center gap-2 px-3.5 py-1.5 rounded-md text-xs font-medium transition-all ${
            pathname === '/scaling'
              ? 'bg-[var(--border-color)] text-[var(--foreground)] font-bold shadow-sm'
              : 'text-[var(--text-muted)] hover:text-[var(--foreground)] hover:bg-[var(--card-bg)]'
          }`}
        >
          <BarChart2 className="w-3.5 h-3.5" />
          <span>Scaling & Telemetry</span>
        </Link>

        <Link
          href="/docs"
          className={`flex items-center gap-2 px-3.5 py-1.5 rounded-md text-xs font-medium transition-all ${
            pathname.startsWith('/docs')
              ? 'bg-[var(--border-color)] text-[var(--foreground)] font-bold shadow-sm'
              : 'text-[var(--text-muted)] hover:text-[var(--foreground)] hover:bg-[var(--card-bg)]'
          }`}
        >
          <BookOpen className="w-3.5 h-3.5" />
          <span>Docs</span>
        </Link>
      </nav>

      {/* Right Tools: Theme Toggle & Live Status */}
      <div className="flex items-center gap-3">
        {/* Sun / Moon Theme Toggle */}
        <button
          onClick={toggleTheme}
          className="p-2 rounded-lg border border-[var(--border-color)] bg-[var(--panel-bg)] text-[var(--foreground)] hover:bg-[var(--border-color)] transition-all flex items-center justify-center shadow-sm"
          title={`Switch to ${theme === 'dark' ? 'Light' : 'Dark'} Mode`}
        >
          {theme === 'dark' ? (
            <Sun className="w-4 h-4 text-amber-300" />
          ) : (
            <Moon className="w-4 h-4 text-zinc-800" />
          )}
        </button>

        {/* System Health Badge */}
        <div className="hidden sm:flex items-center gap-2 bg-[var(--panel-bg)] px-3 py-1.5 rounded-full border border-[var(--border-color)] text-xs text-[var(--text-muted)] font-mono">
          <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse shadow-[0_0_8px_rgba(16,185,129,0.8)]" />
          <span>3 Replicas Live</span>
        </div>
      </div>
    </header>
  );
};
