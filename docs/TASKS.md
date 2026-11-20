# Phased Implementation Plan — MotoSpec

Execute one phase per agent session to maintain high code quality and prevent context dilution. Always run `go test -race ./...` after completing backend phases.

---

## Phase 1: Infrastructure, Database & Repository Layer
- [x] Create root `docker-compose.yml` running PostgreSQL 16 (`5432`) and Redis 7 (`6379`) with healthchecks and named volumes.
- [x] Initialize Go module in `/backend` (`go mod init github.com/username/motospec/backend`) and install core dependencies (`gin`, `pgx/v5`, `go-redis/v9`, `gobreaker`, `validator/v10`, `cron/v3`, `x/sync/singleflight`).
- [x] Create database migration `backend/db/migrations/000001_init.up.sql` matching the exact DDL in `docs/TECH_SPEC.md`.
- [x] Configure `backend/sqlc.yaml` and write typed SQL queries in `backend/db/queries/` (`specs.sql`, `garage.sql`, `maintenance.sql`), then generate Go code via `sqlc generate`.
- [x] Implement Redis read-through cache wrapper (`GetSpec`, `SetSpec`) in `backend/internal/repository/cache.go`.
- [x] Implement thread-safe `sync/atomic` telemetry store in `backend/internal/repository/telemetry.go` (`cache_hits`, `cache_misses`, `llm_primary_success`, `llm_fallback_success`, `llm_total_failures`).

## Phase 2: LLM Orchestrator & Fault-Tolerance (`internal/llm`)
- [x] Define `SpecGenerator` interface, `GeneratedBikeSpec` structs with `validator` tags, and context outage-simulation helpers (`WithSimulatedPrimaryOutage`) in `backend/internal/llm/types.go`.
- [x] Implement JSON sanitizer & domain sanity validator in `backend/internal/llm/validator.go` (strip markdown fences, unmarshal JSON, run struct validation, ensure `engine_oil` presence and unique component codes).
- [x] Implement Primary LLM provider adapter (e.g., Gemini 2.5 Flash REST client with `4s` context timeout and strict JSON prompt) in `backend/internal/llm/primary_provider.go`.
- [x] Implement Fallback LLM provider adapter (e.g., Groq Llama 3.3 70B / OpenAI-compatible client with `4s` context timeout) in `backend/internal/llm/fallback_provider.go`.
- [x] Implement `Orchestrator` in `backend/internal/llm/orchestrator.go` wrapping Primary with `sony/gobreaker` (trip after 3 consecutive failures, 60s open timeout) and automatic fallback routing.
- [x] Write table-driven unit tests in `backend/internal/llm/orchestrator_test.go` covering:
  - Happy path primary generation.
  - Fallback triggered by primary HTTP error or timeout.
  - Fallback triggered by malformed JSON or sanity check failure (e.g., `interval_km < 500` or missing `engine_oil`).
  - Fallback triggered by `X-Simulate-Outage` context flag.
  - Circuit breaker tripping to `Open` after 3 consecutive failures and bypassing primary.

## Phase 3: Domain Services, REST API & Cron Worker
- [x] Implement deterministic query normalizer and Indonesian brand-prefix mapper in `backend/internal/service/normalizer.go` + unit tests (`normalizer_test.go`).
- [x] Implement `SpecService` in `backend/internal/service/spec_service.go` orchestrating `Redis -> PostgreSQL -> singleflight -> LLM Orchestrator -> Persist & Backfill` with latency tracking.
- [x] Implement `GarageService` in `backend/internal/service/garage_service.go` covering:
  - Dual-trigger wear calculation (`max(Wear_km, Wear_time)` uncapped above 100%).
  - Priority status classification (`unknown_needs_inspection`, `critical`, `warning`, `healthy`).
  - Cold-start onboarding (`NULL` baselines), odometer monotonicity enforcement (`ErrInvalidOdometer`), and transactional batch maintenance logging.
- [x] Write table-driven unit tests in `backend/internal/service/garage_service_test.go` verifying all wear formulas and edge cases.
- [x] Implement Gin HTTP handlers, `X-User-ID` / `X-Simulate-Outage` middleware, error-to-HTTP mapper, and graceful shutdown in `backend/internal/handler/` and `backend/cmd/api/main.go` matching `docs/API_CONTRACT.md`.
- [x] Implement daily odometer auto-increment cron worker (`0 0 * * *`) in `backend/internal/worker/odometer_worker.go`.

## Phase 4: Offline-First PWA & Telemetry Inspector (`/web`)
- [ ] Scaffold React + TypeScript (`strict: true`) + Vite + Tailwind CSS in `/web`.
- [ ] Configure `vite-plugin-pwa` (web manifest, app icons, Workbox runtime caching) and IndexedDB wrapper (`idb`) in `web/src/storage/offlineGarage.ts`.
- [ ] Implement typed REST API client in `web/src/api/client.ts` strictly matching `docs/API_CONTRACT.md` (injecting `X-User-ID` and `X-Simulate-Outage` headers).
- [ ] Build mobile-first PWA views:
  - **Spec Search & Seeder:** Search bar with instant model seeding and component interval preview.
  - **Garage Dashboard:** Motorcycle selector, odometer summary, and component cards sorted by urgency with color-coded wear progress bars.
  - **Odometer Quick-Update Modal:** Fast mileage input and daily auto-increment toggle.
  - **Batch Maintenance Logger:** Multi-select component replacement form with cost and notes.
- [ ] Build **Portfolio Telemetry Drawer / Inspector** displaying live `meta` telemetry (`source`, `provider_used`, `latency_ms`), cache hit ratio, circuit breaker status badges, and the **"Simulate Primary LLM Outage"** toggle switch.

## Phase 5: Portfolio Packaging & Final Verification
- [ ] Create `backend/.env.example` and `web/.env.example` documenting all required environment variables.
- [ ] Write root `README.md` featuring a system architecture diagram, database ERD, API latency comparison table (Redis Hit vs. Postgres Hit vs. LLM Cold Seed), and single-command local setup instructions.
- [ ] Run end-to-end verification across all endpoints, offline PWA reload, and simulated LLM outage failover.