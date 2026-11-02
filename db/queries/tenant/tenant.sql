-- name: CreateTenant :one
INSERT INTO tenants (id, name) VALUES ($1, $2) RETURNING *;

-- name: GetTenant :one
SELECT * FROM tenants WHERE id = $1 AND deleted_at IS NULL;

-- name: ListTenants :many
SELECT * FROM tenants
WHERE deleted_at IS NULL
  AND ($1::uuid IS NULL OR id = $1)
  AND ($2::uuid IS NULL OR id > $2)
ORDER BY id
LIMIT $3;

-- name: UpdateTenant :one
UPDATE tenants SET name = COALESCE($2, name)
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteTenant :execrows
UPDATE tenants SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;
