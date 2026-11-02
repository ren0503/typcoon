# Implementation Plan: Initialize Business Modules and Database Models

**Branch**: `002-business-modules` | **Date**: 2026-11-02 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/002-business-modules/spec.md`

## Summary

Implement the business-layer foundation of the Yacht Management System backend as a Go modular monolith. Assign every one of the 41 tables in the authoritative PostgreSQL schema (`docs/schema.sql`) to exactly one of 12 modules (`tenant`, `ship`, `voyage`, `staff`, `partner`, `booking`, `guest`, `auth`, `sync`, `audit`, `ai`, `feedback`), deliver the schema as a versioned Goose migration plus sqlc-generated models (schema fidelity, no edits), and expose a narrow `/v1` REST surface with cursor pagination, a consistent error envelope, `Idempotency-Key` handling only where the schema backs it, default-deny auth with a safe dev principal provider, and transactional audit emission for `guest`/`booking`/`staff` writes. Repositories live behind hand-written interfaces so a SQLite twin can be introduced later without changing services.

## Technical Context

**Language/Version**: Go 1.26 (module `github.com/ren0503/tycoon`)

**Primary Dependencies**: chi v5 (router), pgx v5 (driver), sqlc v1.30 (`pgx/v5` package, `db/queries` + `db/schema.sql` inputs), Goose v3.29 (migrations, Provider API + `embed.FS`), `github.com/google/uuid` (promote to direct dep; `NewV7()`), `shopspring/decimal` + `jackc/pgx-shopspring-decimal` (numeric overridate; registered via `AfterConnect`), `jackc/pgerrcode` (SQLSTATE constants), `go-playground/validator/v10` (DTO shape validation only), viper (config), testcontainers-go + Postgres module (integration tests).

**Storage**: PostgreSQL 15+ (PostgreSQL 16 containerized for tests), per `db/schema.sql` baseline derived verbatim from `docs/schema.sql`.

**Testing**: `go test ./...` (unit) and `go test -tags=integration ./...` (testcontainers Postgres + Goose Provider migrations); same commands locally and in CI.

**Target Platform**: Linux server (EC2 + Auto Scaling later); local development uses Docker/Docker Compose; no AWS account required.

**Project Type**: Web service (REST API, modular monolith, single binary + separate migrate binary).

**Performance Goals**: Not a goal this phase. Bounded payloads only: cursor pagination default 20 / max 100 (FR-039).

**Constraints**: Constitution v2.0.0 governs (see Constitution Check). Sensitive fields stored only as encrypted bytes; refresh/PIN/token hashes never returned; default-deny `/v1`; no advanced workflows (FR-036); no schema inventions; no new distributed components.

**Scale/Scope**: 41 tables → 12 modules; 7 modules expose endpoints; ownership map already fixed in the spec; integration coverage for constraints/soft-delete/pagination/transactions/boundaries.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Post-design re-check (after data-model/contracts/quickstart): PASS — all ten principles unchanged; no violations, no Complexity Tracking entries.** The design aligns with the gates above (module seams per data-model cross-module graph; audit emission same-tx per FR-057; no new distributed components; only three new pinned dependencies justified in research.md R1/R3/R8).

| Principle | Gate question | Status | Evidence/Notes |
|---|---|---|---|
| I. Modular Monolith | One package per module, explicit public interfaces, acyclic deps, DI at composition root | PASS | Structure below: `internal/<module>/` with a small public interface file; `internal/app` composes; depguard + boundary test enforce FR-006/007 |
| II. Database Integrity | Schema delivered verbatim as migration; UUID v7 app-side; status text+CHECK; soft delete only; exclusion constraints authoritative | PASS | Migration 00002 contains `docs/schema.sql` content unchanged; `uuid.NewV7()` before insert; DB remains final authority (FR-027/032) |
| III. Offline-First | Cloud writes `change_log`+`ship_seq` in the same tx via DB triggers; hub-initiated only; idempotency headers honored | PASS | Triggers shipped unmodified; app never writes `change_log` directly; writes accept `Idempotency-Key` where schema-backed (FR-042) |
| IV. Data Ownership | Exactly one write owner per table; `ownership` respected; no dual writes | PASS | Ownership map (spec) is implemented as module ownership; `service_bookings`/`bookings` status flows kept to basic ops; sync tables read-only or system-managed |
| V. AWS Readiness | AWS behind interfaces; local dev without AWS; fail-fast config | PASS | No AWS calls added this phase; guest encryption behind an interface with a non-KMS local implementation |
| VI. Security | mTLS/TLS out of scope here; tokens/hashes never exposed; envelope-encrypted bytes opaque; tenant/ship scope enforced | PASS | FR-043/046-048 map to middleware + DTO whitelisting; dev principal provider cannot activate in production (config fail-fast) |
| VII. Reliability | Idempotent jobs; timeouts; graceful shutdown; migrations as a separate step | PASS | Reuses foundation; single-request transactions only; no new workers; `cmd/migrate` already separate |
| VIII. Testability | Business logic testable with fakes; integration tests on real Postgres; sync/parity tests | PASS | Repo interfaces enable fakes; boundary test + sqlite-parity contract tests planned within this feature's test suite |
| IX. Specification-Driven | Spec/plan/tasks exist; deviations update docs | PASS | This plan + spec clarifications; tasks.md to follow via `/speckit.tasks` |
| X. Simplicity | No microservices/K8s/Kafka/CRDTs; simplest structure; YAGNI | PASS | Plain packages + DI; no new infra components; unbounded dependencies rejected (only decimal, pgx-decimal, pgerrcode, promoted uuid) |

## Project Structure

### Documentation (this feature)

```text
specs/002-business-modules/
├── plan.md              # This file
├── research.md          # Phase 0 output: key decisions
├── data-model.md        # Phase 1 output: entity/module model
├── quickstart.md        # Phase 1 output: validation guide
├── contracts/
│   ├── openapi.yaml          # /v1 REST contract (resource inventory + shared components)
│   └── module-interfaces.md  # Public Go interfaces per module (cross-module contract)
└── tasks.md             # Phase 2 output (/speckit.tasks) — not created here
```

### Source Code (repository root)

```text
db/
├── schema.sql               # becomes the full 41-table sqlc input (currently empty placeholder)
├── migrations/
│   └── 00002_business_schema.sql   # verbatim authoritative schema as versioned migration
└── queries/
    ├── <module>/            # one .sql file set per owning module (sqlc named queries)
    └── health.sql           # existing

internal/
├── app/app.go               # composition root: builds every module, injects deps (FR-008)
├── apperrors/               # consistent error codes/envelope reuse (exists)
├── config/                  # auth.mode=dev|jwt, dev principal allowlist, fail-fast validation
├── database/                # pool, migrate harness (Goose Provider), sqlc output pkg, pg error mapper
│   └── sqlc/                # generated models + queries (never hand-edited)
├── auth/                    # module principal provider interface; dev provider guarded by config
├── tenant/                  # module: public api.go + service + repo + HTTP handlers (RegisterRoutes)
├── ship/                    # modules ship/decks/areas/rooms
├── voyage/                  # ports, voyages, port_calls
├── staff/                   # staff, staff_leaves, staff_credentials, shift_templates,
│                            # coverage_requirements, labor_rules, autonomy_policies
├── partner/                 # partners, contracts, staff, services, slots
├── booking/                 # bookings, service_bookings
├── guest/                   # guests, boarding_passes
├── audit/                   # audit_log append/query (owns audit; emits on behalf via interface)
├── sync/                    # ship_seq, change_log, hub_devices, ship_devices, revocations,
│                            # snapshots, job_runs, ops_events (internal ops only)
├── ai/                      # ai_params metadata CRUD only
├── feedback/                # feedback_responses, feedback_insights (internal)
└── httpserver/              # router, middleware (auth principal, request-id, recover, CORS)
```

Each module package follows the same shape: `module.go` (public interface + factory), `service.go` (business validation), `repo.go` (transactional data access over generated queries), `handlers.go` (chi routes for its exposed resources, mounted under `/v1`), plus module-scoped test fakes. Cross-module use is interface-only (`partner` calls `ship` interface to validate ship scope, `booking` calls `partner`/`voyage`/`ship` interfaces, `guest` calls `booking` and `auth` interfaces, `audit` emits through the `audit` interface).

**Structure Decision**: Extend the existing single-binary Go layout. Ownership map from the spec drives package boundaries; composition stays in `internal/app`; HTTP transport lives inside the owning module package behind a `RegisterRoutes(chi.Router)` seam so route ownership matches table ownership while `internal/httpserver` keeps one an authenticated `/v1` tree.

## Complexity Tracking

No constitution violations. New dependencies (`shopspring/decimal`, `pgx-shopspring-decimal`, `pgerrcode`) are justified in `research.md`; no ADR-triggering complexity introduced.
