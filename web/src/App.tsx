import { useState, useEffect, useCallback } from 'react'
import {
  Bike,
  Gauge,
  Wrench,
  Plus,
  RefreshCw,
  Search,
  AlertCircle,
  AlertTriangle,
  CheckCircle2,
  HelpCircle,
  Edit2,
  WifiOff,
} from 'lucide-react'
import {
  UserBikeSummary,
  BikeHealthDashboard,
  BikeSpec,
  ComponentHealthItem,
  Meta,
} from './api/types'
import {
  api,
  getActiveUserID,
  setActiveUserID,
  subscribeTelemetry,
} from './api/client'
import { Header } from './components/Header'
import { ComponentCard } from './components/ComponentCard'
import { SpecSearchModal } from './components/SpecSearchModal'
import { AddBikeModal } from './components/AddBikeModal'
import { OdometerModal } from './components/OdometerModal'
import { MaintenanceModal } from './components/MaintenanceModal'
import { TelemetryDrawer } from './components/TelemetryDrawer'

type ComponentFilter = 'all' | 'needs_attention' | 'healthy' | 'unknown'

export function App() {
  const [activeUserId, setActiveUserIdState] = useState<string>(getActiveUserID())
  const [isOnline, setIsOnline] = useState<boolean>(navigator.onLine)
  const [lastMeta, setLastMeta] = useState<Meta | null>(null)

  // Garage state
  const [bikes, setBikes] = useState<UserBikeSummary[]>([])
  const [selectedBikeId, setSelectedBikeId] = useState<string | null>(null)
  const [dashboard, setDashboard] = useState<BikeHealthDashboard | null>(null)
  const [loading, setLoading] = useState<boolean>(true)
  const [refreshing, setRefreshing] = useState<boolean>(false)
  const [error, setError] = useState<string | null>(null)

  // Modals state
  const [isSearchOpen, setIsSearchOpen] = useState<boolean>(false)
  const [isAddBikeOpen, setIsAddBikeOpen] = useState<boolean>(false)
  const [selectedSpecForAdd, setSelectedSpecForAdd] = useState<BikeSpec | null>(null)
  const [isOdometerOpen, setIsOdometerOpen] = useState<boolean>(false)
  const [isMaintenanceOpen, setIsMaintenanceOpen] = useState<boolean>(false)
  const [preselectedComponentId, setPreselectedComponentId] = useState<string | null>(null)
  const [isTelemetryOpen, setIsTelemetryOpen] = useState<boolean>(false)

  // Filter state
  const [componentFilter, setComponentFilter] = useState<ComponentFilter>('all')

  // Listen to network status
  useEffect(() => {
    const handleOnline = () => setIsOnline(true)
    const handleOffline = () => setIsOnline(false)

    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)
    return () => {
      window.removeEventListener('online', handleOnline)
      window.removeEventListener('offline', handleOffline)
    }
  }, [])

  // Listen to live telemetry from API requests
  useEffect(() => {
    const unsubscribe = subscribeTelemetry((meta) => {
      setLastMeta(meta)
    })
    return unsubscribe
  }, [])

  // Load bikes list
  const loadBikes = useCallback(async (preferredBikeId?: string) => {
    try {
      setError(null)
      const bikeList = await api.listUserBikes()
      setBikes(bikeList)

      if (bikeList.length > 0) {
        const targetId = preferredBikeId && bikeList.some((b) => b.id === preferredBikeId)
          ? preferredBikeId
          : bikeList[0].id
        setSelectedBikeId(targetId)
        await loadBikeDashboard(targetId)
      } else {
        setSelectedBikeId(null)
        setDashboard(null)
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Gagal memuat daftar motor garasi'
      setError(msg)
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [])

  // Load dashboard for selected bike
  const loadBikeDashboard = async (bikeId: string) => {
    try {
      setLoading(true)
      const data = await api.getBikeDashboard(bikeId)
      setDashboard(data)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Gagal memuat rincian motor'
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  // Handle switching active user
  const handleUserIdChange = (newId: string) => {
    setActiveUserIdState(newId)
    setActiveUserID(newId)
    setLoading(true)
    loadBikes()
  }

  // Initial load
  useEffect(() => {
    loadBikes()
  }, [loadBikes])

  // Change selected bike
  const handleSelectBike = (bikeId: string) => {
    if (bikeId !== selectedBikeId) {
      setSelectedBikeId(bikeId)
      loadBikeDashboard(bikeId)
    }
  }

  // Refresh current bike data
  const handleRefresh = async () => {
    if (selectedBikeId) {
      setRefreshing(true)
      await loadBikeDashboard(selectedBikeId)
      setRefreshing(false)
    } else {
      loadBikes()
    }
  }

  // Handle opening add bike from spec search modal
  const handleSelectSpecToAdd = (spec: BikeSpec) => {
    setIsSearchOpen(false)
    setSelectedSpecForAdd(spec)
    setIsAddBikeOpen(true)
  }

  // Handle single component maintenance click
  const handleLogComponentMaintenance = (component: ComponentHealthItem) => {
    setPreselectedComponentId(component.spec_component_id)
    setIsMaintenanceOpen(true)
  }

  // Filter components based on status
  const filteredComponents = dashboard?.components.filter((c) => {
    if (componentFilter === 'all') return true
    if (componentFilter === 'needs_attention') {
      return c.status === 'critical' || c.status === 'warning'
    }
    if (componentFilter === 'healthy') return c.status === 'healthy'
    if (componentFilter === 'unknown') return c.status === 'unknown_needs_inspection'
    return true
  }) || []

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col font-sans">
      {/* Top Application Header */}
      <Header
        isOnline={isOnline}
        lastMeta={lastMeta}
        activeUserId={activeUserId}
        onOpenSearch={() => setIsSearchOpen(true)}
        onOpenTelemetry={() => setIsTelemetryOpen(true)}
        onChangeUserId={handleUserIdChange}
      />

      {/* Offline Notice Banner */}
      {!isOnline && (
        <div className="bg-amber-500/10 border-b border-amber-500/20 px-4 py-2 text-xs text-amber-300 flex items-center justify-center gap-2">
          <WifiOff className="w-3.5 h-3.5" />
          <span>
            Mode Offline Aktif — Menampilkan data garasi lokal (IndexedDB). Aksi mutasi akan otomatis tersinkron saat koneksi kembali.
          </span>
        </div>
      )}

      {/* Main Container */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6">
        {/* Error Notification */}
        {error && (
          <div className="flex items-center justify-between p-4 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-xs">
            <div className="flex items-center gap-2">
              <AlertCircle className="w-4 h-4 text-rose-400 shrink-0" />
              <span>{error}</span>
            </div>
            <button
              onClick={() => setError(null)}
              className="text-rose-400 hover:text-white ml-2 text-xs font-semibold"
            >
              Tutup
            </button>
          </div>
        )}

        {/* Motorcycle Selection Tabs & Actions */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-800 pb-4">
          <div className="flex items-center gap-2 overflow-x-auto pb-1 sm:pb-0 scrollbar-none">
            {bikes.map((bike) => {
              const isSelected = bike.id === selectedBikeId
              const hasCritical = bike.summary.critical_count > 0
              const hasWarning = bike.summary.warning_count > 0

              return (
                <button
                  key={bike.id}
                  onClick={() => handleSelectBike(bike.id)}
                  className={`flex items-center gap-2 px-3.5 py-2 rounded-xl text-xs font-semibold whitespace-nowrap transition-all border ${
                    isSelected
                      ? 'bg-indigo-600 border-indigo-500 text-white shadow-md shadow-indigo-600/30'
                      : 'bg-slate-900 border-slate-800 text-slate-300 hover:bg-slate-850 hover:border-slate-700'
                  }`}
                >
                  <Bike className="w-3.5 h-3.5" />
                  <span>{bike.nickname}</span>
                  {hasCritical ? (
                    <span className="w-2 h-2 rounded-full bg-rose-400 animate-pulse" />
                  ) : hasWarning ? (
                    <span className="w-2 h-2 rounded-full bg-amber-400" />
                  ) : null}
                </button>
              )
            })}

            {/* Quick Add Button */}
            <button
              onClick={() => setIsSearchOpen(true)}
              className="flex items-center gap-1 px-3 py-2 rounded-xl text-xs font-medium text-slate-400 hover:text-white bg-slate-900/60 hover:bg-slate-800 border border-dashed border-slate-700/80 whitespace-nowrap transition-colors"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>Tambah Motor</span>
            </button>
          </div>

          {/* Refresh action */}
          <div className="flex items-center gap-2 self-end sm:self-auto">
            <button
              onClick={handleRefresh}
              disabled={refreshing}
              className="p-2 rounded-lg bg-slate-900 hover:bg-slate-800 border border-slate-800 text-slate-400 hover:text-slate-200 transition-colors"
              title="Refresh Garasi"
            >
              <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin text-indigo-400' : ''}`} />
            </button>
          </div>
        </div>

        {/* Empty State: No bikes yet */}
        {bikes.length === 0 && !loading && (
          <div className="py-16 px-4 text-center max-w-lg mx-auto space-y-4 bg-slate-900/60 border border-slate-800 rounded-2xl p-8">
            <div className="w-14 h-14 mx-auto rounded-2xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
              <Bike className="w-8 h-8" />
            </div>
            <div>
              <h2 className="text-lg font-bold text-white">Garasi Masih Kosong</h2>
              <p className="text-xs text-slate-400 mt-1">
                Gunakan pencarian pintar untuk menemukan spesifikasi motor pabrikan resmi Indonesia. Sistem akan memanfaatkan cache multi-layer (Redis + PostgreSQL) atau memanggil LLM Orchestrator jika data belum ada.
              </p>
            </div>
            <button
              onClick={() => setIsSearchOpen(true)}
              className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-lg shadow-indigo-600/30 transition-all"
            >
              <Search className="w-4 h-4" />
              <span>Cari & Tambahkan Motor Pertama</span>
            </button>
          </div>
        )}

        {/* Active Motorcycle Dashboard */}
        {dashboard && (
          <div className="space-y-6">
            {/* Top Bike Summary Card */}
            <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 sm:p-6 shadow-xl relative overflow-hidden">
              <div className="absolute top-0 right-0 w-64 h-64 bg-indigo-500/5 rounded-full blur-3xl -z-0 pointer-events-none" />

              <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-6 relative z-10">
                {/* Bike Identity & Odometer */}
                <div>
                  <div className="flex items-center gap-2 text-xs text-slate-400 font-mono mb-1">
                    <span className="capitalize">{dashboard.category}</span>
                    <span>•</span>
                    <span>{dashboard.engine_cc} cc</span>
                    <span>•</span>
                    <span className="text-slate-500">{dashboard.model_display_name}</span>
                  </div>
                  <h2 className="text-2xl font-black text-white tracking-tight">
                    {dashboard.nickname}
                  </h2>

                  {/* Odometer Display & Quick Edit */}
                  <div className="flex items-center gap-3 mt-3">
                    <div className="flex items-baseline gap-1.5 bg-slate-950 px-3.5 py-1.5 rounded-xl border border-slate-800">
                      <Gauge className="w-4 h-4 text-indigo-400 self-center" />
                      <span className="text-xl font-bold font-mono text-white">
                        {dashboard.current_odometer_km.toLocaleString()}
                      </span>
                      <span className="text-xs font-mono text-slate-500">km</span>
                    </div>

                    <button
                      onClick={() => setIsOdometerOpen(true)}
                      className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 border border-slate-700 text-xs font-medium text-slate-300 hover:text-white transition-colors"
                      title="Update Odometer"
                    >
                      <Edit2 className="w-3 h-3 text-indigo-400" />
                      <span>Update KM</span>
                    </button>

                    {/* Auto increment status */}
                    <span className="text-[11px] text-slate-400 hidden sm:inline">
                      (+{dashboard.daily_estimate_km} km/hari{dashboard.auto_increment_enabled ? ', cron aktif' : ', manual'})
                    </span>
                  </div>
                </div>

                {/* Health Summary Status Counters */}
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
                  {/* Critical */}
                  <div className="bg-rose-500/10 border border-rose-500/25 p-3 rounded-xl flex items-center gap-3">
                    <AlertCircle className="w-5 h-5 text-rose-400 shrink-0" />
                    <div>
                      <span className="text-lg font-bold font-mono text-rose-300 leading-none block">
                        {dashboard.summary.critical_count}
                      </span>
                      <span className="text-[10px] text-rose-400/80 uppercase font-semibold">
                        Kritis (&gt;=100%)
                      </span>
                    </div>
                  </div>

                  {/* Warning */}
                  <div className="bg-amber-500/10 border border-amber-500/25 p-3 rounded-xl flex items-center gap-3">
                    <AlertTriangle className="w-5 h-5 text-amber-400 shrink-0" />
                    <div>
                      <span className="text-lg font-bold font-mono text-amber-300 leading-none block">
                        {dashboard.summary.warning_count}
                      </span>
                      <span className="text-[10px] text-amber-400/80 uppercase font-semibold">
                        Waspada (&gt;=75%)
                      </span>
                    </div>
                  </div>

                  {/* Healthy */}
                  <div className="bg-emerald-500/10 border border-emerald-500/25 p-3 rounded-xl flex items-center gap-3">
                    <CheckCircle2 className="w-5 h-5 text-emerald-400 shrink-0" />
                    <div>
                      <span className="text-lg font-bold font-mono text-emerald-300 leading-none block">
                        {dashboard.summary.healthy_count}
                      </span>
                      <span className="text-[10px] text-emerald-400/80 uppercase font-semibold">
                        Kondisi Baik
                      </span>
                    </div>
                  </div>

                  {/* Unknown */}
                  <div className="bg-slate-800/60 border border-slate-700/60 p-3 rounded-xl flex items-center gap-3">
                    <HelpCircle className="w-5 h-5 text-slate-400 shrink-0" />
                    <div>
                      <span className="text-lg font-bold font-mono text-slate-300 leading-none block">
                        {dashboard.summary.unknown_count}
                      </span>
                      <span className="text-[10px] text-slate-400 uppercase font-semibold">
                        Perlu Inspeksi
                      </span>
                    </div>
                  </div>
                </div>

                {/* Primary Action Button */}
                <div className="flex flex-col sm:flex-row gap-2 shrink-0">
                  <button
                    onClick={() => {
                      setPreselectedComponentId(null)
                      setIsMaintenanceOpen(true)
                    }}
                    className="flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md shadow-indigo-600/30 transition-all"
                  >
                    <Wrench className="w-4 h-4" />
                    <span>Catat Servis / Ganti Part</span>
                  </button>
                </div>
              </div>
            </div>

            {/* Component Section Header & Filter Tabs */}
            <div className="space-y-4">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div className="flex items-center gap-2">
                  <h3 className="text-base font-bold text-white">Status Keausan Komponen</h3>
                  <span className="text-xs font-mono text-slate-500">
                    ({filteredComponents.length} part)
                  </span>
                </div>

                {/* Filter Pills */}
                <div className="flex items-center gap-1.5 overflow-x-auto pb-1 sm:pb-0 scrollbar-none">
                  <button
                    onClick={() => setComponentFilter('all')}
                    className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors ${
                      componentFilter === 'all'
                        ? 'bg-slate-800 text-white border border-slate-700'
                        : 'text-slate-400 hover:text-slate-200'
                    }`}
                  >
                    Semua ({dashboard.components.length})
                  </button>
                  <button
                    onClick={() => setComponentFilter('needs_attention')}
                    className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors ${
                      componentFilter === 'needs_attention'
                        ? 'bg-rose-500/20 text-rose-300 border border-rose-500/30'
                        : 'text-slate-400 hover:text-rose-300'
                    }`}
                  >
                    Butuh Servis ({dashboard.summary.critical_count + dashboard.summary.warning_count})
                  </button>
                  <button
                    onClick={() => setComponentFilter('healthy')}
                    className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors ${
                      componentFilter === 'healthy'
                        ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30'
                        : 'text-slate-400 hover:text-emerald-300'
                    }`}
                  >
                    Baik ({dashboard.summary.healthy_count})
                  </button>
                  <button
                    onClick={() => setComponentFilter('unknown')}
                    className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors ${
                      componentFilter === 'unknown'
                        ? 'bg-slate-800 text-slate-300 border border-slate-700'
                        : 'text-slate-400 hover:text-slate-200'
                    }`}
                  >
                    Perlu Inspeksi ({dashboard.summary.unknown_count})
                  </button>
                </div>
              </div>

              {/* Component Cards Grid */}
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {filteredComponents.map((component) => (
                  <ComponentCard
                    key={component.spec_component_id}
                    component={component}
                    onLogMaintenance={handleLogComponentMaintenance}
                  />
                ))}
              </div>

              {filteredComponents.length === 0 && (
                <div className="p-8 text-center text-slate-500 bg-slate-900/40 rounded-xl border border-slate-850">
                  <p className="text-xs">Tidak ada komponen pada filter ini.</p>
                </div>
              )}
            </div>
          </div>
        )}
      </main>

      {/* Modals & Overlays */}
      <SpecSearchModal
        isOpen={isSearchOpen}
        onClose={() => setIsSearchOpen(false)}
        onSelectSpecToAdd={handleSelectSpecToAdd}
      />

      {selectedSpecForAdd && (
        <AddBikeModal
          isOpen={isAddBikeOpen}
          onClose={() => {
            setIsAddBikeOpen(false)
            setSelectedSpecForAdd(null)
          }}
          spec={selectedSpecForAdd}
          onSuccess={(newDashboard) => {
            loadBikes(newDashboard.id)
          }}
        />
      )}

      {dashboard && (
        <OdometerModal
          isOpen={isOdometerOpen}
          onClose={() => setIsOdometerOpen(false)}
          bike={dashboard}
          onSuccess={(updated) => {
            setDashboard(updated)
            loadBikes(updated.id)
          }}
        />
      )}

      {dashboard && (
        <MaintenanceModal
          isOpen={isMaintenanceOpen}
          onClose={() => {
            setIsMaintenanceOpen(false)
            setPreselectedComponentId(null)
          }}
          bike={dashboard}
          preselectedComponentId={preselectedComponentId}
          onSuccess={(updated) => {
            setDashboard(updated)
            loadBikes(updated.id)
          }}
        />
      )}

      <TelemetryDrawer
        isOpen={isTelemetryOpen}
        onClose={() => setIsTelemetryOpen(false)}
      />
    </div>
  )
}

export default App
