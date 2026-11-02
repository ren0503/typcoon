-- name: CreateStaff :one
INSERT INTO staff (id, tenant_id, ship_id, employee_code, full_name, department, role_title,
                   is_supervisor, languages, phone, certifications, status, hired_on)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: GetStaff :one
SELECT * FROM staff WHERE id = $1 AND deleted_at IS NULL;

-- name: ListStaff :many
SELECT * FROM staff
WHERE deleted_at IS NULL
  AND (sqlc.narg('tenant_id')::uuid IS NULL OR tenant_id = sqlc.narg('tenant_id'))
  AND (sqlc.narg('ship_id')::uuid IS NULL OR ship_id = sqlc.narg('ship_id'))
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateStaff :one
UPDATE staff
SET ship_id = sqlc.narg('ship_id'), employee_code = sqlc.arg('employee_code'), full_name = sqlc.arg('full_name'),
    department = sqlc.arg('department'), role_title = sqlc.narg('role_title'), is_supervisor = sqlc.arg('is_supervisor'),
    languages = sqlc.arg('languages'), phone = sqlc.narg('phone'), certifications = sqlc.arg('certifications'),
    status = sqlc.arg('status'), hired_on = sqlc.narg('hired_on')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteStaff :execrows
UPDATE staff SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: StaffExists :one
SELECT EXISTS (SELECT 1 FROM staff WHERE id = sqlc.arg('id') AND deleted_at IS NULL);

-- name: CreateStaffLeave :one
INSERT INTO staff_leaves (id, staff_id, leave_type, starts_at, ends_at, note)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetStaffLeave :one
SELECT * FROM staff_leaves WHERE id = $1 AND deleted_at IS NULL;

-- name: ListStaffLeaves :many
SELECT * FROM staff_leaves
WHERE staff_id = sqlc.arg('staff_id') AND deleted_at IS NULL
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateStaffLeave :one
UPDATE staff_leaves
SET leave_type = sqlc.arg('leave_type'), starts_at = sqlc.arg('starts_at'), ends_at = sqlc.arg('ends_at'), note = sqlc.narg('note')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteStaffLeave :execrows
UPDATE staff_leaves SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: UpsertStaffCredential :one
INSERT INTO staff_credentials (id, staff_id, pin_hash)
VALUES ($1, $2, $3)
ON CONFLICT (staff_id) DO UPDATE SET pin_hash = EXCLUDED.pin_hash, pin_updated_at = now()
RETURNING *;

-- name: GetStaffCredential :one
SELECT * FROM staff_credentials WHERE staff_id = $1 AND deleted_at IS NULL;

-- name: CreateShiftTemplate :one
INSERT INTO shift_templates (id, ship_id, department, code, name, start_time, end_time)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetShiftTemplate :one
SELECT * FROM shift_templates WHERE id = $1 AND deleted_at IS NULL;

-- name: ListShiftTemplates :many
SELECT * FROM shift_templates
WHERE ship_id = sqlc.arg('ship_id') AND deleted_at IS NULL
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateShiftTemplate :one
UPDATE shift_templates
SET department = sqlc.arg('department'), code = sqlc.arg('code'), name = sqlc.arg('name'),
    start_time = sqlc.arg('start_time'), end_time = sqlc.arg('end_time')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteShiftTemplate :execrows
UPDATE shift_templates SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateCoverageRequirement :one
INSERT INTO coverage_requirements (id, ship_id, department, shift_template_id, position_code, deck_id, area_id,
                                   min_staff, max_staff, days_of_week, valid_from, valid_to)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetCoverageRequirement :one
SELECT * FROM coverage_requirements WHERE id = $1 AND deleted_at IS NULL;

-- name: ListCoverageRequirements :many
SELECT * FROM coverage_requirements
WHERE ship_id = sqlc.arg('ship_id') AND deleted_at IS NULL
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateCoverageRequirement :one
UPDATE coverage_requirements
SET department = sqlc.arg('department'), shift_template_id = sqlc.arg('shift_template_id'),
    position_code = sqlc.arg('position_code'), deck_id = sqlc.narg('deck_id'), area_id = sqlc.narg('area_id'),
    min_staff = sqlc.arg('min_staff'), max_staff = sqlc.narg('max_staff'), days_of_week = sqlc.arg('days_of_week'),
    valid_from = sqlc.narg('valid_from'), valid_to = sqlc.narg('valid_to')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteCoverageRequirement :execrows
UPDATE coverage_requirements SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateLaborRule :one
INSERT INTO labor_rules (id, tenant_id, ship_id, rule_code, severity, params, valid_from, valid_to)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetLaborRule :one
SELECT * FROM labor_rules WHERE id = $1 AND deleted_at IS NULL;

-- name: ListLaborRules :many
SELECT * FROM labor_rules
WHERE deleted_at IS NULL
  AND (sqlc.narg('tenant_id')::uuid IS NULL OR tenant_id = sqlc.narg('tenant_id'))
  AND (sqlc.narg('ship_id')::uuid IS NULL OR ship_id = sqlc.narg('ship_id'))
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateLaborRule :one
UPDATE labor_rules
SET ship_id = sqlc.narg('ship_id'), rule_code = sqlc.arg('rule_code'), severity = sqlc.arg('severity'),
    params = sqlc.arg('params'), valid_from = sqlc.narg('valid_from'), valid_to = sqlc.narg('valid_to')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteLaborRule :execrows
UPDATE labor_rules SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateAutonomyPolicy :one
INSERT INTO autonomy_policies (id, tenant_id, ship_id, change_type, level, conditions)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetAutonomyPolicy :one
SELECT * FROM autonomy_policies WHERE id = $1 AND deleted_at IS NULL;

-- name: ListAutonomyPolicies :many
SELECT * FROM autonomy_policies
WHERE deleted_at IS NULL
  AND (sqlc.narg('tenant_id')::uuid IS NULL OR tenant_id = sqlc.narg('tenant_id'))
  AND (sqlc.narg('ship_id')::uuid IS NULL OR ship_id = sqlc.narg('ship_id'))
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateAutonomyPolicy :one
UPDATE autonomy_policies
SET ship_id = sqlc.narg('ship_id'), change_type = sqlc.arg('change_type'), level = sqlc.arg('level'), conditions = sqlc.arg('conditions')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteAutonomyPolicy :execrows
UPDATE autonomy_policies SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;
