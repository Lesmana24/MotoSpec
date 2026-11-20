# Product Requirements Document (PRD) — MotoSpec

## 1. Overview
**MotoSpec** is a motorcycle maintenance tracking and specification discovery application. Instead of maintaining a static, manually curated database of every motorcycle model, the system uses an LLM orchestrator with automatic fallback to seed structured component maintenance schedules on demand, caching results in PostgreSQL and Redis for sub-20ms subsequent lookups.

## 2. Core User Flows
1. **Self-Populating Spec Search:**
   - User enters a free-text motorcycle name (e.g., `"vario 150"`, `"Yamaha NMAX 155cc"`).
   - Backend normalizes the query into a canonical slug (`honda-vario-150`, `yamaha-nmax-155`), checks Redis and PostgreSQL, and triggers the LLM fallback pipeline only on a cache miss.
   - User receives the motorcycle profile along with its standardized list of wear-and-tear components (`engine_oil`, `gear_oil`, `v_belt`, `brake_pads_front`, `spark_plug`, `air_filter`, `coolant`, etc.).
2. **Garage Registration (Onboarding):**
   - User adds a motorcycle to their personal garage by selecting a spec model, assigning a nickname, inputting `current_odometer_km`, and optionally enabling `auto_increment_enabled` with a `daily_estimate_km` value.
   - User may optionally provide `last_replaced_km` and/or `last_replaced_at` for components they already know; omitted components default to an uninitialized baseline.
3. **Component Health Dashboard:**
   - User inspects all tracked components for their motorcycle, sorted by urgency (`critical` $\rightarrow$ `warning` $\rightarrow$ `unknown_needs_inspection` $\rightarrow$ `healthy`).
   - Each component displays remaining kilometers, next due odometer, wear percentage, and status badge.
4. **Odometer Quick-Update & Daily Auto-Increment:**
   - User manually updates `current_odometer_km` at any time.
   - If `auto_increment_enabled` is `true`, a daily background worker automatically increments `current_odometer_km` by `daily_estimate_km` so users do not abandon the app due to manual entry fatigue.
5. **Component Maintenance Logging (Batch Supported):**
   - After servicing their motorcycle, the user selects one or more replaced components, inputs the service odometer (`odometer_km`), service timestamp, optional cost (`cost_idr`), and notes.
   - Logging resets the baseline (`last_replaced_km` and `last_replaced_at`) for every serviced component and appends an immutable record to `maintenance_logs`.
6. **Portfolio Resilience Simulator:**
   - Reviewers can toggle a **"Simulate Primary LLM Outage"** switch in the PWA telemetry panel, which attaches the `X-Simulate-Outage: primary` header to search requests so the fallback provider and latency metrics can be observed live.

## 3. Domain Formulas & Status Thresholds
All calculations below MUST execute exclusively in `backend/internal/service`.

### 3.1 Wear Calculation (Dual-Trigger: Distance OR Time)
- **Distance Wear (%):**
  $$\text{Wear}_{km} = \left( \frac{\text{current\_odometer\_km} - \text{last\_replaced\_km}}{\text{interval\_km}} \right) \times 100$$
- **Time Wear (%):** *(Evaluated only if `interval_months > 0` AND `last_replaced_at IS NOT NULL`)*
  $$\text{Wear}_{time} = \left( \frac{\text{days\_since\_replacement}}{\text{interval\_months} \times 30} \right) \times 100$$
- **Effective Wear (%):**
  $$\text{EffectiveWear} = \max(\text{Wear}_{km}, \text{Wear}_{time})$$
  - *Rule:* `EffectiveWear` is **not capped at 100%** (e.g., `125.0%` indicates the part is 25% past its recommended lifespan). Minimum value is `0.0%`.
- **Due & Remaining Distance:**
  - $\text{NextDueKM} = \text{last\_replaced\_km} + \text{interval\_km}$
  - $\text{RemainingKM} = \text{NextDueKM} - \text{current\_odometer\_km}$ *(Negative values indicate overdue kilometers).*

### 3.2 Component Status Classification
Evaluated in exact priority order:

| Priority | Status Code | Condition | UI Meaning |
| :--- | :--- | :--- | :--- |
| 1 | `unknown_needs_inspection` | `last_replaced_km IS NULL` AND `last_replaced_at IS NULL` | Cold start / replacement history unknown |
| 2 | `critical` | `EffectiveWear >= 100.0` | Overdue / immediate replacement required |
| 3 | `warning` | `75.0 <= EffectiveWear < 100.0` | Approaching limit / prepare budget |
| 4 | `healthy` | `EffectiveWear < 75.0` | Safe operating condition |

## 4. Edge Cases & Business Rules
1. **Cold Start (Used Motorcycle Onboarding):**
   - When a user registers a motorcycle with `current_odometer_km = 24000` and skips entering component history, `last_replaced_km` and `last_replaced_at` MUST remain `NULL`.
   - Do NOT default `last_replaced_km` to `0` (which falsely flags all components as severely overdue) or to `24000` (which falsely assumes every part is brand new). The status MUST resolve to `unknown_needs_inspection` until the user logs a replacement or baseline check.
2. **Odometer Monotonicity (No Rollback):**
   - Any manual odometer update where `new_odometer_km < current_odometer_km` MUST be rejected with HTTP `400 Bad Request` (`INVALID_ODOMETER`).
3. **Maintenance Logged Above Current Odometer:**
   - If a user logs a maintenance event at `odometer_km = 19500` while `user_bikes.current_odometer_km` is `19000`, the service MUST atomically update `user_bikes.current_odometer_km` to `19500` within the same database transaction.
4. **Partial Baseline (Date-Only or KM-Only):**
   - If `last_replaced_km` is known (`!= NULL`) while `last_replaced_at` is `NULL`, compute `EffectiveWear` solely from $\text{Wear}_{km}$.
   - If `last_replaced_at` is known (`!= NULL`) while `last_replaced_km` is `NULL` (and `interval_months > 0`), compute `EffectiveWear` solely from $\text{Wear}_{time}$, leaving `next_due_km` and `remaining_km` as `null`.
5. **Query & Canonical Slug Normalization:**
   - Trim leading/trailing spaces, convert to lowercase, strip standalone unit suffixes (`cc`), collapse multiple spaces/hyphens into a single hyphen, and map common Indonesian model aliases to standard brand prefixes (e.g., `"vario 150"` $\rightarrow$ `"honda-vario-150"`, `"nmax 155"` $\rightarrow$ `"yamaha-nmax-155"`, `"beat fi"` $\rightarrow$ `"honda-beat-fi"`).
6. **Offline PWA Sync Conflict:**
   - When the PWA reconnects after offline usage in a workshop, queued odometer updates or maintenance logs are replayed sequentially to the backend. If a queued odometer value is stale relative to a newer server state, the server enforces Rule #2 (`INVALID_ODOMETER`) and the client refreshes its local IndexedDB cache from the latest `GET /api/v1/garage/bikes/:id` response.