# Research: 004-ship-structure-import

Phase 0 output. Decisions below resolve the plan's open questions and record the
trade-offs that keep the feature inside the existing modular monolith, PostgreSQL
baseline, synchronization contract, and constitution.

## R1. Excel library and workbook safety

**Decision**: Use `github.com/xuri/excelize/v2` pinned to `v2.11.1` as a new direct
dependency. Parse with the streaming `Rows` API, not `GetRows`, so the 5,000-row cap
is enforced while iterating. Generate the template with the same library.

**Rationale**: excelize is pure Go, actively maintained, reads OOXML without CGO,
and writes styled workbooks for the template. Keeping one library for reading and
writing guarantees the template and validator stay aligned (FR-013). The library
does not execute macros and never evaluates formulas.

**Safety checks before handing bytes to excelize** (FR-026/FR-028):

1. Reject files that do not start with the ZIP local-file signature `PK\x03\x04`.
   OLE/compound files (encrypted OOXML) start with `D0 CF 11 E0` and are rejected.
2. Open the archive with `archive/zip`, reject any entry whose lower-cased name is
   `xl/vbaproject.bin` (macro-enabled), and reject archives with abnormal paths.
3. Sum `UncompressedSize64`; reject if the archive exceeds 64 MiB uncompressed or
   any entry exceeds 32 MiB. This bounds zip-bomb expansion before parsing.
4. Enforce 10 MiB compressed upload and 5,000 non-blank data rows per required sheet.
5. Use `excelize.OpenReader` with default options; reject any open/parse error as a
   safe validation failure. Never echo the library error text to the client.

**Alternatives considered**: `tealeg/xlsx` (smaller feature set, weaker template
support); shelling out to LibreOffice (new infrastructure, violates simplicity);
CSV (explicitly out of scope); accepting any file whose extension is `.xlsx`
(rejected by FR-026).

## R2. Import tracking table and validated payload

**Decision**: Add one cloud-only table, `ship_structure_imports`, in migration
`00004_ship_structure_imports.sql`, and update both schema baselines
(`docs/schema.sql`, `db/schema.sql`). The table owns lifecycle state, safe file
metadata, per-entity counts, typed findings, an optional idempotency key, and a
`validated_payload jsonb` containing only the normalized rows that passed validation.

**Payload justification (FR-048)**: The payload is required import metadata, not a
staging copy of business records:

- FR-036 requires preview data to remain available after the upload response so the
  administrator can review and confirm later. The feature deliberately does not add
  S3 or local file storage (no concrete requirement, and constitution X favors the
  simplest implementation), so the normalized result must be persisted.
- FR-041/FR-043 require later transactional confirmation. Re-parsing the file at
  confirmation would make the confirm transaction depend on a file that no longer
  exists and would allow validation and persistence to diverge.
- The payload is never read as the source of truth for decks, areas, or rooms, is
  never synchronized to a hub, is excluded from every snapshot/change-log path, and
  is superseded by the real rows created at confirmation. It is retained only as
  import history, exactly like the findings and counts.
- The spec assumption is refined to state this explicitly (see `spec.md` Assumptions).

**Table shape**:

```sql
CREATE TABLE ship_structure_imports (
  id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  ship_id           uuid NOT NULL REFERENCES ships(id),
  idempotency_key   text,
  status            text NOT NULL DEFAULT 'validating'
                      CHECK (status IN ('uploaded','validating','ready','failed','processing','completed')),
  file_name         text NOT NULL,
  file_size_bytes   bigint NOT NULL CHECK (file_size_bytes >= 0),
  file_sha256       text NOT NULL CHECK (file_sha256 ~ '^[0-9a-f]{64}$'),
  decks_total       integer NOT NULL DEFAULT 0 CHECK (decks_total >= 0),
  decks_valid       integer NOT NULL DEFAULT 0 CHECK (decks_valid >= 0),
  decks_invalid     integer NOT NULL DEFAULT 0 CHECK (decks_invalid >= 0),
  areas_total       integer NOT NULL DEFAULT 0 CHECK (areas_total >= 0),
  areas_valid       integer NOT NULL DEFAULT 0 CHECK (areas_valid >= 0),
  areas_invalid     integer NOT NULL DEFAULT 0 CHECK (areas_invalid >= 0),
  rooms_total       integer NOT NULL DEFAULT 0 CHECK (rooms_total >= 0),
  rooms_valid       integer NOT NULL DEFAULT 0 CHECK (rooms_valid >= 0),
  rooms_invalid     integer NOT NULL DEFAULT 0 CHECK (rooms_invalid >= 0),
  validated_payload jsonb,
  validation_errors jsonb NOT NULL DEFAULT '[]'::jsonb,
  error_message     text,
  created_by        text,
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now(),
  completed_at      timestamptz,
  CHECK (decks_valid + decks_invalid = decks_total),
  CHECK (areas_valid + areas_invalid = areas_total),
  CHECK (rooms_valid + rooms_invalid = rooms_total)
);
CREATE UNIQUE INDEX uq_ship_structure_imports_idem
  ON ship_structure_imports (ship_id, idempotency_key)
  WHERE idempotency_key IS NOT NULL;
CREATE INDEX ix_ship_structure_imports_ship_created
  ON ship_structure_imports (ship_id, created_at DESC);
```

There is deliberately no `version`/`deleted_at`/sync trigger: the table is not
synced master data. `updated_at` is set explicitly by writes in the same style as
other non-synced operational tables, avoiding `touch_row()` (which assumes `version`).

**Ownership**: Map the table to `ModuleShip` in `internal/ownership/ownership.go`.
Update `internal/ownership/ownership_test.go` expectations (schema now has 42
tables) and `internal/database/sqlc/parity_integration_test.go` (add
`ShipStructureImport` and change the count from 41 to 42).

**Alternatives considered**: no tracking table (fails FR-035/036/047); store only
the file in S3 (adds infrastructure and credentials, no requirement); store the file
bytes in PostgreSQL (larger, still a file copy, and unnecessary); store only the file
hash and re-upload on confirm (breaks confirm-after-preview and FR-036).

## R3. Synchronous validation and lifecycle persistence

**Decision**: Upload creates the import row in `validating` in an autocommit write,
reads and validates the workbook synchronously, then updates the row to `ready` or
`failed` with counts, findings, payload, and a safe summary before responding. The
`uploaded` state remains in the CHECK for future asynchronous handoff but is not used
by this synchronous flow; `processing` is used only inside the confirmation
transaction.

**Rationale**: The clarification selected synchronous validation because no worker
or queue exists. The import record still survives restarts and supports history
(FR-047). A request that is interrupted during validation leaves a `validating` row;
because no other process validates the same import, any `validating`/`uploaded`
import older than a stale threshold can safely be marked `failed` when it is next
read or confirmed. That is a valid `validating → failed` transition (FR-039).

**Retry semantics**:

- Validation failure → `failed`; corrected workbook is uploaded as a new import with
  a new `importId` (clarification; FR-045). Failed imports stay visible as history.
- Persistence or audit failure during confirmation → the transaction rolls back, the
  import remains `ready`, a safe `error_message` is recorded (in a short follow-up
  transaction that re-locks the row and only writes if it is still `ready`), and the
  request returns an error. The same import may be confirmed again safely; no partial
  structure exists. This satisfies US3.2 and FR-045 without inventing a durable
  `processing` orphan.

**Alternatives considered**: asynchronous queue (new infrastructure, out of scope);
persisting `processing` before the write transaction (creates crash-orphaned states
and recovery complexity); marking a ready import `failed` after a transient database
error (forces a needless re-upload).

## R4. Upload transport, content validation, and file lifecycle

**Decision**: Use `multipart/form-data` with a single required field `file`. Add a
small helper to `internal/httpx` that:

1. Wraps the body with `http.MaxBytesReader` at 10 MiB plus multipart overhead.
2. Calls `ParseMultipartForm` with a 10 MiB memory limit, then `RemoveAll` on cleanup.
3. Reads the part with `io.LimitReader` at 10 MiB + 1 byte and rejects overflow.
4. Returns the original filename, bytes, and SHA-256.

The handler checks the filename extension case-insensitively (`.xlsx` only), then
runs the R1 content checks. The file is never written to disk and no S3 integration
is added, so there are no temporary files to clean up (FR-061). The in-memory buffer
is released after validation.

**Error behavior**:

- Missing field, malformed multipart, or oversized body → 400/413 with the shared
  error envelope, no import record (the request never represented a usable workbook).
- Any accepted file that fails format/content validation → the import is created and
  ends `failed`; upload responds `201` with the preview so the administrator keeps
  the `importId`, sees the findings, and can retry with a corrected workbook.
  Confirmation remains blocked (SC-005). This is deliberately not an HTTP 4xx, because
  the import resource is successfully created and recorded as history.

**Alternatives considered**: rejecting all invalid files with 422 and no import
(no history, loses the importId); storing the file in S3 and validating asynchronously
(needs AWS infrastructure and a worker); trusting `Content-Type` (not authoritative).

## R5. Validation rules, findings, and counts

**Decision**: Implement validation as a pure function over a decoded workbook plus
sets of existing non-deleted deck levels and room numbers, returning normalized
records, counts, and typed findings. Findings use stable codes and coordinates:

```json
{ "sheet": "Areas", "row": 5, "column": "area_type",
  "code": "invalid_enum", "message": "Area type must be one of: ..." }
```

`row` is the Excel 1-based row (1 = header); `row` is 0 and `column` is `""` for
file/sheet-level problems. Codes: `file_invalid`, `file_too_large`, `unsupported_format`,
`macro_enabled`, `archive_too_large`, `missing_sheet`, `duplicate_sheet`,
`missing_column`, `duplicate_column`, `too_many_rows`, `required`, `invalid_type`,
`invalid_enum`, `invalid_capacity`, `invalid_json`, `invalid_boolean`, `duplicate`,
`conflict`, `invalid_reference`, `missing_deck_records`, `no_importable_records`.

Rules:

- Required sheets `Decks`, `Areas`, `Rooms`, matched case-insensitively after
  trimming. Duplicate normalized sheet names are errors; extra sheets are ignored.
- Header row is row 1. Required columns are matched case-insensitively after trimming;
  extra columns are ignored; missing or duplicate required columns are errors.
- Completely blank data rows are skipped. More than 5,000 non-blank data rows in a
  sheet is an error and parsing stops for that sheet.
- `Decks`: `level` (whole number ≥ 0, int32) and `name` (non-blank). Levels unique in
  the workbook and not equal to an existing non-deleted deck level of the ship.
- `Areas`: `deck_level`, `name`, `area_type`, optional `capacity`, optional `layout`.
  Capacity must be a whole number ≥ 0 when present; type must be one of the existing
  `area_type` values; layout must be syntactically valid JSON when present.
- `Rooms`: `deck_level`, `number`, `room_type`, required `capacity`, optional
  `is_bookable`, optional `layout`. Number non-blank and unique in the workbook and
  against existing non-deleted room numbers; capacity a whole number 1..32767;
  type from the existing `room_type` values; `is_bookable` accepts
  true/false/yes/no/1/0 case-insensitively and defaults to `true`; layout valid JSON
  when present.
- Every Area/Room `deck_level` must reference a Deck row that is valid and importable
  in the same workbook. References to missing, duplicated, invalid, or database-
  conflicting decks are errors (FR-023, and the spec assumption that workbook
  Areas/Rooms only attach to workbook Decks).
- Duplicate keys mark every row in the duplicate group invalid. Existing-record
  conflicts mark the conflicting workbook rows invalid. No invalid row is skipped
  (FR-034).
- A workbook with no Decks but Areas/Rooms is an error. A workbook with no non-blank
  data rows at all is an error (FR-020).
- Counts are `total = valid + invalid` per entity; `invalid` counts a row with one or
  more findings. Import becomes `ready` only when there is at least one record and
  every entity has zero invalid rows and there are no global/file errors.

Values are trimmed before storage, matching the service layer. Names/numbers keep
their trimmed text. Layout is stored as raw JSON and sent to the existing jsonb column.

**Alternatives considered**: validating row-by-row with database lookups (N queries);
silently dropping invalid rows (forbidden by FR-034); accepting decimal capacities
(the existing domain is integer); requiring Areas/Rooms sheets to be non-empty
(not required by the spec).

## R6. Transactional confirmation and concurrency

**Decision**: Confirmation is one database transaction:

1. `Begin`.
2. `SELECT ... FROM ship_structure_imports WHERE id=$1 AND ship_id=$2 FOR UPDATE`.
   This is the serialization point for concurrent confirmations.
3. Re-verify the ship is accessible and not soft-deleted; re-read active deck levels
   and room numbers to close the validation-to-confirm race.
4. If status is `completed`, return the existing import (idempotent success) without
   writing. If status is not `ready`, return a conflict; if the import is stale
   `validating`/`uploaded`, mark `failed` first.
5. Verify the payload still has no conflicts; then set status `processing`.
6. Insert Decks with `uuid.NewV7()`, building a `level → deckID` map.
7. Insert Areas and Rooms using the mapped deck IDs, every row carrying `ship_id`.
8. Append the confirmation audit entry through `audit.Appender.AppendTx`.
9. Set status `completed`, clear `error_message`, set `completed_at`.
10. `Commit`.

On any error, roll back the whole transaction, then open a short follow-up
transaction that re-locks the row and records a safe `error_message` only if the
import is still `ready`. No structure exists because the write transaction is
all-or-nothing. A later confirmation may retry the same import; `completed` remains
terminal.

Because `processing` and `completed` are written in the same transaction as the
structure rows, a crash leaves either the old `ready` state with no structure or the
new `completed` state with the full structure. There is no durable orphan
`processing` state and no stale-processing recovery race. Concurrent confirmations
serialize on the row lock: the first commits, the others observe `completed` and
return the same outcome (FR-044, SC-007).

**Alternatives considered**: claiming `processing` in a separate committed write
(needs leases/recovery and can race with a long transaction); inserting with raw
SQL/COPY (bypasses sqlc and risks trigger mistakes); deferring Areas/Rooms to a
background job (breaks atomicity and needs a worker).

## R7. Audit emission

**Decision**: Extend the ship service with an injectable `audit.Appender`. On
successful confirmation, inside the same transaction, append one entry:

- `source = "cloud"`
- `actor_type = string(principal.ActorType)` and `actor_id = principal.ActorID`
  when present
- `action = "ship.structure_import.confirmed"`
- `entity = "ship_structure_import"`, `entity_id = importId`
- `ship_id = shipId`
- `details = {"import_id": "...", "created": {"decks": N, "areas": N, "rooms": N},
  "outcome": "completed"}`

No raw workbook content, file name, file hash, or file path is included. If the
append fails, the transaction rolls back and confirmation fails (FR-071). Ship CRUD
and upload/preview do not add audit entries; their observability comes from the
import record, structured logs, and the existing change log.

**Alternatives considered**: best-effort audit outside the transaction (violates
FR-071); auditing upload/preview (explicitly out of scope per clarification).

## R8. Synchronization preservation

**Decision**: Persist decks, areas, and rooms through the existing sqlc
`CreateDeck`/`CreateArea`/`CreateRoom` statements on the confirmation transaction.
Do not use `COPY`, do not disable triggers, do not write `change_log` or `ship_seq`
directly, and do not touch the synchronization tables. `ship_structure_imports` is
cloud-only and has no capture trigger, so import bookkeeping never reaches a hub.

**Rationale**: The existing `capture_change()` triggers run on normal inserts and
allocate per-ship sequence numbers in the same transaction, preserving ordering,
version, and delete semantics (FR-051–FR-053, SC-008). The import table is
orchestration data, not master data (FR-054).

**Alternatives considered**: bulk `COPY` (would require trigger workarounds);
direct `change_log` writes (forbidden by the constitution); adding a sync trigger to
the import table (leaks internal state to hubs).

## R9. API prefix reconciliation (`/api` vs `/v1`)

**Decision**: Treat `/api` as canonical, as required by FR-010 and the spec
assumption. Keep the repository-wide OpenAPI paths prefix-neutral, but update the
document metadata to say `/api`, add `servers: [{url: /api}]`, and change every
contract/integration test that hard-codes `/v1` to `/api`. Update comments in
`internal/httpserver/router.go`, `internal/app/modules.go`, and the OpenAPI title.

**Rationale**: Runtime routes are mounted under `/api` in
`internal/httpserver/router.go`; the current tests and title still say `/v1`, so the
contract suite is already inconsistent. New import/template paths must be mounted and
tested under `/api`. Reusing the existing authenticated tree avoids a second router.

**Alternatives considered**: mounting both `/v1` and `/api` (duplicate surface and
ambiguous contract); changing runtime to `/v1` (contradicts FR-010).

## R10. Idempotency

**Decision**:

- Upload accepts the existing `Idempotency-Key` header. The new import table has a
  nullable `idempotency_key` with a partial unique index per ship. In the upload
  service, compute the file SHA-256 first, then look up `(ship_id, key)`:
  - same key + same hash → return the existing import preview without re-validating;
  - same key + different hash → `409 idempotency_conflict`;
  - no key → always create a new import.
  A concurrent insert race is resolved by catching `23505` and re-reading the row.
- Confirmation is idempotent through the import lifecycle: a completed import
  returns its outcome with no new writes; a repeated confirm while ready serializes
  on the row lock and creates exactly one structure.
- Ship writes keep the existing convention: no shared idempotency store and no new
  idempotency column on `ships`; IMO uniqueness remains the only existing persistent
  guard. This matches the clarification and FR-062/FR-063.

**Rationale**: A per-import key is the same pattern the schema uses for bookings and
service bookings; it is not a shared idempotency store. It makes repeated uploads of
the same workbook cheap and prevents duplicate import history.

**Alternatives considered**: a global `idempotency_keys` table (explicitly rejected);
no upload key (duplicate history on retries); re-validating on every replay (wasteful
and can diverge if the database changed).

## R11. Ship creation status and constructor wiring

**Decision**:

- Add `Status string` to `ShipInput` and `status` to `shipCreateRequest` with the
  existing `oneof=active maintenance retired` validation. Default blank to `active`,
  validate in the service, and include `status` in the `CreateShip` sqlc INSERT.
- Change `ship.New` to `New(pool *pgxpool.Pool, logger *slog.Logger, appender audit.Appender) Service`.
  Inject `audit.NewStore(pool, logger)` from `internal/app/modules.go`.
- Keep the existing deck/area/room CRUD endpoints intact; the onboarding workflow
  adds import endpoints rather than removing the existing per-record APIs.
- Update every stale `ship.New(h.Pool)` call in integration tests to pass a logger
  and appender. Add `testutil.Logger(t)` (discard/test handler) and
  `testutil.AuditStore(h)` helpers so the call sites stay short.

**Rationale**: FR-002/003 require status on creation. The audit appender is required
by FR-071 and must be injected to keep the module testable. The stale constructor
calls currently make the integration build fail; fixing them is a prerequisite for
the feature's tests.

**Alternatives considered**: a variadic/options constructor (hides a required
dependency); keeping `ship.New` unchanged and using a package-level audit store
(hidden coupling); removing the existing deck/area/room endpoints (out of scope and
would break other modules/tests).

## R12. Template generation

**Decision**: Generate the workbook dynamically with excelize on each request:

- Sheets `Decks`, `Areas`, `Rooms`, plus a `Reference` sheet.
- Row 1 on each required sheet is the exact header row the validator expects:
  - Decks: `level`, `name`
  - Areas: `deck_level`, `name`, `area_type`, `capacity`, `layout`
  - Rooms: `deck_level`, `number`, `room_type`, `capacity`, `is_bookable`, `layout`
- Rows 2–3 contain valid examples (including a blank optional capacity/layout and a
  `TRUE` boolean) that pass validation unchanged for a fresh ship.
- `Reference` documents each column, the full `area_type` and `room_type` value
  lists, the `level ≥ 0` rule, the `capacity` rules, and that `deck_level` in Areas
  and Rooms must match a `level` in the Decks sheet.
- Freeze the header row and set reasonable column widths.
- Respond with
  `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
  and `Content-Disposition: attachment; filename="ship-structure-import-template.xlsx"`.

Extra sheets are ignored by validation, so `Reference` is safe. A test uploads the
generated examples and asserts a `ready` preview (US5.3), and a test compares the
documented enums with the service maps (FR-013).

**Alternatives considered**: a checked-in binary template (drifts from validation);
putting descriptions in data rows (would be parsed as records); omitting the
Reference sheet (fails FR-012).

## R13. Migration, baselines, and generated code

**Decision**: Deliver `db/migrations/00004_ship_structure_imports.sql` with the table
and indexes. Append the same DDL to `docs/schema.sql` (authoritative baseline) and
`db/schema.sql` (sqlc input), so the baseline and migration agree. Run
`make sqlc` to regenerate `internal/database/sqlc`. Update ownership and parity
tests as described in R2. The migration must apply cleanly on top of `00003`, and
the integration harness applies it automatically.

**Rationale**: Constitution II requires versioned migrations and baseline updates;
sqlc must see the new table for typed queries. No existing schema object is changed.

**Alternatives considered**: putting the table only in the baseline (breaks migration
history); putting it only in the migration (sqlc cannot generate models); editing
`00002` (forbidden).

## R14. Test strategy and existing-suite repairs

**Decision**: Add:

- Unit tests in `internal/ship` for workbook decoding/validation using in-memory
  generated `.xlsx` fixtures: valid workbook, missing/duplicate sheets and columns,
  bad types/enums, duplicates, DB conflicts, unknown deck references, capacity and
  JSON rules, boolean forms, row limits, empty/only-areas workbooks, macro/encrypted/
  corrupt/OLE/oversized archives.
- Integration tests (`//go:build integration`) using the existing testcontainers
  harness:
  - ship create/list/get/update/delete including status default/validation and
    duplicate IMO;
  - upload valid/invalid files, preview counts/findings, no structure before confirm;
  - confirm success with correct ship/deck FKs, counts, completed status;
  - persistence/audit failure rollback (inject a failing appender or constraint),
    leaving no partial structure and a safe `error_message`;
  - concurrent confirmations produce one structure;
  - completed confirm is idempotent; failed validation cannot be confirmed; stale
    `validating` recovery;
  - upload idempotency replay and hash mismatch;
  - change-log entries for every imported deck/area/room, plus version/deletion
    semantics after update/soft delete;
  - cross-tenant and cross-ship import access returns not-found/forbidden without
    leaking existence;
  - template download and re-upload of its examples;
  - ownership/parity/migration re-run.
- Contract tests: add the new paths to `openapi/openapi.yaml`, exercise upload,
  status, preview, confirm, and template, and switch all `/v1` test prefixes to
  `/api` (R9).

**Repairs required for the existing suite**:

- Replace every `ship.New(h.Pool)` in integration tests with the new constructor
  (logger + audit appender).
- Replace hard-coded `/v1` request paths in
  `internal/httpserver/contract_integration_test.go`,
  `internal/httpserver/auth_contract_integration_test.go`, and
  `internal/httpserver/idempotency_integration_test.go` with `/api`, and trim
  `/api` instead of `/v1` when matching the prefix-neutral OpenAPI paths.
- Add `testutil.Logger(t)` and a small audit-appender helper if needed.

**Commands**: `make sqlc`, `make test`, `make test-integration`, `make lint`.

**Alternatives considered**: testing validation only through HTTP (slow, poor
diagnostics); mocking the database for conflict rules (the constraints are the
authority); leaving the stale `/v1` and constructor issues for a separate change
(the feature's own contract suite would not compile/run).
