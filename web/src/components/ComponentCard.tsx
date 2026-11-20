import React from 'react'
import { AlertCircle, AlertTriangle, CheckCircle2, HelpCircle, Wrench } from 'lucide-react'
import { ComponentHealthItem } from '../api/types'

interface ComponentCardProps {
  component: ComponentHealthItem
  onLogMaintenance: (component: ComponentHealthItem) => void
}

export const ComponentCard: React.FC<ComponentCardProps> = ({ component, onLogMaintenance }) => {
  // Status styling configurations
  const getStatusConfig = () => {
    switch (component.status) {
      case 'critical':
        return {
          border: 'border-rose-500/40 hover:border-rose-500/70',
          badgeBg: 'bg-rose-500/15 text-rose-300 border-rose-500/30',
          barColor: 'bg-rose-500 shadow-sm shadow-rose-500/50',
          icon: <AlertCircle className="w-3.5 h-3.5 text-rose-400" />,
          label: 'Kritis / Overdue',
        }
      case 'warning':
        return {
          border: 'border-amber-500/40 hover:border-amber-500/70',
          badgeBg: 'bg-amber-500/15 text-amber-300 border-amber-500/30',
          barColor: 'bg-amber-500 shadow-sm shadow-amber-500/50',
          icon: <AlertTriangle className="w-3.5 h-3.5 text-amber-400" />,
          label: 'Waspada (>=75%)',
        }
      case 'healthy':
        return {
          border: 'border-slate-800 hover:border-slate-700',
          badgeBg: 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30',
          barColor: 'bg-emerald-500',
          icon: <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />,
          label: 'Kondisi Baik',
        }
      case 'unknown_needs_inspection':
      default:
        return {
          border: 'border-slate-800 hover:border-slate-700',
          badgeBg: 'bg-slate-700/50 text-slate-300 border-slate-600/50',
          barColor: 'bg-slate-600',
          icon: <HelpCircle className="w-3.5 h-3.5 text-slate-400" />,
          label: 'Perlu Cek / Baseline Kosong',
        }
    }
  }

  const config = getStatusConfig()
  const hasWear = component.wear_percentage !== null
  const wearPct = hasWear ? component.wear_percentage! : 0
  const barWidth = Math.min(Math.max(wearPct, 0), 100)

  // Urgency tag styling
  const getUrgencyBadge = () => {
    switch (component.urgency_level) {
      case 'high':
        return 'text-rose-400 bg-rose-950/40 border-rose-800/40'
      case 'medium':
        return 'text-amber-400 bg-amber-950/40 border-amber-800/40'
      case 'low':
      default:
        return 'text-slate-400 bg-slate-800/60 border-slate-700/40'
    }
  }

  return (
    <div
      className={`bg-slate-900/90 border rounded-xl p-4 transition-all duration-200 flex flex-col justify-between ${config.border}`}
    >
      <div>
        {/* Top Header: Name, Code & Status Badge */}
        <div className="flex items-start justify-between gap-2 mb-2">
          <div>
            <div className="flex items-center gap-2">
              <h3 className="font-semibold text-white text-base tracking-tight">{component.name}</h3>
              <span className={`text-[10px] uppercase font-mono px-1.5 py-0.5 rounded border ${getUrgencyBadge()}`}>
                {component.urgency_level}
              </span>
            </div>
            <div className="flex items-center gap-2 text-xs text-slate-400 mt-0.5 font-mono">
              <span>{component.code}</span>
              <span>•</span>
              <span className="capitalize">{component.category}</span>
            </div>
          </div>

          <div
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${config.badgeBg}`}
          >
            {config.icon}
            <span>{config.label}</span>
          </div>
        </div>

        {/* Wear Percentage Progress Bar */}
        <div className="my-3">
          <div className="flex justify-between items-center text-xs mb-1">
            <span className="text-slate-400">Tingkat Keausan</span>
            {hasWear ? (
              <span
                className={`font-mono font-semibold ${
                  wearPct >= 100
                    ? 'text-rose-400'
                    : wearPct >= 75
                      ? 'text-amber-400'
                      : 'text-emerald-400'
                }`}
              >
                {wearPct.toFixed(1)}%
              </span>
            ) : (
              <span className="text-slate-500 font-mono italic">Belum terhitung</span>
            )}
          </div>
          <div className="w-full bg-slate-800 rounded-full h-2 overflow-hidden border border-slate-700/50">
            {hasWear ? (
              <div
                className={`h-full transition-all duration-500 rounded-full ${config.barColor}`}
                style={{ width: `${barWidth}%` }}
              />
            ) : (
              <div className="h-full bg-slate-700/60 w-full animate-pulse" />
            )}
          </div>
        </div>

        {/* Specs & Metrics Grid */}
        <div className="grid grid-cols-2 gap-2 text-xs bg-slate-950/60 p-2.5 rounded-lg border border-slate-800/60 mb-3">
          <div>
            <span className="text-slate-500 block">Interval Servis:</span>
            <span className="text-slate-200 font-mono font-medium">
              {component.interval_km.toLocaleString()} km / {component.interval_months} bln
            </span>
          </div>

          <div>
            <span className="text-slate-500 block">Jatuh Tempo:</span>
            <span className="text-slate-200 font-mono font-medium">
              {component.next_due_km !== null
                ? `${component.next_due_km.toLocaleString()} km`
                : '—'}
            </span>
          </div>

          <div>
            <span className="text-slate-500 block">Terakhir Diganti:</span>
            <span className="text-slate-300 font-mono">
              {component.last_replaced_km !== null
                ? `${component.last_replaced_km.toLocaleString()} km`
                : 'Belum ada data'}
            </span>
            {component.last_replaced_at && (
              <span className="text-[10px] text-slate-500 block">
                {new Date(component.last_replaced_at).toLocaleDateString('id-ID')}
              </span>
            )}
          </div>

          <div>
            <span className="text-slate-500 block">Sisa Jarak:</span>
            {component.remaining_km !== null ? (
              <span
                className={`font-mono font-medium ${
                  component.remaining_km < 0
                    ? 'text-rose-400'
                    : component.remaining_km <= 500
                      ? 'text-amber-400'
                      : 'text-emerald-400'
                }`}
              >
                {component.remaining_km < 0
                  ? `Terlewat ${Math.abs(component.remaining_km).toLocaleString()} km`
                  : `${component.remaining_km.toLocaleString()} km`}
              </span>
            ) : (
              <span className="text-slate-500 font-mono">—</span>
            )}
          </div>
        </div>

        {/* Notes */}
        {component.notes && (
          <p className="text-xs text-slate-400 italic bg-slate-900/50 px-2 py-1.5 rounded border border-slate-800/50 mb-3">
            "{component.notes}"
          </p>
        )}
      </div>

      {/* Action: Log Service */}
      <button
        onClick={() => onLogMaintenance(component)}
        className="w-full flex items-center justify-center gap-1.5 py-1.5 px-3 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 hover:text-white border border-slate-700 hover:border-slate-600 text-xs font-medium transition-colors"
      >
        <Wrench className="w-3.5 h-3.5 text-indigo-400" />
        <span>Catat Penggantian Part</span>
      </button>
    </div>
  )
}
