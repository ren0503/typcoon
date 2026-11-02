-- name: CreateShip :one
INSERT INTO ships (id, tenant_id, name, imo_number, flag_state, timezone)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetShip :one
SELECT * FROM ships WHERE id = $1 AND deleted_at IS NULL;

-- name: ListShips :many
SELECT * FROM ships
WHERE deleted_at IS NULL
  AND (sqlc.narg('tenant_id')::uuid IS NULL OR tenant_id = sqlc.narg('tenant_id'))
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateShip :one
UPDATE ships
SET name = sqlc.arg('name'),
    imo_number = sqlc.narg('imo_number'),
    flag_state = sqlc.narg('flag_state'),
    timezone = sqlc.arg('timezone'),
    status = sqlc.arg('status')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteShip :execrows
UPDATE ships SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: ShipExists :one
SELECT EXISTS (
  SELECT 1 FROM ships
  WHERE id = sqlc.arg('id') AND deleted_at IS NULL
    AND (sqlc.narg('tenant_id')::uuid IS NULL OR tenant_id = sqlc.narg('tenant_id'))
);

-- name: CreateDeck :one
INSERT INTO decks (id, ship_id, level, name) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetDeck :one
SELECT * FROM decks WHERE id = $1 AND deleted_at IS NULL;

-- name: ListDecks :many
SELECT * FROM decks
WHERE ship_id = sqlc.arg('ship_id') AND deleted_at IS NULL
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateDeck :one
UPDATE decks SET level = sqlc.arg('level'), name = sqlc.arg('name')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteDeck :execrows
UPDATE decks SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: DeckExistsInShip :one
SELECT EXISTS (SELECT 1 FROM decks WHERE id = sqlc.arg('id') AND ship_id = sqlc.arg('ship_id') AND deleted_at IS NULL);

-- name: CreateArea :one
INSERT INTO areas (id, ship_id, deck_id, name, area_type, capacity, layout)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetArea :one
SELECT * FROM areas WHERE id = $1 AND deleted_at IS NULL;

-- name: ListAreas :many
SELECT * FROM areas
WHERE ship_id = sqlc.arg('ship_id') AND deleted_at IS NULL
  AND (sqlc.narg('deck_id')::uuid IS NULL OR deck_id = sqlc.narg('deck_id'))
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateArea :one
UPDATE areas
SET deck_id = sqlc.arg('deck_id'), name = sqlc.arg('name'), area_type = sqlc.arg('area_type'),
    capacity = sqlc.narg('capacity'), layout = sqlc.narg('layout')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteArea :execrows
UPDATE areas SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateRoom :one
INSERT INTO rooms (id, ship_id, deck_id, number, room_type, capacity, is_bookable, layout)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetRoom :one
SELECT * FROM rooms WHERE id = $1 AND deleted_at IS NULL;

-- name: ListRooms :many
SELECT * FROM rooms
WHERE ship_id = sqlc.arg('ship_id') AND deleted_at IS NULL
  AND (sqlc.narg('deck_id')::uuid IS NULL OR deck_id = sqlc.narg('deck_id'))
  AND id > COALESCE(sqlc.narg('after')::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY id
LIMIT sqlc.arg('page_limit');

-- name: UpdateRoom :one
UPDATE rooms
SET deck_id = sqlc.arg('deck_id'), number = sqlc.arg('number'), room_type = sqlc.arg('room_type'),
    capacity = sqlc.arg('capacity'), is_bookable = sqlc.arg('is_bookable'), layout = sqlc.narg('layout')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteRoom :execrows
UPDATE rooms SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: RoomExists :one
SELECT EXISTS (SELECT 1 FROM rooms WHERE id = sqlc.arg('id') AND deleted_at IS NULL);
