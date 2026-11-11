# Import Lifecycle Contract

This document defines the observable behavior of the structure-import endpoints. It
complements `contracts/openapi.yaml` and is binding for implementation and tests.

## Endpoints

| Method | Path | Behavior |
|---|---|---|
| POST | `/api/ships` | Create a ship with optional `status` (default `active`); no structure required |
| GET | `/api/ships` | Cursor-paginated list scoped to the principal |
| GET | `/api/ships/{shipId}` | Ship detail; 404 when missing, deleted, or out of scope |
| PATCH | `/api/ships/{shipId}` | Partial update, including `status` and `imo_number` |
| DELETE | `/api/ships/{shipId}` | Soft delete; excludes the ship from reads and imports |
| POST | `/api/ships/{shipId}/structure/imports` | Upload one `.xlsx` (`file` part) and validate synchronously |
| GET | `/api/ships/{shipId}/structure/imports/{importId}` | Import status |
| GET | `/api/ships/{shipId}/structure/imports/{importId}/preview` | Full preview with valid records and findings |
| POST | `/api/ships/{shipId}/structure/imports/{importId}/confirm` | Atomically create all structure |
| GET | `/api/ships/structure/import-template` | Download the XLSX template |

The existing per-record deck/area/room endpoints remain available and unchanged; the
onboarding workflow does not remove them.

## Lifecycle states

```text
uploaded ──► validating ──► ready ──► processing ──► completed
                  │                        │
                  └──► failed              └── (rollback) ready + error_message
```

- **uploaded**: reserved for a future asynchronous upload handoff; the synchronous
  implementation creates the row directly in `validating`.
- **validating**: the upload request is parsing/validating. A row left in this state
  past the stale threshold (10 minutes) means the process was interrupted.
- **ready**: every record is valid and the import is confirmable.
- **failed**: validation/file failure, or an interrupted validation recovered as
  failed. Terminal for that `importId`; retry is a new upload with a new `importId`.
- **processing**: transient state written inside the confirmation transaction while
  the row is locked. It commits only with `completed`.
- **completed**: terminal; all structure and the audit entry are committed.

Allowed transitions enforced by the service:

| From | To | Trigger |
|---|---|---|
| `validating` | `ready` | All file/sheet/row checks pass and at least one record exists |
| `validating` | `failed` | Any file/sheet/row finding, or stale recovery |
| `uploaded` | `failed` | Stale recovery (only if a future async path creates this state) |
| `ready` | `processing` | Confirmation transaction after row lock and conflict re-check |
| `processing` | `completed` | Structure + audit inserted and transaction commits |
| `processing` | `ready` (rollback) | Any persistence/audit error; all writes rolled back |
| `completed` | `completed` | Repeated confirmation; returns the existing outcome, no writes |

A confirmation failure does **not** persist `failed`. It records a safe
`error_message` in a follow-up transaction that only writes while the row is still
`ready`. This makes the same import safely retryable (FR-045, US4.4) and avoids a
crash-orphaned `processing` row.

## Upload behavior

1. Resolve the ship, verify it exists, is not soft-deleted, and is accessible.
2. Parse the multipart request (field `file`); enforce 10 MiB. Missing/malformed/
   oversized transport → 400/413 with no import row.
3. Read the bytes, compute SHA-256, sanitize the filename for display, and create the
   import row in `validating`.
4. If `Idempotency-Key` is present and an import for `(ship_id, key)` exists:
   - same SHA-256 → return that import's preview; do not re-validate;
   - different SHA-256 → `409 idempotency_conflict`;
   - race on insert (`23505`) → re-read and apply the same rule.
5. Validate the workbook synchronously (see `template-workbook.md`).
6. Persist counts, findings, safe summary, and `validated_payload`; set `ready` or
   `failed`.
7. Return `201` with the import preview in both cases. A `failed` import is history;
   confirmation is blocked.

A corrected workbook is a new upload with a new `importId`. Because the unique key is
per `(ship_id, idempotency_key)`, the client must use a new `Idempotency-Key` (or omit
it) when the file content changes; reusing the key with a different SHA-256 is a
`409 idempotency_conflict` by design.

Accepted-but-invalid files return `201` rather than `4xx` because the import resource
was created and the client needs the `importId` to inspect the findings. Transport
failures (no file, malformed multipart, >10 MiB) return `400`/`413` and create no
import.

## Confirmation behavior

One transaction:

1. `SELECT ... FOR UPDATE` the import by `id` and `ship_id`.
2. Re-check ship access and non-deleted state.
3. Completed → return the completed import (idempotent, no writes).
4. Stale `validating`/`uploaded` → mark `failed`, return `409`.
5. Not `ready` → `409` (includes `failed` and in-flight `processing`).
6. Re-query existing non-deleted deck levels and room numbers; if the payload now
   conflicts, roll back and record a safe `error_message`, return `409`.
7. Set `processing`.
8. Insert Decks with UUID v7; build `level → deckID`.
9. Insert Areas and Rooms with the mapped deck IDs and `ship_id`.
10. Append the audit entry in the same transaction.
11. Set `completed`, clear `error_message`, set `completed_at`; commit.

Any error → rollback everything, then record a safe `error_message` if the row is
still `ready`, and return `500` (or `409` for a re-checked conflict). No partial
structure may remain. Repeating confirmation after a failure is safe.

## Concurrency

- Two confirmations: the second blocks on the row lock; after the first commits it
  sees `completed` and returns the same outcome. Exactly one structure is created.
- Confirmation vs. unrelated structure creation: the conflict re-check and the unique
  indexes (`uq_decks_level`, `uq_rooms_number`) are authoritative; a conflict rolls
  back the whole import.
- Upload vs. confirmation: confirmation re-checks all conflicts, so a change after
  validation cannot produce an invalid structure.
- Crash during confirmation: the transaction rolls back; the import stays `ready`
  (or is already `completed` if it committed). There is no durable `processing` orphan.
- Crash during validation: the row stays `validating`; a later status/preview/confirm
  marks it `failed` once stale (>10 minutes) with a safe message. No structure exists.

## Access and error mapping

Every import operation requires both the path `shipId` and `importId`. The service
first verifies ship access, then looks up `(importId, shipId)`. A missing import, an
import belonging to another ship, a soft-deleted ship, or an out-of-scope tenant/ship
all return the same non-revealing `404 not_found`.

| Situation | Status | Error code |
|---|---|---|
| Missing/malformed multipart or missing `file` | 400 | `validation_failed` |
| Upload >10 MiB | 413 | `validation_failed` |
| Invalid ship payload | 422 | `validation_failed` |
| Duplicate IMO | 409 | `conflict` |
| Upload idempotency key with different file | 409 | `idempotency_conflict` |
| Ship/import missing or not accessible | 404 | `not_found` |
| Confirm a failed/validating/processing import | 409 | `conflict` |
| Confirm while a concurrent confirmation is in flight | 409 | `conflict` |
| Confirmation persistence/audit failure | 500 | `internal_error` |
| Confirmation discovers a new structure conflict | 409 | `conflict` |
| Repeat confirm of a completed import | 200 | — (completed outcome) |

Responses never include raw workbook content, file paths, SQL, stack traces, library
errors, or secrets. `error_summary` and finding messages are curated.

## Audit entry

On successful confirmation only, appended through `audit.Appender.AppendTx` in the
same transaction:

```json
{
  "source": "cloud",
  "actor_type": "admin",
  "actor_id": "<principal actor id, when present>",
  "action": "ship.structure_import.confirmed",
  "entity": "ship_structure_import",
  "entity_id": "<importId>",
  "ship_id": "<shipId>",
  "details": {
    "import_id": "<importId>",
    "created": { "decks": 2, "areas": 10, "rooms": 100 },
    "outcome": "completed"
  }
}
```

If the audit append fails, the confirmation fails and rolls back (FR-071).
