import {
  Meta,
  BikeSpec,
  UserBikeSummary,
  BikeHealthDashboard,
  SystemTelemetryData,
  AddBikeRequest,
  UpdateOdometerRequest,
  LogMaintenanceRequest,
  ApiSuccessEnvelope,
  ApiErrorEnvelope,
} from './types'
import {
  cacheOfflineBikes,
  getCachedBikes,
  cacheOfflineDashboard,
  getCachedDashboard,
} from '../storage/offlineGarage'

// Global client state for headers & telemetry tracking
let activeUserID: string = localStorage.getItem('motospec_user_id') || 'demo-user'
let simulateOutage: boolean = localStorage.getItem('motospec_simulate_outage') === 'true'

type TelemetryListener = (meta: Meta) => void
const telemetryListeners: Set<TelemetryListener> = new Set()

export function getActiveUserID(): string {
  return activeUserID
}

export function setActiveUserID(id: string): void {
  activeUserID = id.trim() || 'demo-user'
  localStorage.setItem('motospec_user_id', activeUserID)
}

export function isSimulateOutageEnabled(): boolean {
  return simulateOutage
}

export function setSimulateOutage(enabled: boolean): void {
  simulateOutage = enabled
  localStorage.setItem('motospec_simulate_outage', enabled ? 'true' : 'false')
}

export function subscribeTelemetry(listener: TelemetryListener): () => void {
  telemetryListeners.add(listener)
  return () => {
    telemetryListeners.delete(listener)
  }
}

function notifyTelemetry(meta?: Meta): void {
  if (meta) {
    telemetryListeners.forEach((fn) => fn(meta))
  }
}

const API_BASE = (import.meta.env.VITE_API_URL as string) || ''

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<ApiSuccessEnvelope<T>> {
  const headers = new Headers(options.headers || {})
  headers.set('Content-Type', 'application/json')
  headers.set('X-User-ID', activeUserID)

  if (simulateOutage) {
    headers.set('X-Simulate-Outage', 'primary')
  }

  const url = `${API_BASE}${endpoint}`

  try {
    const res = await fetch(url, { ...options, headers })
    const json = (await res.json()) as ApiSuccessEnvelope<T> | ApiErrorEnvelope

    if (!res.ok) {
      const errJson = json as ApiErrorEnvelope
      const errCode = errJson?.error?.code || 'UNKNOWN_ERROR'
      const errMsg = errJson?.error?.message || `HTTP ${res.status}`
      throw new Error(`[${errCode}] ${errMsg}`)
    }

    const successJson = json as ApiSuccessEnvelope<T>
    if (successJson.meta) {
      notifyTelemetry(successJson.meta)
    }

    return successJson
  } catch (err) {
    throw err
  }
}

export const api = {
  // Spec search & seeding
  async searchSpec(query: string): Promise<ApiSuccessEnvelope<BikeSpec>> {
    return request<BikeSpec>(`/api/v1/specs/search?q=${encodeURIComponent(query)}`)
  },

  async listSeededModels(): Promise<ApiSuccessEnvelope<BikeSpec[]>> {
    return request<BikeSpec[]>(`/api/v1/specs`)
  },

  // Garage operations with offline fallbacks
  async listUserBikes(): Promise<UserBikeSummary[]> {
    try {
      const res = await request<UserBikeSummary[]>('/api/v1/garage/bikes')
      await cacheOfflineBikes(res.data)
      return res.data
    } catch (err) {
      // Fallback to IndexedDB when offline
      const cached = await getCachedBikes()
      if (cached.length > 0) {
        return cached
      }
      throw err
    }
  },

  async addBike(req: AddBikeRequest): Promise<BikeHealthDashboard> {
    const res = await request<BikeHealthDashboard>('/api/v1/garage/bikes', {
      method: 'POST',
      body: JSON.stringify(req),
    })
    await cacheOfflineDashboard(res.data)
    return res.data
  },

  async getBikeDashboard(bikeId: string): Promise<BikeHealthDashboard> {
    try {
      const res = await request<BikeHealthDashboard>(`/api/v1/garage/bikes/${bikeId}`)
      await cacheOfflineDashboard(res.data)
      return res.data
    } catch (err) {
      const cached = await getCachedDashboard(bikeId)
      if (cached) {
        return cached
      }
      throw err
    }
  },

  async updateOdometer(bikeId: string, req: UpdateOdometerRequest): Promise<BikeHealthDashboard> {
    const res = await request<BikeHealthDashboard>(`/api/v1/garage/bikes/${bikeId}/odometer`, {
      method: 'PATCH',
      body: JSON.stringify(req),
    })
    await cacheOfflineDashboard(res.data)
    return res.data
  },

  async logMaintenance(bikeId: string, req: LogMaintenanceRequest): Promise<BikeHealthDashboard> {
    const res = await request<BikeHealthDashboard>(`/api/v1/garage/bikes/${bikeId}/maintenance`, {
      method: 'POST',
      body: JSON.stringify(req),
    })
    await cacheOfflineDashboard(res.data)
    return res.data
  },

  // Telemetry endpoint
  async getSystemTelemetry(): Promise<SystemTelemetryData> {
    const res = await request<SystemTelemetryData>('/api/v1/system/telemetry')
    return res.data
  },
}
