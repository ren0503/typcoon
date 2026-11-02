package ship

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ren0503/tycoon/internal/database"
	"github.com/ren0503/tycoon/internal/database/sqlc"
)

type repo struct {
	q *sqlc.Queries
}

func newRepo(pool *pgxpool.Pool) *repo {
	return &repo{q: sqlc.New(pool)}
}

func (r *repo) createShip(ctx context.Context, p sqlc.CreateShipParams) (sqlc.Ship, error) {
	row, err := r.q.CreateShip(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getShip(ctx context.Context, id uuid.UUID) (sqlc.Ship, error) {
	row, err := r.q.GetShip(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listShips(ctx context.Context, p sqlc.ListShipsParams) ([]sqlc.Ship, error) {
	rows, err := r.q.ListShips(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateShip(ctx context.Context, p sqlc.UpdateShipParams) (sqlc.Ship, error) {
	row, err := r.q.UpdateShip(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteShip(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteShip(ctx, id)
	return execResult(n, err)
}

func (r *repo) shipExists(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (bool, error) {
	ok, err := r.q.ShipExists(ctx, sqlc.ShipExistsParams{ID: id, TenantID: nullUUID(tenantID)})
	return ok, database.MapPgError(err)
}

func (r *repo) createDeck(ctx context.Context, p sqlc.CreateDeckParams) (sqlc.Deck, error) {
	row, err := r.q.CreateDeck(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getDeck(ctx context.Context, id uuid.UUID) (sqlc.Deck, error) {
	row, err := r.q.GetDeck(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listDecks(ctx context.Context, p sqlc.ListDecksParams) ([]sqlc.Deck, error) {
	rows, err := r.q.ListDecks(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateDeck(ctx context.Context, p sqlc.UpdateDeckParams) (sqlc.Deck, error) {
	row, err := r.q.UpdateDeck(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteDeck(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteDeck(ctx, id)
	return execResult(n, err)
}

func (r *repo) deckExistsInShip(ctx context.Context, id, shipID uuid.UUID) (bool, error) {
	ok, err := r.q.DeckExistsInShip(ctx, sqlc.DeckExistsInShipParams{ID: id, ShipID: shipID})
	return ok, database.MapPgError(err)
}

func (r *repo) createArea(ctx context.Context, p sqlc.CreateAreaParams) (sqlc.Area, error) {
	row, err := r.q.CreateArea(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getArea(ctx context.Context, id uuid.UUID) (sqlc.Area, error) {
	row, err := r.q.GetArea(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listAreas(ctx context.Context, p sqlc.ListAreasParams) ([]sqlc.Area, error) {
	rows, err := r.q.ListAreas(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateArea(ctx context.Context, p sqlc.UpdateAreaParams) (sqlc.Area, error) {
	row, err := r.q.UpdateArea(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteArea(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteArea(ctx, id)
	return execResult(n, err)
}

func (r *repo) createRoom(ctx context.Context, p sqlc.CreateRoomParams) (sqlc.Room, error) {
	row, err := r.q.CreateRoom(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getRoom(ctx context.Context, id uuid.UUID) (sqlc.Room, error) {
	row, err := r.q.GetRoom(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listRooms(ctx context.Context, p sqlc.ListRoomsParams) ([]sqlc.Room, error) {
	rows, err := r.q.ListRooms(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateRoom(ctx context.Context, p sqlc.UpdateRoomParams) (sqlc.Room, error) {
	row, err := r.q.UpdateRoom(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteRoom(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteRoom(ctx, id)
	return execResult(n, err)
}

func nullUUID(id uuid.UUID) uuid.NullUUID {
	if id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}
