package ship

import (
	"context"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/database"
	"github.com/ren0503/tycoon/internal/database/sqlc"
)

func (r *repo) createImport(ctx context.Context, p sqlc.CreateShipStructureImportParams) (sqlc.ShipStructureImport, error) {
	row, err := r.q.CreateShipStructureImport(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getImport(ctx context.Context, id, shipID uuid.UUID) (sqlc.ShipStructureImport, error) {
	row, err := r.q.GetShipStructureImport(ctx, sqlc.GetShipStructureImportParams{ID: id, ShipID: shipID})
	return row, database.MapPgError(err)
}

func (r *repo) getImportByIdempotencyKey(ctx context.Context, shipID uuid.UUID, key string) (sqlc.ShipStructureImport, error) {
	row, err := r.q.GetShipStructureImportByIdempotencyKey(ctx, sqlc.GetShipStructureImportByIdempotencyKeyParams{
		ShipID: shipID, IdempotencyKey: &key,
	})
	return row, database.MapPgError(err)
}

func (r *repo) updateImportValidation(ctx context.Context, p sqlc.UpdateShipStructureImportValidationParams) (sqlc.ShipStructureImport, error) {
	row, err := r.q.UpdateShipStructureImportValidation(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) failImport(ctx context.Context, id, shipID uuid.UUID, message string) (sqlc.ShipStructureImport, error) {
	row, err := r.q.FailShipStructureImport(ctx, sqlc.FailShipStructureImportParams{
		ID: id, ShipID: shipID, ErrorMessage: &message,
	})
	return row, database.MapPgError(err)
}

func (r *repo) setImportError(ctx context.Context, id, shipID uuid.UUID, message string) error {
	_, err := r.q.SetShipStructureImportError(ctx, sqlc.SetShipStructureImportErrorParams{
		ID: id, ShipID: shipID, ErrorMessage: &message,
	})
	return database.MapPgError(err)
}

func (r *repo) existingDeckLevels(ctx context.Context, shipID uuid.UUID) (map[int32]bool, error) {
	levels, err := r.q.ListExistingDeckLevels(ctx, shipID)
	if err != nil {
		return nil, database.MapPgError(err)
	}
	out := make(map[int32]bool, len(levels))
	for _, level := range levels {
		out[level] = true
	}
	return out, nil
}

func (r *repo) existingRoomNumbers(ctx context.Context, shipID uuid.UUID) (map[string]bool, error) {
	numbers, err := r.q.ListExistingRoomNumbers(ctx, shipID)
	if err != nil {
		return nil, database.MapPgError(err)
	}
	out := make(map[string]bool, len(numbers))
	for _, number := range numbers {
		out[number] = true
	}
	return out, nil
}
