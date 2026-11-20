import React, { useState, useEffect } from 'react'
import { X, Gauge, AlertCircle, Loader2, Clock } from 'lucide-react'
import { BikeHealthDashboard, UpdateOdometerRequest } from '../api/types'
import { api } from '../api/client'

interface OdometerModalProps {
  isOpen: boolean
  onClose: () => void
  bike: BikeHealthDashboard
  onSuccess: (dashboard: BikeHealthDashboard) => void
}

export const OdometerModal: React.FC<OdometerModalProps> = ({
  isOpen,
  onClose,
  bike,
  onSuccess,
}) => {
  const [currentOdometerKm, setCurrentOdometerKm] = useState<number>(bike.current_odometer_km)
  const [dailyEstimateKm, setDailyEstimateKm] = useState<number>(bike.daily_estimate_km)
  const [autoIncrementEnabled, setAutoIncrementEnabled] = useState<boolean>(bike.auto_increment_enabled)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setCurrentOdometerKm(bike.current_odometer_km)
    setDailyEstimateKm(bike.daily_estimate_km)
    setAutoIncrementEnabled(bike.auto_increment_enabled)
    setError(null)
  }, [bike, isOpen])

  if (!isOpen) return null

  const isRollback = currentOdometerKm < bike.current_odometer_km

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (isRollback) {
      setError(`Anti-rollback: Odometer baru tidak boleh lebih kecil dari KM saat ini (${bike.current_odometer_km.toLocaleString()} km)`)
      return
    }

    setSubmitting(true)
    setError(null)

    try {
      const req: UpdateOdometerRequest = {
        current_odometer_km: Number(currentOdometerKm),
        daily_estimate_km: Number(dailyEstimateKm),
        auto_increment_enabled: autoIncrementEnabled,
      }

      const updated = await api.updateOdometer(bike.id, req)
      onSuccess(updated)
      onClose()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Gagal memperbarui odometer'
      setError(msg)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-md shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
          <div className="flex items-center gap-2">
            <div className="p-2 bg-indigo-500/10 rounded-lg text-indigo-400 border border-indigo-500/20">
              <Gauge className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-semibold text-white">Update Odometer</h2>
              <p className="text-xs text-slate-400">{bike.nickname} ({bike.model_display_name})</p>
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
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {error && (
            <div className="flex items-start gap-2.5 p-3 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-xs">
              <AlertCircle className="w-4 h-4 shrink-0 mt-0.5 text-rose-400" />
              <span>{error}</span>
            </div>
          )}

          {/* Odometer Input */}
          <div>
            <div className="flex justify-between items-center mb-1">
              <label className="text-xs font-medium text-slate-300">
                Odometer Baru (KM) <span className="text-rose-400">*</span>
              </label>
              <span className="text-[11px] font-mono text-slate-500">
                Saat ini: {bike.current_odometer_km.toLocaleString()} km
              </span>
            </div>
            <div className="relative">
              <input
                type="number"
                min={bike.current_odometer_km}
                value={currentOdometerKm}
                onChange={(e) => setCurrentOdometerKm(Number(e.target.value))}
                className={`w-full bg-slate-950 border rounded-xl px-4 py-2.5 text-base font-mono font-bold text-white focus:outline-none transition-all ${
                  isRollback ? 'border-rose-500 text-rose-400' : 'border-slate-700 focus:border-indigo-500'
                }`}
                required
                autoFocus
              />
              <span className="absolute right-3.5 top-1/2 -translate-y-1/2 text-xs font-mono text-slate-400">
                km
              </span>
            </div>
            {isRollback && (
              <p className="text-[11px] text-rose-400 mt-1">
                Angka tidak boleh lebih rendah dari odometer tercatat.
              </p>
            )}
          </div>

          {/* Daily Estimate */}
          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1">
              Estimasi Penggunaan Harian
            </label>
            <div className="relative">
              <input
                type="number"
                min="0"
                max="1000"
                value={dailyEstimateKm}
                onChange={(e) => setDailyEstimateKm(Number(e.target.value))}
                className="w-full bg-slate-950 border border-slate-700 rounded-xl px-4 py-2 text-sm text-white font-mono focus:outline-none focus:border-indigo-500 transition-all"
              />
              <span className="absolute right-3.5 top-1/2 -translate-y-1/2 text-xs text-slate-400">
                km/hari
              </span>
            </div>
          </div>

          {/* Auto-Increment Toggle */}
          <div className="flex items-center justify-between p-3 rounded-xl bg-slate-950 border border-slate-800">
            <div>
              <p className="text-xs font-medium text-white">Auto-Increment Cron Odometer</p>
              <p className="text-[11px] text-slate-400">
                Tambah {dailyEstimateKm} km otomatis tiap 00:00 UTC
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

          {/* Last sync info */}
          <div className="flex items-center gap-1.5 text-[11px] text-slate-500 pt-1">
            <Clock className="w-3.5 h-3.5" />
            <span>
              Sinkronisasi terakhir:{' '}
              {new Date(bike.last_odometer_sync_at).toLocaleString('id-ID')}
            </span>
          </div>

          {/* Submit */}
          <div className="pt-2">
            <button
              type="submit"
              disabled={submitting || isRollback}
              className="w-full flex items-center justify-center gap-2 py-2.5 px-4 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm font-semibold transition-all shadow-md shadow-indigo-600/25"
            >
              {submitting ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  <span>Memperbarui Odometer...</span>
                </>
              ) : (
                <span>Simpan Perubahan</span>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
