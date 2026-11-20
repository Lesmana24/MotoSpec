import React, { useEffect, useState } from 'react'
import {
  X,
  Zap,
  ShieldAlert,
  Database,
  Cpu,
  RefreshCw,
  CheckCircle2,
  AlertTriangle,
  Flame,
} from 'lucide-react'
import { Meta, SystemTelemetryData } from '../api/types'
import {
  api,
  isSimulateOutageEnabled,
  setSimulateOutage,
  subscribeTelemetry,
} from '../api/client'

interface TelemetryDrawerProps {
  isOpen: boolean
  onClose: () => void
}

export const TelemetryDrawer: React.FC<TelemetryDrawerProps> = ({ isOpen, onClose }) => {
  const [lastMeta, setLastMeta] = useState<Meta | null>(null)
  const [telemetry, setTelemetry] = useState<SystemTelemetryData | null>(null)
  const [loading, setLoading] = useState(false)
  const [simulateOutage, setSimulateOutageState] = useState(isSimulateOutageEnabled())

  // Subscribe to live API request telemetry
  useEffect(() => {
    const unsubscribe = subscribeTelemetry((meta) => {
      setLastMeta(meta)
    })
    return unsubscribe
  }, [])

  // Poll telemetry metrics when drawer is open
  const fetchTelemetry = async () => {
    try {
      setLoading(true)
      const data = await api.getSystemTelemetry()
      setTelemetry(data)
    } catch (err) {
      console.warn('Failed to fetch system telemetry:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (isOpen) {
      fetchTelemetry()
      const interval = setInterval(fetchTelemetry, 3000)
      return () => clearInterval(interval)
    }
  }, [isOpen])

  const handleToggleOutage = (enabled: boolean) => {
    setSimulateOutage(enabled)
    setSimulateOutageState(enabled)
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 overflow-hidden">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm transition-opacity" onClick={onClose} />

      <div className="fixed inset-y-0 right-0 max-w-full flex pl-10">
        <div className="w-screen max-w-md bg-slate-900 border-l border-slate-800 p-6 flex flex-col justify-between shadow-2xl overflow-y-auto">
          {/* Header */}
          <div>
            <div className="flex items-center justify-between border-b border-slate-800 pb-4">
              <div className="flex items-center gap-2">
                <div className="w-8 h-8 rounded-lg bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
                  <Zap className="w-4 h-4" />
                </div>
                <div>
                  <h2 className="text-base font-bold text-white">System Telemetry</h2>
                  <p className="text-xs text-slate-400">Fault-Tolerance & Multi-Layer Cache</p>
                </div>
              </div>
              <button
                onClick={onClose}
                className="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Outage Simulation Switch */}
            <div className="mt-6 p-4 rounded-xl bg-gradient-to-br from-rose-950/40 to-slate-900 border border-rose-500/30">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2.5">
                  <Flame className="w-5 h-5 text-rose-400" />
                  <div>
                    <span className="text-xs font-bold text-white block">Simulate Primary Outage</span>
                    <span className="text-[11px] text-slate-400 block">Forces fallback via X-Simulate-Outage</span>
                  </div>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={simulateOutage}
                    onChange={(e) => handleToggleOutage(e.target.checked)}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-slate-800 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-rose-600"></div>
                </label>
              </div>
              {simulateOutage && (
                <div className="mt-3 text-[11px] font-medium text-rose-300 bg-rose-500/10 border border-rose-500/20 rounded-lg p-2 flex items-center gap-1.5">
                  <AlertTriangle className="w-3.5 h-3.5 shrink-0" />
                  Active: Primary LLM will be bypassed; requests will route to Groq Llama 3.3.
                </div>
              )}
            </div>

            {/* Latest Request Meta */}
            <div className="mt-6">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-400 mb-3">
                Latest Request Telemetry
              </h3>
              {lastMeta ? (
                <div className="bg-slate-800/60 rounded-xl p-4 border border-slate-700/60 space-y-2.5">
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-slate-400">Data Source:</span>
                    <span
                      className={`font-semibold px-2 py-0.5 rounded text-[11px] ${
                        lastMeta.source === 'redis_cache'
                          ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                          : lastMeta.source === 'postgres_db'
                          ? 'bg-blue-500/10 text-blue-400 border border-blue-500/20'
                          : lastMeta.source === 'llm_fallback'
                          ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                          : 'bg-purple-500/10 text-purple-400 border border-purple-500/20'
                      }`}
                    >
                      {lastMeta.source}
                    </span>
                  </div>

                  <div className="flex items-center justify-between text-xs">
                    <span className="text-slate-400">Response Latency:</span>
                    <span className="font-mono font-medium text-white">{lastMeta.latency_ms} ms</span>
                  </div>

                  {lastMeta.canonical_slug && (
                    <div className="flex items-center justify-between text-xs">
                      <span className="text-slate-400">Resolved Slug:</span>
                      <span className="font-mono text-indigo-300 text-[11px]">{lastMeta.canonical_slug}</span>
                    </div>
                  )}

                  {lastMeta.provider_used && (
                    <div className="flex items-center justify-between text-xs">
                      <span className="text-slate-400">Provider Used:</span>
                      <span className="text-slate-200 font-medium">{lastMeta.provider_used}</span>
                    </div>
                  )}

                  {lastMeta.fallback_triggered && (
                    <div className="mt-2 pt-2 border-t border-slate-700 text-xs text-amber-300">
                      <div className="flex items-center gap-1 font-semibold">
                        <ShieldAlert className="w-3.5 h-3.5" /> Fallback Triggered
                      </div>
                      <p className="text-[11px] text-amber-400/80 mt-0.5">{lastMeta.fallback_reason}</p>
                    </div>
                  )}
                </div>
              ) : (
                <div className="bg-slate-800/40 rounded-xl p-4 border border-slate-700/40 text-center text-xs text-slate-500">
                  No requests recorded yet. Try searching for a motorcycle specification!
                </div>
              )}
            </div>

            {/* In-Memory Cache & Circuit Breaker Metrics */}
            <div className="mt-6">
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-400">
                  Backend Metrics
                </h3>
                <button
                  onClick={fetchTelemetry}
                  disabled={loading}
                  className="text-xs text-indigo-400 hover:text-indigo-300 flex items-center gap-1"
                >
                  <RefreshCw className={`w-3 h-3 ${loading ? 'animate-spin' : ''}`} /> Refresh
                </button>
              </div>

              {telemetry ? (
                <div className="space-y-3">
                  {/* Cache Metrics */}
                  <div className="bg-slate-800/60 rounded-xl p-4 border border-slate-700/60">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-xs font-medium text-slate-300 flex items-center gap-1.5">
                        <Database className="w-3.5 h-3.5 text-emerald-400" /> Redis Cache Hit Ratio
                      </span>
                      <span className="text-xs font-bold text-emerald-400">
                        {telemetry.cache_hit_ratio_pct}%
                      </span>
                    </div>
                    {/* Progress Bar */}
                    <div className="w-full bg-slate-700/50 rounded-full h-2 overflow-hidden">
                      <div
                        className="bg-emerald-500 h-2 rounded-full transition-all duration-500"
                        style={{ width: `${Math.min(telemetry.cache_hit_ratio_pct, 100)}%` }}
                      />
                    </div>
                    <div className="flex justify-between items-center text-[11px] text-slate-400 mt-2">
                      <span>Hits: {telemetry.cache_hits}</span>
                      <span>Misses: {telemetry.cache_misses}</span>
                    </div>
                  </div>

                  {/* Provider Circuit Breakers */}
                  <div className="bg-slate-800/60 rounded-xl p-4 border border-slate-700/60 space-y-3">
                    <span className="text-xs font-medium text-slate-300 flex items-center gap-1.5">
                      <Cpu className="w-3.5 h-3.5 text-indigo-400" /> LLM Circuit Breakers
                    </span>
                    {telemetry.llm_providers.map((p) => (
                      <div key={p.name} className="flex items-center justify-between text-xs bg-slate-900/60 p-2.5 rounded-lg border border-slate-800">
                        <div>
                          <div className="font-medium text-slate-200 capitalize">{p.role}: {p.name}</div>
                          <div className="text-[11px] text-slate-500">Consecutive Failures: {p.consecutive_failures}</div>
                        </div>
                        <span
                          className={`text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded ${
                            p.circuit_state === 'closed'
                              ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                              : p.circuit_state === 'open'
                              ? 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                              : 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                          }`}
                        >
                          {p.circuit_state}
                        </span>
                      </div>
                    ))}
                    <div className="grid grid-cols-3 gap-2 pt-1 text-center text-[10px] text-slate-400">
                      <div className="bg-slate-900/40 p-2 rounded border border-slate-800">
                        <div className="font-semibold text-white">{telemetry.llm_primary_success}</div>
                        <div>Primary</div>
                      </div>
                      <div className="bg-slate-900/40 p-2 rounded border border-slate-800">
                        <div className="font-semibold text-white">{telemetry.llm_fallback_success}</div>
                        <div>Fallback</div>
                      </div>
                      <div className="bg-slate-900/40 p-2 rounded border border-slate-800">
                        <div className="font-semibold text-rose-400">{telemetry.llm_total_failures}</div>
                        <div>Failures</div>
                      </div>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="bg-slate-800/40 rounded-xl p-4 border border-slate-700/40 text-center text-xs text-slate-500">
                  Loading telemetry...
                </div>
              )}
            </div>
          </div>

          <div className="pt-6 border-t border-slate-800 text-[11px] text-slate-500 flex items-center justify-between">
            <span>MotoSpec v1.0</span>
            <span className="flex items-center gap-1 text-emerald-400">
              <CheckCircle2 className="w-3 h-3" /> System Operational
            </span>
          </div>
        </div>
      </div>
    </div>
  )
}
