# Data Model: 004-ship-structure-import

Phase 1 output. The existing `ships`, `decks`, `areas`, and `rooms` tables are the
source of truth and are not redesigned. This feature adds one cloud-only orchestration
table and reuses the existing audit/change-log mechanisms.

## 1. Existing entities (unchanged)

### Ship (`ships`)

| Column | Type | Rules |
|---|---|---|
| `id` | uuid PK | Generated with `uuid.NewV7()` in the service |
| `tenant_id` | uuid FK → `tenants` | From the authenticated principal; required |
| `name` | text | Non-blank after trimming |
| `imo_number` | text, nullable unique | Existing uniqueness only; no new checksum |
| `flag_state` | text, nullable | Trimmed; blank becomes NULL |
| `timezone` | text, default `UTC` | Valid IANA name; blank becomes `UTC` |
| `status` | text, default `active` | `active` \| `maintenance` \| `retired`; accepted on create and update |
| `created_at`, `updated_at`, `version`, `deleted_at` | existing | Managed by `touch_row()` and soft-delete paths |

A ship can exist with no decks, areas, or rooms. Soft-deleting a ship excludes it
from normal reads and makes all its import endpoints return not-found.

### Deck (`decks`)

Existing columns and constraints. Import writes `id` (UUID v7), `ship_id`, `level`
(int32, app rule ≥ 0), and `name` (non-blank). `uq_decks_level (ship_id, level) WHERE
deleted_at IS NULL` is the final conflict authority. Only conflicts with non-deleted
levels block an import; soft-deleted levels may be reused.

### Area (`areas`)

Existing columns and constraints. Import writes `id`, `ship_id`, `deck_id` resolved
from the workbook `deck_level`, `name`, `area_type`, optional `capacity` (≥ 0), and
optional `layout` (valid jsonb). Areas always attach to a Deck created in the same
workbook, per the feature assumption.

### Room (`rooms`)

Existing columns and constraints. Import writes `id`, `ship_id`, `deck_id`, `number`,
`room_type`, `capacity` (smallint, > 0), `is_bookable` (default true), and optional
`layout`. `uq_rooms_number (ship_id, number) WHERE deleted_at IS NULL` is the final
conflict authority.

### Change log / audit

`capture_change()` triggers on `ships`, `decks`, `areas`, and `rooms` continue to
write `change_log` and allocate `ship_seq` in the same transaction. `audit_log` gets
one confirmation entry through the existing `audit.Appender`. Neither is modified.

## 2. New entity: Ship Structure Import (`ship_structure_imports`)

A tracked import attempt for one ship. Cloud-only, not synchronized, no capture
trigger, no soft delete (history is retained; there is no delete endpoint).

### DDL

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

### Field semantics

| Field | Semantics |
|---|---|
| `id` | Import identifier returned to the client; UUID v7. |
| `ship_id` | Owning ship. Every read/write is scoped by both `id` and `ship_id`. |
| `idempotency_key` | Optional upload `Idempotency-Key`; unique per ship when present. Not a shared store. |
| `status` | Lifecycle state (below). |
| `file_name` | Original client filename, sanitized only for response safety; never used as a path. |
| `file_size_bytes` | Uploaded byte count (compressed). |
| `file_sha256` | Lowercase hex SHA-256 of the uploaded bytes; used for idempotent replay comparison. |
| `*_total` | Non-blank data rows considered, including invalid rows. |
| `*_valid` | Rows with zero findings. |
| `*_invalid` | Rows with one or more findings. `valid + invalid = total`. |
| `validated_payload` | Normalized valid rows for preview/confirm; `NULL` until validation completes. Never a source of truth. |
| `validation_errors` | Array of findings (schema below); `[]` on success. |
| `error_message` | Safe administrator-facing summary (validation failure, interrupted validation, or confirmation retry hint). Never raw SQL/file/library text. |
| `created_by` | Principal actor id (text) when available; for history only. |
| `created_at`, `updated_at` | Set explicitly; `updated_at` changes on every state/count/payload update. |
| `completed_at` | Set once when status becomes `completed`. |

### Lifecycle

```text
                 (upload + synchronous validation)
upload ──► validating ──► ready ──► [confirmation tx: processing ──► completed]
                 │                         │
                 └──► failed               └── (any error) rollback to ready + error_message
```

- `uploaded` is reserved for a future asynchronous handoff and is not used by the
  synchronous flow.
- `failed` is the terminal state for validation/file failures (and for stale
  `validating`/`uploaded` imports recovered after an interrupted validation). A failed
  import is history; retry means a new upload with a new `importId`.
- `processing` is written inside the confirmation transaction after the row lock and a
  final conflict re-check. It is only durable together with `completed`; a rollback
  leaves the row `ready`.
- `completed` is terminal. A repeated confirmation returns the completed outcome
  without writes.
- A confirmation failure rolls back all structure, leaves `ready`, and records a safe
  `error_message`; the same import may be retried.

Valid transitions (FR-039): `validating → ready|failed`; `ready → processing`;
`processing → completed`. Recovery of an interrupted validation may
`uploaded|validating → failed`. Confirmation failures do not transition to `failed`;
the transaction leaves `ready`.

## 3. Import payload (`validated_payload`)

Only normalized, valid rows are stored. Rows are trimmed and typed. The payload is a
JSON object:

```json
{
  "decks": [
    { "level": 1, "name": "Main Deck" }
  ],
  "areas": [
    {
      "deck_level": 1,
      "name": "Main Lobby",
      "area_type": "lobby",
      "capacity": 40,
      "layout": { "x": 0, "y": 0, "w": 10, "h": 8 }
    }
  ],
  "rooms": [
    {
      "deck_level": 2,
      "number": "201",
      "room_type": "balcony",
      "capacity": 2,
      "is_bookable": true,
      "layout": null
    }
  ]
}
```

- `capacity` is a JSON number or `null` for Areas; Rooms always have a number.
- `layout` is the parsed JSON value or `null`.
- `deck_level` is the workbook reference key, not a database id. Confirmation maps it
  to the newly created deck id.
- Empty arrays are allowed for Areas/Rooms, but at least one record must exist overall.

## 4. Validation finding

```json
{
  "sheet": "Rooms",
  "row": 7,
  "column": "capacity",
  "code": "invalid_capacity",
  "message": "Room capacity must be a whole number greater than zero."
}
```

| Field | Semantics |
|---|---|
| `sheet` | `Decks`, `Areas`, `Rooms`, or `""` for file-level findings. |
| `row` | Excel 1-based row (1 = header); `0` for file/sheet-level findings. |
| `column` | Expected/actual canonical column name; `""` for sheet/file-level findings. |
| `code` | Stable machine-readable code (see `contracts/template-workbook.md`). |
| `message` | Safe, actionable text; never includes raw SQL, file paths, or library errors. |

## 5. Counts and confirmability

```json
{
  "decks": { "total": 2, "valid": 2, "invalid": 0 },
  "areas": { "total": 3, "valid": 3, "invalid": 0 },
  "rooms": { "total": 8, "valid": 8, "invalid": 0 }
}
```

An import is `ready` and `can_confirm = true` only when:

1. The file-level checks passed (format, size, archive safety, no macro).
2. All required sheets/columns exist exactly once.
3. There is at least one non-blank data row across the workbook.
4. Every entity has `invalid = 0`.
5. Every Area/Room `deck_level` resolves to a valid Deck in the workbook.
6. No Deck level or Room number conflicts with an existing non-deleted record of the
   ship at validation time.

Confirmation re-checks conditions 5–6 and the database unique indexes, so a concurrent
change cannot produce a partial/invalid structure.

## 6. Relationships

```text
tenants 1 ─── * ships 1 ─── * decks 1 ─── * areas
                         │            └── * rooms
                         └── * ship_structure_imports
```

- `ship_structure_imports.ship_id → ships.id` is the only new foreign key.
- Imported Areas/Rooms reference the decks created in the same confirmation; no
  workbook record references a pre-existing deck.
- The import row is never referenced by business tables; no cascade/soft-delete
  coupling is added.

## 7. Idempotency and concurrency invariants

- Upload with `Idempotency-Key`: unique `(ship_id, idempotency_key)` when present.
  Same key + same SHA-256 returns the stored import preview; same key + different
  SHA-256 returns `409 idempotency_conflict`; a `23505` race is resolved by re-reading.
- Confirmation: `SELECT ... FOR UPDATE` on the import row serializes all confirmers.
  Only one transaction can move `ready → processing → completed`; others see
  `completed` (idempotent success) or a non-ready state (conflict).
- Structure inserts and the audit entry are in the same transaction. There is no
  durable `processing` orphan, because the state change commits only with the
  structure and audit.
- Soft-deleted ships/imports are not accessible; import lookups always require both
  `importId` and the path `shipId`, and then re-check principal tenant/ship access.
