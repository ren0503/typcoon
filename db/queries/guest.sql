-- name: CreateGuest :one
INSERT INTO guests (id, booking_id, full_name, date_of_birth, nationality, document_type, document_enc,
                    document_key_id, emergency_contact_enc, special_needs, is_lead, muster_station)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetGuest :one
SELECT * FROM guests WHERE id = $1 AND deleted_at IS NULL;

-- name: ListGuests :many
SELECT * FROM guests
WHERE booking_id = sqlc.arg('booking_id') AND deleted_at IS NULL
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateGuest :one
UPDATE guests
SET full_name = sqlc.arg('full_name'), date_of_birth = sqlc.narg('date_of_birth'), nationality = sqlc.narg('nationality'),
    document_type = sqlc.narg('document_type'), document_enc = sqlc.narg('document_enc'), document_key_id = sqlc.narg('document_key_id'),
    emergency_contact_enc = sqlc.narg('emergency_contact_enc'), special_needs = sqlc.narg('special_needs'),
    is_lead = sqlc.arg('is_lead'), muster_station = sqlc.narg('muster_station')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteGuest :execrows
UPDATE guests SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: GuestExists :one
SELECT EXISTS (SELECT 1 FROM guests WHERE id = sqlc.arg('id') AND deleted_at IS NULL);

-- name: CreateBoardingPass :one
INSERT INTO boarding_passes (id, booking_id, guest_id, jti, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetBoardingPass :one
SELECT * FROM boarding_passes WHERE id = $1;

-- name: ListBoardingPasses :many
SELECT * FROM boarding_passes
WHERE booking_id = sqlc.arg('booking_id')
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: RevokeBoardingPass :one
UPDATE boarding_passes SET revoked_at = COALESCE(revoked_at, now()) WHERE id = $1 RETURNING *;
