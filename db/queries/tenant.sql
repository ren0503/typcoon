-- name: CreateTenant :one
INSERT INTO tenants (id, name) VALUES ($1, $2) RETURNING *;

-- name: GetTenant :one
SELECT * FROM tenants WHERE id = $1 AND deleted_at IS NULL;

-- name: ListTenants :many
SELECT * FROM tenants
WHERE deleted_at IS NULL
  AND (sqlc.narg('only_id')::uuid IS NULL OR id = sqlc.narg('only_id'))
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateTenant :one
UPDATE tenants SET name = COALESCE(sqlc.narg('name'), name)
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteTenant :execrows
UPDATE tenants SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;
