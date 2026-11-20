import React from 'react'
import { Activity, Search, Wifi, WifiOff } from 'lucide-react'
import { Meta } from '../api/types'

interface HeaderProps {
  isOnline: boolean
  lastMeta: Meta | null
  activeUserId: string
  onOpenSearch: () => void
  onOpenTelemetry: () => void
  onChangeUserId: (newId: string) => void
}

export const Header: React.FC<HeaderProps> = ({
  isOnline,
  lastMeta,
  activeUserId,
  onOpenSearch,
  onOpenTelemetry,
  onChangeUserId,
}) => {
  return (
    <header className="sticky top-0 z-30 bg-slate-900/80 backdrop-blur-md border-b border-slate-800">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between gap-4">
        {/* Brand & Identity */}
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center shadow-lg shadow-indigo-500/20 font-black text-xl text-white tracking-wider">
            M
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="font-bold text-lg text-white tracking-tight">MotoSpec</h1>
              {isOnline ? (
                <span className="inline-flex items-center gap-1 text-[11px] font-medium text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full border border-emerald-500/20">
                  <Wifi className="w-3 h-3" /> Online
                </span>
              ) : (
                <span className="inline-flex items-center gap-1 text-[11px] font-medium text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded-full border border-amber-500/20">
                  <WifiOff className="w-3 h-3" /> Offline (Cached)
                </span>
              )}
            </div>
            <p className="text-xs text-slate-400 hidden sm:block">Motorcycle Maintenance & Resilience Tracker</p>
          </div>
        </div>

        {/* Action Controls */}
        <div className="flex items-center gap-2 sm:gap-3">
          {/* User ID Selector */}
          <div className="flex items-center gap-1.5 bg-slate-800/80 border border-slate-700 rounded-lg px-2.5 py-1 text-xs text-slate-300">
            <span className="text-slate-500 hidden sm:inline">User:</span>
            <input
              type="text"
              value={activeUserId}
              onChange={(e) => onChangeUserId(e.target.value)}
              className="bg-transparent font-medium text-white focus:outline-none w-20 sm:w-24 text-xs"
              placeholder="User ID"
              title="X-User-ID scope"
            />
          </div>

          {/* Spec Search & Seeder Button */}
          <button
            onClick={onOpenSearch}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md shadow-indigo-600/20 transition-colors"
          >
            <Search className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">Find & Seed Spec</span>
            <span className="sm:hidden">Search</span>
          </button>

          {/* Telemetry Drawer Trigger */}
          <button
            onClick={onOpenTelemetry}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 text-xs font-medium transition-all"
            title="Inspect Live System Telemetry"
          >
            <Activity className="w-3.5 h-3.5 text-indigo-400" />
            <span className="hidden sm:inline">Telemetry</span>
            {lastMeta && (
              <span className="bg-slate-700 text-[11px] font-mono px-1.5 py-0.5 rounded text-indigo-300 border border-slate-600">
                {lastMeta.latency_ms}ms
              </span>
            )}
          </button>
        </div>
      </div>
    </header>
  )
}
