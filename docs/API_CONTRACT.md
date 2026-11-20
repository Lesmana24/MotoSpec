# REST API Contract — MotoSpec

## 1. Conventions & Global Headers
- **Base URL:** `/api/v1`
- **Content-Type:** `application/json`
- **Global Request Headers:**
  - `X-User-ID` *(optional)*: String identifier isolating user garage data. Defaults to `"demo-user"` if omitted or blank.
  - `X-Simulate-Outage` *(optional)*: Set to `"primary"` on `GET /api/v1/specs/search` to force a primary LLM outage and trigger the fallback chain for portfolio demonstration.
- **Success Response Envelope:** Every `2xx` response wraps the domain payload inside `data` and backend execution telemetry inside `meta`.
- **Error Response Envelope:** Every `4xx` and `5xx` response returns a structured `error` object.

---

## 2. Specification Endpoints

### 2.1 Search & Seed Motorcycle Specification
`GET /api/v1/specs/search?q=vario%20150`

Resolves the query via normalized slug lookup (`Redis -> PostgreSQL -> Primary LLM -> Fallback LLM`).

- **Query Parameters:**
  - `q` *(required)*: Free-text motorcycle name (minimum 2 characters).

- **Response `200 OK`:**

      {
        "meta": {
          "source": "llm_fallback",
          "provider_used": "groq-llama-3.3-70b",
          "fallback_triggered": true,
          "fallback_reason": "simulated_primary_outage",
          "latency_ms": 1120,
          "canonical_slug": "honda-vario-150"
        },
        "data": {
          "id": "550e8400-e29b-41d4-a716-446655440000",
          "canonical_slug": "honda-vario-150",
          "brand": "Honda",
          "display_name": "Honda Vario 150",
          "category": "matic",
          "engine_cc": 150,
          "is_verified": false,
          "components": [
            {
              "id": "770e8400-e29b-41d4-a716-446655440001",
              "code": "engine_oil",
              "name": "Oli Mesin",
              "category": "lubrication",
              "interval_km": 4000,
              "interval_months": 4,
              "urgency_level": "high",
              "notes": "Gunakan oli spesifikasi 10W-30 JASO MB"
            },
            {
              "id": "770e8400-e29b-41d4-a716-446655440002",
              "code": "v_belt",
              "name": "V-Belt & Roller CVT",
              "category": "transmission",
              "interval_km": 24000,
              "interval_months": 24,
              "urgency_level": "high",
              "notes": "Periksa keausan karet dan kepeangan roller"
            }
          ]
        }
      }

*(Note: When `meta.source` is `"redis_cache"` or `"postgres_db"`, `provider_used` is `null` and `fallback_triggered` is `false`.)*

---

### 2.2 List Seeded Motorcycle Models
`GET /api/v1/specs`

Returns all motorcycle models already persisted in PostgreSQL for instant selection without invoking LLM providers.

- **Response `200 OK`:**

      {
        "meta": {
          "source": "postgres_db",
          "latency_ms": 8
        },
        "data": [
          {
            "id": "550e8400-e29b-41d4-a716-446655440000",
            "canonical_slug": "honda-vario-150",
            "brand": "Honda",
            "display_name": "Honda Vario 150",
            "category": "matic",
            "engine_cc": 150,
            "is_verified": false
          }
        ]
      }

---

## 3. User Garage Endpoints

### 3.1 List User Motorcycles
`GET /api/v1/garage/bikes`

Lists all motorcycles registered under the active `X-User-ID` along with high-level health counters.

- **Response `200 OK`:**

      {
        "meta": {
          "source": "postgres_db",
          "latency_ms": 11
        },
        "data": [
          {
            "id": "990e8400-e29b-41d4-a716-446655440099",
            "nickname": "Vario Harian Kantor",
            "model_display_name": "Honda Vario 150",
            "category": "matic",
            "current_odometer_km": 18500,
            "daily_estimate_km": 35,
            "auto_increment_enabled": true,
            "summary": {
              "healthy_count": 5,
              "warning_count": 1,
              "critical_count": 0,
              "unknown_count": 1
            }
          }
        ]
      }

---

### 3.2 Add Motorcycle to Garage
`POST /api/v1/garage/bikes`

Registers a motorcycle to the user's garage. Any component omitted from `initial_baselines` is initialized with `last_replaced_km: null` and `last_replaced_at: null` (`unknown_needs_inspection`).

- **Request Body:**

      {
        "bike_model_id": "550e8400-e29b-41d4-a716-446655440000",
        "nickname": "Vario Harian Kantor",
        "current_odometer_km": 18500,
        "daily_estimate_km": 35,
        "auto_increment_enabled": true,
        "initial_baselines": [
          {
            "spec_component_id": "770e8400-e29b-41d4-a716-446655440001",
            "last_replaced_km": 16000,
            "last_replaced_at": "2026-10-15T08:00:00Z"
          }
        ]
      }

- **Response `201 Created`:** Returns the full Bike Health Dashboard payload (identical schema to Endpoint 3.3).

---

### 3.3 Get Bike Health Dashboard
`GET /api/v1/garage/bikes/:id`

Returns full component wear metrics computed by `backend/internal/service`, sorted by urgency (`critical` -> `warning` -> `unknown_needs_inspection` -> `healthy`).

- **Response `200 OK`:**

      {
        "meta": {
          "source": "postgres_db",
          "latency_ms": 14
        },
        "data": {
          "id": "990e8400-e29b-41d4-a716-446655440099",
          "nickname": "Vario Harian Kantor",
          "model_display_name": "Honda Vario 150",
          "category": "matic",
          "engine_cc": 150,
          "current_odometer_km": 18500,
          "daily_estimate_km": 35,
          "auto_increment_enabled": true,
          "last_odometer_sync_at": "2026-11-20T00:00:00Z",
          "summary": {
            "healthy_count": 1,
            "warning_count": 0,
            "critical_count": 0,
            "unknown_count": 1
          },
          "components": [
            {
              "spec_component_id": "770e8400-e29b-41d4-a716-446655440001",
              "code": "engine_oil",
              "name": "Oli Mesin",
              "category": "lubrication",
              "interval_km": 4000,
              "interval_months": 4,
              "urgency_level": "high",
              "notes": "Gunakan oli spesifikasi 10W-30 JASO MB",
              "last_replaced_km": 16000,
              "last_replaced_at": "2026-10-15T08:00:00Z",
              "next_due_km": 20000,
              "remaining_km": 1500,
              "wear_percentage": 62.5,
              "status": "healthy"
            },
            {
              "spec_component_id": "770e8400-e29b-41d4-a716-446655440002",
              "code": "v_belt",
              "name": "V-Belt & Roller CVT",
              "category": "transmission",
              "interval_km": 24000,
              "interval_months": 24,
              "urgency_level": "high",
              "notes": "Periksa keausan karet dan kepeangan roller",
              "last_replaced_km": null,
              "last_replaced_at": null,
              "next_due_km": null,
              "remaining_km": null,
              "wear_percentage": null,
              "status": "unknown_needs_inspection"
            }
          ]
        }
      }

---

### 3.4 Update Odometer & Daily Auto-Increment
`PATCH /api/v1/garage/bikes/:id/odometer`

Updates the motorcycle's current odometer reading and daily estimation settings. Rejects values lower than the current odometer.

- **Request Body:**

      {
        "current_odometer_km": 19200,
        "daily_estimate_km": 40,
        "auto_increment_enabled": true
      }

- **Response `200 OK`:** Returns the updated Bike Health Dashboard payload (Endpoint 3.3).

---

### 3.5 Log Component Maintenance (Batch Supported)
`POST /api/v1/garage/bikes/:id/maintenance`

Logs one or more component replacements in a single atomic transaction, resetting their `last_replaced_km` and `last_replaced_at` baselines. If `odometer_km > current_odometer_km`, `current_odometer_km` is automatically bumped to match.

- **Request Body:**

      {
        "odometer_km": 19200,
        "serviced_at": "2026-11-20T10:30:00Z",
        "items": [
          {
            "spec_component_id": "770e8400-e29b-41d4-a716-446655440001",
            "cost_idr": 65000,
            "notes": "Ganti oli MPX2 + ring baut"
          },
          {
            "spec_component_id": "770e8400-e29b-41d4-a716-446655440002",
            "cost_idr": 185000,
            "notes": "Ganti V-belt set original"
          }
        ]
      }

- **Response `201 Created`:** Returns the updated Bike Health Dashboard payload (Endpoint 3.3).

---

### 3.6 Get Maintenance History Logs
`GET /api/v1/garage/bikes/:id/maintenance`

Returns historical maintenance records ordered by `serviced_at DESC`.

- **Response `200 OK`:**

      {
        "meta": {
          "source": "postgres_db",
          "latency_ms": 9
        },
        "data": [
          {
            "id": "aa0e8400-e29b-41d4-a716-446655440011",
            "component_code": "engine_oil",
            "component_name": "Oli Mesin",
            "odometer_km": 19200,
            "serviced_at": "2026-11-20T10:30:00Z",
            "cost_idr": 65000,
            "notes": "Ganti oli MPX2 + ring baut"
          }
        ]
      }

---

## 4. System & Telemetry Endpoints

### 4.1 Get Backend Resilience & Cache Telemetry
`GET /api/v1/system/telemetry`

Exposes real-time cache hit ratios and circuit breaker states for the PWA telemetry drawer.

- **Response `200 OK`:**

      {
        "meta": {
          "source": "in_memory_metrics",
          "latency_ms": 1
        },
        "data": {
          "cache_hits": 42,
          "cache_misses": 6,
          "cache_hit_ratio_pct": 87.5,
          "llm_primary_success": 4,
          "llm_fallback_success": 2,
          "llm_total_failures": 0,
          "llm_providers": [
            {
              "role": "primary",
              "name": "gemini-2.5-flash",
              "circuit_state": "closed",
              "consecutive_failures": 0
            },
            {
              "role": "fallback",
              "name": "groq-llama-3.3-70b",
              "circuit_state": "closed",
              "consecutive_failures": 0
            }
          ]
        }
      }

### 4.2 Health Check
`GET /healthz`

- **Response `200 OK`:**

      {
        "status": "ok",
        "postgres": "connected",
        "redis": "connected"
      }

---

## 5. Standard Error Codes

### Error Envelope Format

    {
      "error": {
        "code": "INVALID_ODOMETER",
        "message": "New odometer (17000 km) cannot be lower than current odometer (18500 km)."
      }
    }

### Error Code Catalog

| HTTP Status | Error Code | Trigger Condition |
| :--- | :--- | :--- |
| `400` | `INVALID_REQUEST` | Malformed JSON body, missing required fields, or invalid UUID format |
| `400` | `INVALID_ODOMETER` | Submitted odometer reading is lower than `current_odometer_km` |
| `404` | `NOT_FOUND` | Motorcycle ID or component ID does not exist or belongs to another `X-User-ID` |
| `503` | `ALL_PROVIDERS_FAILED` | Cache miss where both primary and fallback LLM providers failed or timed out |
| `500` | `INTERNAL_ERROR` | Unexpected database, Redis, or server error |