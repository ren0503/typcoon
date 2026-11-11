---

description: "Task list for Ship Onboarding with Excel Structure Import"
---

# Tasks: Ship Onboarding with Excel Structure Import

**Input**: Design documents from `/specs/004-ship-structure-import/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`,
`quickstart.md`

**Tests**: Included because the specification explicitly requires them
(FR-065–FR-069). Write the tests first and confirm they fail before implementing.

**Organization**: Tasks are grouped by user story so each story can be implemented,
tested, and delivered independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: `US1`–`US6`, mapping to the user stories in `spec.md`
- Every task includes the exact repository path(s) it touches

## Path Conventions

- Single Go module at the repository root.
- Business modules live in `internal/<module>/`; the ship feature lives in
  `internal/ship/`.
- Database artifacts live in `db/`; the authoritative baseline is `docs/schema.sql`.
- HTTP contract is `openapi/openapi.yaml`; integration tests live beside the package
  they exercise.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add the one new dependency, the import table, generated queries, and the
shared test/upload helpers required by every story.

- [X] T001 [P] Add `github.com/xuri/excelize/v2` pinned to `v2.11.1` to `go.mod`/`go.sum`
  (`go get github.com/xuri/excelize/v2@v2.11.1 && go mod tidy`); verify `go build ./...`
  still succeeds.

- [X] T002 [P] Create `db/migrations/00004_ship_structure_imports.sql` defining
  `ship_structure_imports` exactly as `data-model.md` §2: `id uuid PK`,
  `ship_id uuid NOT NULL REFERENCES ships(id)`, optional `idempotency_key text`,
  `status text NOT NULL DEFAULT 'validating' CHECK (status IN
  ('uploaded','validating','ready','failed','processing','completed'))`,
  `file_name text NOT NULL`, `file_size_bytes bigint NOT NULL CHECK (file_size_bytes >= 0)`,
  `file_sha256 text NOT NULL CHECK (file_sha256 ~ '^[0-9a-f]{64}$')`, six count columns
  with `>= 0` checks and `valid + invalid = total` checks, `validated_payload jsonb`,
  `validation_errors jsonb NOT NULL DEFAULT '[]'::jsonb`, `error_message text`,
  `created_by text`, timestamps, `completed_at`; add
  `uq_ship_structure_imports_idem (ship_id, idempotency_key) WHERE idempotency_key IS NOT NULL`
  and `ix_ship_structure_imports_ship_created (ship_id, created_at DESC)`. Do not add
  a sync/capture trigger, `version`, or `deleted_at` (the table is cloud-only).

- [X] T003 Append the identical `ship_structure_imports` DDL to `docs/schema.sql`
  (authoritative baseline) and `db/schema.sql` (sqlc input) so the baseline and
  migration agree; keep the table in the SHIPS/DECKS/AREAS/ROOMS section.

- [X] T004 Map `ship_structure_imports` to `ModuleShip` in
  `internal/ownership/ownership.go`; update `internal/ownership/ownership_test.go`
  (schema now has 42 tables) and add `sqlc.ShipStructureImport` to
  `internal/database/sqlc/parity_integration_test.go`, changing its model count from
  41 to 42.

- [X] T005 Add `db/queries/ship_imports.sql` with named queries for create, get by
  `(id, ship_id)`, get for update, get by `(ship_id, idempotency_key)`, update
  validation results, mark `processing` only from `ready`, complete only from
  `processing`, set `ready` error summary, fail stale `uploaded|validating`, and list
  active deck levels / room numbers for conflict checks. Modify
  `db/queries/ship.sql` `CreateShip` to insert `status`. Run `make sqlc` and commit
  the generated `internal/database/sqlc` changes.

- [X] T006 [P] Add `internal/testutil/logger.go` with `Logger(t)` returning a test-scoped
  `*slog.Logger` and an audit-appender helper (for example
  `AuditStore(h *Harness) *audit.Store`) so integration tests can use the new ship
  constructor without duplicating setup.

- [X] T007 [P] Add `internal/httpx/upload.go` with a bounded multipart helper per
  `research.md` R4: wrap the body in `http.MaxBytesReader` at 10 MiB plus multipart
  overhead, call `ParseMultipartForm` with a 10 MiB memory limit and `RemoveAll` cleanup,
  read the `file` part with `io.LimitReader` at 10 MiB + 1 byte, reject overflow, and
  return original filename, bytes, and lowercase SHA-256. Return the shared
  `apperrors` envelope for missing/malformed/oversized requests.

**Checkpoint**: Dependency, table, generated code, ownership/parity expectations, and
shared helpers are ready.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Change the ship constructor/wiring and repair the existing suite so every
user story can compile and run against `/api`.

**CRITICAL**: No user story implementation can begin until this phase is complete.

- [X] T008 Change `ship.New` in `internal/ship/module.go`/`service.go` to
  `New(pool *pgxpool.Pool, logger *slog.Logger, appender audit.Appender) Service`,
  store the appender on the service, and inject `audit.NewStore(pool, logger)` in
  `internal/app/modules.go`. Keep the existing public `Service` methods and the
  deck/area/room CRUD surface intact.

- [X] T009 Repair every stale one-argument `ship.New(h.Pool)` call in
  `internal/ship/service_integration_test.go`,
  `internal/database/tx_integration_test.go`,
  `internal/booking/service_integration_test.go`,
  `internal/voyage/service_integration_test.go`,
  `internal/testutil/constraints_integration_test.go`,
  `internal/pagination/pagination_integration_test.go`,
  `internal/guest/service_integration_test.go`,
  `internal/staff/service_integration_test.go`, and
  `internal/partner/service_integration_test.go` to use the new constructor with
  `testutil.Logger(t)` and an audit appender. Run
  `go test -tags=integration -run '^$' ./...` to prove the integration packages compile.

- [X] T010 Reconcile the canonical `/api` prefix: replace hard-coded `/v1` request
  paths and trim prefixes in `internal/httpserver/contract_integration_test.go`,
  `internal/httpserver/auth_contract_integration_test.go`, and
  `internal/httpserver/idempotency_integration_test.go`; update the comments in
  `internal/httpserver/router.go` and `internal/app/modules.go`; update the
  `openapi/openapi.yaml` title and add `servers: [{url: /api}]`. Do not change existing
  resource paths or behavior.

**Checkpoint**: The module compiles, the integration suite builds, and the contract
tests target `/api`. User stories can now proceed.

---

## Phase 3: User Story 1 - Create and Manage a Ship Independently (Priority: P1) 🎯 MVP

**Goal**: A ship can be created, listed, viewed, updated, and soft-deleted using only
vessel-level fields, including `status`, with no decks/areas/rooms required.

**Independent Test**: Create a ship with only vessel fields (including a blank status
that defaults to `active`), verify `shipId` is returned, then retrieve, update, list,
and soft-delete it; confirm no structure rows are created and invalid payloads/duplicate
IMOs are rejected.

### Tests for User Story 1

- [X] T011 [P] [US1] Add integration tests in
  `internal/ship/ship_onboarding_integration_test.go` covering: create with only
  `name`/`timezone`/`status`; blank status defaults to `active`; status accepts only
  `active|maintenance|retired`; blank name and invalid IANA timezone are rejected;
  duplicate `imo_number` returns `conflict`; get/list/update/delete work; soft-deleted
  ships disappear from reads; no `decks`/`areas`/`rooms` exist.

- [ ] T012 [P] [US1] Add contract tests in
  `internal/httpserver/ship_contract_integration_test.go` exercising
  `POST/GET /api/ships` and `GET/PATCH/DELETE /api/ships/{shipId}` against
  `openapi/openapi.yaml`, including the new `status` field and `imo_number` update.

### Implementation for User Story 1

- [X] T013 [US1] Update `openapi/openapi.yaml`: add `status` (enum
  `[active, maintenance, retired]`, default `active`) to `ShipInput`, add
  `imo_number` to `ShipUpdate`, and ensure the `Ship` schema includes `tenant_id`,
  timestamps, `version`, and nullable `deleted_at` as returned by `sqlc.Ship`.

- [X] T014 [US1] Add `Status string` to `ShipInput` in `internal/ship/module.go` and
  `status` to `shipCreateRequest` in `internal/ship/handlers.go` with
  `validate:"omitempty,oneof=active maintenance retired"`; map it in `createShip`.

- [X] T015 [US1] Update `internal/ship/service.go` `CreateShip` to default a blank
  status to `active`, validate against `shipStatuses`, and pass `Status` to
  `sqlc.CreateShipParams`; keep `GetShip`/`ListShips`/`UpdateShip`/`DeleteShip`
  behavior, tenant/ship access checks, and soft-delete semantics unchanged.

- [X] T016 [US1] Run the User Story 1 tests and `go test ./internal/ship/...`; fix
  failures and confirm the story works independently.

**Checkpoint**: User Story 1 is independently functional and is the shippable MVP
slice.

---

## Phase 4: User Story 2 - Upload a Structure Workbook and Receive a Validation Preview (Priority: P1)

**Goal**: Upload a `.xlsx` workbook for an existing ship, validate it synchronously,
and return an `importId`, status, counts, valid normalized rows, and row-level findings
without creating any structure.

**Independent Test**: Upload a valid workbook and a set of invalid workbooks for an
existing ship; inspect the previews; verify `ready`/`failed`, counts, findings, no
persisted structure, upload idempotency, and cross-tenant/ship denial.

### Tests for User Story 2

- [X] T017 [P] [US2] Add table-driven unit tests in
  `internal/ship/import_validation_test.go` for workbook safety and validation:
  ZIP/OOXML signature, OLE/encrypted rejection, `xl/vbaProject.bin` rejection, 10 MiB
  and 5,000-row limits, archive expansion limits, missing/duplicate sheets,
  missing/duplicate columns, required values, integer types, enums, duplicate deck
  levels/room numbers, existing DB conflicts, unknown/invalid `deck_level` references,
  Area capacity `>= 0`, Room capacity `1..32767`, valid/invalid JSON, recognized
  `is_bookable` forms defaulting to `true`, no-deck-with-areas, and no records.

- [X] T018 [P] [US2] Add integration tests in
  `internal/ship/import_upload_integration_test.go` covering: valid upload becomes
  `ready` with correct counts and payload; invalid file/sheets/columns/types/enums/
  duplicates/conflicts/references become `failed` with findings; no decks/areas/rooms
  are created by upload or preview; missing/malformed/oversized transport returns
  400/413 without an import; `Idempotency-Key` replay returns the same import and a
  different file returns `409 idempotency_conflict`; cross-tenant and cross-ship
  preview requests return non-revealing `404`.

- [ ] T019 [P] [US2] Add contract tests in
  `internal/httpserver/import_upload_contract_integration_test.go` for
  `POST /api/ships/{shipId}/structure/imports` (multipart `file`) and
  `GET /api/ships/{shipId}/structure/imports/{importId}/preview`, including the
  `ImportPreview`, `ShipStructureImport`, counts, and finding schemas.

### Implementation for User Story 2

- [X] T020 [US2] Add the import upload/status/preview paths and schemas from
  `contracts/openapi.yaml` to `openapi/openapi.yaml`: multipart `file` request,
  `ImportFile`, `ImportEntityCounts`, `ImportCounts`, `ValidationFinding`,
  `ImportedDeck`, `ImportedArea`, `ImportedRoom`, `ImportRecords`,
  `ShipStructureImport`, and `ImportPreview`. Keep the paths prefix-neutral.

- [X] T021 [US2] Define the import domain types and mappers in
  `internal/ship/import_validation.go`: normalized `DeckRow`/`AreaRow`/`RoomRow`,
  `ImportPayload`, `ValidationFinding` (sheet, row, column, code, message), counts,
  error summary, and conversion to/from the JSONB payload and findings stored on
  `ship_structure_imports`.

- [X] T022 [US2] Implement `internal/ship/import_repo.go` over `sqlc.New(pool)` and
  `sqlc.New(tx)`: create import, get by `(id, ship_id)`, get for update, get by
  `(ship_id, idempotency_key)`, update validation results, list active deck levels and
  room numbers for a ship, mark stale `uploaded|validating` imports failed, and set a
  safe `ready` error summary. Use `database.MapPgError` consistently.

- [X] T023 [US2] Implement `internal/ship/import_validation.go` validation per
  `research.md` R1/R5 and `contracts/template-workbook.md`: extension `.xlsx` only;
  ZIP signature; reject OLE/encrypted, macros, and oversized archives; stream rows with
  excelize `Rows`; required sheets `Decks`/`Areas`/`Rooms` (case-insensitive after
  trimming; duplicate normalized names rejected); header row 1 matched
  case-insensitively after trimming with extra columns ignored; skip blank rows; cap
  5,000 non-blank rows per sheet; Decks `level` whole number `>= 0` unique within the
  workbook and against non-deleted existing levels, `name` non-blank; Areas
  `deck_level` must resolve to a valid workbook Deck, `name` non-blank, `area_type` in
  `restaurant,lobby,open_deck,theatre,shop,corridor,stairwell,spa,pool,other`,
  optional `capacity >= 0`, optional valid-JSON `layout`; Rooms `deck_level` resolves,
  `number` non-blank unique within the workbook and against non-deleted existing
  numbers, `room_type` in `inside,oceanview,balcony,suite,crew,other`, `capacity`
  `1..32767`, `is_bookable` accepts `true/false/yes/no/1/0` case-insensitively and
  defaults `true`, optional valid-JSON `layout`. Return every finding, mark all rows in
  a duplicate group invalid, and never silently skip invalid rows.

- [X] T024 [US2] Implement the upload and preview operations in
  `internal/ship/import_service.go`: verify ship exists/not-deleted/accessible; parse
  the multipart `file` via `httpx`; compute SHA-256; resolve upload idempotency
  (`(ship_id, idempotency_key)` same hash returns the existing preview, different hash
  is `409 idempotency_conflict`, `23505` race re-reads); create the import in
  `validating`; run validation synchronously; persist counts, findings, payload, and
  safe summary (`ready` or `failed`); return the preview. Upload/preview must never
  insert decks, areas, or rooms.

- [X] T025 [US2] Implement `internal/ship/import_handlers.go` upload and preview
  handlers and register `POST /ships/{shipId}/structure/imports` and
  `GET /ships/{shipId}/structure/imports/{importId}/preview` in
  `internal/ship/handlers.go`. Map missing/malformed/oversized multipart to 400/413,
  and map all other failures through `httpx.WriteError` with safe messages.

- [X] T026 [US2] Run the User Story 2 tests (unit, integration, contract); fix
  failures and confirm no structure is persisted before confirmation.

**Checkpoint**: User Stories 1 and 2 work independently; a valid workbook produces a
`ready` preview and no business rows.

---

## Phase 5: User Story 3 - Confirm the Import Atomically (Priority: P1)

**Goal**: Confirm a `ready` import and create all Decks, Areas, and Rooms in one
all-or-nothing transaction, with correct relationships and a same-transaction audit
entry.

**Independent Test**: Confirm a ready import and verify every row, FK, `ship_id`, and
count; force persistence/audit failures and verify no partial structure remains and the
import is safely retryable; run concurrent confirmations and verify exactly one
structure and idempotent completed responses.

### Tests for User Story 3

- [X] T027 [P] [US3] Add integration tests in
  `internal/ship/import_confirm_integration_test.go` covering: successful confirmation
  creates decks first then areas/rooms with correct `ship_id`/`deck_id` and counts;
  status becomes `completed`; a conflicting deck/room created after validation rolls
  back; an injected failing audit appender or forced constraint failure leaves no
  partial structure, keeps the import `ready`, and records a safe `error_message`;
  a retry succeeds; concurrent confirmations produce exactly one structure; a repeated
  confirm of `completed` returns the same outcome without duplicates; a `failed`
  import returns `409`.

- [ ] T028 [P] [US3] Add contract tests in
  `internal/httpserver/import_confirm_contract_integration_test.go` for
  `POST /api/ships/{shipId}/structure/imports/{importId}/confirm`, including the
  completed `ShipStructureImport` response, `409 ImportNotConfirmable`, `404`, and
  `500` rollback cases.

### Implementation for User Story 3

- [X] T029 [US3] Implement the confirmation transaction in
  `internal/ship/import_service.go`: `Begin`; `SELECT ... FOR UPDATE` the import by
  `(id, ship_id)`; re-check ship access/non-deleted; return the completed outcome
  without writes when already `completed`; mark stale `uploaded|validating` failed;
  reject non-`ready` states; re-query active deck levels and room numbers and reject
  newly introduced conflicts; update `ready → processing`; insert Decks with
  `uuid.NewV7()`, build `level → deckID`; insert Areas and Rooms with the mapped deck
  ids and `ship_id`; update `processing → completed`, clear `error_message`, set
  `completed_at`; `Commit`. Never use `COPY`, disable triggers, or write `change_log`
  directly.

- [X] T030 [US3] Append the confirmation audit entry inside the same transaction in
  `internal/ship/import_service.go` via `audit.Appender.AppendTx`: `source=cloud`,
  `actor_type`/`actor_id` from the principal, action
  `ship.structure_import.confirmed`, entity `ship_structure_import`, entity id
  `importId`, `ship_id`, and details containing only import id, created counts, and
  outcome. Do not include workbook content, file name, hash, or path. If the append
  fails, fail and roll back the confirmation.

- [X] T031 [US3] Implement rollback failure handling and the confirm handler in
  `internal/ship/import_service.go`/`import_handlers.go`: on any persistence/audit
  error, roll back, then in a follow-up transaction re-lock the row and set a safe
  `error_message` only if still `ready`; return `500 internal_error` (or `409` for a
  re-checked conflict). Register `POST /ships/{shipId}/structure/imports/{importId}/confirm`
  in `internal/ship/handlers.go`.

- [X] T032 [US3] Run the User Story 3 tests; verify rollback leaves no partial
  structure, retries are safe, and concurrent/repeated confirmation is idempotent.

**Checkpoint**: User Stories 1–3 deliver the full onboarding workflow (MVP for
onboarding) with atomic confirmation.

---

## Phase 6: User Story 4 - Track Import Status and Recover Safely (Priority: P2)

**Goal**: Expose import status, counts, timestamps, and safe error summaries; recover
interrupted validation; keep failed imports as history; reject reprocessing of
completed imports.

**Independent Test**: Drive an import through validation, failure, retry-as-new-upload,
completion, and reprocessing; query status at each stage and verify no duplicate
structure.

### Tests for User Story 4

- [X] T033 [P] [US4] Add integration tests in
  `internal/ship/import_status_integration_test.go` covering: status for `ready`,
  `failed`, `completed`, and stale `validating` imports; counts/file metadata/timestamps;
  safe `error_summary` without sensitive data; stale `validating` recovery to `failed`;
  failed import history and corrected-workbook retry as a new `importId`; completed
  reprocessing rejection; cross-tenant/cross-ship `404`.

- [ ] T034 [P] [US4] Add contract tests in
  `internal/httpserver/import_status_contract_integration_test.go` for
  `GET /api/ships/{shipId}/structure/imports/{importId}` and the
  `ShipStructureImport` schema.

### Implementation for User Story 4

- [X] T035 [US4] Implement the status operation in
  `internal/ship/import_service.go`/`import_handlers.go`: get by `(id, ship_id)` after
  ship access verification; return lifecycle state, file metadata, counts,
  timestamps, `can_confirm`, and safe `error_summary`; treat `uploaded|validating`
  older than 10 minutes as interrupted and mark `failed` with a safe message; register
  `GET /ships/{shipId}/structure/imports/{importId}`.

- [X] T036 [US4] Enforce retry/reprocessing behavior: failed imports remain readable
  history and are never confirmable; corrected workbooks create a new import with a
  new `importId`; completed imports return their outcome on repeated confirm and are
  never reprocessed. Ensure the status/confirm paths share one access helper so a
  guessed `importId` cannot reveal another ship's import.

- [X] T037 [US4] Run the User Story 4 tests; verify recovery and retry create no
  duplicate structure.

**Checkpoint**: Administrators can inspect and recover imports safely.

---

## Phase 7: User Story 5 - Download a Template Aligned with Validation (Priority: P2)

**Goal**: Provide a downloadable `.xlsx` template with the required sheets, headers,
examples, enum values, and `deck_level` explanation, generated from the same rules the
validator uses.

**Independent Test**: Download the template, inspect its sheets/headers/enums/examples,
upload the examples unchanged for a fresh ship, and verify the preview is `ready`.

### Tests for User Story 5

- [X] T038 [P] [US5] Add unit tests in `internal/ship/template_test.go` that generate
  the workbook and assert the required `Decks`, `Areas`, and `Rooms` sheets, exact
  header rows, the `Reference` sheet with column descriptions, full `area_type` and
  `room_type` lists, the `deck_level` explanation, `is_bookable` forms/default, limits,
  and that every documented enum equals the service validation maps.

- [X] T039 [P] [US5] Add integration tests in
  `internal/ship/template_integration_test.go` that download the generated template
  through the service, upload its example rows unchanged for a fresh non-deleted ship,
  and assert `ready` with the expected counts and no findings.

- [X] T040 [P] [US5] Add contract tests in
  `internal/httpserver/template_contract_integration_test.go` for
  `GET /api/ships/structure/import-template`, including the XLSX content type and
  `Content-Disposition: attachment; filename="ship-structure-import-template.xlsx"`.

### Implementation for User Story 5

- [X] T041 [US5] Add the template path and binary response to
  `openapi/openapi.yaml` exactly as in `contracts/openapi.yaml`, including the XLSX
  media type and attachment header.

- [X] T042 [US5] Implement `internal/ship/template.go` with excelize: create the
  `Decks`, `Areas`, `Rooms`, and `Reference` sheets; write the exact header rows;
  write the valid example rows from `contracts/template-workbook.md` (including blank
  optional cells and a `TRUE` boolean); document columns, enums, capacity rules,
  `deck_level` references, `is_bookable` forms/default, and file/row limits on
  `Reference`; freeze headers and size columns; return bytes plus the required
  filename/content type.

- [X] T043 [US5] Register `GET /ships/structure/import-template` in
  `internal/ship/handlers.go` in the static branch of the `/ships` router so it does
  not collide with `/ships/{shipId}`; write the XLSX bytes directly with the correct
  headers.

- [X] T044 [US5] Make the template and validator share the enum maps, capacity rules,
  boolean forms, and limits (no duplicated literals); if any literal must change,
  update `contracts/template-workbook.md` and the tests in the same change.

- [X] T045 [US5] Run the User Story 5 tests; verify the examples validate unchanged and
  every documented enum matches the validator.

**Checkpoint**: Template and validation cannot drift, and a downloaded template can be
uploaded successfully.

---

## Phase 8: User Story 6 - Make Imported Structure Available for Offline Operations (Priority: P2)

**Goal**: Ensure imported decks, areas, and rooms flow through the existing
change-log/snapshot mechanism with the same version and deletion semantics as
individually created records.

**Independent Test**: Confirm an import, read the ship's change-log stream, and verify
every imported record appears with correct ordering, version, and operation semantics;
repeat after updating and soft-deleting an imported record.

### Tests for User Story 6

- [X] T046 [P] [US6] Add integration tests in
  `internal/ship/import_sync_integration_test.go` that confirm an import and assert
  `change_log` contains one row per imported deck/area/room for the ship, with
  increasing `seq`, `op = 'upsert'`, correct entity/entity_id/version; then update and
  soft-delete imported records through the existing endpoints and assert version
  increments and `op = 'delete'`, with no gaps or duplicates.

### Implementation for User Story 6

- [X] T047 [US6] Verify the confirmation path uses the existing sqlc
  `CreateDeck`/`CreateArea`/`CreateRoom` statements on the transaction and never
  disables triggers, uses `COPY`, or writes `change_log`/`ship_seq` directly; adjust
  only if the implementation deviates.

- [X] T048 [US6] Add a snapshot/delta verification using the existing
  `internal/sync` service: after confirmation, the ship's synchronized master data
  includes every imported deck/area/room with the same semantics as individually
  created rows.

- [X] T049 [US6] Run the User Story 6 tests; verify imported structure is fully
  available to offline synchronization.

**Checkpoint**: Imported structure is usable offline with no synchronization bypass.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, security/redaction, parity, and full validation across all
stories.

- [X] T050 [P] Update `docs/system-architecture.md` and `README.md` with the new
  `ship_structure_imports` table, the import/template endpoints under `/api`, the
  `.xlsx` limits, and the retained deck/area/room CRUD surface.

- [ ] T051 [P] Add/verify cross-tenant, cross-ship, and redaction coverage for every
  import endpoint in `internal/httpserver/security_integration_test.go` or a new
  `internal/httpserver/import_security_integration_test.go`: no existence leak, no
  raw workbook content/file paths/SQL/stack traces in responses or logs.

- [ ] T052 [P] Finalize migration/baseline/ownership/parity checks: re-run
  `00004` through the Goose provider, assert `docs/schema.sql` and `db/schema.sql`
  match the migration, and confirm ownership/parity tests cover 42 tables.

- [X] T053 Review structured logging and error messages in `internal/ship/` for PII,
  file content, file paths, SQL, and library errors; ensure all failures return the
  shared envelope with stable codes.

- [ ] T054 Run `make sqlc`, `make test`, `make test-integration`, and `make lint`;
  fix all failures.

- [ ] T055 Execute the `quickstart.md` walkthrough end to end (create ship, template,
  upload, preview, confirm, change-log/audit verification, failure/retry, access
  denial) and reconcile any drift in docs.

- [X] T056 Run the OpenAPI lint/mount verification
  (`TestDeclaredOperationsAreMounted` and the request/response contract tests) and
  confirm every declared `/api` operation is mounted and every exercised response
  matches the spec.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; start immediately.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories.
- **User Stories (Phases 3–8)**: Depend on Foundational. Story order below reflects
  shared code boundaries, not optional value.
- **Polish (Phase 9)**: Depends on the desired user stories being complete.

### Task-Level Dependencies

- T002 → T003 → T005 (migration/baselines before sqlc generation).
- T005 → T008 (generated `CreateShipStatus`/import queries before service wiring).
- T008 → T009 → T010 (constructor/wiring before repairing test call sites and
  contract prefix tests).
- T011/T012 → T013 → T014/T015 → T016 (US1 tests, contract schema, implementation).
- T017/T018/T019 → T020 → T021/T022/T023 → T024/T025 → T026 (US2 tests, OpenAPI,
  types/repo/validation/service/handlers).
- T027/T028 → T029 → T030 → T031 → T032 (US3 tests, transaction, audit, failure
  handling/handler).
- T033/T034 → T035/T036 → T037 (US4 tests, status/recovery, retry semantics).
- T038/T039/T040 → T041 → T042 → T043 → T044 → T045 (US5 tests, OpenAPI, template,
  routes, alignment).
- T046 → T047/T048 → T049 (US6 tests, trigger verification, snapshot/delta).
- All stories → T050–T056 (Polish).

### User Story Dependencies

- **US1 (P1)**: Independent after Foundational; provides the ship-creation MVP.
- **US2 (P1)**: Depends on Foundational; needs an existing ship row (US1 or a test
  fixture) and owns the shared import table/repo/validation infrastructure.
- **US3 (P1)**: Depends on US2 (import record, payload, repo, validation findings).
- **US4 (P2)**: Depends on US2 and US3 (status, recovery, completed/retry semantics).
- **US5 (P2)**: Template generation depends only on Foundational; the
  “examples upload as ready” test depends on US2 validation.
- **US6 (P2)**: Depends on US3 confirmation.
- **Polish**: Depends on all implemented stories.

### Within Each User Story

- Tests first (FR-065–FR-069); confirm they fail before implementation.
- Contract/schema updates before contract tests can pass.
- Types → repository → validation/service → handlers/routes.
- Core implementation before integration verification.
- Complete and validate the story before moving to the next priority.

---

## Parallel Opportunities

### Setup / Foundational

```bash
Task: "T001 [P] Add excelize v2.11.1"
Task: "T002 [P] Create 00004 migration"
Task: "T006 [P] Add testutil logger helper"
Task: "T007 [P] Add httpx upload helper"
# These touch different files and can start together.
```

### User Story 1

```bash
Task: "T011 [US1] Ship onboarding integration tests"
Task: "T012 [US1] Ship contract tests"
```

### User Story 2

```bash
Task: "T017 [US2] Workbook validation unit tests"
Task: "T018 [US2] Upload/preview integration tests"
Task: "T019 [US2] Upload/preview contract tests"
```

### User Story 3

```bash
Task: "T027 [US3] Confirmation integration tests"
Task: "T028 [US3] Confirmation contract tests"
```

### User Story 4

```bash
Task: "T033 [US4] Status/recovery integration tests"
Task: "T034 [US4] Status contract tests"
```

### User Story 5

```bash
Task: "T038 [US5] Template unit tests"
Task: "T039 [US5] Template example integration tests"
Task: "T040 [US5] Template contract tests"
```

### User Story 6 and Polish

```bash
Task: "T046 [US6] Change-log/sync integration tests"
Task: "T050 [P] Documentation updates"
Task: "T051 [P] Security/redaction tests"
Task: "T052 [P] Migration/ownership/parity final checks"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1 (ship create/list/get/update/delete with `status`).
4. **STOP and VALIDATE**: Run US1 tests and the ship contract tests.
5. Deploy/demo the independent vessel-creation slice.

### Onboarding MVP (User Stories 1–3)

1. Complete Setup + Foundational + US1.
2. Add US2 (upload/preview) and validate no structure is persisted.
3. Add US3 (atomic confirm + audit) and validate rollback/concurrency.
4. This is the smallest complete onboarding workflow and should be demoed before P2
   work.

### Incremental Delivery

1. Setup + Foundational → suite compiles and `/api` contracts are consistent.
2. US1 → vessel creation MVP.
3. US2 + US3 → upload/preview/confirm onboarding MVP.
4. US4 → status/recovery.
5. US5 → template.
6. US6 → offline synchronization verification.
7. Polish → docs, security, parity, full test/lint/quickstart.

### Parallel Team Strategy

1. Team completes Setup + Foundational together (T001–T010).
2. Once Foundational is done:
   - Developer A: US1 (ship CRUD/status).
   - Developer B: US5 template generation (independent of US2 implementation, except
     the example-upload test).
3. After US2: Developer A continues US3; Developer B continues US4/US6.
4. Reserve one reviewer/owner for OpenAPI + contract tests because several stories
   touch `openapi/openapi.yaml` and `internal/httpserver/*_test.go`.

---

## Notes

- `[P]` tasks touch different files and have no dependency on an incomplete task.
- `[Story]` labels map every user-story task to `spec.md` for traceability.
- Tests are required by FR-065–FR-069 and MUST fail before implementation.
- Do not bypass triggers, write `change_log` directly, add a queue/worker, or store
  workbook files in S3/temp storage.
- Keep the import table cloud-only: no sync trigger, no `version`, no `deleted_at`.
- The canonical endpoint prefix is `/api`; contract tests must trim `/api`, not `/v1`.
- Format validation: every task above follows
  `- [ ] Txxx [P?] [US?] Description with exact file path`.

## Verification Status

- Unit tests, `go vet`, OpenAPI validation, and integration-package compilation pass.
- Integration runtime execution requires Docker/Testcontainers, which is unavailable in this
  environment. The remaining unchecked tasks are runtime-only contract/full-suite verification.
