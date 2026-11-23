import React, { useState } from 'react'
import {
  Search,
  X,
  Sparkles,
  Database,
  CheckCircle,
  Plus,
  Loader2,
  AlertCircle,
  Layers,
} from 'lucide-react'
import { BikeSpec, Meta } from '../api/types'
import { api } from '../api/client'

interface SpecSearchModalProps {
  isOpen: boolean
  onClose: () => void
  onSelectSpecToAdd: (spec: BikeSpec) => void
}

const SUGGESTIONS = [
  'Honda Vario 150',
  'Yamaha NMAX 155',
  'Honda Beat FI',
  'Yamaha Aerox 155',
  'Kawasaki Ninja 250',
  'Mio Gear 125 S',
]

export const SpecSearchModal: React.FC<SpecSearchModalProps> = ({
  isOpen,
  onClose,
  onSelectSpecToAdd,
}) => {
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(false)
  const [spec, setSpec] = useState<BikeSpec | null>(null)
  const [meta, setMeta] = useState<Meta | null>(null)
  const [error, setError] = useState<string | null>(null)

  if (!isOpen) return null

  const handleSearch = async (searchQuery: string) => {
    const q = searchQuery.trim()
    if (!q) return

    setLoading(true)
    setError(null)
    setSpec(null)
    setMeta(null)

    try {
      const response = await api.searchSpec(q)
      setSpec(response.data)
      if (response.meta) {
        setMeta(response.meta)
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Gagal mencari spesifikasi'
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    handleSearch(query)
  }

  const getSourceBadge = () => {
    if (!meta) return null

    switch (meta.source) {
      case 'redis_cache':
        return (
          <span className="inline-flex items-center gap-1 text-[11px] font-mono text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded border border-emerald-500/20">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
            Redis Cache ({meta.latency_ms}ms)
          </span>
        )
      case 'postgres_db':
        return (
          <span className="inline-flex items-center gap-1 text-[11px] font-mono text-sky-400 bg-sky-500/10 px-2 py-0.5 rounded border border-sky-500/20">
            <Database className="w-3 h-3 text-sky-400" />
            PostgreSQL DB ({meta.latency_ms}ms)
          </span>
        )
      case 'llm_primary':
        return (
          <span className="inline-flex items-center gap-1 text-[11px] font-mono text-indigo-400 bg-indigo-500/10 px-2 py-0.5 rounded border border-indigo-500/20">
            <Sparkles className="w-3 h-3 text-indigo-400" />
            LLM Primary ({meta.provider_used || 'Gemini'}) ({meta.latency_ms}ms)
          </span>
        )
      case 'llm_fallback':
        return (
          <span className="inline-flex items-center gap-1 text-[11px] font-mono text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded border border-amber-500/20">
            <Sparkles className="w-3 h-3 text-amber-400" />
            LLM Fallback ({meta.provider_used || 'Groq'}) ({meta.latency_ms}ms)
          </span>
        )
      default:
        return null
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-2xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
          <div className="flex items-center gap-2">
            <div className="p-2 bg-indigo-500/10 rounded-lg text-indigo-400 border border-indigo-500/20">
              <Search className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-semibold text-white">Cari & Seed Spesifikasi Motor</h2>
              <p className="text-xs text-slate-400">
                Data akan diambil dari Redis, DB, atau di-generate otomatis via LLM Orchestrator
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Search Input Bar */}
        <div className="p-6 pb-3 border-b border-slate-800/60 bg-slate-900/60">
          <form onSubmit={handleSubmit} className="flex gap-2">
            <div className="relative flex-1">
              <input
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Contoh: Honda Vario 150, NMAX 155, Beat FI..."
                className="w-full bg-slate-950 border border-slate-700 rounded-xl px-4 py-2.5 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all font-medium"
                autoFocus
              />
              {query && (
                <button
                  type="button"
                  onClick={() => setQuery('')}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white"
                >
                  <X className="w-4 h-4" />
                </button>
              )}
            </div>
            <button
              type="submit"
              disabled={loading || !query.trim()}
              className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm font-semibold transition-all shadow-md shadow-indigo-600/20"
            >
              {loading ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  <span>Mencari...</span>
                </>
              ) : (
                <>
                  <Search className="w-4 h-4" />
                  <span>Cari</span>
                </>
              )}
            </button>
          </form>

          {/* Quick Suggestions */}
          <div className="flex flex-wrap items-center gap-1.5 mt-3">
            <span className="text-[11px] text-slate-500 mr-1">Rekomendasi:</span>
            {SUGGESTIONS.map((sug) => (
              <button
                key={sug}
                type="button"
                onClick={() => {
                  setQuery(sug)
                  handleSearch(sug)
                }}
                className="text-xs bg-slate-800 hover:bg-slate-700 text-slate-300 px-2.5 py-1 rounded-lg border border-slate-700/60 transition-colors"
              >
                {sug}
              </button>
            ))}
          </div>
        </div>

        {/* Content Body */}
        <div className="p-6 overflow-y-auto space-y-4 flex-1">
          {error && (
            <div className="flex items-start gap-2.5 p-3.5 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-xs">
              <AlertCircle className="w-4 h-4 shrink-0 mt-0.5 text-rose-400" />
              <div>
                <p className="font-semibold">Pencarian Gagal</p>
                <p className="text-rose-400 mt-0.5">{error}</p>
              </div>
            </div>
          )}

          {loading && (
            <div className="py-12 flex flex-col items-center justify-center text-slate-400 space-y-3">
              <Loader2 className="w-8 h-8 animate-spin text-indigo-400" />
              <div className="text-center">
                <p className="text-sm font-medium text-slate-300">Menghubungkan ke Motor Spec Pipeline...</p>
                <p className="text-xs text-slate-500 mt-1">
                  Mengecek Cache Redis &rarr; Database &rarr; LLM Orchestrator
                </p>
              </div>
            </div>
          )}

          {!loading && !spec && !error && (
            <div className="py-12 text-center text-slate-500 space-y-2">
              <Layers className="w-10 h-10 mx-auto text-slate-600 mb-2" />
              <p className="text-sm">Ketik nama model motor di atas untuk melihat spesifikasi komponen.</p>
              <p className="text-xs text-slate-600">
                Sistem akan otomatis menormalkan query (misal: "vario150" &rarr; "Honda Vario 150").
              </p>
            </div>
          )}

          {spec && (
            <div className="space-y-4 animate-in fade-in-50 duration-200">
              {/* Bike Meta Card */}
              <div className="bg-slate-950 p-4 rounded-xl border border-slate-800 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="text-lg font-bold text-white">{spec.display_name}</h3>
                    {spec.is_verified && (
                      <span className="inline-flex items-center gap-1 text-[11px] text-sky-400 bg-sky-500/10 px-2 py-0.5 rounded-full border border-sky-500/20">
                        <CheckCircle className="w-3 h-3" /> Terverifikasi
                      </span>
                    )}
                  </div>
                  <div className="flex items-center gap-2 text-xs text-slate-400 mt-1 font-mono">
                    <span className="capitalize">{spec.brand}</span>
                    <span>•</span>
                    <span>{spec.engine_cc} cc</span>
                    <span>•</span>
                    <span className="capitalize">{spec.category}</span>
                    <span>•</span>
                    <span className="text-slate-500">{spec.canonical_slug}</span>
                  </div>
                </div>

                <div className="flex flex-col sm:items-end gap-2">
                  {getSourceBadge()}
                  <button
                    onClick={() => onSelectSpecToAdd(spec)}
                    className="flex items-center justify-center gap-1.5 px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold shadow-md shadow-emerald-600/20 transition-all"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    <span>Tambahkan ke Garasi</span>
                  </button>
                </div>
              </div>

              {/* Component Specifications Table */}
              <div>
                <h4 className="text-xs font-semibold uppercase tracking-wider text-slate-400 mb-2">
                  Daftar Interval Komponen ({spec.components.length})
                </h4>
                <div className="space-y-2">
                  {spec.components.map((comp) => (
                    <div
                      key={comp.id || comp.code}
                      className="bg-slate-950/70 border border-slate-800/80 rounded-xl p-3 flex items-center justify-between gap-3 text-xs"
                    >
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="font-semibold text-slate-200">{comp.name}</span>
                          <span className="text-[10px] font-mono px-1.5 py-0.2 bg-slate-800 text-slate-400 rounded">
                            {comp.code}
                          </span>
                        </div>
                        <p className="text-slate-400 text-[11px] mt-0.5">{comp.notes}</p>
                      </div>

                      <div className="text-right shrink-0 font-mono">
                        <span className="text-slate-300 font-medium">
                          {comp.interval_km.toLocaleString()} km
                        </span>
                        <span className="text-slate-500 block text-[11px]">
                          {comp.interval_months} bulan
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
