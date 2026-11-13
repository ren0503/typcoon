---

description: "Task list for separating business application from infrastructure"
---

# Tasks: Separate Business Application from Infrastructure Layers

**Input**: Design documents from `specs/005-app-infra-separation/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`

**Tests**: Existing tests are mandatory preservation gates and must not be weakened. The
architecture-boundary test is explicitly required by FR-039/FR-040/FR-041 and SC-008.

**Organization**: Tasks are grouped by user story. US1 and US2 are the two P1 halves of the
same architectural refactor: US1 establishes the dependency direction; US2 proves that the
refactor preserved behavior. US3–US5 complete runtime composition, ownership/guardrails, and
documentation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: `US1`–`US5` for user-story phases; omitted in setup, foundational, and polish
- Every task includes an exact repository path

## Path Conventions

- Repository root: `/home/ren0503/tycoon/server`
- Go module: `github.com/ren0503/tycoon`
- Application: `internal/application/<module>/`
- Infrastructure: `internal/infrastructure/...`
- Transport: `internal/transport/http/...`
- Platform: `internal/platform/...`
- Composition root: `internal/app/`
- Authoritative schema/migrations/queries: `db/`, `docs/schema.sql`
- API contract: `openapi/openapi.yaml`

---

## Phase 1: Setup (Shared Infrastructure Moves)

**Purpose**: Relocate shared technical packages so every later module move has a stable home.
Update imports and tests in the same task so compilation remains coherent.

- [X] T001 Move `internal/database` pool/DSN/migrations/error mapping and `internal/database/sqlc` to `internal/infrastructure/persistence/postgres` and `internal/infrastructure/persistence/postgres/sqlc`; update `sqlc.yaml` `out`/package paths, run `make sqlc`, update all imports and `internal/database/*_test.go` locations.
- [X] T002 Move `internal/apperrors` to `internal/platform/errors` while retaining Go package name `apperrors`; update all imports, `internal/platform/errors/errors_test.go`, and stable error codes/messages unchanged.
- [X] T003 Move `internal/config` to `internal/platform/config`; update imports, `internal/platform/config/config_test.go`, environment variables, defaults, and fail-fast validation unchanged.
- [X] T004 Move `internal/pagination` to `internal/platform/pagination`; update imports, `internal/platform/pagination/pagination_test.go`, `pagination_integration_test.go`, cursor encoding, and filter hashes unchanged.
- [X] T005 Move `internal/httpx` to `internal/transport/http/httpx`; update imports, `internal/transport/http/httpx/upload_test.go`, JSON/error/upload helpers, and upload limits unchanged.
- [X] T006 Move `internal/logging` to `internal/infrastructure/observability/logging`; update `cmd/server`, `cmd/migrate`, and composition imports with log levels/fields unchanged.
- [X] T007 Move `internal/health` to `internal/infrastructure/health`; update imports and preserve `Pinger`, intervals, timeout, logging, and failure behavior in `internal/infrastructure/health/checker.go`.
- [X] T008 Move `internal/awsconfig` to `internal/infrastructure/aws`; update imports and `config_test.go`; preserve AWS SDK configuration and fail-fast behavior.
- [X] T009 Move `internal/testutil` to `internal/platform/testutil`; update integration-test imports, harness/migration application, fixtures, logger, and auth helpers without changing test behavior.
- [X] T010 Move `internal/httpserver` router/server/middleware/technical handlers/tests to `internal/transport/http/{router.go,server.go,middleware/,handlers/health/}`; keep Chi route composition and `/health`/`/ready` behavior unchanged.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish shared application metadata and audit policy/value contracts needed by
all module moves.

**⚠️ CRITICAL**: Complete this phase before starting US1 module extraction.

- [X] T011 Move `internal/ownership` to `internal/application/ownership`; update `internal/application/ownership/ownership_test.go` and imports; preserve all 12 modules and every table-owner mapping verbatim.
- [X] T012 Move audit `Entry`, `Record`, `Filter`, `Validate`, and audit action policy to `internal/application/audit/audit.go`; keep a temporary compatibility re-export in `internal/audit/audit.go` so not-yet-moved modules compile; update auth/ship/staff/booking/guest tests for the new `Entry` import only.

**Checkpoint**: Shared paths and audit value contracts are ready; module extraction can begin.

---

## Phase 3: User Story 1 - Enforce Dependency Direction Across Layers (Priority: P1) 🎯 MVP

**Goal**: Every business module lives under `internal/application`, every concrete
implementation under `internal/infrastructure`, every HTTP surface under
`internal/transport/http`, and the automated boundary test rejects forbidden imports and
cycles.

**Independent Test**: Run the layer-boundary test and `go list` import checks; temporarily
add a forbidden application import and confirm the test fails, then remove it and confirm it
passes. The complete package graph must be acyclic.

### Boundary and Audit Foundations for US1

- [X] T013 [US1] Replace `internal/app/boundaries_test.go` with a layer-aware `go list -json ./...` test that classifies `application`, `infrastructure`, `transport`, `platform`, and `app`, rejects forbidden external prefixes, checks import direction, reports offending package/import, and detects cycles.
- [X] T014 [US1] Move audit SQL persistence from `internal/audit` to `internal/infrastructure/persistence/postgres/audit/store.go`; expose `AppendTx(ctx, pgx.Tx, application/audit.Entry)` and `List`; update current construction/call sites through the temporary `internal/audit` facade and move audit tests.

### Tenant Module

- [X] T015 [US1] Create `internal/application/tenant/{types.go,ports.go}` and move `internal/tenant/{module.go,service.go}` to `internal/application/tenant/`; replace `sqlc.Tenant`/`sqlc.*Params` with application `Tenant`/`CreateParams`/`ListParams`/`UpdateParams`, keep validation/errors/pagination unchanged, and move `service_integration_test.go` with import-only changes.
- [X] T016 [US1] Create `internal/infrastructure/persistence/postgres/tenant/repo.go` implementing the tenant repository port with `sqlc.New` and `postgres.MapPgError`; map rows field-for-field and preserve create/list/get/update/soft-delete SQL behavior.
- [X] T017 [US1] Move `internal/tenant/handlers.go` to `internal/transport/http/handlers/tenant/handlers.go`; keep `/api/tenants` routes, DTOs, validation messages, envelopes, and response fields unchanged; update route registration.

### Voyage Module

- [X] T018 [US1] Create `internal/application/voyage/{types.go,ports.go}` and move `internal/voyage/{module.go,service.go}` to `internal/application/voyage/`; replace `sqlc.Port`/`sqlc.Voyage`/`sqlc.PortCall` and params with app types, convert `decimal.NullDecimal` latitude/longitude to `*float64` at the port boundary, and move `service_integration_test.go` with import-only changes.
- [X] T019 [US1] Create `internal/infrastructure/persistence/postgres/voyage/repo.go` implementing the voyage repository port with pgx/sqlc; map `decimal.NullDecimal` ↔ `*float64`, preserve port/voyage/port-call SQL, pagination, and error translation.
- [X] T020 [US1] Move `internal/voyage/handlers.go` to `internal/transport/http/handlers/voyage/handlers.go`; preserve `/api/ports`, `/api/voyages`, `/api/voyages/{voyageId}/port-calls`, and `/api/port-calls/{id}` routes, DTOs, time formatting, and envelopes unchanged.

### Ship Module

- [X] T021 [US1] Create `internal/application/ship/{types.go,ports.go}` and move `internal/ship/{module.go,service.go,import_service.go,import_types.go,import_validation.go,template.go}` to `internal/application/ship/`; replace sqlc/pgx types, define application `Repository` plus `Tx` with `AppendAudit`/`Commit`/`Rollback` for import confirmation, and move ship tests with import-only changes.
- [X] T022 [US1] Create `internal/infrastructure/persistence/postgres/ship/{repo.go,import_repo.go,tx.go}`; implement ship/deck/area/room/import SQL with `sqlc.New`, `postgres.MapPgError`, same-transaction audit via `infrastructure/.../postgres/audit.AppendTx`, and preserve import row locks, conflict checks, terminal commits, and rollback behavior.
- [X] T023 [US1] Move `internal/ship/{handlers.go,import_handlers.go}` to `internal/transport/http/handlers/ship/`; preserve `/api/ships`, structure import, decks/areas/rooms routes, multipart/upload handling, DTOs, status codes, and error envelopes unchanged.

### Staff Module

- [X] T024 [US1] Create `internal/application/staff/{types.go,ports.go}` and move `internal/staff/{module.go,service.go,pin.go}` to `internal/application/staff/`; replace `pgtype.Date` with `*time.Time`, `pgtype.Time` with `staff.TimeOfDay{Microseconds,Valid}`, replace sqlc/pgx types, define app `Repository` plus audited `Tx`, and move staff tests with import-only changes.
- [X] T025 [US1] Create `internal/infrastructure/persistence/postgres/staff/{repo.go,tx.go}`; implement all staff/leave/credential/shift/coverage/labor/autonomy SQL, map date/time values, use same-transaction audit through `Tx.AppendAudit`, and preserve error translation and soft-delete semantics.
- [X] T026 [US1] Move `internal/staff/handlers.go` to `internal/transport/http/handlers/staff/handlers.go`; keep `/api/staff`, leaves, credential, shift-template, coverage, labor-rule, and autonomy routes, date/time parsing and formatting, DTOs, and envelopes unchanged.

### Partner Module

- [X] T027 [US1] Create `internal/application/partner/{types.go,ports.go}` and move `internal/partner/{module.go,service.go}` to `internal/application/partner/`; replace sqlc/pgx types, map `pgtype.Date` to `*time.Time` at the port boundary, declare a consumer-owned ship tenant lookup, and move partner tests with import-only changes.
- [X] T028 [US1] Create `internal/infrastructure/persistence/postgres/partner/repo.go`; implement partner/contract/staff/service/slot SQL with row/date mapping, pagination, and `postgres.MapPgError`; preserve exclusion/overlap and status behavior.
- [X] T029 [US1] Move `internal/partner/handlers.go` to `internal/transport/http/handlers/partner/handlers.go`; preserve `/api/partners`, nested contracts/staff/services/slots, standalone update/delete routes, DTOs, date parsing, and envelopes unchanged.

### Booking Module

- [X] T030 [US1] Create `internal/application/booking/{types.go,ports.go}` and move `internal/booking/{module.go,service.go}` to `internal/application/booking/`; replace sqlc/pgx types, define app `Repository` plus audited `Tx`, keep idempotency decisions in the service, and move `service_integration_test.go` with import-only changes.
- [X] T031 [US1] Create `internal/infrastructure/persistence/postgres/booking/{repo.go,tx.go}`; implement booking/service-booking SQL, idempotency lookups/conflicts, same-transaction audit through `Tx.AppendAudit`, row mapping, and error translation unchanged.
- [X] T032 [US1] Move `internal/booking/handlers.go` to `internal/transport/http/handlers/booking/handlers.go`; preserve `/api/bookings`, `/api/service-bookings`, idempotency headers, DTOs, status codes, and envelopes unchanged.

### Guest Module

- [X] T033 [US1] Create `internal/application/guest/{types.go,ports.go}` and move `internal/guest/{module.go,service.go}` to `internal/application/guest/`; keep `DocumentCipher` and encryption-presence rules in application, replace sqlc/pgx types, map `pgtype.Date` to `*time.Time`, define app `Repository` plus audited `Tx`, and move guest tests with import-only changes.
- [X] T034 [US1] Create `internal/infrastructure/persistence/postgres/guest/{repo.go,tx.go}`; implement guest/boarding-pass SQL, date mapping, same-transaction audit via `Tx.AppendAudit`, and preserve secrecy/revocation behavior and errors.
- [X] T035 [US1] Move `internal/guest/{dto.go,handlers.go}` to `internal/transport/http/handlers/guest/`; preserve `/api/bookings/{bookingId}/guests`, boarding-pass issue/list/revoke/detail routes, DTOs, date parsing, document handling, and envelopes unchanged.

### Authentication Module

- [X] T036 [US1] Create `internal/application/auth/{types.go,ports.go}` and move `internal/auth/{module.go,service.go,principal.go,tokens.go,refresh.go,audit.go}` to `internal/application/auth/`; declare `Store`, `Verifier`, `Signer`, `KeySet`, `RateLimiter`, `AccountLookup`, and `Clock` ports; remove `net/http`, pgx, sqlc, AWS, Redis, and Firebase imports from application/auth; move unit tests with import-only changes.
- [X] T037 [US1] Create `internal/infrastructure/persistence/postgres/auth/{store.go,accounts.go}` implementing the application `Store`/account lookup with pgx/sqlc, application row types, and same-transaction audit; move `store_test.go` and auth persistence integration tests with construction/import changes only.
- [X] T038 [US1] Create external auth adapters: `internal/infrastructure/identity/firebase/verifier.go`, `internal/infrastructure/security/kms/signer.go`, `internal/infrastructure/security/local/{signer.go,document_cipher.go}`, and `internal/infrastructure/cache/redis/limiter.go`; move Firebase/signing/rate-limit/local-cipher tests without assertion changes.
- [X] T039 [US1] Move `internal/auth/{handlers.go,dto.go,provider.go}` to `internal/transport/http/handlers/auth/` and `internal/transport/http/middleware/`; keep `AccessTokenProvider`, `DevHeaderProvider`, `DenyAllProvider`, bearer parsing, public/protected auth routes, DTOs, error responses, and middleware behavior unchanged.

### Internal Modules and Audit Cleanup

- [X] T040 [US1] Remove the temporary `internal/audit` facade after all call sites use `application/audit` and module `Tx.AppendAudit`/auth Store transactions; confirm no `pgx`/`pgconn` remains in application contracts.
- [X] T041 [US1] Move `internal/ai/{module.go,service.go}` to `internal/application/ai/`; create app row/param types and a repository port, remove pgx/sqlc, and preserve AI metadata use cases. No LLM/provider integration is added.
- [X] T042 [US1] Create `internal/infrastructure/persistence/postgres/ai/repo.go`; implement AI parameter SQL and error mapping unchanged; do not add transport or provider packages.
- [X] T043 [US1] Move `internal/feedback/{module.go,service.go}` to `internal/application/feedback/`; replace sqlc/pgx types, declare consumer-owned voyage/ship lookup ports, and preserve submit/get/insight behavior.
- [X] T044 [US1] Create `internal/infrastructure/persistence/postgres/feedback/repo.go`; implement feedback response/insight SQL and error mapping unchanged; no HTTP transport package is created.
- [X] T045 [US1] Move `internal/sync/{module.go,repo.go,service.go}` to `internal/application/sync/`; replace sqlc/pgx types, declare consumer-owned ship/voyage ports, and preserve device/revocation/snapshot/job/ops/change-log behavior.
- [X] T046 [US1] Create `internal/infrastructure/persistence/postgres/sync/repo.go`; implement sync SQL and error mapping unchanged; no worker/hub package is created.
- [X] T047 [US1] Finalize `internal/app/boundaries_test.go` to require absence of legacy `internal/{tenant,ship,...}` business paths, forbid application `net/http`/drivers/SDKs, enforce platform/testutil exception, and keep the full cycle check.
- [X] T048 [US1] Run `go test ./internal/app -run 'Test.*Boundar' -count=1` and `go list` checks; fix every remaining application-to-infrastructure/transport import and any import cycle without weakening the rule.

**Checkpoint**: US1 is independently verifiable by the boundary test and an acyclic package graph.

---

## Phase 4: User Story 2 - Preserve Existing Business Behavior and API Contracts (Priority: P1)

**Goal**: Existing endpoints, envelopes, auth, audit, idempotency, database, and sync behavior
remain unchanged after relocation.

**Independent Test**: Existing unit, route-inventory, OpenAPI, auth, idempotency, migration,
trigger, audit, and sync tests pass with only import/package/construction changes. The route
inventory and OpenAPI operation set are unchanged.

- [X] T049 [US2] Move/update `internal/transport/http/{route_inventory_test.go,openapi_spec_test.go,contract_integration_test.go,auth_contract_integration_test.go,template_contract_integration_test.go}`; assert `/api`, `/health`, `/ready`, route set, methods, status codes, and envelopes are unchanged.
- [X] T050 [US2] Move/update auth exchange, refresh rotation/reuse, logout-all, identity link/unlink, rate-limit, JWKS, and signing tests across `internal/application/auth/`, `internal/infrastructure/{identity/firebase,security/kms,security/local,cache/redis}/`, and `internal/transport/http/handlers/auth/`; do not change assertions or skip conditions.
- [X] T051 [US2] Move/update migration, trigger, transaction, `pgerrors`, schema-parity, and sqlc tests under `internal/infrastructure/persistence/postgres/`; confirm no migration, schema, trigger, change-log, or constraint behavior changed.
- [X] T052 [US2] Move/update module integration tests for tenant, ship/import, voyage, staff, partner, booking, and guest under `internal/application/*/`, `internal/infrastructure/persistence/postgres/*/`, and `internal/transport/http/handlers/*/`; retain all business assertions, including rollback and conflict cases.
- [X] T053 [US2] Add or update contract-level assertions for date (`*time.Time`), time-of-day (`staff.TimeOfDay`), decimal (`*float64`), nullable JSON, pagination, and response envelopes in `internal/transport/http/handlers/`; preserve existing API formatting exactly.
- [X] T054 [US2] Verify same-transaction audit and rollback for ship import confirmation, staff, booking, and guest in `internal/application/{ship,staff,booking,guest}/` and `internal/infrastructure/persistence/postgres/{ship,staff,booking,guest}/` integration tests; confirm failed audit append still rolls back the business write and commit ordering is unchanged.
- [X] T055 [US2] Run `make test` and `make test-integration`; fix only imports, package locations, and construction wiring. Record any unchanged environment-driven skips; do not add suppressions.

**Checkpoint**: US1 structure and US2 behavior are both verified; MVP is complete.

---

## Phase 5: User Story 3 - Keep Runtime Composition Explicit and Deployment-Specific (Priority: P2)

**Goal**: API and migration runtimes assemble only the infrastructure they need; command
entrypoints remain thin.

**Independent Test**: Build and start both runtimes with existing configuration. Inspect the
composition root to confirm PostgreSQL/auth/HTTP wiring for API and PostgreSQL-only wiring
for migration; no worker/hub runtime is created.

- [X] T056 [US3] Create `internal/app/api.go` to assemble config/logger, PostgreSQL pool, AWS config, auth adapters, application modules, transport router/server, and health monitor.
- [X] T057 [US3] Create `internal/app/migrate.go` to assemble only PostgreSQL configuration/pool/Goose and expose the migration runtime entrypoint.
- [X] T058 [US3] Refactor `internal/app/modules.go` to construct application services from injected ports only; remove `pgxpool.Pool`/`audit.Store` arguments and direct SQL/HTTP dependencies from composition.
- [X] T059 [US3] Refactor `cmd/server/main.go` to load config/logger/signals and call the API composition root only; remove direct infrastructure/client construction.
- [X] T060 [US3] Refactor `cmd/migrate/main.go` to load config/logger/signals and call the migration composition root only; ensure it initializes no Firebase, Redis, KMS, HTTP, or auth provider.
- [X] T061 [US3] Move `cmd/server/auth.go` construction (`buildAuthService`, `buildKeySet`, `buildRateLimiter`, verifier/store/provider wiring) into `internal/app/api.go` plus infrastructure adapters; delete `cmd/server/auth.go`.
- [X] T062 [US3] Update/add `internal/app/app_test.go` and composition checks to prove API wiring and migration-only wiring, graceful shutdown, and no unused cloud dependency initialization; run `make build`.

**Checkpoint**: API and migration are independently initializable and deployment-specific.

---

## Phase 6: User Story 4 - Establish Clear Ownership of Business and Technical Code (Priority: P2)

**Goal**: Every final package has one responsibility, route registration is out of application,
persistence is physically separate, and no placeholder package exists.

**Independent Test**: Review the final tree and package imports; verify no application package
contains handlers/SQL/SDK construction, no ai/feedback/sync transport directory exists, no
empty worker/hub/SQLite/SNS/SQS/S3/Redshift/EventBridge package exists, and ownership metadata
still matches all tables.

- [X] T063 [US4] Verify `internal/application/ownership/ownership.go` still maps every table to the same module and update `internal/application/ownership/ownership_test.go` only for imports; confirm no business table owner changed.
- [X] T064 [US4] Audit `internal/application/` for leftover `net/http`, chi, pgx, sqlc, AWS, Redis, Firebase, SQL strings, route registration, or adapter construction; move any remainder to `internal/transport/http/`, `internal/infrastructure/`, or the composition root.
- [X] T065 [US4] Confirm no empty/placeholder directories or packages were created for worker, hub, SQLite, SNS, SQS, S3, Redshift, EventBridge, certificates, metrics, or route-less modules; update `internal/app/boundaries_test.go` to reject legacy paths if not already done.
- [X] T066 [US4] Review package comments and ownership boundaries in `internal/application/*/module.go` and `internal/application/*/ports.go`; update comments only where needed to state rules/services vs transport/persistence, with no behavior changes.

**Checkpoint**: Every package maps to a real responsibility and the target tree contains no placeholders.

---

## Phase 7: User Story 5 - Prevent Regressions with Documentation and Guardrails (Priority: P3)

**Goal**: Durable documentation, ADR, and an executable guardrail prevent boundary erosion.

**Independent Test**: Read the ADR and architecture docs, run the boundary test, introduce a
deliberate forbidden import, confirm failure, remove it, and confirm pass. Final validation
commands and schema/OpenAPI diffs are clean.

- [X] T067 [US5] Create `docs/adr/ADR-09-application-infrastructure-layers.md` documenting the five layers, application-owned row types, transaction/audit ports, runtime composition split, alternatives rejected, and deferred SQLite/worker/hub decisions.
- [X] T068 [US5] Update `README.md` project layout and architecture summary to show `internal/application`, `internal/infrastructure`, `internal/transport`, `internal/platform`, and `internal/app`.
- [X] T069 [US5] Update `docs/system-architecture.md` package/module mapping, dependency rules, runtime composition, and persistence/audit ownership; keep synchronization and data-ownership descriptions unchanged.
- [X] T070 [US5] Mutation-test `internal/app/boundaries_test.go`: add a forbidden application import (for example `internal/infrastructure/persistence/postgres` and `net/http`), confirm failure names the package/import, remove it, and confirm pass; record the result.
- [X] T071 [US5] Run final `go fmt ./...`, `go vet ./...`, `go test ./...`, `go build ./...`, and integration tests where available; verify `docs/schema.sql`, `db/migrations/`, `db/queries/`, and `openapi/` have no functional diff and report moved/split/merged packages and intentionally absent packages.

**Checkpoint**: Documentation, ADR, guardrails, and validation gates are complete.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; must complete before module extraction.
- **Foundational (Phase 2)**: Depends on Setup; blocks US1 module work.
- **US1 (Phase 3)**: Depends on Foundational; establishes the target structure.
- **US2 (Phase 4)**: Depends on US1; verifies preserved behavior and API contracts.
- **US3 (Phase 5)**: Depends on US1 (application/infrastructure split) and US2 (behavior green).
- **US4 (Phase 6)**: Depends on US1–US3; final ownership/placement audit.
- **US5 (Phase 7)**: Depends on US4; final docs, mutation test, and validation.

### User Story Dependencies

- **US1 (P1)**: Required base for all other stories. Independently testable via boundary test.
- **US2 (P1)**: Same refactor milestone as US1; depends on US1 structure, independently testable via existing behavior suites.
- **US3 (P2)**: Depends on US1/US2; independently testable by starting API and migration runtimes.
- **US4 (P2)**: Depends on US1–US3; independently testable by tree/package inspection and ownership test.
- **US5 (P3)**: Depends on US1–US4; independently testable via docs review and mutation test.

### Parallel Opportunities

- Setup package moves touch shared imports and should run sequentially unless explicitly split by package.
- Within US1, application type extraction for different modules can run in parallel after T013/T014 if developers avoid shared composition files.
- Infrastructure adapters for different modules can run in parallel once their module ports exist.
- Transport handler moves for different modules can run in parallel once their application interfaces exist.
- US2 test moves can be parallel by test suite, but final `make test`/integration runs must be serialized after edits.
- US3 composition files are shared and should not be edited in parallel without coordination.
- US5 documentation files can be written in parallel; mutation/final validation must be serialized.

### Within Each User Story

- Boundary/audit prerequisites before module extraction.
- Application types and ports before infrastructure adapters.
- Infrastructure adapters before transport handlers that depend on their responses.
- Module tests move with the code they cover; assertions stay unchanged.
- Run the smallest relevant test after each module group; run full gates at checkpoints.

---

## Parallel Examples

### US1: Independent Module Extraction

```bash
# After T013/T014 and shared conventions are in place, different developers can work on:
Task: "T015-T017 tenant application types, adapter, transport"
Task: "T018-T020 voyage application types, adapter, transport"
Task: "T027-T029 partner application types, adapter, transport"
```

Do not parallelize edits to `internal/app/modules.go`, `internal/app/api.go`, or shared
boundary tests.

### US1: Infrastructure Adapters

```bash
# Once each module's ports exist, adapters are separate files:
Task: "T016 tenant postgres adapter"
Task: "T019 voyage postgres adapter"
Task: "T028 partner postgres adapter"
```

### US2: Behavior Suites

```bash
# Test suites are separate files after relocation:
Task: "T049 route inventory and OpenAPI tests"
Task: "T051 migration/trigger/error tests"
Task: "T052 module integration tests"
```

---

## Implementation Strategy

### MVP First (US1 + US2 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete US1: application/infrastructure/transport split and boundary test.
4. Complete US2: run existing behavior and contract suites with import/wiring-only changes.
5. **STOP and VALIDATE**: boundary test passes, route inventory/OpenAPI unchanged, existing
   unit/integration tests pass. This is the MVP for the architectural refactor.

### Incremental Delivery

1. Setup + Foundational → shared paths ready.
2. US1 → dependency direction enforced.
3. US2 → behavior preservation proven.
4. US3 → runtime composition split.
5. US4 → ownership and placement audited.
6. US5 → ADR/docs/guardrail mutation test and final validation.

### Parallel Team Strategy

1. One developer completes Setup/Foundational and shared boundary/audit conventions.
2. After that, module groups can be split across developers:
   - Developer A: tenant, voyage, partner.
   - Developer B: ship, staff.
   - Developer C: booking, guest.
   - Developer D: auth + external adapters.
3. Internal modules (`ai`, `feedback`, `sync`) can proceed in parallel once audit/persistence
   conventions are stable.
4. One owner completes composition (US3) and docs (US5) after module merges.

---

## Notes

- [P] marks tasks in different files with no incomplete dependency.
- Existing tests are preservation evidence; do not relax assertions, delete coverage, or add
  new skips.
- The app-owned `Tx` must keep audit in the same PostgreSQL transaction.
- No schema, migration, query, OpenAPI, event, audit payload, route, or auth behavior change.
- No empty package or placeholder runtime is allowed.
- Commit after logical groups; stop at checkpoints to validate independently.
