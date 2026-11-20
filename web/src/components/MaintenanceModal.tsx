import React, { useState, useEffect } from 'react'
import {
  X,
  Wrench,
  CheckSquare,
  Square,
  AlertCircle,
  Loader2,
} from 'lucide-react'
import {
  BikeHealthDashboard,
  LogMaintenanceRequest,
  MaintenanceLogItemInput,
} from '../api/types'
import { api } from '../api/client'

interface MaintenanceModalProps {
  isOpen: boolean
  onClose: () => void
  bike: BikeHealthDashboard
  preselectedComponentId?: string | null
  onSuccess: (dashboard: BikeHealthDashboard) => void
}

export const MaintenanceModal: React.FC<MaintenanceModalProps> = ({
  isOpen,
  onClose,
  bike,
  preselectedComponentId,
  onSuccess,
}) => {
  const [odometerKm, setOdometerKm] = useState<number>(bike.current_odometer_km)
  const [servicedAt, setServicedAt] = useState<string>(
    new Date().toISOString().split('T')[0]
  )
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [itemsData, setItemsData] = useState<
    Record<string, { cost?: number; notes: string }>
  >({})
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setOdometerKm(bike.current_odometer_km)
    setServicedAt(new Date().toISOString().split('T')[0])
    setError(null)

    const initialSet = new Set<string>()
    const initialData: Record<string, { cost?: number; notes: string }> = {}

    if (preselectedComponentId) {
      initialSet.add(preselectedComponentId)
      initialData[preselectedComponentId] = {
        cost: undefined,
        notes: 'Penggantian part berkala',
      }
    }

    setSelectedIds(initialSet)
    setItemsData(initialData)
  }, [bike, isOpen, preselectedComponentId])

  if (!isOpen) return null

  const toggleComponent = (compId: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(compId)) {
        next.delete(compId)
      } else {
        next.add(compId)
        if (!itemsData[compId]) {
          setItemsData((d) => ({
            ...d,
            [compId]: { cost: undefined, notes: 'Penggantian part berkala' },
          }))
        }
      }
      return next
    })
  }

  const selectAllCriticalAndWarning = () => {
    const next = new Set(selectedIds)
    const nextData = { ...itemsData }

    bike.components.forEach((c) => {
      if (c.status === 'critical' || c.status === 'warning') {
        next.add(c.spec_component_id)
        if (!nextData[c.spec_component_id]) {
          nextData[c.spec_component_id] = {
            cost: undefined,
            notes: 'Penggantian part terjadwal',
          }
        }
      }
    })

    setSelectedIds(next)
    setItemsData(nextData)
  }

  const handleItemDataChange = (
    compId: string,
    field: 'cost' | 'notes',
    val: string | number
  ) => {
    setItemsData((prev) => ({
      ...prev,
      [compId]: {
        ...prev[compId],
        [field]: field === 'cost' ? Number(val) : String(val),
      },
    }))
  }

  // Calculate total estimated cost
  const totalCost = Array.from(selectedIds).reduce((sum, id) => {
    const item = itemsData[id]
    return sum + (item?.cost || 0)
  }, 0)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (selectedIds.size === 0) {
      setError('Pilih minimal satu komponen yang diservis atau diganti')
      return
    }

    setSubmitting(true)
    setError(null)

    try {
      const items: MaintenanceLogItemInput[] = Array.from(selectedIds).map((id) => {
        const item = itemsData[id]
        return {
          spec_component_id: id,
          cost_idr: item?.cost && item.cost > 0 ? item.cost : null,
          notes: item?.notes?.trim() || 'Penggantian komponen',
        }
      })

      const req: LogMaintenanceRequest = {
        odometer_km: Number(odometerKm),
        serviced_at: new Date(servicedAt).toISOString(),
        items,
      }

      const updated = await api.logMaintenance(bike.id, req)
      onSuccess(updated)
      onClose()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Gagal mencatat servis'
      setError(msg)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
          <div className="flex items-center gap-2">
            <div className="p-2 bg-indigo-500/10 rounded-lg text-indigo-400 border border-indigo-500/20">
              <Wrench className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-semibold text-white">Catat Servis / Ganti Part</h2>
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
        <form onSubmit={handleSubmit} className="p-6 overflow-y-auto space-y-4 flex-1">
          {error && (
            <div className="flex items-start gap-2.5 p-3 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-xs">
              <AlertCircle className="w-4 h-4 shrink-0 mt-0.5 text-rose-400" />
              <span>{error}</span>
            </div>
          )}

          {/* Odometer & Service Date */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1">
                Odometer Saat Servis <span className="text-rose-400">*</span>
              </label>
              <div className="relative">
                <input
                  type="number"
                  min="0"
                  value={odometerKm}
                  onChange={(e) => setOdometerKm(Number(e.target.value))}
                  className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3.5 py-2 text-sm text-white font-mono focus:outline-none focus:border-indigo-500 transition-all"
                  required
                />
                <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-slate-500">km</span>
              </div>
              <p className="text-[10px] text-slate-500 mt-1">
                Jika angka lebih tinggi dari odometer garasi, odometer akan otomatis dinaikkan.
              </p>
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1">
                Tanggal Servis <span className="text-rose-400">*</span>
              </label>
              <input
                type="date"
                value={servicedAt}
                onChange={(e) => setServicedAt(e.target.value)}
                className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3.5 py-2 text-sm text-white focus:outline-none focus:border-indigo-500 transition-all"
                required
              />
            </div>
          </div>

          {/* Component Selection Header */}
          <div className="border-t border-slate-800 pt-3">
            <div className="flex items-center justify-between mb-2">
              <label className="text-xs font-semibold uppercase tracking-wider text-slate-300">
                Pilih Komponen yang Diganti ({selectedIds.size}/{bike.components.length})
              </label>
              <button
                type="button"
                onClick={selectAllCriticalAndWarning}
                className="text-[11px] text-indigo-400 hover:text-indigo-300 font-medium transition-colors"
              >
                + Pilih Semua Kritis / Waspada
              </button>
            </div>

            {/* Component Multi-Select List */}
            <div className="space-y-2 max-h-56 overflow-y-auto pr-1">
              {bike.components.map((c) => {
                const isSelected = selectedIds.has(c.spec_component_id)
                const cData = itemsData[c.spec_component_id] || { notes: '' }

                return (
                  <div
                    key={c.spec_component_id}
                    className={`p-3 rounded-xl border transition-all ${
                      isSelected
                        ? 'bg-slate-950 border-indigo-500/50 shadow-sm shadow-indigo-500/10'
                        : 'bg-slate-950/60 border-slate-800 hover:border-slate-700'
                    }`}
                  >
                    <div
                      className="flex items-center justify-between cursor-pointer"
                      onClick={() => toggleComponent(c.spec_component_id)}
                    >
                      <div className="flex items-center gap-2.5">
                        <div className="text-indigo-400">
                          {isSelected ? (
                            <CheckSquare className="w-4 h-4 text-indigo-400" />
                          ) : (
                            <Square className="w-4 h-4 text-slate-600" />
                          )}
                        </div>
                        <div>
                          <span className="text-xs font-semibold text-white block">
                            {c.name}
                          </span>
                          <span className="text-[10px] text-slate-500 font-mono">
                            Interval: {c.interval_km.toLocaleString()} km / {c.interval_months} bln
                          </span>
                        </div>
                      </div>

                      <span
                        className={`text-[10px] font-medium px-2 py-0.5 rounded-full border ${
                          c.status === 'critical'
                            ? 'text-rose-400 bg-rose-500/10 border-rose-500/20'
                            : c.status === 'warning'
                              ? 'text-amber-400 bg-amber-500/10 border-amber-500/20'
                              : 'text-slate-400 bg-slate-800 border-slate-700'
                        }`}
                      >
                        {c.status}
                      </span>
                    </div>

                    {/* Expanded details when checked */}
                    {isSelected && (
                      <div className="mt-3 pt-3 border-t border-slate-800/80 grid grid-cols-2 gap-2 text-xs">
                        <div>
                          <label className="text-[10px] text-slate-400 block mb-0.5">
                            Biaya Part (IDR):
                          </label>
                          <input
                            type="number"
                            placeholder="Contoh: 65000"
                            value={cData.cost || ''}
                            onChange={(e) =>
                              handleItemDataChange(
                                c.spec_component_id,
                                'cost',
                                e.target.value
                              )
                            }
                            className="w-full bg-slate-900 border border-slate-700 rounded-lg px-2.5 py-1 text-slate-200 font-mono text-xs focus:outline-none focus:border-indigo-500"
                          />
                        </div>

                        <div>
                          <label className="text-[10px] text-slate-400 block mb-0.5">
                            Catatan / Merek:
                          </label>
                          <input
                            type="text"
                            placeholder="Merek, spesifikasi part"
                            value={cData.notes}
                            onChange={(e) =>
                              handleItemDataChange(
                                c.spec_component_id,
                                'notes',
                                e.target.value
                              )
                            }
                            className="w-full bg-slate-900 border border-slate-700 rounded-lg px-2.5 py-1 text-slate-200 text-xs focus:outline-none focus:border-indigo-500"
                          />
                        </div>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          </div>

          {/* Total Cost Display */}
          {totalCost > 0 && (
            <div className="flex justify-between items-center p-3 rounded-xl bg-slate-950 border border-slate-800 text-xs">
              <span className="text-slate-400">Total Estimasi Biaya Servis:</span>
              <span className="font-mono font-bold text-white text-sm">
                Rp {totalCost.toLocaleString('id-ID')}
              </span>
            </div>
          )}

          {/* Submit */}
          <div className="pt-2">
            <button
              type="submit"
              disabled={submitting || selectedIds.size === 0}
              className="w-full flex items-center justify-center gap-2 py-2.5 px-4 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm font-semibold transition-all shadow-md shadow-indigo-600/25"
            >
              {submitting ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  <span>Mencatat Servis Atomik...</span>
                </>
              ) : (
                <span>Catat Servis ({selectedIds.size} Komponen)</span>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
