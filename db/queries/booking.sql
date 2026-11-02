-- name: CreateBooking :one
INSERT INTO bookings (id, ship_id, voyage_id, account_id, room_id, embark_seq, disembark_seq, expires_at, idempotency_key, special_requests)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetBooking :one
SELECT * FROM bookings WHERE id = $1 AND deleted_at IS NULL;

-- name: GetBookingByIdempotency :one
SELECT * FROM bookings
WHERE account_id = sqlc.narg('account_id') AND idempotency_key = sqlc.arg('idempotency_key') AND deleted_at IS NULL;

-- name: ListBookings :many
SELECT * FROM bookings
WHERE deleted_at IS NULL
  AND (sqlc.narg('ship_id')::uuid IS NULL OR ship_id = sqlc.narg('ship_id'))
  AND (sqlc.narg('voyage_id')::uuid IS NULL OR voyage_id = sqlc.narg('voyage_id'))
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateBooking :one
UPDATE bookings
SET room_id = sqlc.narg('room_id'), embark_seq = sqlc.arg('embark_seq'), disembark_seq = sqlc.arg('disembark_seq'),
    expires_at = sqlc.narg('expires_at'), special_requests = sqlc.narg('special_requests')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteBooking :execrows
UPDATE bookings SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: BookingExists :one
SELECT EXISTS (SELECT 1 FROM bookings WHERE id = sqlc.arg('id') AND deleted_at IS NULL);

-- name: CreateServiceBooking :one
INSERT INTO service_bookings (id, booking_id, partner_service_id, slot_id, guest_id, quantity, idempotency_key, price_cents)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetServiceBooking :one
SELECT * FROM service_bookings WHERE id = $1 AND deleted_at IS NULL;

-- name: GetServiceBookingByIdempotency :one
SELECT * FROM service_bookings
WHERE booking_id = sqlc.arg('booking_id') AND idempotency_key = sqlc.arg('idempotency_key') AND deleted_at IS NULL;

-- name: ListServiceBookings :many
SELECT * FROM service_bookings
WHERE deleted_at IS NULL
  AND (sqlc.narg('booking_id')::uuid IS NULL OR booking_id = sqlc.narg('booking_id'))
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateServiceBooking :one
UPDATE service_bookings
SET slot_id = sqlc.narg('slot_id'), guest_id = sqlc.narg('guest_id'), quantity = sqlc.arg('quantity'),
    status = sqlc.arg('status'), price_cents = sqlc.narg('price_cents')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteServiceBooking :execrows
UPDATE service_bookings SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: ServiceBookingExists :one
SELECT EXISTS (SELECT 1 FROM service_bookings WHERE id = sqlc.arg('id') AND deleted_at IS NULL);

-- name: ListBookingsForTenant :many
SELECT b.* FROM bookings b
JOIN ships s ON s.id = b.ship_id
WHERE b.deleted_at IS NULL AND s.deleted_at IS NULL AND s.tenant_id = sqlc.arg('tenant_id')
  AND b.id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY b.id
LIMIT sqlc.arg('page_limit');
