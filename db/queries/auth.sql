-- name: CreateAdminUser :one
INSERT INTO admin_users (id, tenant_id, email, display_name, idp_subject, role, mfa_enabled, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetAdminUser :one
SELECT * FROM admin_users WHERE id = $1 AND deleted_at IS NULL;

-- name: GetActiveAdminUserByAccountID :one
SELECT * FROM admin_users
WHERE account_id = $1 AND status = 'active' AND deleted_at IS NULL;

-- name: ListAdminUsers :many
SELECT * FROM admin_users
WHERE tenant_id = sqlc.arg('tenant_id') AND deleted_at IS NULL
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateAdminUser :one
UPDATE admin_users
SET email = sqlc.arg('email'), display_name = sqlc.arg('display_name'), idp_subject = sqlc.narg('idp_subject'),
    role = sqlc.arg('role'), mfa_enabled = sqlc.arg('mfa_enabled'), status = sqlc.arg('status')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteAdminUser :execrows
UPDATE admin_users SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: AssignAdminShipAccess :exec
INSERT INTO admin_ship_access (admin_user_id, ship_id) VALUES ($1, $2)
ON CONFLICT (admin_user_id, ship_id) DO NOTHING;

-- name: RevokeAdminShipAccess :execrows
DELETE FROM admin_ship_access WHERE admin_user_id = $1 AND ship_id = $2;

-- name: ListAdminShipAccess :many
SELECT * FROM admin_ship_access WHERE admin_user_id = $1;

-- name: CreateAccount :one
INSERT INTO accounts (id, email, display_name, phone, locale, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetAccount :one
SELECT * FROM accounts WHERE id = $1 AND deleted_at IS NULL;

-- name: ListAccounts :many
SELECT * FROM accounts
WHERE deleted_at IS NULL
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateAccount :one
UPDATE accounts
SET email = sqlc.narg('email'), display_name = sqlc.narg('display_name'), phone = sqlc.narg('phone'),
    locale = sqlc.arg('locale'), status = sqlc.arg('status')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteAccount :execrows
UPDATE accounts SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: AccountExists :one
SELECT EXISTS (SELECT 1 FROM accounts WHERE id = $1 AND deleted_at IS NULL);

-- name: LinkAccountIdentity :one
INSERT INTO account_identities (id, account_id, provider, provider_sub, email)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetAccountIdentity :one
SELECT * FROM account_identities WHERE id = $1;

-- name: GetAccountIdentityByProviderSubject :one
SELECT * FROM account_identities WHERE provider = $1 AND provider_sub = $2;

-- name: ListAccountIdentities :many
SELECT * FROM account_identities WHERE account_id = $1 ORDER BY created_at;

-- name: CountAccountIdentities :one
SELECT count(*) FROM account_identities WHERE account_id = $1;

-- name: UnlinkAccountIdentity :execrows
DELETE FROM account_identities WHERE id = $1;

-- name: DeleteAccountIdentityForAccount :execrows
DELETE FROM account_identities WHERE id = $1 AND account_id = $2;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (id, subject_type, subject_id, family_id, client, token_hash, expires_at, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetRefreshTokenByHash :one
SELECT * FROM refresh_tokens WHERE token_hash = $1;

-- name: GetRefreshTokenByHashForUpdate :one
SELECT * FROM refresh_tokens WHERE token_hash = $1 FOR UPDATE;

-- name: MarkRefreshTokenUsed :one
UPDATE refresh_tokens SET used_at = now(), replaced_by = sqlc.narg('replaced_by')
WHERE id = sqlc.arg('id') AND used_at IS NULL AND revoked_at IS NULL
RETURNING *;

-- name: RevokeRefreshTokenFamily :execrows
UPDATE refresh_tokens SET revoked_at = now() WHERE family_id = $1 AND revoked_at IS NULL;

-- name: RevokeAccountRefreshTokens :execrows
UPDATE refresh_tokens SET revoked_at = now()
WHERE subject_type = 'account' AND subject_id = $1 AND revoked_at IS NULL;
