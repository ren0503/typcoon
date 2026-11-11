# Quickstart: Ship Onboarding with Excel Structure Import

This is the implementation validation guide for `004-ship-structure-import`. It
assumes the repository's existing local setup from `README.md`.

## Prerequisites

- Go 1.26+, Docker, `make`
- PostgreSQL/Redis via `docker compose up -d postgres redis`
- A dev principal allowlist, for example in `.env`:

```bash
AUTH_MODE=dev
AUTH_DEV_PRINCIPALS=system:dev-1:00000000-0000-0000-0000-000000000000
```

For the examples below:

```bash
BASE=http://localhost:8080/api
PRINCIPAL='X-Dev-Principal: system:dev-1:00000000-0000-0000-0000-000000000000'
TENANT_ID=<an existing tenant uuid accessible to the principal>
```

Create a tenant first if the database is empty (`POST $BASE/tenants` with the same
header), then use its id as `TENANT_ID`.

## Build and migrate

```bash
make sqlc            # regenerate sqlc after adding db/queries/ship_imports.sql
make migrate-up      # applies 00001..00004
make test            # unit + OpenAPI lint, no Docker
make test-integration  # testcontainers Postgres; requires Docker
make lint
```

`00004_ship_structure_imports.sql` must apply cleanly after `00003` and must be
idempotent when re-run through the Goose provider (the harness already proves this).

## 1. Create a ship without structure

```bash
curl -sS -X POST "$BASE/ships" \
  -H "$PRINCIPAL" -H 'Content-Type: application/json' \
  -d "{\"name\":\"MV Onboarding\",\"timezone\":\"Europe/Lisbon\",\"status\":\"active\"}"
```

Expected: `201` with the ship, including `id` (UUID v7), `status`, `version: 1`, and
no decks/areas/rooms. Repeat with a duplicate `imo_number` to confirm `409 conflict`;
omit `status` to confirm it defaults to `active`.

```bash
SHIP_ID=<id from the response>
curl -sS -H "$PRINCIPAL" "$BASE/ships/$SHIP_ID"
curl -sS -H "$PRINCIPAL" "$BASE/ships"
```

## 2. Download and fill the template

```bash
curl -sS -H "$PRINCIPAL" -o /tmp/ship-template.xlsx \
  "$BASE/ships/structure/import-template"
```

The workbook has `Decks`, `Areas`, `Rooms` (required) and `Reference` (documentation)
sheets. Fill row 2 onward on each required sheet. See
`contracts/template-workbook.md` for every column, enum, limit, and example. The
example rows already shipped in the template must validate unchanged for a fresh ship.

## 3. Upload and preview

```bash
curl -sS -X POST "$BASE/ships/$SHIP_ID/structure/imports" \
  -H "$PRINCIPAL" \
  -H "Idempotency-Key: onboarding-$SHIP_ID-v1" \
  -F "file=@/tmp/ship.xlsx" | tee /tmp/import.json

IMPORT_ID=$(python3 -c 'import json;print(json.load(open("/tmp/import.json"))["import"]["id"])')
curl -sS -H "$PRINCIPAL" "$BASE/ships/$SHIP_ID/structure/imports/$IMPORT_ID"
curl -sS -H "$PRINCIPAL" "$BASE/ships/$SHIP_ID/structure/imports/$IMPORT_ID/preview"
```

Expected for a valid workbook:

- `201` on upload; `import.status == "ready"`, `can_confirm == true`;
- `counts.*.invalid == 0` and `counts.*.valid == counts.*.total`;
- `errors == []`;
- `records` contains the normalized rows;
- `GET /ships/$SHIP_ID/decks` (and areas/rooms) still returns nothing.

For an invalid workbook, the upload still returns `201` but `status == "failed"`,
`can_confirm == false`, and `errors` lists every finding with `sheet`, `row`,
`column`, `code`, and a safe `message`. No structure is created.

Repeating the same upload with the same `Idempotency-Key` and file returns the same
import preview (no new row). Using the key with a different file returns
`409 idempotency_conflict`. When retrying a corrected workbook, use a new
`Idempotency-Key` (or omit it) so the new import gets a new `importId`.

## 4. Confirm atomically

```bash
curl -sS -X POST "$BASE/ships/$SHIP_ID/structure/imports/$IMPORT_ID/confirm" \
  -H "$PRINCIPAL" | tee /tmp/confirm.json
```

Expected: `200`, `status == "completed"`, `completed_at` set, and the created counts.
Verify:

```bash
curl -sS -H "$PRINCIPAL" "$BASE/ships/$SHIP_ID/decks"
curl -sS -H "$PRINCIPAL" "$BASE/ships/$SHIP_ID/areas"
curl -sS -H "$PRINCIPAL" "$BASE/ships/$SHIP_ID/rooms"
```

Every row must have the correct `ship_id`, and each area/room `deck_id` must be one of
the imported decks. Confirm again: it returns the same completed outcome, with no
duplicates. Confirm a `failed` import: `409 conflict`.

## 5. Verify synchronization and audit

After confirmation, the existing triggers must have emitted one change per imported
record:

```sql
SELECT entity, entity_id, op, version, seq
FROM change_log
WHERE ship_id = '<SHIP_ID>'
ORDER BY seq;
```

Expect `decks`, `areas`, and `rooms` rows with increasing `seq`. Update and soft-delete
an imported deck/room through the existing endpoints and verify `version` increments
and `op = 'delete'` appears, with no bypass of the triggers.

Audit (confirmation only):

```sql
SELECT action, actor_type, actor_id, entity, entity_id, ship_id, details
FROM audit_log
WHERE entity = 'ship_structure_import' AND entity_id = '<IMPORT_ID>';
```

Expect one `ship.structure_import.confirmed` entry with created counts and no raw
workbook content or file details.

## 6. Verify rollback and recovery

- Force a persistence failure in an integration test (for example, inject a failing
  audit appender or create a conflicting deck after validation). Expect no partial
  decks/areas/rooms, the import still `ready` with a safe `error_message`, and a
  retryable confirmation.
- Simulate an interrupted validation by leaving an import in `validating` older than
  the stale threshold. A status/preview/confirm call must mark it `failed` with a safe
  message; no structure exists.
- Run two confirmations concurrently; exactly one structure is created and both
  responses observe the same completed outcome (one may be `409` while in flight).

## 7. Access checks

Use a principal for another tenant and another ship:

- `GET/POST` import endpoints for a ship outside the principal → `404 not_found`.
- Use a valid `importId` with a different path `shipId` → `404 not_found` (existence
  is not revealed).
- Soft-delete the ship, then upload/preview/confirm → `404 not_found`.

## 8. Acceptance checklist

- [ ] Ship create/list/get/update/soft-delete work with `status`; no structure needed.
- [ ] Template downloads as `.xlsx` with the three required sheets, headers, examples,
      enum lists, and the `deck_level` explanation.
- [ ] Template examples upload as `ready`.
- [ ] Invalid files (`.xls`, CSV renamed, OLE/encrypted, macro, corrupt, >10 MiB) and
      invalid sheets/columns/values produce the documented findings and never become
      confirmable.
- [ ] Duplicate workbook keys and conflicts with existing non-deleted decks/rooms are
      reported; soft-deleted keys may be reused.
- [ ] Unknown/duplicated/invalid `deck_level` references block Areas/Rooms.
- [ ] No structure exists after upload/preview; only confirmation creates rows.
- [ ] Confirmation creates decks first, then areas/rooms, with correct FKs and
      `ship_id`, in one transaction.
- [ ] Persistence/audit failure leaves no partial structure and the import is retryable.
- [ ] Concurrent/repeated confirmation creates exactly one structure.
- [ ] `change_log` contains every imported record and preserves version/delete semantics.
- [ ] Confirmation writes one audit entry; audit failure rolls back.
- [ ] Cross-tenant/cross-ship imports return non-revealing not-found.
- [ ] `make test`, `make test-integration`, and `make lint` pass; contract tests use
      `/api`.

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| Upload `400` | Missing `file` part or malformed multipart |
| Upload `413` | File exceeds 10 MiB compressed |
| Import `failed`, `unsupported_format` | `.xls`/CSV/renamed non-ZIP content |
| Import `failed`, `macro_enabled` | Workbook contains `xl/vbaProject.bin` |
| Import `failed`, `archive_too_large` | Compressed file expands beyond the safety limit |
| Import `failed`, `missing_sheet`/`missing_column` | Sheet or header name does not match the template |
| Import `failed`, `conflict` | Deck level or room number already exists as non-deleted |
| Import `failed`, `invalid_reference` | Area/Room `deck_level` has no valid Decks row |
| Confirm `409` | Import is failed, validating, processing, or not ready |
| Confirm `500` | Persistence/audit failed; nothing was committed; inspect `error_summary` and retry |
| `404` on import | Wrong ship/import pairing, soft-deleted ship, or no tenant/ship access |
