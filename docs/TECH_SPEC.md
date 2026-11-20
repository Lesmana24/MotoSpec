# Technical Specification — MotoSpec

## 1. Read-Through Cache & LLM Seeding Pipeline

    GET /api/v1/specs/search?q="Honda Vario 150cc"
           │
           ▼
    [1. Normalize Query] ──► canonical_slug: "honda-vario-150"
           │
           ▼
    [2. Redis Lookup] ─── Key: "bike:spec:honda-vario-150" (TTL: 24h)
           ├─► HIT  ──► Increment cache_hits ──► Return Spec (meta.source = "redis_cache")
           └─► MISS ──► Increment cache_misses
                │
                ▼
    [3. PostgreSQL Lookup] ── Query: bike_models + bike_spec_components
           ├─► HIT  ──► Backfill Redis (24h TTL) ──► Return Spec (meta.source = "postgres_db")
           └─► MISS
                │
                ▼
    [4. LLM Orchestrator] ── Primary Provider (4s timeout + gobreaker)
           ├─► SUCCESS + Valid Sanity Check ──► Persist DB & Redis ──► Return ("llm_primary")
           └─► FAIL / Timeout / Invalid Schema / X-Simulate-Outage: primary
                │
                ▼
    [5. Fallback LLM Provider] ── Secondary Provider (4s timeout)
           ├─► SUCCESS + Valid Sanity Check ──► Persist DB & Redis ──► Return ("llm_fallback")
           └─► FAIL ──► Return HTTP 503 Service Unavailable (ALL_PROVIDERS_FAILED)

### Concurrency Protection (Singleflight)
- Wrap cache-miss database/LLM resolution in `golang.org/x/sync/singleflight` keyed by `canonical_slug`.
- If 10 concurrent requests query an uncached `"yamaha-aerox-155"` simultaneously, only **1** LLM call is executed; the remaining 9 await the shared result to prevent duplicate LLM billing and database unique-constraint collisions.

---

## 2. Query Normalization Rules (`internal/service/normalizer.go`)
Before querying Redis, PostgreSQL, or an LLM, raw user input MUST pass through deterministic normalization:
1. Trim leading/trailing whitespace and convert to lowercase.
2. Strip standalone displacement suffixes (`cc`, `ccs`) and punctuation (keep alphanumeric, spaces, and hyphens).
3. Collapse consecutive whitespace or hyphens into a single hyphen (`-`).
4. Apply Indonesian market brand-alias prefix mapping when the brand name is omitted:
   - `vario`, `beat`, `scoopy`, `pcx`, `adv`, `cbr`, `cb150`, `crf`, `supra`, `revo`, `genio`, `stylo` -> prefix `honda-`
   - `nmax`, `aerox`, `lexi`, `fazzio`, `filano`, `r15`, `r25`, `mt15`, `mt25`, `vixion`, `mx-king`, `mio`, `xsr` -> prefix `yamaha-`
   - `nex`, `address`, `burgman`, `satria`, `gsx` -> prefix `suzuki-`
   - `ninja`, `klx`, `w175`, `zx25r` -> prefix `kawasaki-`
   - `vespa`, `sprint`, `primavera`, `lx125` -> prefix `vespa-`
5. Example: `"  Vario 150cc  "` -> `"honda-vario-150"`.

---

## 3. LLM Orchestrator & Resilience (`internal/llm`)

### 3.1 Core Interface & Data Structures

    type SpecGenerator interface {
        Name() string
        GenerateSpec(ctx context.Context, canonicalSlug string, rawQuery string) (*GeneratedBikeSpec, error)
    }

    type GeneratedBikeSpec struct {
        Brand       string                   `json:"brand" validate:"required,min=2,max=60"`
        DisplayName string                   `json:"display_name" validate:"required,min=3,max=120"`
        Category    string                   `json:"category" validate:"required,oneof=matic bebek sport naked trail cruiser"`
        EngineCC    int                      `json:"engine_cc" validate:"required,gte=50,lte=2500"`
        Components  []GeneratedSpecComponent `json:"components" validate:"required,min=4,max=20,dive"`
    }

    type GeneratedSpecComponent struct {
        Code           string `json:"code" validate:"required,min=2,max=60"`
        Name           string `json:"name" validate:"required,min=2,max=100"`
        Category       string `json:"category" validate:"required,oneof=lubrication transmission brakes ignition filtration cooling tires"`
        IntervalKM     int    `json:"interval_km" validate:"required,gte=500,lte=100000"`
        IntervalMonths int    `json:"interval_months" validate:"gte=0,lte=60"`
        UrgencyLevel   string `json:"urgency_level" validate:"required,oneof=high medium low"`
        Notes          string `json:"notes" validate:"max=300"`
    }

### 3.2 Provider Configuration & Fault-Tolerance
- **Primary Provider:** Fast low-latency model (e.g., Gemini 2.5 Flash via REST API with `responseMimeType: "application/json"`).
- **Fallback Provider:** Independent secondary model/vendor (e.g., Groq Llama 3.3 70B or OpenAI GPT-4o-mini with JSON mode).
- **Per-Provider Timeout:** `4 * time.Second` via `context.WithTimeout`.
- **Circuit Breaker (`sony/gobreaker` on Primary):**
  - `MaxRequests` (Half-Open probe limit): `1`
  - `Interval` (Cyclic failure window): `60 * time.Second`
  - `Timeout` (Open state duration before Half-Open): `60 * time.Second`
  - `ReadyToTrip`: trips to `Open` when `counts.ConsecutiveFailures >= 3`
- **Sanity Check Validator (Triggers Fallback on Failure):**
  1. Strip markdown code fences if the LLM wraps its JSON response.
  2. Unmarshal strictly into `GeneratedBikeSpec`.
  3. Run `validator.New().Struct(spec)`.
  4. Verify `engine_oil` exists in `Components` (every internal combustion motorcycle requires engine oil tracking).
  5. Ensure no duplicate `code` entries exist within `Components`.
- **Outage Simulation Hook:**
  - When `X-Simulate-Outage: primary` is present on the HTTP request, `internal/handler` injects a context key (`llm.WithSimulatedPrimaryOutage(ctx)`).
  - The orchestrator detects this flag, skips the primary provider call, sets `fallback_triggered = true` and `fallback_reason = "simulated_primary_outage"`, and executes the fallback provider directly.

---

## 4. Redis Caching Strategy (`internal/repository/cache.go`)
- **Spec Cache Key:** `bike:spec:{canonical_slug}`
  - **Value:** JSON-serialized `BikeSpecDTO` (including database UUIDs for `bike_model_id` and `bike_spec_components.id`).
  - **TTL:** `24 * time.Hour`
- **Telemetry Counters (In-Memory Atomic + Optional Redis Persistence):**
  - Track `cache_hits`, `cache_misses`, `llm_primary_success`, `llm_fallback_success`, and `llm_total_failures` using `sync/atomic` counters exposed via `GET /api/v1/system/telemetry`.

---

## 5. Background Worker (`internal/worker/odometer_worker.go`)
- **Scheduler:** `github.com/robfig/cron/v3`
- **Cron Expression:** `0 0 * * *` (Daily at 00:00 local server time).
- **Behavior:**
  - Executes a single set-based SQL `UPDATE` on `user_bikes` where `auto_increment_enabled = TRUE` AND `daily_estimate_km > 0` AND `last_odometer_sync_at < CURRENT_DATE`.
  - Atomically increments `current_odometer_km = current_odometer_km + daily_estimate_km` and sets `last_odometer_sync_at = NOW()`, `updated_at = NOW()`.
  - The `last_odometer_sync_at < CURRENT_DATE` predicate guarantees idempotency if the worker is manually triggered or restarted on the same day.

---

## 6. PostgreSQL Database Schema (DDL)
File location: `backend/db/migrations/000001_init.up.sql`

    CREATE EXTENSION IF NOT EXISTS "pgcrypto";

    -- 1. Master Motorcycle Catalog (Seeded by LLM or Admin)
    CREATE TABLE bike_models (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        canonical_slug VARCHAR(120) UNIQUE NOT NULL,
        brand VARCHAR(60) NOT NULL,
        display_name VARCHAR(120) NOT NULL,
        category VARCHAR(40) NOT NULL,
        engine_cc INT NOT NULL CHECK (engine_cc > 0),
        is_verified BOOLEAN NOT NULL DEFAULT FALSE,
        source_provider VARCHAR(80) NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

    CREATE INDEX idx_bike_models_slug ON bike_models(canonical_slug);

    -- 2. Master Component Intervals per Bike Model
    CREATE TABLE bike_spec_components (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        bike_model_id UUID NOT NULL REFERENCES bike_models(id) ON DELETE CASCADE,
        code VARCHAR(60) NOT NULL,
        name VARCHAR(100) NOT NULL,
        category VARCHAR(50) NOT NULL,
        interval_km INT NOT NULL CHECK (interval_km >= 500),
        interval_months INT NOT NULL DEFAULT 0 CHECK (interval_months >= 0),
        urgency_level VARCHAR(20) NOT NULL DEFAULT 'medium',
        notes TEXT NOT NULL DEFAULT '',
        UNIQUE (bike_model_id, code)
    );

    CREATE INDEX idx_spec_components_model_id ON bike_spec_components(bike_model_id);

    -- 3. User Garage Motorcycles
    CREATE TABLE user_bikes (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        user_id VARCHAR(100) NOT NULL,
        bike_model_id UUID NOT NULL REFERENCES bike_models(id) ON DELETE RESTRICT,
        nickname VARCHAR(100) NOT NULL,
        current_odometer_km INT NOT NULL CHECK (current_odometer_km >= 0),
        daily_estimate_km INT NOT NULL DEFAULT 0 CHECK (daily_estimate_km >= 0),
        auto_increment_enabled BOOLEAN NOT NULL DEFAULT FALSE,
        last_odometer_sync_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

    CREATE INDEX idx_user_bikes_user_id ON user_bikes(user_id);

    -- 4. Current Component Baseline per User Bike
    CREATE TABLE user_component_states (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        user_bike_id UUID NOT NULL REFERENCES user_bikes(id) ON DELETE CASCADE,
        spec_component_id UUID NOT NULL REFERENCES bike_spec_components(id) ON DELETE CASCADE,
        last_replaced_km INT NULL CHECK (last_replaced_km >= 0),
        last_replaced_at TIMESTAMPTZ NULL,
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        UNIQUE (user_bike_id, spec_component_id)
    );

    CREATE INDEX idx_user_comp_states_bike_id ON user_component_states(user_bike_id);

    -- 5. Historical Maintenance Audit Logs
    CREATE TABLE maintenance_logs (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        user_bike_id UUID NOT NULL REFERENCES user_bikes(id) ON DELETE CASCADE,
        spec_component_id UUID NOT NULL REFERENCES bike_spec_components(id) ON DELETE CASCADE,
        odometer_km INT NOT NULL CHECK (odometer_km >= 0),
        serviced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        cost_idr INT NULL CHECK (cost_idr >= 0),
        notes TEXT NOT NULL DEFAULT '',
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

    CREATE INDEX idx_maintenance_logs_bike_id ON maintenance_logs(user_bike_id);
    CREATE INDEX idx_maintenance_logs_serviced_at ON maintenance_logs(serviced_at DESC);

---

## 7. `sqlc` Configuration Reference
File location: `backend/sqlc.yaml`

    version: "2"
    sql:
      - engine: "postgresql"
        queries: "db/queries"
        schema: "db/migrations"
        gen:
          go:
            package: "db"
            out: "internal/repository/db"
            sql_package: "pgx/v5"
            emit_json_tags: true
            emit_empty_slices: true