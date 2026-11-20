import React, { useState, useEffect } from 'react'
import { X, Bike, Loader2, AlertCircle } from 'lucide-react'
import { BikeSpec, BikeHealthDashboard, AddBikeRequest, ComponentBaselineInput } from '../api/types'
import { api } from '../api/client'

interface AddBikeModalProps {
  isOpen: boolean
  onClose: () => void
  spec: BikeSpec | null
  onSuccess: (dashboard: BikeHealthDashboard) => void
}

export const AddBikeModal: React.FC<AddBikeModalProps> = ({
  isOpen,
  onClose,
  spec,
  onSuccess,
}) => {
  const [nickname, setNickname] = useState('')
  const [currentOdometerKm, setCurrentOdometerKm] = useState<number>(10000)
  const [dailyEstimateKm, setDailyEstimateKm] = useState<number>(25)
  const [autoIncrementEnabled, setAutoIncrementEnabled] = useState<boolean>(true)
  const [enableBaselines, setEnableBaselines] = useState<boolean>(false)
  const [baselines, setBaselines] = useState<Record<string, { km?: number; date?: string }>>({})

  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (spec) {
      setNickname(`${spec.display_name}`)
      setError(null)
      setBaselines({})
      setEnableBaselines(false)
    }
  }, [spec])

  if (!isOpen || !spec) return null

  const handleBaselineChange = (componentId: string, field: 'km' | 'date', value: string | number) => {
    setBaselines((prev) => {
      const existing = prev[componentId] || {}
      return {
        ...prev,
        [componentId]: {
          ...existing,
          [field]: field === 'km' ? Number(value) : String(value),
        },
      }
    })
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!nickname.trim()) {
      setError('Nama panggilan motor wajib diisi')
      return
    }

    if (currentOdometerKm < 0) {
      setError('Odometer tidak boleh bernilai negatif')
      return
    }

    setSubmitting(true)
    setError(null)

    try {
      const initialBaselines: ComponentBaselineInput[] = []
      if (enableBaselines) {
        Object.entries(baselines).forEach(([compId, data]) => {
          if (data.km !== undefined || (data.date && data.date.trim() !== '')) {
            let isoDate: string | null = null
            if (data.date && data.date.trim() !== '') {
              const d = new Date(data.date)
              if (!isNaN(d.getTime())) {
                isoDate = d.toISOString()
              }
            }
            initialBaselines.push({
              spec_component_id: compId,
              last_replaced_km: data.km !== undefined && !isNaN(data.km) ? data.km : null,
              last_replaced_at: isoDate,
            })
          }
        })
      }

      const req: AddBikeRequest = {
        bike_model_id: spec.id,
        nickname: nickname.trim(),
        current_odometer_km: Number(currentOdometerKm),
        daily_estimate_km: Number(dailyEstimateKm),
        auto_increment_enabled: autoIncrementEnabled,
        initial_baselines: initialBaselines.length > 0 ? initialBaselines : undefined,
      }

      const dashboard = await api.addBike(req)
      onSuccess(dashboard)
      onClose()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Gagal menambahkan motor ke garasi'
      setError(msg)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-lg max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
          <div className="flex items-center gap-2">
            <div className="p-2 bg-emerald-500/10 rounded-lg text-emerald-400 border border-emerald-500/20">
              <Bike className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-semibold text-white">Tambahkan ke Garasi</h2>
              <p className="text-xs text-slate-400">{spec.display_name} ({spec.engine_cc} cc)</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Form Body */}
        <form onSubmit={handleSubmit} className="p-6 overflow-y-auto space-y-4 flex-1">
          {error && (
            <div className="flex items-start gap-2.5 p-3 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-xs">
              <AlertCircle className="w-4 h-4 shrink-0 mt-0.5 text-rose-400" />
              <span>{error}</span>
            </div>
          )}

          {/* Nickname Input */}
          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1">
              Nama Panggilan Motor <span className="text-rose-400">*</span>
            </label>
            <input
              type="text"
              value={nickname}
              onChange={(e) => setNickname(e.target.value)}
              placeholder="Contoh: Vario Harian, Beat Istri, Ninja Sunmori"
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3.5 py-2 text-sm text-white focus:outline-none focus:border-indigo-500 transition-all font-medium"
              required
            />
          </div>

          {/* Odometer & Daily Estimate */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1">
                Odometer Saat Ini (KM) <span className="text-rose-400">*</span>
              </label>
              <div className="relative">
                <input
                  type="number"
                  min="0"
                  value={currentOdometerKm}
                  onChange={(e) => setCurrentOdometerKm(Number(e.target.value))}
                  className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3.5 py-2 text-sm text-white font-mono focus:outline-none focus:border-indigo-500 transition-all"
                  required
                />
                <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-slate-500">km</span>
              </div>
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1">
                Estimasi KM Harian
              </label>
              <div className="relative">
                <input
                  type="number"
                  min="0"
                  max="1000"
                  value={dailyEstimateKm}
                  onChange={(e) => setDailyEstimateKm(Number(e.target.value))}
                  className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3.5 py-2 text-sm text-white font-mono focus:outline-none focus:border-indigo-500 transition-all"
                />
                <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-slate-500">km/hari</span>
              </div>
            </div>
          </div>

          {/* Auto-Increment Toggle */}
          <div className="flex items-center justify-between p-3 rounded-xl bg-slate-950 border border-slate-800">
            <div>
              <p className="text-xs font-medium text-white">Auto-Increment Odometer Tiap Malam</p>
              <p className="text-[11px] text-slate-400">
                Odometer bertambah otomatis sebesar {dailyEstimateKm} km setiap tengah malam (cron worker)
              </p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={autoIncrementEnabled}
                onChange={(e) => setAutoIncrementEnabled(e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-slate-800 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-indigo-600"></div>
            </label>
          </div>

          {/* Optional Baseline toggle */}
          <div className="border-t border-slate-800/80 pt-3">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs font-medium text-slate-200">Set Riwayat Awal Servis (Opsional)</p>
                <p className="text-[11px] text-slate-500">
                  Jika dikosongkan, status komponen berstatus "Perlu Inspeksi" (cold-start)
                </p>
              </div>
              <input
                type="checkbox"
                checked={enableBaselines}
                onChange={(e) => setEnableBaselines(e.target.checked)}
                className="w-4 h-4 rounded border-slate-700 bg-slate-950 text-indigo-600 focus:ring-0 cursor-pointer"
              />
            </div>

            {enableBaselines && (
              <div className="mt-3 space-y-2 max-h-48 overflow-y-auto pr-1">
                {spec.components.map((c) => (
                  <div key={c.id} className="p-2.5 rounded-lg bg-slate-950/80 border border-slate-800 text-xs">
                    <span className="font-semibold text-slate-300 block mb-1.5">{c.name}</span>
                    <div className="grid grid-cols-2 gap-2">
                      <input
                        type="number"
                        placeholder="KM terakhir ganti"
                        onChange={(e) => handleBaselineChange(c.id, 'km', e.target.value)}
                        className="bg-slate-900 border border-slate-700 rounded-lg px-2.5 py-1 text-slate-200 font-mono text-[11px]"
                      />
                      <input
                        type="date"
                        onChange={(e) => handleBaselineChange(c.id, 'date', e.target.value)}
                        className="bg-slate-900 border border-slate-700 rounded-lg px-2 py-1 text-slate-200 text-[11px]"
                      />
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Submit */}
          <div className="pt-2">
            <button
              type="submit"
              disabled={submitting}
              className="w-full flex items-center justify-center gap-2 py-2.5 px-4 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-sm font-semibold transition-all shadow-md shadow-indigo-600/25"
            >
              {submitting ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  <span>Menyimpan ke Garasi...</span>
                </>
              ) : (
                <span>Simpan Motor ke Garasi</span>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
