# Implementation Plan: Ship Onboarding with Excel Structure Import

**Branch**: `004-ship-structure-import` | **Date**: 2026-11-11 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/004-ship-structure-import/spec.md`

## Summary

Refactor the existing ship module so a vessel can be created, listed, viewed, updated,
and soft-deleted independently of its physical structure, then onboarded through a
multi-sheet Excel workbook. The feature adds Excel template download, synchronous
workbook validation with an `importId`-scoped preview, and an all-or-nothing
confirmation that creates Decks first, then Areas and Rooms with correct ship/deck
foreign keys. It preserves the existing PostgreSQL schema conventions, UUID v7 ids,
change-capture triggers, soft deletion, and cloud-to-hub synchronization, and emits a
confirmation audit entry in the same transaction. New business endpoints are exposed
under `/api` (the runtime prefix), and the currently stale `/v1` tests and OpenAPI
metadata are reconciled as part of the work.

The only new persistence is a cloud-only `ship_structure_imports` table that records
lifecycle state, safe file metadata, counts, findings, and the normalized validated
rows needed to render a later preview and perform confirmation. It is import
orchestration data, not synchronized master data, and the actual decks/areas/rooms are
still created through the normal SQL paths only at confirmation.

## Technical Context

**Language/Version**: Go 1.26 (module `github.com/ren0503/tycoon`)

**Primary Dependencies**: chi v5 (router), pgx v5 + pgxpool (driver), sqlc (typed
queries, `pgx/v5`), Goose v3.29 (migrations), `google/uuid` (`NewV7()`), `slog`,
`go-playground/validator/v10` (DTO shape only), `testcontainers-go` + Postgres module
(integration tests), and **new**: `github.com/xuri/excelize/v2` pinned to `v2.11.1`
for OOXML parsing and template generation.

**Storage**: PostgreSQL 15+ (PostgreSQL 16 containerized in tests). Existing `ships`,
`decks`, `areas`, `rooms`, triggers, `change_log`, and `audit_log` are authoritative
and unchanged. One versioned migration adds `ship_structure_imports`.

**Testing**: `go test ./...` (`make test`, infrastructure-free unit + OpenAPI lint) and
`go test -tags=integration ./...` (`make test-integration`, testcontainers Postgres +
Goose migrations). Add unit tests for workbook validation and integration tests for
upload/preview/confirm, concurrency, rollback, audit, idempotency, template, ownership,
and change-log behavior. Existing integration tests that call the old one-argument
`ship.New` and the `/v1` contract paths must be repaired so the suite compiles and
matches the `/api` runtime.

**Target Platform**: Linux server (EC2 + Auto Scaling later); local development via
Docker Compose; no AWS account required.

**Project Type**: Web service / modular monolith (single Go binary plus separate
migrate binary).

**Performance Goals**: Synchronous upload+validation returns a complete preview within
30 seconds for at least 95% of supported workbooks; confirmation completes within
60 seconds for the reference volume (20 decks, 100 areas, 500 rooms). Upload cap is
10 MiB; cap is 5,000 non-blank data rows per required sheet.

**Constraints**: Constitutions v2.0.0 governs. No new microservice, queue, worker,
Kafka, CRDT, or distributed component. No redesign of existing tables. `.xlsx` only;
no `.xls`, CSV, encrypted, or macro-enabled workbooks. The workbook is held in memory
during validation and never stored in S3/temp files. Validation is synchronous; the
import record still survives restarts. Confirmation is one transaction and must not
bypass triggers. New endpoints use `/api`; OpenAPI/contract tests must be reconciled
from `/v1` to `/api`.

**Scale/Scope**: Reference onboarding volume 20 decks / 100 areas / 500 rooms; hard
limits 5,000 rows per sheet and 10 MiB per file. Existing ship/deck/area/room CRUD is
retained; this feature adds import operations and a creation-time `status` field.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Post-design re-check (after `research.md`, `data-model.md`, contracts, quickstart):
PASS.** All ten principles remain satisfied. No violations require `Complexity
Tracking`. The one new dependency (`excelize/v2`) and one new cloud-only table are
justified by concrete requirements (FR-011–FR-013, FR-035–FR-036, FR-047–FR-048) in
`research.md` R1/R2. There are no architecture, ownership, synchronization, or
infrastructure changes that require an ADR.

| Principle | Gate question | Status | Evidence/Notes |
|---|---|---|---|
| I. Modular Monolith | One package per module, explicit interfaces, acyclic deps, DI at composition root | PASS | Work stays in `internal/ship/`; import files are package-internal. `audit.Appender` is injected in `internal/app/modules.go`; no cross-module table access. Ship service remains the public seam. |
| II. Database Integrity | Constraints authoritative; versioned migrations; UUID v7; status text+CHECK; soft delete on synced tables | PASS | `00004` adds one table with status CHECK and count CHECKs; existing tables untouched. Decks/areas/rooms use `uuid.NewV7()` and normal inserts. Import table is not synced, so no soft-delete columns are invented. |
| III. Offline-First | Cloud writes `change_log`/`ship_seq` in the same tx via triggers; idempotency headers honored | PASS | Confirmation uses normal INSERTs so `capture_change()` fires per row (FR-051–FR-053). The import row has no capture trigger and never reaches a hub. Upload uses `Idempotency-Key` backed by a per-import key (R10). |
| IV. Data Ownership | Exactly one write owner per table; no dual writes | PASS | `ship_structure_imports` is owned by the `ship` module and mapped in `ownership.TableOwner`; all writes are cloud-side through the ship service. Business structure remains cloud-owned. |
| V. AWS Readiness | AWS behind interfaces; local dev without AWS; fail-fast config | PASS | No AWS calls added. No S3; workbook bytes stay in memory. Existing config behavior unchanged. |
| VI. Security | Tenant/ship boundaries; no sensitive leakage; audit for sensitive operations | PASS | Every endpoint resolves the ship and import under `principal.CanAccessTenant/CanAccessShip`; cross-ship/tenant imports return a non-revealing not-found. Errors use safe messages and never expose file paths, SQL, stack traces, or raw workbook content. Confirmation is audited (FR-071). |
| VII. Reliability | Idempotent writes; timeouts; graceful shutdown; migrations separate | PASS | Upload idempotency key + lifecycle; completed confirm is idempotent; concurrent confirms serialize on a row lock. No new workers. Migration remains a separate `cmd/migrate` step. Existing HTTP server timeouts apply. |
| VIII. Testability | Business logic testable with fakes; real Postgres integration; sync/parity | PASS | Validation is a pure function over decoded rows + existing-key sets. `audit.Appender` is injectable for failure tests. Integration tests cover constraints, triggers, rollback, concurrency, and parity (42 tables). |
| IX. Specification-Driven | Spec/plan/tasks; deviations update docs | PASS | This plan + `research.md` + spec assumption refinement. `tasks.md` follows via `/speckit.tasks`. Implementation must keep `docs/schema.sql`/`docs/system-architecture.md` aligned. |
| X. Simplicity | No microservices/K8s/Kafka/CRDTs; simplest structure; YAGNI | PASS | One table, one library, synchronous validation, no file store, no queue, no new module. Import is additive; existing CRUD remains. Complexity Tracking is empty. |

## Project Structure

### Documentation (this feature)

```text
specs/004-ship-structure-import/
├── plan.md              # This file
├── research.md          # Phase 0 output: decisions R1–R14
├── data-model.md        # Phase 1 output: entities, lifecycle, payload/finding schemas, DDL
├── quickstart.md        # Phase 1 output: migration, run, curl, validation guide
├── contracts/
│   ├── openapi.yaml             # Feature OpenAPI contract (ship + import + template)
│   ├── import-lifecycle.md      # State machine, idempotency, concurrency, errors
│   └── template-workbook.md     # Sheet/column layout, examples, enum/finding reference
└── tasks.md             # Phase 2 output (/speckit.tasks) — not created here
```

### Source Code (repository root)

```text
db/
├── schema.sql                              # append ship_structure_imports (sqlc input)
├── migrations/
│   ├── 00001_init.sql
│   ├── 00002_business_schema.sql
│   ├── 00003_firebase_sso_auth.sql
│   └── 00004_ship_structure_imports.sql    # NEW: table + indexes
└── queries/
    ├── ship.sql                            # add status to CreateShip
    └── ship_imports.sql                    # NEW: import CRUD, lock, counts, active key sets

docs/
└── schema.sql                              # append ship_structure_imports (authoritative baseline)

internal/
├── app/
│   └── modules.go                          # inject audit store into ship.New; /api comment
├── audit/
│   └── audit.go                            # unchanged; Appender used by ship service
├── database/sqlc/                          # regenerated models + queries (never hand-edited)
├── httpx/
│   └── upload.go                           # NEW: bounded multipart helper + MaxUploadBytes
├── httpserver/
│   ├── router.go                           # /api remains canonical; update comments
│   ├── contract_integration_test.go        # /v1 -> /api; exercise new paths
│   ├── auth_contract_integration_test.go   # /v1 -> /api
│   └── idempotency_integration_test.go     # /v1 -> /api
├── ownership/
│   ├── ownership.go                        # ship_structure_imports -> ModuleShip
│   └── ownership_test.go                   # 42-table expectation
├── ship/
│   ├── module.go                           # ShipInput.Status; constructor + Service surface
│   ├── service.go                          # status default/validation; import methods
│   ├── repo.go                             # pool + tx helpers; normal CRUD
│   ├── import_repo.go                      # NEW: import persistence/lock/active-key queries
│   ├── import_service.go                   # NEW: upload, status, preview, confirm
│   ├── import_validation.go                # NEW: pure workbook validation + findings
│   ├── template.go                         # NEW: excelize template generation
│   ├── import_handlers.go                  # NEW: multipart/status/preview/confirm/template
│   ├── handlers.go                         # ship CRUD/template route registration
│   ├── service_integration_test.go         # update constructor
│   └── import_integration_test.go          # NEW: import lifecycle/rollback/concurrency
└── testutil/
    ├── auth.go                             # existing
    ├── harness.go                          # applies migration 00004 automatically
    ├── fixtures.go                         # add helpers if needed
    └── logger.go                           # NEW: Logger(t) / audit helper for tests

openapi/
└── openapi.yaml                            # add paths/schemas; title/servers /api
```

Each module keeps its existing shape. The ship package stays flat and adds import
files beside `service.go`/`handlers.go`; there is no new business module. The import
repo obtains transaction-scoped queries with `sqlc.New(tx)` exactly like the booking
module. Cross-module effects are limited to the injected `audit.Appender`. HTTP
transport remains inside the ship package behind `RegisterRoutes`, mounted on the one
authenticated `/api` tree by `internal/httpserver`.

**Structure Decision**: Extend the existing single-binary Go layout and the existing
`ship` module. Keep `docs/schema.sql` and `db/schema.sql` in lockstep with migration
`00004`, regenerate sqlc, and add feature-specific files rather than a new package or
service. This preserves module boundaries, keeps the import workflow transactionally
close to the ship tables it writes, and avoids new infrastructure.

## Complexity Tracking

No constitution violations. The new `excelize/v2` dependency and the
`ship_structure_imports` table are directly required by FR-011–FR-013 and
FR-035–FR-048 and are justified in `research.md` R1/R2. No ADR-triggering complexity
is introduced.
