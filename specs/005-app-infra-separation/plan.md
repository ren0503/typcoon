# Implementation Plan: Separate Business Application from Infrastructure Layers

**Branch**: `005-app-infra-separation` | **Date**: 2026-11-12 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/005-app-infra-separation/spec.md`

## Summary

Refactor the existing Go modular monolith into five explicit layers without changing
externally observable behavior: business modules move to `internal/application`, concrete
PostgreSQL/AWS/Redis/Firebase/local-security implementations move to
`internal/infrastructure`, HTTP routing/handlers/middleware move to `internal/transport/http`,
generic utilities move to `internal/platform`, and runtime assembly stays in `internal/app`.
No schema, migration, trigger, route, response envelope, auth, audit, idempotency, or sync
semantics change.

The key mechanism is an application-owned persistence port per module. Application code
declares the row-shaped types, query parameter types, repository methods, and — where a
use case spans multiple writes — an application-owned `Tx` interface with module operations,
`AppendAudit`, `Commit`, and `Rollback`. The PostgreSQL adapter implements those interfaces
with pgx/sqlc, maps `pgtype`/`decimal` values to standard Go values, maps database errors,
and appends audit rows through the same `pgx.Tx`. Application services keep deciding when
to begin, commit, roll back, and audit; they never see `pgx.Tx`, `pgconn.CommandTag`,
`pgtype`, `decimal.NullDecimal`, or `sqlc` types.

Existing concrete implementations are retained and relocated rather than replaced: pgx,
sqlc, Goose, chi, AWS SDK configuration, KMS signing, Redis rate limiting, Firebase
verification, local signing, and the local guest document cipher. No SQLite, SNS, SQS, S3,
Redshift, EventBridge, worker, or hub code is created because none exists today.

## Technical Context

**Language/Version**: Go 1.26 (module `github.com/ren0503/tycoon`)

**Primary Dependencies**: chi v5, pgx v5 + pgxpool, sqlc-generated `pgx/v5` queries, Goose
v3.29, `google/uuid`, `slog`, `go-playground/validator/v10` (transport DTOs only),
`samber/lo`, `golang-jwt/jwt/v5`, `shopspring/decimal` (adapter conversion only),
AWS SDK for Go v2 (configuration and KMS), `redis/go-redis/v9`, Firebase Admin SDK,
testcontainers-go + PostgreSQL module for integration tests.

**Storage**: PostgreSQL 15+ remains the only implemented store. `docs/schema.sql` and
`db/migrations/` remain authoritative; this feature adds no tables, columns, indexes,
triggers, queries, or migrations. The SQLite hub store exists only as future architecture
direction and is not implemented here.

**Testing**: `go test ./...` for infrastructure-free unit, route-inventory, OpenAPI, and
signature tests; `go test -tags=integration ./...` with testcontainers PostgreSQL for
repositories, transactions, constraints, migrations, auth rotation, audit, and sync. The
architecture-boundary test uses `go list -json ./...` and must prove layer direction,
forbidden application imports, and acyclicity.

**Target Platform**: Linux server (EC2 + Auto Scaling later); local development via Docker
Compose; no AWS account required for unit tests or local signing mode.

**Project Type**: Web service / modular monolith with two independently initialized
runtimes: API (`cmd/server`) and migration (`cmd/migrate`).

**Performance Goals**: No additional network or database round trips for business
operations. The refactor adds only interface dispatch and value mapping; p95 API latency,
transaction size, and query counts must remain equivalent. Startup and migration time must
not regress materially.

**Constraints**: Constitution v2.0.0 governs. Preserve all behavior required by FR-029
through FR-038. Do not introduce a DI framework, generic database abstraction, Microservice,
Kafka, CRDT, SQLite placeholder, empty infrastructure package, or placeholder worker/hub.
Application packages must not import transport, infrastructure, `net/http`, chi, database
drivers, sqlc, AWS SDK, Redis, Firebase, or testcontainers. Tests may change imports,
package locations, and construction only.

**Scale/Scope**: Move/split 12 business modules, 42 table row types, 12 repository/query
surfaces, 14 HTTP handler/route surfaces, auth/signing/rate-limit/identity adapters, audit
persistence, database lifecycle/migrations/error mapping, health checking, configuration,
pagination, HTTP helpers, and test support. The delivered package graph must contain no
cycles and no application-to-infrastructure edges.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Post-design re-check (after `research.md`, `data-model.md`, contracts, quickstart):
PASS.** All ten principles remain satisfied. No constitution violations require
`Complexity Tracking`. This is an architecture-boundary change, so an ADR is required by
Principle IX and FR-045; the ADR is a delivery artifact of implementation.

| Principle | Gate question | Status | Evidence/Notes |
|---|---|---|---|
| I. Modular Monolith | One package per business module, explicit interfaces, acyclic graph, DI at composition root | PASS | All 12 modules move intact to `internal/application/<module>`. Cross-module consumers declare minimal local interfaces; consumers may import only public value types such as `auth.Principal`. No module imports another module's persistence/transport. `internal/app` is the only both-layers importer. |
| II. Database Integrity | Constraints authoritative; versioned migrations; UUID v7; status text+CHECK; soft delete on synced tables | PASS | No schema, migration, query, or trigger change. Adapters continue to use existing SQL and `uuid.NewV7()`. `database.MapPgError` moves to the PostgreSQL adapter; constraints remain the final authority. |
| III. Offline-First | `change_log`/`ship_seq` written in the same transaction; idempotency; snapshots/deltas unchanged | PASS | The app-owned `Tx` callback preserves exactly one PostgreSQL transaction for business writes and audit. Triggers remain untouched. Idempotency keys, change-log sequencing, and sync services are moved, not redesigned. |
| IV. Data Ownership | Exactly one write owner per table; no dual writes | PASS | `ownership.TableOwner` and all mappings remain unchanged under `application/ownership`. Application services retain write-policy decisions; only persistence mechanics move to adapters. |
| V. AWS Readiness | AWS behind project-owned interfaces; local dev without AWS; fail-fast config | PASS | AWS SDK remains only in `infrastructure/aws` and `infrastructure/security/kms`. Auth depends on `Signer`/`KeySet` contracts. Local signing/cipher and dev principal implementations move outside application packages. Configuration behavior is unchanged. |
| VI. Security | Tenant/ship boundaries; encryption; token rotation; audit | PASS | Authorization checks stay in application services. KMS/Redis/Firebase/local security adapters move behind application-owned interfaces. Guest document cipher rule stays in `application/guest`; implementation moves to infrastructure. Audit remains same-transaction. |
| VII. Reliability | Idempotent jobs, timeouts, graceful shutdown, separate migrations | PASS | HTTP timeouts and graceful shutdown are unchanged. API and migration runtimes get separate composition paths; migration is still a separate command and never runs at startup. No new async work is introduced. |
| VIII. Testability | Business logic testable with fakes; real PostgreSQL integration | PASS | Application services depend only on consumer-owned ports, so unit fakes remain possible. Existing integration tests continue against real PostgreSQL through relocated test support. The boundary test is executable and mutation-tested. |
| IX. Specification-Driven | Spec/plan/tasks; ADR for architecture; docs updated | PASS | This plan and Phase 0/1 artifacts trace to the spec. Implementation adds an ADR and updates `README.md` and `docs/system-architecture.md`. |
| X. Simplicity | Simplest structure; no microservices/K8s/Kafka/CRDTs; YAGNI | PASS | Moves existing code; adds no technology. Interfaces exist only where consumption occurs. No empty packages or generic database abstraction. `Complexity Tracking` is empty. |

## Project Structure

### Documentation (this feature)

```text
specs/005-app-infra-separation/
├── plan.md                       # This file
├── research.md                   # Phase 0 output: decisions R1–R14
├── data-model.md                 # Phase 1 output: application row types, mappings, transaction model
├── quickstart.md                 # Phase 1 output: refactor validation and runtime checks
├── contracts/
│   ├── layer-boundaries.md       # Dependency matrix, allowed imports, architecture-test rules
│   └── http-surface.md           # Preserved API surface and transport ownership
└── tasks.md                      # Phase 2 output (/speckit.tasks) — not created here
```

### Source Code (repository root)

```text
cmd/
├── server/
│   └── main.go                   # thin: config/logger/signal -> app.RunAPI
└── migrate/
    └── main.go                   # thin: config/logger/signal -> app.RunMigrations

internal/
├── application/                  # business modules and use cases only
│   ├── ai/
│   ├── audit/                    # Entry, validation policy, app-owned writer contract
│   ├── auth/                     # service, Principal, tokens, store/verifier/limiter/signer ports
│   ├── booking/
│   ├── feedback/
│   ├── guest/                    # includes DocumentCipher contract and encryption rule
│   ├── ownership/
│   ├── partner/
│   ├── ship/
│   ├── staff/
│   ├── sync/
│   ├── tenant/
│   └── voyage/
├── infrastructure/               # concrete adapters only
│   ├── aws/                      # AWS SDK configuration loader
│   ├── cache/redis/              # auth rate limiter adapter
│   ├── health/                   # database connectivity monitor
│   ├── identity/firebase/        # Firebase token verifier
│   ├── observability/logging/    # slog construction
│   ├── persistence/postgres/
│   │   ├── ai/
│   │   ├── audit/
│   │   ├── auth/
│   │   ├── booking/
│   │   ├── feedback/
│   │   ├── guest/
│   │   ├── partner/
│   │   ├── ship/
│   │   ├── staff/
│   │   ├── sync/
│   │   ├── tenant/
│   │   ├── voyage/
│   │   ├── errors.go             # pgx/pgconn -> platform app errors
│   │   ├── migrate.go            # Goose runner
│   │   ├── pool.go               # pgxpool lifecycle/DSN
│   │   └── sqlc/                 # generated code, unchanged except package location
│   └── security/
│       ├── kms/                  # KMS signer
│       └── local/                # local signer + local document cipher
├── transport/
│   └── http/
│       ├── handlers/             # only modules with an existing HTTP surface
│       │   ├── auth/
│       │   ├── booking/
│       │   ├── guest/
│       │   ├── health/           # /health and /ready
│       │   ├── partner/
│       │   ├── ship/
│       │   ├── staff/
│       │   ├── tenant/
│       │   └── voyage/           # ai/feedback/sync expose no routes; no dirs created
│       ├── httpx/                # JSON/upload helpers
│       ├── middleware/           # auth, CORS, recovery, request logging
│       ├── router.go
│       └── server.go
├── platform/
│   ├── config/
│   ├── errors/                   # package name apperrors retained for symbol stability
│   ├── pagination/
│   └── testutil/                 # test-only; may import infrastructure
└── app/                          # composition root only
    ├── api.go                    # API runtime assembly
    ├── app.go                    # lifecycle/shutdown
    ├── migrate.go                # migration runtime assembly
    └── modules.go                # application services from injected adapters

db/                               # unchanged authoritative migrations and queries
docs/                             # schema/architecture documentation to update
openapi/                          # unchanged API contract
```

**Structure Decision**: Use one Go module with explicit layer directories. The application
tree contains only business modules, row-shaped types, use-case services, audit policy, and
consumer-owned interfaces. Infrastructure adapters are grouped by technology while PostgreSQL
repositories are grouped by owning module to keep each module's SQL translation local.
Transport is grouped by protocol and business route surface. `internal/app` is the only
package allowed to import application and infrastructure together. No `worker`, `hub`,
SQLite, SNS, SQS, S3, Redshift, EventBridge, certificate, or metrics package is created.

### Package Migration Map

| Current path | Target path | Notes |
|---|---|---|
| `internal/{tenant,ship,voyage,staff,partner,booking,guest,sync,ai,feedback,auth,audit}` | `internal/application/<module>` | Services, rules, row types, ports; handlers/repos removed or split |
| `internal/ownership` | `internal/application/ownership` | Table-to-module map unchanged |
| `internal/database` (pool/DSN/migrations/tests) | `internal/infrastructure/persistence/postgres` | Goose remains authoritative |
| `internal/database/pgerrors.go` | `internal/infrastructure/persistence/postgres/errors.go` | Same mappings; adapter-owned |
| `internal/database/sqlc` | `internal/infrastructure/persistence/postgres/sqlc` | Generated code regenerated at new import path |
| `internal/audit` | `application/audit` + `infrastructure/persistence/postgres/audit` | Policy/Entry vs SQL persistence |
| `internal/auth/store.go`, `accounts.go` | `application/auth` ports + `infrastructure/persistence/postgres/auth` | Application types, adapter transactions |
| `internal/auth/firebase.go` | `infrastructure/identity/firebase` | Implements application `Verifier` |
| `internal/auth/signing.go` (Local*) | `infrastructure/security/local` | Implements application `Signer`/`KeySet` |
| `internal/auth/signing.go` (KMS*) | `infrastructure/security/kms` | Implements application `Signer` |
| `internal/auth/ratelimit.go` | `application/auth` interface + `infrastructure/cache/redis` | Redis adapter only |
| `internal/auth/provider.go` | `transport/http/middleware` | HTTP request parsing; `Principal` stays in app/auth |
| `internal/guest/module.go` local cipher | `infrastructure/security/local` | `DocumentCipher` contract stays in app/guest |
| `internal/apperrors` | `internal/platform/errors` | Package name `apperrors` retained |
| `internal/config` | `internal/platform/config` | No behavior change |
| `internal/pagination` | `internal/platform/pagination` | No behavior change |
| `internal/httpx` | `internal/transport/http/httpx` | HTTP-only helpers |
| `internal/httpserver/router.go`, `server.go` | `internal/transport/http/` | Transport |
| `internal/httpserver/handlers` | `internal/transport/http/handlers/health` | Technical endpoints |
| `internal/httpserver/middleware` | `internal/transport/http/middleware` | Transport |
| `internal/health` | `internal/infrastructure/health` | Database probe |
| `internal/logging` | `internal/infrastructure/observability/logging` | Concrete slog setup |
| `internal/awsconfig` | `internal/infrastructure/aws` | AWS SDK configuration only |
| `internal/testutil` | `internal/platform/testutil` | Test-only infrastructure exception |
| `internal/app` | `internal/app` | Composition root; command construction moves here |

### Application Persistence and Transaction Mechanism

Every module owns its row-shaped data types and persistence contract. The refactor replaces
`sqlc` row types and query parameter structs with module-local equivalents, for example:

```go
// application/staff
type Staff struct { /* same fields as sqlc.Staff, standard Go/pure UUID types */ }
type CreateStaffParams struct { /* same fields as sqlc.CreateStaffParams */ }

type Tx interface {
    CreateStaff(context.Context, CreateStaffParams) (Staff, error)
    UpdateStaff(context.Context, UpdateStaffParams) (Staff, error)
    SoftDeleteStaff(context.Context, uuid.UUID) (int64, error)
    AppendAudit(context.Context, audit.Entry) error
    Commit(context.Context) error
    Rollback(context.Context) error
}

type Repository interface {
    Begin(context.Context) (Tx, error)
    GetStaff(context.Context, uuid.UUID) (Staff, error)
    ListStaff(context.Context, ListStaffParams) ([]Staff, error)
    // ...
}
```

- `Begin` returns an application-owned transaction handle. No driver type crosses the port.
- Services keep the existing control flow: begin, execute writes, append audit, commit, and
  defer rollback where the current code does so.
- The PostgreSQL adapter's `Tx` holds one `pgx.Tx`; its module methods call `sqlc.New(tx)`;
  `AppendAudit` calls the PostgreSQL audit writer with that same `pgx.Tx`.
- `Commit`/`Rollback` translate driver errors through the adapter error mapper.
- Modules without multi-write transactions use plain repository methods and do not get a
  fabricated `Tx`.
- `application/auth` uses a high-level transactional `Store` port because account
  provisioning, refresh rotation, reuse revocation, and logout-all already bundle
  orchestration-specific multi-step writes. The adapter opens a transaction and appends
  audit rows inside it; application code passes `audit.Entry` values and never sees SQL.
- Cross-module calls use consumer-declared interfaces. Consumers may import another
  application module's public row value (for example `booking.Booking` or `voyage.Voyage`)
  when required by the method result; they must not import its repository, adapter, or
  transport package.

### Incremental Delivery Phases

1. **Scaffold and move shared utilities**: create platform/transport/infrastructure trees;
   move config, errors, pagination, logging, health, httpx, and generated sqlc; update
   imports and test harness. Build and test.
2. **Split HTTP transport**: move route registration and handler/DTO files per module to
   `transport/http/handlers/<module>`; wire route registration through composition. Keep
   route inventory and OpenAPI tests green.
3. **Extract application types and ports module by module**: start with independent modules
   (`tenant`, `ai`, `feedback`, `sync`), then `partner`, `voyage`, `ship`, `staff`,
   `booking`, `guest`; finally `auth` and `audit`. Move PostgreSQL adapters in lockstep.
4. **Replace transaction and audit coupling**: introduce application `Tx` ports in `ship`
   import confirmation, `staff`, `booking`, and `guest`; move `audit.Execer` persistence;
   prove same-transaction rollback with existing integration tests.
5. **Split auth and external adapters**: move Firebase, KMS, local signing, Redis limiter,
   account store, and local guest cipher. Move HTTP principal providers to transport
   middleware. Keep token formats and rotation behavior unchanged.
6. **Split runtime composition**: make API and migration composition independent; reduce
   `cmd/*` to configuration, logging, signals, and one composition-root call.
7. **Guardrails and documentation**: replace the flat boundaries test with layer rules,
   mutation-check it, add the ADR, update README/architecture docs, run all validation.

### Validation Strategy

- Run `go fmt ./...`, `go vet ./...`, `go test ./...`, and `go build ./...` after each
  phase; run `go test -tags=integration ./...` when Docker is available.
- Reuse existing unit, route-inventory, OpenAPI, auth, idempotency, and sync tests with
  import/wiring changes only. Do not relax assertions.
- Add architecture-test assertions that fail on application -> infrastructure/transport,
  application -> forbidden SDK/driver/`net/http`, infrastructure -> transport/app,
  transport -> infrastructure/app, platform -> upper layers, and any import cycle.
- Temporarily add a forbidden import (for example `internal/infrastructure/persistence/postgres`
  in an application package) and confirm the boundary test fails; remove it.
- Verify API and migration runtimes separately: the migration runtime must not initialize
  Firebase, Redis, KMS, or HTTP.

## Complexity Tracking

No constitution violations. The layer split and application-owned persistence ports are
mandated by the specification and replace existing direct coupling; they do not add runtime
infrastructure, technologies, or placeholder packages. Complexity Tracking is empty.
