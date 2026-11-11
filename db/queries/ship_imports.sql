-- name: CreateShipStructureImport :one
INSERT INTO ship_structure_imports (
  id, ship_id, idempotency_key, status, file_name, file_size_bytes, file_sha256,
  decks_total, decks_valid, decks_invalid,
  areas_total, areas_valid, areas_invalid,
  rooms_total, rooms_valid, rooms_invalid,
  validated_payload, validation_errors, error_message, created_by
) VALUES (
  sqlc.arg('id'), sqlc.arg('ship_id'), sqlc.narg('idempotency_key'), sqlc.arg('status'),
  sqlc.arg('file_name'), sqlc.arg('file_size_bytes'), sqlc.arg('file_sha256'),
  sqlc.arg('decks_total'), sqlc.arg('decks_valid'), sqlc.arg('decks_invalid'),
  sqlc.arg('areas_total'), sqlc.arg('areas_valid'), sqlc.arg('areas_invalid'),
  sqlc.arg('rooms_total'), sqlc.arg('rooms_valid'), sqlc.arg('rooms_invalid'),
  sqlc.narg('validated_payload'), sqlc.arg('validation_errors'), sqlc.narg('error_message'),
  sqlc.narg('created_by')
) RETURNING *;

-- name: GetShipStructureImport :one
SELECT * FROM ship_structure_imports WHERE id = sqlc.arg('id') AND ship_id = sqlc.arg('ship_id');

-- name: GetShipStructureImportForUpdate :one
SELECT * FROM ship_structure_imports
WHERE id = sqlc.arg('id') AND ship_id = sqlc.arg('ship_id')
FOR UPDATE;

-- name: GetShipStructureImportByIdempotencyKey :one
SELECT * FROM ship_structure_imports
WHERE ship_id = sqlc.arg('ship_id') AND idempotency_key = sqlc.arg('idempotency_key');

-- name: UpdateShipStructureImportValidation :one
UPDATE ship_structure_imports
SET status = sqlc.arg('status'),
    decks_total = sqlc.arg('decks_total'), decks_valid = sqlc.arg('decks_valid'), decks_invalid = sqlc.arg('decks_invalid'),
    areas_total = sqlc.arg('areas_total'), areas_valid = sqlc.arg('areas_valid'), areas_invalid = sqlc.arg('areas_invalid'),
    rooms_total = sqlc.arg('rooms_total'), rooms_valid = sqlc.arg('rooms_valid'), rooms_invalid = sqlc.arg('rooms_invalid'),
    validated_payload = sqlc.narg('validated_payload'),
    validation_errors = sqlc.arg('validation_errors'),
    error_message = sqlc.narg('error_message'),
    updated_at = now()
WHERE id = sqlc.arg('id') AND ship_id = sqlc.arg('ship_id')
RETURNING *;

-- name: SetShipStructureImportProcessing :execrows
UPDATE ship_structure_imports
SET status = 'processing', error_message = NULL, updated_at = now()
WHERE id = sqlc.arg('id') AND ship_id = sqlc.arg('ship_id') AND status = 'ready';

-- name: CompleteShipStructureImport :one
UPDATE ship_structure_imports
SET status = 'completed', error_message = NULL, completed_at = now(), updated_at = now()
WHERE id = sqlc.arg('id') AND ship_id = sqlc.arg('ship_id')
RETURNING *;

-- name: SetShipStructureImportError :execrows
UPDATE ship_structure_imports
SET error_message = sqlc.arg('error_message'), updated_at = now()
WHERE id = sqlc.arg('id') AND ship_id = sqlc.arg('ship_id') AND status = 'ready';

-- name: FailShipStructureImport :one
UPDATE ship_structure_imports
SET status = 'failed', error_message = sqlc.arg('error_message'), updated_at = now()
WHERE id = sqlc.arg('id') AND ship_id = sqlc.arg('ship_id')
  AND status IN ('uploaded', 'validating')
RETURNING *;

-- name: ListExistingDeckLevels :many
SELECT level FROM decks WHERE ship_id = sqlc.arg('ship_id') AND deleted_at IS NULL ORDER BY level;

-- name: ListExistingRoomNumbers :many
SELECT number FROM rooms WHERE ship_id = sqlc.arg('ship_id') AND deleted_at IS NULL ORDER BY number;
