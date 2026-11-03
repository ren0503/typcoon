# Tasks: 002-business-modules — Initialize Business Modules and Database Models

**Input**: Design documents from `/specs/002-business-modules/`

**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Included — the spec explicitly requires them (FR-049–FR-053, US5, SC-007). Integration tests use real PostgreSQL via testcontainers.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

## Path Conventions

Single Go project (per plan.md): modules in `internal/<module>/`, shared infra under `internal/{app,apperrors,config,database,httpserver,testutil}`, schema/queries under `db/`, docs under `specs/002-business-modules/`.

<!-- Tasks below replace template samples; ownership map in spec.md is the authority for module assignments. -->

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Dependency and toolchain preparation (no business behavior).

- [X] T001 [P] Add pinned dependencies in `go.mod`: `github.com/shopspring/decimal`, `github.com/jackc/pgx-shopspring-decimal`, `github.com/jackc/pgerrcode`; promote `github.com/google/uuid` and `golang.org/x/crypto` (Argon2id PIN hashing) to direct requires per research.md R2/R3
- [X] T002 [P] Update `sqlc.yaml` per research R1: keep `pgx/v5`, set `emit_json_tags: true`, `emit_interface: true`, `emit_pointers_for_null_types: true`; add override pairs (nullable AND non-nullable listed separately): `uuid→uuid.UUID` / `uuid.NullUUID`, `pg_catalog.timestamptz/{timestamp}→time.Time` / `*time.Time`, nullable `text`/`varchar→*string`, `jsonb→json.RawMessage` / `*json.RawMessage`, `pg_catalog.numeric→decimal.Decimal` / `decimal.NullDecimal`; leave `date` as `pgtype.Date`, `time` as `pgtype.Time`, `bpchar` as `string`
- [X] T003 [P] Extend `.golangci.yml` with depguard rules: only module-private or public-interface files may be imported cross-module (`internal/<module>/module.go` is the public face); forbid any business module importing `internal/<other>/repo*, queries, sqlcmodel*` (FR-006)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T004 Extend `internal/testutil/postgres.go`: testcontainers Postgres harness — start container, connect via pgx pool, apply migrations with Goose **Provider API** over `db/migrations` (`goose.NewProvider(goose.DialectPostgres, db, embed or os.DirFS)` — commit embed of migrations via `db/embed.go`), expose `Start(ctx) (*Harness, t.Cleanup)`, pool accessor, and a `TruncateAll()` helper for test isolation (FR-050, TC-002)
- [X] T005 [P] Extend `internal/apperrors/errors.go`: add constructors `NotFound(code, msg)`, `Conflict(code, msg)`, `Validation(code, msg, field)`, `Unauthorized(code, msg)`, and forbidden code constants `unauthorized`, `validation_failed`, `invalid_reference`, `invalid_cursor`, `idempotency_conflict`, `overlap_conflict` with envelope-compatible HTTP statuses (FR-025/033/040)
- [X] T006 [P] Create `internal/database/pgerrors.go`: map `*pgconn.PgError` via `pgerrcode` constants — `23505→409 conflict`, `23503→422 invalid_reference`, `23514→422 validation_failed` (constraint name→field hint switch), `23P01→409 overlap_conflict`, `40001/40P01→bounded retry then 503`, `22P02→400 validation_failed`, `22032→422 validation_failed`, `pgx.ErrNoRows→404 not_found`; else `apperrors.Internal` (research R8)
- [X] T007 [P] Create `internal/pagination/pagination.go`: opaque cursor codec — base64url JSON `{v, sort..., id, f}` with filter-set hash `f`; `Endpoint(limit)` clamps default 20 / max 100 (FR-039); decode errors, unknown version, filter-hash mismatch, or >256-byte cursors return `400 invalid_cursor`; `limit+1` fetch helper to set `next_cursor` null at page end
- [X] T008 Create `internal/auth/` module: `Principal{TenantID, Roles, ShipIDs, ActorType, ActorID}` with `CanAccessTenant(id)`, `CanAccessShip(id)`; `PrincipalProvider` interface; `context` helpers (`PrincipalFrom(ctx)`); `DevHeaderProvider` resolving principals from an explicit allowlist; extend `internal/config/config.go` with `auth.mode` (`dev`|`jwt`) + `auth.dev.principals`, and fail-fast validation: `auth.mode=dev` with production environment (or empty allowlist) is a startup error (FR-043)
- [X] T009 Create `internal/audit/` module (internal `audit_log` owner, non-UUID bigint identity PK): `module.go` exposing `Appender{ AppendTx(ctx, tx, Entry) error }` plus query functions; `repo.go` with `INSERT INTO audit_log` and filterable list reads; `Entry{OccurredAt, Source(cloud/hub), ActorType(admin/account/staff/hub/system/ai), ActorID, Action, Entity, EntityID, ShipID, VoyageID, Details jsonb}` — details MUST never carry sensitive values (FR-046/048 data rules; R13)

**Checkpoint**: Foundation ready — test harness, error mapping, cursor codec, principal model, and audit seam all exist; user stories can start in parallel.

---

## Phase 3: User Story 1 — Find One Clear Owner for Every Table (Priority: P1) 🎯 MVP

**Goal**: Every table has exactly one owning module; module boundaries are structural (packages + interfaces) and machine-checked.

**Independent Test**: Run ownership + boundary tests without any endpoints: verify 41/41 table assignments and an acyclic import graph (spec US1 Independent Test).

### Implementation for User Story 1

- [X] T010 [P] [US1] Create public interface skeleton `module.go` for modules `tenant`, `ship`, `voyage`, `staff`, `partner`, `booking`, `guest`, `sync`, `ai`, `feedback` in `internal/<module>/module.go`: package, module-owning responsibility comment, public service interface per `contracts/module-interfaces.md` (constructor signatures only; no implementations yet). `auth` and `audit` skeletons already exist from T008/T009
- [X] T011 [US1] Create `internal/ownership/ownership.go`: constants for the 12 modules and a `TableOwner` map assigning all 41 tables from `docs/schema.sql` exactly as specified in the spec's Table-to-Module Ownership Map (e.g. `ports→voyage`, `guests`+`boarding_passes→guest`, `autonomy_policies→staff`, `job_runs→sync`, `ai_params→ai`); plus `internal/ownership/ownership_test.go` `TestOwnershipCoversAllTables`: parses table names from `docs/schema.sql` and fails on zero/multiple owners (FR-003, SC-001, new tables must fail CI)
- [X] T012 [US1] Create `internal/app/boundaries_test.go` `TestModuleBoundaries`: use `go list ./...` + import graph assertions — cross-module imports only touch the owning module's `module.go` surface (or `internal/{app,apperrors,config,database,httpserver,pagination,testutil,ownership}` infra); dependency graph acyclic (FR-006/007)
- [X] T013 [US1] Extend composition root `internal/app/app.go` to construct all 12 modules with dependency injection — each module built behind its public interface, wiring matches the acyclic graph in `data-model.md` (partner→ship; booking→{voyage,ship,partner}; guest→{booking,auth}; guest/booking/staff→audit) (FR-008/009)

**Checkpoint**: Ownership + boundary tests pass with skeleton modules; app still boots (existing health routes intact).

---

## Phase 4: User Story 2 — Trust That Models Match the Real Schema (Priority: P1)

**Goal**: Authoritative schema delivered verbatim as a migration; generated models cover 100% of tables/columns; DB-managed behavior proven.

**Independent Test**: Apply migration to empty testcontainers Postgres, assert 41 tables with columns/indexes/triggers; re-run migration harmlessly (spec US2 Independent Test).

### Implementation for User Story 2

- [X] T014 [P] [US2] Create `db/migrations/00002_business_schema.sql`: `-- +goose Up / StatementBegin` containing the authoritative DDL from `docs/schema.sql` VERBATIM (extension, all 41 tables, indexes, check constraints, exclusion constraints `ex_voyages_no_overlap`/`ex_partner_slots_no_overlap`/`ex_bookings_room_no_overlap`, identity `audit_log.id`, functions `touch_row`/`capture_change`/`touch_partner_on_contract`, all `trg_*` triggers); `-- +goose Down` drops objects in reverse-dependency order (FR-010/011, TC-005)
- [X] T015 [P] [US2] Replace placeholder `db/schema.sql` (currently "foundation defines no business tables") with the same authoritative schema content as sqlc input, keeping it byte-equivalent to the migration body
- [X] T016 [P] [US2] Create `db/queries/tenant/*.sql` named queries for `tenants`: insert (id passed explicitly, UUID v7 app-side), get by id, list keyset `(id)` pages, update name, soft delete `SET deleted_at = now()`, restore-free; counts/exists as needed — statuses like `status IN ('active','suspended')` are DB CHECK-enforced, mirrors in service constants
- [X] T017 [P] [US2] Create `db/queries/ship/*.sql` for `ships`, `decks`, `areas`, `rooms` (CRUD + keyset list + soft delete; deck uniqueness `uq_decks_level WHERE deleted_at IS NULL`, room `uq_rooms_number`); include FK joins used for parent-reference validation helpers (`deck_exists_in_ship`)
- [X] T018 [P] [US2] Create `db/queries/voyage/*.sql` for `ports`, `voyages`, `port_calls` (CRUD + keyset list; list ordered `(departure_at, id)` for voyages, `(sequence, id)` for port_calls; port-call sequence uniqueness via `uq_port_calls_seq`); ports list ordered `(un_locode, id)` (global, no ship scope)
- [X] T019 [P] [US2] Create `db/queries/staff/*.sql` for `staff`, `staff_leaves`, `staff_credentials`, `shift_templates`, `coverage_requirements`, `labor_rules`, `autonomy_policies` (CRUD + keyset lists; credential set/rotate + metadata read WITHOUT hash + staff_exists; autonomy insert uses real `ship_id` NOT the `00000000…` coalesce placeholder); languages `text[]` default `'{}'` and `certifications` JSONB `CHECK (jsonb_typeof(certifications) = 'array')` honored via `json.RawMessage`
- [X] T020 [P] [US2] Create `db/queries/partner/*.sql` for `partners`, `partner_contracts`, `partner_staff`, `partner_services`, `partner_slots` (CRUD + keyset lists; slot create/update constrained by `status IN ('requested','approved','rejected','cancelled')`; contract `starts_on` + optional `ends_on`; `partner_exists` helper)
- [X] T021 [P] [US2] Create `db/queries/booking/*.sql` and `db/queries/guest/*.sql`: bookings/service_bookings/guests CRUD (booking status NOT updated; service_bookings status update allowed), boarding_passes insert/get/list/revoke (`revoked_at` only); keyset lists; `booking_exists`; guests list per booking; no query writes `document_enc` as plaintext path (bytea param only)
- [X] T022 [P] [US2] Create `db/queries/auth/*.sql`, `db/queries/sync/*.sql`, `db/queries/ai/*.sql`, `db/queries/feedback/*.sql`: admin_users/admin_ship_access/accounts/account_identities/refresh_tokens CRUD-ish ops per ownership map ops column (tokens hash-only: insert/read-by-hash/rotate/revoke); sync tables: `ship_seq`/`change_log` reads only (no app writes; trigger-owned), hub_devices/ship_devices/revocations/snapshots/job_runs/ops_events defined ops (ops_events insert on-conflict `(hub_id, local_seq) DO NOTHING` dedup); `ai_params` metadata CRUD; feedback submit/get/list (no update/delete)
- [X] T023 [US2] Run `make sqlc` and commit generated `internal/database/sqlc/*`; fix any generation errors — zero errors, no handwritten duplicate models, generated code used directly (SC-003)

### Tests for User Story 2 ⚠ (write before/with T023 completion)

- [X] T024 [P] [US2] Migration tests `internal/database/migrate_integration_test.go`: apply to empty DB via harness (build-tag `integration`); assert table count = 41 by information_schema, presence of the 3 exclusion constraints and `capture_change`/`touch_row` functions; re-run `provider.Up` → harmless, version idempotent (FR-053, SC-004)
- [X] T025 [P] [US2] Schema parity test `internal/database/sqlc/parity_integration_test.go`: reflect over generated models and `information_schema` — every table has a model, every column represented, composite PKs (`change_log`, `ops_events`, `ship_seq`, `admin_ship_access`, `job_runs`) and bigint identity `audit_log` represented; nullable columns typed nullable (FR-013/016/014)
- [X] T026 [US2] Trigger behavior test `internal/database/trigger_integration_test.go`: insert+update a ship → `updated_at` changed, `version` incremented by trigger, one `change_log` row per op with per-ship `ship_seq` monotonic in commit order; verify app code does NOT write `version`/`updated_at` (FR-018)

**Checkpoint**: SC-002/SC-003/SC-004 provable via these three tests alone.

---

## Phase 5: User Story 3 — Use Basic Data Access Without Raw SQL (Priority: P2)

**Goal**: Repo+service layers per module with only appropriate operations; validation (incl. uniform parent-reference rule); transactions; audit emission.

**Independent Test**: Service/repo integration tests against real Postgres — create/get/list/update/soft-delete + expected errors, with no HTTP layer engaged (spec US3 Independent Test).

### Implementation for User Story 3

- [X] T027 [P] [US3] Implement `internal/tenant/` service+repo (`service.go`, `repo.go`): create/get/list/update/soft delete of tenants; UUID v7 via `uuid.NewV7()` before insert; list returns cursor pages (T007); soft-deleted rows excluded from all normal reads
- [X] T028 [P] [US3] Implement `internal/ship/` service+repo for ships/decks/areas/rooms: service validates `capacity > 0` rooms (`capacity smallint NOT NULL CHECK (capacity > 0)`), parent-references — `deck_id` must resolve to a NON-deleted deck of the SAME ship (uniform rule: nonexistent or soft-deleted parent → `invalid_reference` 422), enum values (`room_type IN ('inside','oceanview','balcony','suite','crew','other')`, `area_type` 10-value enum), `timezone` non-empty IANA; unique-conflict mapping for `uq_decks_level`/`uq_rooms_number`/`imo_number`
- [X] T029 [P] [US3] Implement `internal/voyage/` service+repo for ports/voyages/port_calls: validate `arrival_at > departure_at` in service BEFORE insert (CHECK also authoritative), `sequence >= 1`, port-call times optional, `status` values constrained; map `ex_voyages_no_overlap` and 23P01 to `overlap_conflict` 409; voyages unchanged-unique `code` per ship even after soft-delete surfaces 409 (not auto-reuse)
- [X] T030 [US3] Implement `internal/staff/` service+repo for staff/leaves/credentials/shift_templates/coverage/labor_rules/autonomy: `ends_at > starts_at` leaves; `min_staff >= 0` and `max_staff` NULL or `>= min_staff`; `days_of_week` values 1..7; `valid_from <= valid_to` when both present; credentials hash plaintext PIN with Argon2id (`golang.org/x/crypto/argon2`) before persist and pin_hash NEVER returned/logged; certifications must parse as JSON array or 422; audit emissions deferred to T036
- [X] T031 [P] [US3] Implement `internal/partner/` service+repo for partners/contracts/staff/services/slots: contract `ends_on > starts_on` when present; slot `end_at > start_at`; slot create starts `requested`; status decisions among requested→approved/rejected/cancelled capture `decided_by/decided_at/decision_note` from principal; map `ex_partner_slots_no_overlap` on approved slots to 409
- [X] T032 [US3] Implement `internal/booking/` service+repo for bookings/service_bookings: create with optional Idempotency-Key — in-tx lookup by `(account_id, key)` / `(booking_id, key)`; found+payload-hash-equal → return existing (no duplicate); found+different payload → 409 `idempotency_conflict` (FR-042); validate `disembark_seq > embark_seq`, room/embark bounds; booking `status` NEVER updated by service (immutable this phase); service_bookings `status` may be set directly to CHECK values; booking parents (ship/voyage/room/account) + service booking parents (booking/service/slot/guest) uniformly validated non-deleted; map `ex_bookings_room_no_overlap` → 409; transactions for booking+child creates (FR-026)
- [X] T033 [US3] Implement `internal/guest/` service+repo for guests/boarding_passes: create requires non-deleted booking; accepts `document_enc`/`emergency_contact_enc` ONLY as ciphertext bytes produced by the `guest.DocumentCipher` interface (envelope abstraction with a non-KMS local impl for dev/tests; NO plaintext storage path, FR-047); responses never include ciphertext or hashes; boarding pass issue validates booking, generates `jti` uuid, requires `expires_at` future; revoke idempotently sets `revoked_at`; travel of boarding passes offlines deferred (schema observation)
- [X] T034 [P] [US3] Implement `internal/sync/` ops for hub_devices/ship_devices/revocations/snapshots/job_runs/ops_events: system-managed lifecycle ops exactly per ownership map (`Provision/Retire`, `Register/Revoke`, `Claim/Finish`); `token_hash`/`cert_serial` never returned; `change_log`/`ship_seq` read-only and trigger-owned (FR-029/035)
- [X] T035 [P] [US3] Implement `internal/ai/` AIParamService and `internal/feedback/` FeedbackService: ai_params metadata CRUD (unique `(tenant_id, kind, param_version)`, kind/status enums; `storage_key`/`sha256`/`kid`/`signature` metadata only, no signing workflow); feedback submit requires voyage scope + `text_redacted` non-empty; insights record-only with severity 1..5, sentiment -2..2; no update/delete (FR-029, spec map)
- [X] T036 [US3] Wire audit emission (FR-057): `staff`, `booking`, `guest` services call `audit.Appender.AppendTx(ctx, tx, entry)` inside the SAME transaction as create/update/soft-delete (actor from principal; `details` = sanitized diff WITHOUT sensitive values); other modules do NOT emit yet

### Tests for User Story 3 ⚠

- [X] T037 [P] [US3] Integration tests `internal/tenant/*_test.go` + `internal/ship/service_integration_test.go` (build tag `integration`): CRUD happy paths, soft-delete exclusion, unique-violation mapping (imo/deck level/room number), invalid parent → 422 `invalid_reference`, cascade-free soft-delete of parent leaves child reusable reference check
- [X] T038 [P] [US3] Integration tests `internal/voyage/service_integration_test.go`: overlap → 409 `overlap_conflict` (exclusion constraint engaged), reversed times → 422, dup sequence → 409, port soft-deleted then referenced → 422
- [X] T039 [P] [US3] Integration tests `internal/staff/service_integration_test.go`: leaves time-order, `days_of_week` 9 → 422, non-array certifications → 422, PIN hashing round-trip + metadata-only read (no pin_hash anywhere), autonomy policy with NULL ship_id insert succeeds (no placeholder leak)
- [X] T040 [P] [US3] Integration tests `internal/partner/service_integration_test.go`: approved-slot overlap → 409, requested slots overlapping allowed by constraint semantics (only approved excluded), contract date window validation
- [X] T041 [US3] Integration tests `internal/booking/service_integration_test.go`: idempotency same-key+same-payload → same record; same-key+different-payload → 409 `idempotency_conflict` (both for bookings and service_bookings); room overlap → 409; booking status immutability; rollback test — chained multi-step tx failure leaves zero partial rows (US3 acceptance 7)
- [X] T042 [US3] Integration tests `internal/guest/service_integration_test.go`: create/update with ciphertext, plaintext rejection (422, never persisted), GET/list responses lack ciphertext+document content, boarding-pass issue→list→revoke idempotent, audit entry written in same tx (forced failure injection proves no staff/guest row persists without audit)

**Checkpoint**: All US3 acceptance scenarios demonstrable through service/repo integration tests without HTTP.

---

## Phase 6: User Story 4 — Consume a Safe API Foundation (Priority: P2)

**Goal**: Narrow documented `/v1` REST surface for the 7 exposed modules: default-deny, scoped, validated, cursored, consistent errors, schema-backed idempotency.

**Independent Test**: Contract+integration tests over the mounted router: unauthorized/cross-scope/invalid calls produce documented responses; internal tables have no routes (spec US4 Independent Test).

### Implementation for User Story 4

- [X] T043 [P] [US4] Author `openapi/openapi.yaml` from `specs/002-business-modules/contracts/openapi.yaml` (copy + finalize; document per-resource schemas/paths; contract for FR-038/039/040/042/043 validated by contract tests)
- [X] T044 [P] [US4] Implement `internal/httpserver/middleware/auth.go`: `Authenticate(provider)` resolving `auth.Principal` into request context for the whole `/v1` tree; default-deny (no principal → 401 `unauthorized` without revealing resource existence); dev-mode requests emit a `dev-principal` log marker (FR-043)
- [X] T045 [US4] Wire route mounting in `internal/httpserver/router.go` + `internal/app`: mount exposed modules' `RegisterRoutes(chi.Router)` under `/v1` with provider injection; `auth`, `sync`, `audit`, `ai`, `feedback` register NO routes; add a route-inventory test asserting zero CRUD routes for `change_log`, `ship_seq`, `refresh_tokens`, `audit_log`, `ops_events`, `snapshots`, `job_runs` (FR-044/045, US4.6)
- [X] T046 [P] [US4] Implement handlers `internal/{tenant,ship}/handlers.go`: DTOs with validator/v10 tags (required, uuid format, oneof for transport enums, min/max), service-attributed validation errors mapped to the error envelope via `apperrors.As` + request-id (`internal/httpserver/handlers/response.go` helpers), cursor list wiring with `pagination` (default 20/clamp 100/`invalid_cursor` 400)
- [X] T047 [P] [US4] Implement handlers `internal/{voyage,partner}/handlers.go`: same DTO/envelope/cursor patterns; port latitude/longitude numeric via decimal; 409 on overlap conflicts surfaced as `overlap_conflict` code
- [X] T048 [P] [US4] Implement handlers `internal/{staff,booking,guest}/handlers.go`: staff credential endpoints (PUT set/rotate — accepts PIN in body, returns ONLY `StaffCredentialMeta`; GET metadata); booking update rejecting `status` with 422 `business_rule_violation` (immutable per Clarification 2026-11-02); `Idempotency-Key` header passthrough to service; guest IO never returns ciphertext; boarding-pass issue/revoke endpoints (revoke → 200 current state)
- [X] T049 [US4] Add OpenAPI contract-test harness (build tag `integration`) `internal/httpserver/contract_test.go`: kin-openapi/openapi3 loads `openapi/openapi.yaml`; a test round-tripper validates requests/responses against the spec for every endpoint exercised; spec lint runs in `make test` (drift fails loudly, R12)

### Tests for User Story 4 ⚠

- [X] T050 [P] [US4] Security contract tests `internal/httpserver/security_integration_test.go`: unauthenticated request → 401 + envelope; cross-tenant principal → denied on tenant-scoped reads (e.g. other tenant's ship) per SC-008; ship-scoped principal cannot read foreign ship's voyages; malformed cursor → 400 `invalid_cursor`; cursor from another filter → 400; limit 500 clipped to 100 in response
- [X] T051 [P] [US4] Idempotency + sensitive-field contract tests `internal/httpserver/idempotency_integration_test.go` + `sensitive_redaction_test.go`: retry same key/payload → same 201 record, no dup; same key different body → 409 `idempotency_conflict`; guest JSON responses and server log lines contain no `document_enc`/`emergency_contact_enc`/PIN/token values (FR-046)

**Checkpoint**: US4 acceptance scenarios 1–7 each covered by a named test above.

---

## Phase 7: User Story 5 — Verify Correctness with Real Database Tests (Priority: P3)

**Goal**: Single comprehensive, reproducible test suite (local + CI) proving foundation properties end-to-end.

**Independent Test**: `make test && make test-integration` from a clean checkout with Docker — all green (spec US5 Independent Test).

### Implementation for User Story 5

- [X] T052 [P] [US5] Cross-cutting constraint matrix test `internal/testutil/constraints_integration_test.go`: dataset-driven cases for every documented constraint class — check violations (invalid status/role/type/department values), unique violations incl. non-partial uniques (`voyages(ship_id, code)`, `ships.imo_number`, `ports.un_locode` block reuse after soft delete), FK violations, exclusion violations (voyages/partner-slots/bookings) → each asserted to surface the mapped `apperrors` code, never a raw SQL error (US5.2)
- [X] T053 [P] [US5] Pagination correctness test `internal/pagination/pagination_integration_test.go`: seed ~150 rows in `tenants` and `bookings`; walk all pages via cursors — no duplicates, no skips at default 20 and boundary limit 100; end-of-list `next_cursor` null; crossing pages while concurrent insert runs keeps keyset stability (US5.4)
- [X] T054 [US5] Transaction rollback test `internal/database/tx_integration_test.go`: multi-write service operation where step N fails (forced via invalid child) → all earlier writes rolled back, `change_log` gains no entries (trigger-side consistency) (US5.5)
- [X] T055 [US5] Verify the same commands run identically locally and in CI: extend `Makefile` docs/footer comments and CI workflow (if present) so `make test` and `make test-integration` are the documented entry points; run `make test && make test-integration` locally from clean state to confirm green (FR-052)

**Checkpoint**: Spec US5 acceptance scenarios 1–6 verified.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T056 [P] Update `docs/system-architecture.md` module map (12 modules, ownership boundaries, audit emission scope) and note the new baseline `db/schema.sql` correspondence per constitution Principle IX
- [X] T057 [P] Update `README` quickstart section: docker-compose Postgres boot, `make sqlc`, `make migrate-up`, `make run`, dev-principal env vars (`AUTH_MODE=dev`, `AUTH_DEV_PRINCIPALS`), and the no-AWS-required statement (SC-009)
- [ ] T058 Run full quickstart.md validation: execute all 8 scenarios in `specs/002-business-modules/quickstart.md` and record outcomes; `golangci-lint run ./...` clean; confirm no FR-036 workflows leaked in diffs (SC-010)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: no dependencies — start immediately
- **Foundational (Phase 2)**: depends on T001–T003; BLOCKS all user stories
- **US1 (Phase 3)**: depends on Phase 2 (needs config/auth/audit types for composition)
- **US2 (Phase 4)**: depends on Phase 2 (harness) + T010 skeletons; does NOT depend on US1 completion except T013 composition (run after T013)
- **US3 (Phase 5)**: depends on Phase 2 + US2 (queries/generated models are the data access)
- **US4 (Phase 6)**: depends on US3 (services are the handlers' dependencies)
- **US5 (Phase 7)**: depends on US3 (+US4 for route-related scenarios already covered)
- **Polish (Phase 8)**: after all stories

### User Story Dependencies

- **US1 (P1)**: independent; blocked only by Phase 2
- **US2 (P1)**: independent of US1; blocked by Phase 2
- **US3 (P2)**: blocked by US2; interface composition from US1 must exist but stories remain testable independently
- **US4 (P2)**: blocked by US3; contract tests also gate on US2's migration harness
- **US5 (P3)**: aggregates proofs; blocked by US3 (test targets exist by then); runs after US4 for full-route matrix

### Within Each User Story

- Skeletons/queries before services; services before handlers; audit wiring after all three emitting services; tests accompany each implementation task (spec requires tests)

### Parallel Opportunities

- Setup: T001, T002, T003 fully parallel
- Foundational: T005, T006, T007 parallel (T004 first; T008, T009 after T004)
- US2: T014–T022 all parallel (distinct files); parity/migration tests parallel after T023
- US3: module pairs (tenant, ship, voyage, partner, sync, ai+feedback) parallel; staff/booking/guest can also parallelize with care on T036 audit wiring at the end
- US4: T043, T044, T046–T048 parallel once T045 mounts structure

---

## Parallel Example: User Story 2

```bash
# All query files are independent (distinct directories):
Task: "Create db/queries/tenant/*.sql for tenants (T016)"
Task: "Create db/queries/ship/*.sql for ships, decks, areas, rooms (T017)"
Task: "Create db/queries/voyage/*.sql for ports, voyages, port_calls (T018)"
Task: "Create db/queries/staff/*.sql for 7 staff-family tables (T019)"
Task: "Create db/queries/partner/*.sql for 5 partner tables (T020)"
Task: "Create db/queries/booking/*.sql, db/queries/guest/*.sql (T021)"
Task: "Create db/queries/{auth,sync,ai,feedback}/*.sql for internal tables (T022)"
```

---

## Implementation Strategy

### MVP First (US1 + US2 only)

1. Phase 1 + Phase 2 complete (toolchain + harness + shared seams)
2. US1: 41/41 ownership assigned and boundary-checked
3. US2: real migration + generated models provably matching the schema
4. **STOP and VALIDATE**: ownership + migration + parity tests green — schema foundation is trustworthy before any feature logic

### Incremental Delivery

1. Setup → Foundational → US1+US2 (schema trust) → Validate
2. US3 (data access) → Validate against real Postgres
3. US4 (safe API) → Validate contract tests
4. US5 (aggregated proofs) → full-suite green → quickstart validation

### Parallel Team Strategy

- Developer A: US1 boundary work → US3 staff/partner modules
- Developer B: US2 queries per module directory → US3 booking/guest
- Developer C: Phase 2 shared infra → US4 middleware/handlers → US5 contract suites
- Merge order: US1 → US2 → US3 → US4 → US5 (dependencies as listed)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to spec user story for traceability
- Tests are MANDATORY here per FR-049–FR-053 and US5 — do not skip
- Never handwritten models/ad hoc SQL in services; all data access through generated queries (FR-012/024)
- Sensitive values (pin_hash, token_hash, document ciphertext) never in logs or responses (FR-046/048)
- Stop at any checkpoint to validate the story independently
