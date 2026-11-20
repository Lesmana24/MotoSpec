// Domain & REST API Contract Type Definitions

export type ComponentStatus = 'healthy' | 'warning' | 'critical' | 'unknown_needs_inspection'

export interface Meta {
  source: 'redis_cache' | 'postgres_db' | 'llm_primary' | 'llm_fallback' | 'in_memory_metrics'
  provider_used?: string | null
  fallback_triggered?: boolean
  fallback_reason?: string | null
  latency_ms: number
  canonical_slug?: string
}

export interface SpecComponent {
  id: string
  code: string
  name: string
  category: string
  interval_km: number
  interval_months: number
  urgency_level: 'high' | 'medium' | 'low'
  notes: string
}

export interface BikeSpec {
  id: string
  canonical_slug: string
  brand: string
  display_name: string
  category: string
  engine_cc: number
  is_verified: boolean
  components: SpecComponent[]
}

export interface HealthSummary {
  healthy_count: number
  warning_count: number
  critical_count: number
  unknown_count: number
}

export interface UserBikeSummary {
  id: string
  nickname: string
  model_display_name: string
  category: string
  current_odometer_km: number
  daily_estimate_km: number
  auto_increment_enabled: boolean
  summary: HealthSummary
}

export interface ComponentHealthItem {
  spec_component_id: string
  code: string
  name: string
  category: string
  interval_km: number
  interval_months: number
  urgency_level: 'high' | 'medium' | 'low'
  notes: string
  last_replaced_km: number | null
  last_replaced_at: string | null
  next_due_km: number | null
  remaining_km: number | null
  wear_percentage: number | null
  status: ComponentStatus
}

export interface BikeHealthDashboard {
  id: string
  nickname: string
  model_display_name: string
  category: string
  engine_cc: number
  current_odometer_km: number
  daily_estimate_km: number
  auto_increment_enabled: boolean
  last_odometer_sync_at: string
  summary: HealthSummary
  components: ComponentHealthItem[]
}

export interface LLMProviderStatus {
  role: 'primary' | 'fallback'
  name: string
  circuit_state: 'closed' | 'open' | 'half-open'
  consecutive_failures: number
}

export interface SystemTelemetryData {
  cache_hits: number
  cache_misses: number
  cache_hit_ratio_pct: number
  llm_primary_success: number
  llm_fallback_success: number
  llm_total_failures: number
  llm_providers: LLMProviderStatus[]
}

export interface ApiSuccessEnvelope<T> {
  meta?: Meta
  data: T
}

export interface ApiErrorEnvelope {
  error: {
    code: string
    message: string
  }
}

// Requests
export interface ComponentBaselineInput {
  spec_component_id: string
  last_replaced_km?: number | null
  last_replaced_at?: string | null
}

export interface AddBikeRequest {
  bike_model_id: string
  nickname: string
  current_odometer_km: number
  daily_estimate_km: number
  auto_increment_enabled: boolean
  initial_baselines?: ComponentBaselineInput[]
}

export interface UpdateOdometerRequest {
  current_odometer_km: number
  daily_estimate_km: number
  auto_increment_enabled: boolean
}

export interface MaintenanceLogItemInput {
  spec_component_id: string
  cost_idr?: number | null
  notes: string
}

export interface LogMaintenanceRequest {
  odometer_km: number
  serviced_at: string
  items: MaintenanceLogItemInput[]
}
