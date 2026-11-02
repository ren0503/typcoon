-- name: CreateAIParam :one
INSERT INTO ai_params (id, tenant_id, kind, param_version, storage_key, sha256, kid, signature, status, released_at, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetAIParam :one
SELECT * FROM ai_params WHERE id = $1 AND deleted_at IS NULL;

-- name: ListAIParams :many
SELECT * FROM ai_params
WHERE tenant_id = sqlc.arg('tenant_id') AND deleted_at IS NULL
  AND (sqlc.narg('kind')::text IS NULL OR kind = sqlc.narg('kind'))
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateAIParamMetadata :one
UPDATE ai_params
SET storage_key = sqlc.arg('storage_key'), sha256 = sqlc.arg('sha256'), kid = sqlc.arg('kid'),
    signature = sqlc.arg('signature'), status = sqlc.arg('status'), released_at = sqlc.narg('released_at'), notes = sqlc.narg('notes')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteAIParam :execrows
UPDATE ai_params SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateFeedbackResponse :one
INSERT INTO feedback_responses (id, ship_id, voyage_id, respondent_type, respondent_ref, language, text_redacted, submitted_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetFeedbackResponse :one
SELECT * FROM feedback_responses WHERE id = $1;

-- name: ListFeedbackResponses :many
SELECT * FROM feedback_responses
WHERE ship_id = sqlc.arg('ship_id')
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: CreateFeedbackInsight :one
INSERT INTO feedback_insights (id, response_id, ship_id, voyage_id, topic, area_type, deck_level, service_code, severity, sentiment, model_version)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetFeedbackInsight :one
SELECT * FROM feedback_insights WHERE id = $1;

-- name: ListFeedbackInsights :many
SELECT * FROM feedback_insights
WHERE ship_id = sqlc.arg('ship_id')
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');
