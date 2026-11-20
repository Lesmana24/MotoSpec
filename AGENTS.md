# AGENTS.md — Steering & Architectural Guardrails

## 1. Project Identity
**MotoSpec** is a motorcycle specification & maintenance tracker engineered as a backend-focused portfolio project. It demonstrates fault-tolerant LLM orchestration (primary/fallback providers, schema sanity validation, circuit breaking), multi-layer read-through caching (Redis + PostgreSQL), and an offline-capable PWA client.

## 2. Tech Stack & Hard Constraints
### Backend (`/backend`)
- **Runtime & Router:** Go 1.23+, `[github.com/gin-gonic/gin](https://github.com/gin-gonic/gin)`
- **Persistence & Cache:** PostgreSQL 16 (docker-compose host port: `5433`, container port: `5432`, `[github.com/jackc/pgx/v5](https://github.com/jackc/pgx/v5)` + `sqlc`), Redis 7 (`6379`, `[github.com/redis/go-redis/v9](https://github.com/redis/go-redis/v9)`)
- **Resilience & Validation:** `[github.com/sony/gobreaker](https://github.com/sony/gobreaker)`, `[github.com/go-playground/validator/v10](https://github.com/go-playground/validator/v10)`, `[github.com/robfig/cron/v3](https://github.com/robfig/cron/v3)`
- **Logging:** Standard library `log/slog` (structured JSON output)
- **STRICTLY FORBIDDEN:**
  - Do NOT use GORM or any reflection-based ORM. Write explicit SQL in `backend/db/queries/` and generate code via `sqlc`.
  - Do NOT use `fmt.Println` or unstructured loggers in production paths.

### Frontend PWA (`/web`)
- **Stack:** React 18+, TypeScript (`"strict": true`), Vite, Tailwind CSS
- **Offline & PWA:** `vite-plugin-pwa` (Workbox), `idb` (IndexedDB wrapper for offline garage caching)
- **STRICTLY FORBIDDEN:**
  - Do NOT compute wear percentages, dual-trigger deadlines, status thresholds, or slug normalization in TypeScript.
  - Do NOT use `any` type assertions.

## 3. Architectural Boundaries
1. **Thin Client Mandate:** All domain math, status classification (`healthy`, `warning`, `critical`, `unknown_needs_inspection`), and model slug normalization MUST execute exclusively in `backend/internal/service`.
2. **Strict Layer Isolation:**
   - `internal/handler`: HTTP routing, JSON binding, header extraction (`X-User-ID`, `X-Simulate-Outage`), and mapping domain errors to HTTP status codes. Zero business logic or direct DB/LLM calls.
   - `internal/service`: Core domain logic, read-through cache orchestration, and wear calculation.
   - `internal/llm`: `SpecGenerator` interface, provider HTTP clients, timeout wrappers, JSON sanity validator, and `gobreaker` fallback chain.
   - `internal/repository`: PostgreSQL (`sqlc`-generated queries) and Redis cache operations.
   - `internal/worker`: Scheduled background cron jobs.
3. **Zero-Friction Auth:** Do NOT implement JWT/OAuth login flows. Scope user data via the `X-User-ID` request header (defaulting to `"demo-user"` when absent).

## 4. Repository Map

    ├── AGENTS.md                  # Global rules (always active)
    ├── .agent/workflows/          # Execution & verification SOPs
    ├── docs/
    │   ├── PRD.md                 # Feature flows, wear formulas, edge cases
    │   ├── TECH_SPEC.md           # Cache pipeline, LLM resilience, PostgreSQL DDL
    │   ├── API_CONTRACT.md        # REST endpoints & JSON response envelopes
    │   └── TASKS.md               # Phased implementation checklist
    ├── backend/
    │   ├── cmd/api/main.go
    │   ├── db/{migrations,queries}/
    │   └── internal/{handler,service,llm,repository,worker}/
    └── web/
        └── src/{api,components,hooks,pages,storage}/

## 5. Coding & Verification Protocol
- **Error Handling:** Wrap internal errors with package context (`fmt.Errorf("service.GetBikeSpec: %w", err)`). Translate sentinel errors (`ErrNotFound`, `ErrInvalidOdometer`, `ErrAllProvidersFailed`) to HTTP responses solely in `internal/handler`.
- **On-Demand Specs:** Read `docs/TECH_SPEC.md` before modifying database schemas or `internal/llm`, and read `docs/API_CONTRACT.md` before changing JSON payloads or frontend API clients.
- **Mandatory Verification:** After creating or editing code in `internal/service` or `internal/llm`, you MUST write/update table-driven unit tests with mocks and attempt `go test -race -v ./internal/...` first. If the local environment lacks a C compiler (GCC/CGO) on Windows (`go: -race requires cgo`), you MUST explicitly document this platform limitation in the verification summary and execute the fallback `go test -v ./internal/...`. Full `-race` testing remains strictly enforced on Linux runners via GitHub Actions CI (`.github/workflows/ci.yml`).