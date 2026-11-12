package ship

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ren0503/tycoon/internal/application/audit"
	appship "github.com/ren0503/tycoon/internal/application/ship"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/convert"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
)

type Tx struct {
	tx pgx.Tx
	q  *sqlc.Queries
}

var _ appship.Tx = (*Tx)(nil)

func (t *Tx) ShipExists(ctx context.Context, p appship.ShipExistsParams) (bool, error) {
	ok, err := t.q.ShipExists(ctx, convert.To[sqlc.ShipExistsParams](p))
	return ok, postgres.MapPgError(err)
}
func (t *Tx) GetShipStructureImportForUpdate(ctx context.Context, p appship.GetShipStructureImportForUpdateParams) (appship.ShipStructureImport, error) {
	row, err := t.q.GetShipStructureImportForUpdate(ctx, convert.To[sqlc.GetShipStructureImportForUpdateParams](p))
	return convert.To[appship.ShipStructureImport](row), postgres.MapPgError(err)
}
func (t *Tx) FailShipStructureImport(ctx context.Context, p appship.FailShipStructureImportParams) (appship.ShipStructureImport, error) {
	row, err := t.q.FailShipStructureImport(ctx, convert.To[sqlc.FailShipStructureImportParams](p))
	return convert.To[appship.ShipStructureImport](row), postgres.MapPgError(err)
}
func (t *Tx) SetShipStructureImportProcessing(ctx context.Context, p appship.SetShipStructureImportProcessingParams) (int64, error) {
	n, err := t.q.SetShipStructureImportProcessing(ctx, convert.To[sqlc.SetShipStructureImportProcessingParams](p))
	return n, postgres.MapPgError(err)
}
func (t *Tx) CreateDeck(ctx context.Context, p appship.CreateDeckParams) (appship.Deck, error) {
	row, err := t.q.CreateDeck(ctx, convert.To[sqlc.CreateDeckParams](p))
	return convert.To[appship.Deck](row), postgres.MapPgError(err)
}
func (t *Tx) CreateArea(ctx context.Context, p appship.CreateAreaParams) (appship.Area, error) {
	row, err := t.q.CreateArea(ctx, convert.To[sqlc.CreateAreaParams](p))
	return convert.To[appship.Area](row), postgres.MapPgError(err)
}
func (t *Tx) CreateRoom(ctx context.Context, p appship.CreateRoomParams) (appship.Room, error) {
	row, err := t.q.CreateRoom(ctx, convert.To[sqlc.CreateRoomParams](p))
	return convert.To[appship.Room](row), postgres.MapPgError(err)
}
func (t *Tx) CompleteShipStructureImport(ctx context.Context, p appship.CompleteShipStructureImportParams) (appship.ShipStructureImport, error) {
	row, err := t.q.CompleteShipStructureImport(ctx, convert.To[sqlc.CompleteShipStructureImportParams](p))
	return convert.To[appship.ShipStructureImport](row), postgres.MapPgError(err)
}
func (t *Tx) ListExistingDeckLevels(ctx context.Context, shipID uuid.UUID) ([]int32, error) {
	rows, err := t.q.ListExistingDeckLevels(ctx, shipID)
	return rows, postgres.MapPgError(err)
}
func (t *Tx) ListExistingRoomNumbers(ctx context.Context, shipID uuid.UUID) ([]string, error) {
	rows, err := t.q.ListExistingRoomNumbers(ctx, shipID)
	return rows, postgres.MapPgError(err)
}
func (t *Tx) AppendAudit(ctx context.Context, entry audit.Entry) error {
	return postgres.AppendAuditTx(ctx, t.tx, entry)
}
func (t *Tx) Commit(ctx context.Context) error   { return postgres.MapPgError(t.tx.Commit(ctx)) }
func (t *Tx) Rollback(ctx context.Context) error { return postgres.MapPgError(t.tx.Rollback(ctx)) }
