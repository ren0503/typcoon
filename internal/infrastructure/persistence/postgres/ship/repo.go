// Package ship implements ship persistence.
package ship

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appship "github.com/ren0503/tycoon/internal/application/ship"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/convert"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
)

type Repo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool, q: sqlc.New(pool)} }

var _ appship.Repository = (*Repo)(nil)

func (r *Repo) Begin(ctx context.Context) (appship.Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, postgres.MapPgError(err)
	}
	return &Tx{tx: tx, q: sqlc.New(tx)}, nil
}

func (r *Repo) CreateShip(ctx context.Context, p appship.CreateShipParams) (appship.Ship, error) {
	row, err := r.q.CreateShip(ctx, convert.To[sqlc.CreateShipParams](p))
	return convert.To[appship.Ship](row), postgres.MapPgError(err)
}
func (r *Repo) GetShip(ctx context.Context, id uuid.UUID) (appship.Ship, error) {
	row, err := r.q.GetShip(ctx, id)
	return convert.To[appship.Ship](row), postgres.MapPgError(err)
}
func (r *Repo) ListShips(ctx context.Context, p appship.ListShipsParams) ([]appship.Ship, error) {
	rows, err := r.q.ListShips(ctx, convert.To[sqlc.ListShipsParams](p))
	return convert.Slice[appship.Ship](rows), postgres.MapPgError(err)
}
func (r *Repo) UpdateShip(ctx context.Context, p appship.UpdateShipParams) (appship.Ship, error) {
	row, err := r.q.UpdateShip(ctx, convert.To[sqlc.UpdateShipParams](p))
	return convert.To[appship.Ship](row), postgres.MapPgError(err)
}
func (r *Repo) DeleteShip(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeleteShip(ctx, id))
}
func (r *Repo) ShipExists(ctx context.Context, id, tenantID uuid.UUID) (bool, error) {
	ok, err := r.q.ShipExists(ctx, sqlc.ShipExistsParams{ID: id, TenantID: nullUUID(tenantID)})
	return ok, postgres.MapPgError(err)
}
func (r *Repo) CreateDeck(ctx context.Context, p appship.CreateDeckParams) (appship.Deck, error) {
	row, err := r.q.CreateDeck(ctx, convert.To[sqlc.CreateDeckParams](p))
	return convert.To[appship.Deck](row), postgres.MapPgError(err)
}
func (r *Repo) GetDeck(ctx context.Context, id uuid.UUID) (appship.Deck, error) {
	row, err := r.q.GetDeck(ctx, id)
	return convert.To[appship.Deck](row), postgres.MapPgError(err)
}
func (r *Repo) ListDecks(ctx context.Context, p appship.ListDecksParams) ([]appship.Deck, error) {
	rows, err := r.q.ListDecks(ctx, convert.To[sqlc.ListDecksParams](p))
	return convert.Slice[appship.Deck](rows), postgres.MapPgError(err)
}
func (r *Repo) UpdateDeck(ctx context.Context, p appship.UpdateDeckParams) (appship.Deck, error) {
	row, err := r.q.UpdateDeck(ctx, convert.To[sqlc.UpdateDeckParams](p))
	return convert.To[appship.Deck](row), postgres.MapPgError(err)
}
func (r *Repo) DeleteDeck(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeleteDeck(ctx, id))
}
func (r *Repo) DeckExistsInShip(ctx context.Context, id, shipID uuid.UUID) (bool, error) {
	ok, err := r.q.DeckExistsInShip(ctx, sqlc.DeckExistsInShipParams{ID: id, ShipID: shipID})
	return ok, postgres.MapPgError(err)
}
func (r *Repo) CreateArea(ctx context.Context, p appship.CreateAreaParams) (appship.Area, error) {
	row, err := r.q.CreateArea(ctx, convert.To[sqlc.CreateAreaParams](p))
	return convert.To[appship.Area](row), postgres.MapPgError(err)
}
func (r *Repo) GetArea(ctx context.Context, id uuid.UUID) (appship.Area, error) {
	row, err := r.q.GetArea(ctx, id)
	return convert.To[appship.Area](row), postgres.MapPgError(err)
}
func (r *Repo) ListAreas(ctx context.Context, p appship.ListAreasParams) ([]appship.Area, error) {
	rows, err := r.q.ListAreas(ctx, convert.To[sqlc.ListAreasParams](p))
	return convert.Slice[appship.Area](rows), postgres.MapPgError(err)
}
func (r *Repo) UpdateArea(ctx context.Context, p appship.UpdateAreaParams) (appship.Area, error) {
	row, err := r.q.UpdateArea(ctx, convert.To[sqlc.UpdateAreaParams](p))
	return convert.To[appship.Area](row), postgres.MapPgError(err)
}
func (r *Repo) DeleteArea(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeleteArea(ctx, id))
}
func (r *Repo) CreateRoom(ctx context.Context, p appship.CreateRoomParams) (appship.Room, error) {
	row, err := r.q.CreateRoom(ctx, convert.To[sqlc.CreateRoomParams](p))
	return convert.To[appship.Room](row), postgres.MapPgError(err)
}
func (r *Repo) GetRoom(ctx context.Context, id uuid.UUID) (appship.Room, error) {
	row, err := r.q.GetRoom(ctx, id)
	return convert.To[appship.Room](row), postgres.MapPgError(err)
}
func (r *Repo) ListRooms(ctx context.Context, p appship.ListRoomsParams) ([]appship.Room, error) {
	rows, err := r.q.ListRooms(ctx, convert.To[sqlc.ListRoomsParams](p))
	return convert.Slice[appship.Room](rows), postgres.MapPgError(err)
}
func (r *Repo) UpdateRoom(ctx context.Context, p appship.UpdateRoomParams) (appship.Room, error) {
	row, err := r.q.UpdateRoom(ctx, convert.To[sqlc.UpdateRoomParams](p))
	return convert.To[appship.Room](row), postgres.MapPgError(err)
}
func (r *Repo) DeleteRoom(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeleteRoom(ctx, id))
}

func nullUUID(id uuid.UUID) uuid.NullUUID {
	if id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}
func mapDelete(_ int64, err error) error {
	if err != nil {
		return postgres.MapPgError(err)
	}
	return nil
}

var _ = pgx.ErrNoRows
