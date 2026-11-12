package ship

import (
	"context"

	"github.com/google/uuid"

	appship "github.com/ren0503/tycoon/internal/application/ship"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/convert"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
)

func (r *Repo) CreateShipStructureImport(ctx context.Context, p appship.CreateShipStructureImportParams) (appship.ShipStructureImport, error) {
	row, err := r.q.CreateShipStructureImport(ctx, convert.To[sqlc.CreateShipStructureImportParams](p))
	return convert.To[appship.ShipStructureImport](row), postgres.MapPgError(err)
}
func (r *Repo) GetShipStructureImport(ctx context.Context, p appship.GetShipStructureImportParams) (appship.ShipStructureImport, error) {
	row, err := r.q.GetShipStructureImport(ctx, convert.To[sqlc.GetShipStructureImportParams](p))
	return convert.To[appship.ShipStructureImport](row), postgres.MapPgError(err)
}
func (r *Repo) GetShipStructureImportByIdempotencyKey(ctx context.Context, p appship.GetShipStructureImportByIdempotencyKeyParams) (appship.ShipStructureImport, error) {
	row, err := r.q.GetShipStructureImportByIdempotencyKey(ctx, convert.To[sqlc.GetShipStructureImportByIdempotencyKeyParams](p))
	return convert.To[appship.ShipStructureImport](row), postgres.MapPgError(err)
}
func (r *Repo) UpdateShipStructureImportValidation(ctx context.Context, p appship.UpdateShipStructureImportValidationParams) (appship.ShipStructureImport, error) {
	row, err := r.q.UpdateShipStructureImportValidation(ctx, convert.To[sqlc.UpdateShipStructureImportValidationParams](p))
	return convert.To[appship.ShipStructureImport](row), postgres.MapPgError(err)
}
func (r *Repo) FailShipStructureImport(ctx context.Context, p appship.FailShipStructureImportParams) (appship.ShipStructureImport, error) {
	row, err := r.q.FailShipStructureImport(ctx, convert.To[sqlc.FailShipStructureImportParams](p))
	return convert.To[appship.ShipStructureImport](row), postgres.MapPgError(err)
}
func (r *Repo) SetShipStructureImportError(ctx context.Context, p appship.SetShipStructureImportErrorParams) error {
	_, err := r.q.SetShipStructureImportError(ctx, convert.To[sqlc.SetShipStructureImportErrorParams](p))
	return postgres.MapPgError(err)
}
func (r *Repo) ExistingDeckLevels(ctx context.Context, shipID uuid.UUID) (map[int32]bool, error) {
	levels, err := r.q.ListExistingDeckLevels(ctx, shipID)
	out := make(map[int32]bool, len(levels))
	for _, level := range levels {
		out[level] = true
	}
	return out, postgres.MapPgError(err)
}
func (r *Repo) ExistingRoomNumbers(ctx context.Context, shipID uuid.UUID) (map[string]bool, error) {
	numbers, err := r.q.ListExistingRoomNumbers(ctx, shipID)
	out := make(map[string]bool, len(numbers))
	for _, number := range numbers {
		out[number] = true
	}
	return out, postgres.MapPgError(err)
}
