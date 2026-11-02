-- name: CreatePort :one
INSERT INTO ports (id, un_locode, name, country_code, latitude, longitude, timezone)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetPort :one
SELECT * FROM ports WHERE id = $1 AND deleted_at IS NULL;

-- name: ListPorts :many
SELECT * FROM ports
WHERE deleted_at IS NULL
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdatePort :one
UPDATE ports
SET name = sqlc.arg('name'), latitude = sqlc.narg('latitude'), longitude = sqlc.narg('longitude'),
    timezone = sqlc.arg('timezone')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeletePort :execrows
UPDATE ports SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: PortExists :one
SELECT EXISTS (SELECT 1 FROM ports WHERE id = sqlc.arg('id') AND deleted_at IS NULL);

-- name: CreateVoyage :one
INSERT INTO voyages (id, ship_id, code, departure_at, arrival_at, boarding_opens_at, manifest_frozen_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetVoyage :one
SELECT * FROM voyages WHERE id = $1 AND deleted_at IS NULL;

-- name: ListVoyages :many
SELECT * FROM voyages
WHERE deleted_at IS NULL
  AND (sqlc.narg('ship_id')::uuid IS NULL OR ship_id = sqlc.narg('ship_id'))
  AND (departure_at, id) > (COALESCE(sqlc.narg('after_sort')::timestamptz, 'epoch'::timestamptz),
                            COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid))
ORDER BY departure_at, id
LIMIT sqlc.arg('page_limit');

-- name: UpdateVoyage :one
UPDATE voyages
SET code = sqlc.arg('code'), departure_at = sqlc.arg('departure_at'), arrival_at = sqlc.arg('arrival_at'),
    boarding_opens_at = sqlc.narg('boarding_opens_at'), manifest_frozen_at = sqlc.narg('manifest_frozen_at')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteVoyage :execrows
UPDATE voyages SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: VoyageExists :one
SELECT EXISTS (SELECT 1 FROM voyages WHERE id = sqlc.arg('id') AND deleted_at IS NULL);

-- name: CreatePortCall :one
INSERT INTO port_calls (id, ship_id, voyage_id, port_id, sequence, eta, etd, ata, atd, allows_embark, allows_disembark, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetPortCall :one
SELECT * FROM port_calls WHERE id = $1 AND deleted_at IS NULL;

-- name: ListPortCalls :many
SELECT * FROM port_calls
WHERE voyage_id = sqlc.arg('voyage_id') AND deleted_at IS NULL
  AND (sequence, id) > (COALESCE(sqlc.narg('after_seq')::bigint, 0),
                        COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid))
ORDER BY sequence, id
LIMIT sqlc.arg('page_limit');

-- name: UpdatePortCall :one
UPDATE port_calls
SET sequence = sqlc.arg('sequence'), eta = sqlc.narg('eta'), etd = sqlc.narg('etd'),
    ata = sqlc.narg('ata'), atd = sqlc.narg('atd'), allows_embark = sqlc.arg('allows_embark'),
    allows_disembark = sqlc.arg('allows_disembark'), status = sqlc.arg('status')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeletePortCall :execrows
UPDATE port_calls SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: PortCallExists :one
SELECT EXISTS (SELECT 1 FROM port_calls WHERE id = sqlc.arg('id') AND deleted_at IS NULL);

-- name: ListVoyagesForTenant :many
SELECT v.* FROM voyages v
JOIN ships s ON s.id = v.ship_id
WHERE v.deleted_at IS NULL AND s.deleted_at IS NULL AND s.tenant_id = sqlc.arg('tenant_id')
  AND (v.departure_at, v.id) > (COALESCE(sqlc.narg('after_sort')::timestamptz, 'epoch'::timestamptz),
                                COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid))
ORDER BY v.departure_at, v.id
LIMIT sqlc.arg('page_limit');
