-- name: CreatePartner :one
INSERT INTO partners (id, tenant_id, name, business_type, contact_name, contact_email, contact_phone)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetPartner :one
SELECT * FROM partners WHERE id = $1 AND deleted_at IS NULL;

-- name: ListPartners :many
SELECT * FROM partners
WHERE deleted_at IS NULL
  AND (sqlc.narg('tenant_id')::uuid IS NULL OR tenant_id = sqlc.narg('tenant_id'))
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdatePartner :one
UPDATE partners
SET name = sqlc.arg('name'), business_type = sqlc.arg('business_type'), contact_name = sqlc.narg('contact_name'),
    contact_email = sqlc.narg('contact_email'), contact_phone = sqlc.narg('contact_phone'), status = sqlc.arg('status')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeletePartner :execrows
UPDATE partners SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: PartnerExists :one
SELECT EXISTS (
  SELECT 1 FROM partners
  WHERE id = sqlc.arg('id') AND deleted_at IS NULL
    AND (sqlc.narg('tenant_id')::uuid IS NULL OR tenant_id = sqlc.narg('tenant_id'))
);

-- name: CreatePartnerContract :one
INSERT INTO partner_contracts (id, ship_id, partner_id, starts_on, ends_on, fee_model, fee_amount_cents, currency, terms_storage_key, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetPartnerContract :one
SELECT * FROM partner_contracts WHERE id = $1 AND deleted_at IS NULL;

-- name: ListPartnerContracts :many
SELECT * FROM partner_contracts
WHERE partner_id = sqlc.arg('partner_id') AND deleted_at IS NULL
  AND (sqlc.narg('ship_id')::uuid IS NULL OR ship_id = sqlc.narg('ship_id'))
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdatePartnerContract :one
UPDATE partner_contracts
SET ship_id = sqlc.arg('ship_id'), starts_on = sqlc.arg('starts_on'), ends_on = sqlc.narg('ends_on'),
    fee_model = sqlc.narg('fee_model'), fee_amount_cents = sqlc.narg('fee_amount_cents'), currency = sqlc.narg('currency'),
    terms_storage_key = sqlc.narg('terms_storage_key'), status = sqlc.arg('status')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeletePartnerContract :execrows
UPDATE partner_contracts SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: CreatePartnerStaff :one
INSERT INTO partner_staff (id, ship_id, partner_id, full_name, role_title, phone, status, valid_from, valid_to)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetPartnerStaff :one
SELECT * FROM partner_staff WHERE id = $1 AND deleted_at IS NULL;

-- name: ListPartnerStaff :many
SELECT * FROM partner_staff
WHERE partner_id = sqlc.arg('partner_id') AND deleted_at IS NULL
  AND (sqlc.narg('ship_id')::uuid IS NULL OR ship_id = sqlc.narg('ship_id'))
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdatePartnerStaff :one
UPDATE partner_staff
SET ship_id = sqlc.arg('ship_id'), full_name = sqlc.arg('full_name'), role_title = sqlc.narg('role_title'),
    phone = sqlc.narg('phone'), status = sqlc.arg('status'), valid_from = sqlc.narg('valid_from'), valid_to = sqlc.narg('valid_to')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeletePartnerStaff :execrows
UPDATE partner_staff SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: CreatePartnerService :one
INSERT INTO partner_services (id, ship_id, partner_id, area_id, name, category, price_cents, currency,
                              requires_slot, default_capacity, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetPartnerService :one
SELECT * FROM partner_services WHERE id = $1 AND deleted_at IS NULL;

-- name: ListPartnerServices :many
SELECT * FROM partner_services
WHERE partner_id = sqlc.arg('partner_id') AND deleted_at IS NULL
  AND (sqlc.narg('ship_id')::uuid IS NULL OR ship_id = sqlc.narg('ship_id'))
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdatePartnerService :one
UPDATE partner_services
SET ship_id = sqlc.arg('ship_id'), area_id = sqlc.narg('area_id'), name = sqlc.arg('name'), category = sqlc.arg('category'),
    price_cents = sqlc.narg('price_cents'), currency = sqlc.narg('currency'), requires_slot = sqlc.arg('requires_slot'),
    default_capacity = sqlc.narg('default_capacity'), is_active = sqlc.arg('is_active')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeletePartnerService :execrows
UPDATE partner_services SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: CreatePartnerSlot :one
INSERT INTO partner_slots (id, ship_id, partner_id, area_id, voyage_id, service_id, start_at, end_at, capacity)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetPartnerSlot :one
SELECT * FROM partner_slots WHERE id = $1 AND deleted_at IS NULL;

-- name: ListPartnerSlots :many
SELECT * FROM partner_slots
WHERE partner_id = sqlc.arg('partner_id') AND deleted_at IS NULL
  AND (sqlc.narg('ship_id')::uuid IS NULL OR ship_id = sqlc.narg('ship_id'))
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdatePartnerSlot :one
UPDATE partner_slots
SET voyage_id = sqlc.narg('voyage_id'), service_id = sqlc.narg('service_id'), start_at = sqlc.arg('start_at'),
    end_at = sqlc.arg('end_at'), capacity = sqlc.narg('capacity'), status = sqlc.arg('status'),
    requested_by = sqlc.narg('requested_by'), decided_by = sqlc.narg('decided_by'), decided_at = sqlc.narg('decided_at'),
    decision_note = sqlc.narg('decision_note')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeletePartnerSlot :execrows
UPDATE partner_slots SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: PartnerServiceExists :one
SELECT EXISTS (SELECT 1 FROM partner_services WHERE id = $1 AND deleted_at IS NULL);
