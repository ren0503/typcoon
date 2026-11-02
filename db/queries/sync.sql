-- name: GetShipSeq :one
SELECT * FROM ship_seq WHERE ship_id = $1;

-- name: ListChangeLog :many
SELECT * FROM change_log
WHERE ship_id = sqlc.arg('ship_id') AND seq > sqlc.arg('after_seq')
ORDER BY seq
LIMIT sqlc.arg('page_limit');

-- name: CreateHubDevice :one
INSERT INTO hub_devices (id, ship_id, name, role, cert_serial)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetHubDevice :one
SELECT * FROM hub_devices WHERE id = $1 AND deleted_at IS NULL;

-- name: ListHubDevices :many
SELECT * FROM hub_devices
WHERE ship_id = sqlc.arg('ship_id') AND deleted_at IS NULL
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateHubDeviceStatus :one
UPDATE hub_devices SET status = sqlc.arg('status'), epoch = sqlc.arg('epoch')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: RetireHubDevice :execrows
UPDATE hub_devices SET status = 'retired', deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateShipDevice :one
INSERT INTO ship_devices (id, ship_id, device_name, device_type, token_hash)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListShipDevices :many
SELECT * FROM ship_devices
WHERE ship_id = sqlc.arg('ship_id') AND deleted_at IS NULL
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: RevokeShipDevice :one
UPDATE ship_devices SET status = 'revoked' WHERE id = $1 AND deleted_at IS NULL RETURNING *;

-- name: CreateRevocation :one
INSERT INTO revocations (id, ship_id, subject_type, subject_id, reason, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetRevocation :one
SELECT * FROM revocations WHERE id = $1 AND deleted_at IS NULL;

-- name: ListRevocations :many
SELECT * FROM revocations
WHERE ship_id = sqlc.arg('ship_id') AND deleted_at IS NULL
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: CreateSnapshot :one
INSERT INTO snapshots (id, ship_id, voyage_id, phase, seq_base, storage_key, sha256, size_bytes, kid, signature, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetSnapshot :one
SELECT * FROM snapshots WHERE id = $1;

-- name: ListSnapshots :many
SELECT * FROM snapshots
WHERE ship_id = sqlc.arg('ship_id')
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: CreateJobRun :one
INSERT INTO job_runs (id, voyage_id, job_type, schedule_version)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: FinishJobRun :one
UPDATE job_runs
SET status = sqlc.arg('status'), finished_at = now(), error = sqlc.narg('error')
WHERE id = sqlc.arg('id') AND status = 'running'
RETURNING *;

-- name: ListJobRuns :many
SELECT * FROM job_runs
WHERE voyage_id = sqlc.arg('voyage_id')
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpsertOpsEvent :execrows
INSERT INTO ops_events (hub_id, local_seq, ship_id, voyage_id, epoch, entity, entity_id, op, payload, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (hub_id, local_seq) DO NOTHING;

-- name: ListOpsEvents :many
SELECT * FROM ops_events
WHERE ship_id = sqlc.arg('ship_id') AND received_at > sqlc.arg('after_received')
ORDER BY received_at, hub_id, local_seq
LIMIT sqlc.arg('page_limit');
