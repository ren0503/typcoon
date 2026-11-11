package ship

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ren0503/tycoon/internal/apperrors"
	"github.com/ren0503/tycoon/internal/audit"
	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/database"
	"github.com/ren0503/tycoon/internal/database/sqlc"
)

const staleImportAfter = 10 * time.Minute

func (s *service) UploadStructureImport(ctx context.Context, p auth.Principal, shipID uuid.UUID, fileName string, data []byte, idempotencyKey string) (ImportPreview, error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		return ImportPreview{}, importNotFound()
	}
	digest := sha256.Sum256(data)
	fileSHA := hex.EncodeToString(digest[:])

	if idempotencyKey != "" {
		existing, err := s.repo.getImportByIdempotencyKey(ctx, shipID, idempotencyKey)
		if err == nil {
			return s.replayImport(ctx, existing, fileSHA)
		}
		if !isNotFound(err) {
			return ImportPreview{}, err
		}
	}

	id, err := uuid.NewV7()
	if err != nil {
		return ImportPreview{}, apperrors.Internal(err)
	}
	var createdBy *string
	if p.ActorID != "" {
		createdBy = &p.ActorID
	}
	var key *string
	if idempotencyKey != "" {
		key = &idempotencyKey
	}
	created, err := s.repo.createImport(ctx, sqlc.CreateShipStructureImportParams{
		ID: id, ShipID: shipID, IdempotencyKey: key, Status: "validating",
		FileName: fileName, FileSizeBytes: int64(len(data)), FileSha256: fileSHA,
		ValidationErrors: json.RawMessage("[]"), CreatedBy: createdBy,
	})
	if err != nil {
		if idempotencyKey != "" && isIdempotencyConflict(err) {
			existing, getErr := s.repo.getImportByIdempotencyKey(ctx, shipID, idempotencyKey)
			if getErr == nil {
				return s.replayImport(ctx, existing, fileSHA)
			}
		}
		return ImportPreview{}, err
	}

	existingLevels, err := s.repo.existingDeckLevels(ctx, shipID)
	if err != nil {
		return ImportPreview{}, err
	}
	existingNumbers, err := s.repo.existingRoomNumbers(ctx, shipID)
	if err != nil {
		return ImportPreview{}, err
	}
	validation := validateWorkbook(data, fileName, existingLevels, existingNumbers)
	payload, err := json.Marshal(validation.Payload)
	if err != nil {
		return ImportPreview{}, apperrors.Internal(err)
	}
	findings, err := json.Marshal(validation.Findings)
	if err != nil {
		return ImportPreview{}, apperrors.Internal(err)
	}
	status := "ready"
	if len(validation.Findings) > 0 {
		status = "failed"
	}
	var summary *string
	if validation.Summary != "" {
		summary = &validation.Summary
	}
	updated, err := s.repo.updateImportValidation(ctx, sqlc.UpdateShipStructureImportValidationParams{
		Status:     status,
		DecksTotal: validation.Counts.Decks.Total, DecksValid: validation.Counts.Decks.Valid, DecksInvalid: validation.Counts.Decks.Invalid,
		AreasTotal: validation.Counts.Areas.Total, AreasValid: validation.Counts.Areas.Valid, AreasInvalid: validation.Counts.Areas.Invalid,
		RoomsTotal: validation.Counts.Rooms.Total, RoomsValid: validation.Counts.Rooms.Valid, RoomsInvalid: validation.Counts.Rooms.Invalid,
		ValidatedPayload: payload, ValidationErrors: findings, ErrorMessage: summary,
		ID: created.ID, ShipID: shipID,
	})
	if err != nil {
		return ImportPreview{}, err
	}
	return importPreview(updated), nil
}

func (s *service) replayImport(ctx context.Context, existing sqlc.ShipStructureImport, fileSHA string) (ImportPreview, error) {
	if existing.FileSha256 != fileSHA {
		return ImportPreview{}, apperrors.Conflict(apperrors.CodeIdempotencyConflict,
			"Idempotency key was already used with a different file.")
	}
	recovered, err := s.recoverStaleImport(ctx, existing)
	if err != nil {
		return ImportPreview{}, err
	}
	return importPreview(recovered), nil
}

func (s *service) GetStructureImport(ctx context.Context, p auth.Principal, shipID, importID uuid.UUID) (ShipStructureImportResponse, error) {
	row, err := s.loadImport(ctx, p, shipID, importID)
	if err != nil {
		return ShipStructureImportResponse{}, err
	}
	return importResponse(row), nil
}

func (s *service) PreviewStructureImport(ctx context.Context, p auth.Principal, shipID, importID uuid.UUID) (ImportPreview, error) {
	row, err := s.loadImport(ctx, p, shipID, importID)
	if err != nil {
		return ImportPreview{}, err
	}
	return importPreview(row), nil
}

func (s *service) loadImport(ctx context.Context, p auth.Principal, shipID, importID uuid.UUID) (sqlc.ShipStructureImport, error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		return sqlc.ShipStructureImport{}, importNotFound()
	}
	row, err := s.repo.getImport(ctx, importID, shipID)
	if err != nil {
		if isNotFound(err) {
			return sqlc.ShipStructureImport{}, importNotFound()
		}
		return sqlc.ShipStructureImport{}, err
	}
	return s.recoverStaleImport(ctx, row)
}

func (s *service) recoverStaleImport(ctx context.Context, row sqlc.ShipStructureImport) (sqlc.ShipStructureImport, error) {
	if (row.Status == "validating" || row.Status == "uploaded") && time.Since(row.UpdatedAt) > staleImportAfter {
		message := "Validation was interrupted. Upload the workbook again to retry."
		recovered, err := s.repo.failImport(ctx, row.ID, row.ShipID, message)
		if err != nil {
			if isNotFound(err) {
				if fresh, getErr := s.repo.getImport(ctx, row.ID, row.ShipID); getErr == nil {
					return fresh, nil
				}
			}
			return sqlc.ShipStructureImport{}, err
		}
		return recovered, nil
	}
	return row, nil
}

func (s *service) ConfirmStructureImport(ctx context.Context, p auth.Principal, shipID, importID uuid.UUID) (ShipStructureImportResponse, error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		return ShipStructureImportResponse{}, importNotFound()
	}
	tx, err := s.repo.begin(ctx)
	if err != nil {
		return ShipStructureImportResponse{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := sqlc.New(tx)
	exists, err := q.ShipExists(ctx, sqlc.ShipExistsParams{ID: shipID, TenantID: nullUUID(p.TenantID)})
	if err != nil {
		return ShipStructureImportResponse{}, database.MapPgError(err)
	}
	if !exists {
		return ShipStructureImportResponse{}, importNotFound()
	}

	row, err := q.GetShipStructureImportForUpdate(ctx, sqlc.GetShipStructureImportForUpdateParams{ID: importID, ShipID: shipID})
	if err != nil {
		if isNotFound(database.MapPgError(err)) {
			return ShipStructureImportResponse{}, importNotFound()
		}
		return ShipStructureImportResponse{}, database.MapPgError(err)
	}
	if (row.Status == "validating" || row.Status == "uploaded") && time.Since(row.UpdatedAt) > staleImportAfter {
		failed, failErr := q.FailShipStructureImport(ctx, sqlc.FailShipStructureImportParams{
			ID: importID, ShipID: shipID, ErrorMessage: stringPtr("Validation was interrupted. Upload the workbook again to retry."),
		})
		if failErr != nil {
			return ShipStructureImportResponse{}, database.MapPgError(failErr)
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return ShipStructureImportResponse{}, database.MapPgError(commitErr)
		}
		_ = failed
		return ShipStructureImportResponse{}, apperrors.Conflict(apperrors.CodeConflict,
			"The import is no longer in a confirmable state.")
	}
	if row.Status == "completed" {
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return ShipStructureImportResponse{}, database.MapPgError(commitErr)
		}
		return importResponse(row), nil
	}
	if row.Status != "ready" {
		return ShipStructureImportResponse{}, apperrors.Conflict(apperrors.CodeConflict,
			"The import is not ready to confirm.")
	}

	var payload ImportPayload
	if len(row.ValidatedPayload) == 0 || json.Unmarshal(row.ValidatedPayload, &payload) != nil {
		_ = tx.Rollback(ctx)
		s.recordConfirmError(ctx, shipID, importID, "The validated import payload is unavailable. Upload the workbook again.")
		return ShipStructureImportResponse{}, apperrors.Internal(fmt.Errorf("validated import payload is missing"))
	}
	if err := checkConfirmationConflicts(ctx, q, shipID, payload); err != nil {
		_ = tx.Rollback(ctx)
		s.recordConfirmError(ctx, shipID, importID, "The ship structure changed since validation. Upload the workbook again.")
		return ShipStructureImportResponse{}, err
	}
	affected, err := q.SetShipStructureImportProcessing(ctx, sqlc.SetShipStructureImportProcessingParams{ID: importID, ShipID: shipID})
	if err != nil {
		return ShipStructureImportResponse{}, database.MapPgError(err)
	}
	if affected != 1 {
		return ShipStructureImportResponse{}, apperrors.Conflict(apperrors.CodeConflict,
			"The import is not ready to confirm.")
	}

	deckIDs := make(map[int32]uuid.UUID, len(payload.Decks))
	for _, deck := range payload.Decks {
		id, idErr := uuid.NewV7()
		if idErr != nil {
			return s.confirmFailed(ctx, tx, shipID, importID, idErr)
		}
		if _, insertErr := q.CreateDeck(ctx, sqlc.CreateDeckParams{ID: id, ShipID: shipID, Level: deck.Level, Name: deck.Name}); insertErr != nil {
			return s.confirmFailed(ctx, tx, shipID, importID, database.MapPgError(insertErr))
		}
		deckIDs[deck.Level] = id
	}
	for _, area := range payload.Areas {
		deckID, ok := deckIDs[area.DeckLevel]
		if !ok {
			return s.confirmFailed(ctx, tx, shipID, importID, apperrors.Conflict(apperrors.CodeConflict,
				"An area references a deck that is not part of this import."))
		}
		id, idErr := uuid.NewV7()
		if idErr != nil {
			return s.confirmFailed(ctx, tx, shipID, importID, idErr)
		}
		if _, insertErr := q.CreateArea(ctx, sqlc.CreateAreaParams{
			ID: id, ShipID: shipID, DeckID: deckID, Name: area.Name, AreaType: area.AreaType,
			Capacity: area.Capacity, Layout: area.Layout,
		}); insertErr != nil {
			return s.confirmFailed(ctx, tx, shipID, importID, database.MapPgError(insertErr))
		}
	}
	for _, room := range payload.Rooms {
		deckID, ok := deckIDs[room.DeckLevel]
		if !ok {
			return s.confirmFailed(ctx, tx, shipID, importID, apperrors.Conflict(apperrors.CodeConflict,
				"A room references a deck that is not part of this import."))
		}
		id, idErr := uuid.NewV7()
		if idErr != nil {
			return s.confirmFailed(ctx, tx, shipID, importID, idErr)
		}
		if _, insertErr := q.CreateRoom(ctx, sqlc.CreateRoomParams{
			ID: id, ShipID: shipID, DeckID: deckID, Number: room.Number, RoomType: room.RoomType,
			Capacity: room.Capacity, IsBookable: room.IsBookable, Layout: room.Layout,
		}); insertErr != nil {
			return s.confirmFailed(ctx, tx, shipID, importID, database.MapPgError(insertErr))
		}
	}

	actorType := string(p.ActorType)
	if actorType == "" {
		actorType = string(auth.ActorSystem)
	}
	var actorID *string
	if p.ActorID != "" {
		actorID = &p.ActorID
	}
	entity := "ship_structure_import"
	entityID := importID.String()
	ship := shipID
	details, _ := json.Marshal(map[string]any{
		"import_id": importID.String(),
		"created": map[string]int{
			"decks": len(payload.Decks), "areas": len(payload.Areas), "rooms": len(payload.Rooms),
		},
		"outcome": "completed",
	})
	if err := s.audit.AppendTx(ctx, tx, audit.Entry{
		Source: "cloud", ActorType: actorType, ActorID: actorID,
		Action: "ship.structure_import.confirmed", Entity: &entity, EntityID: &entityID,
		ShipID: &ship, Details: details,
	}); err != nil {
		return s.confirmFailed(ctx, tx, shipID, importID, err)
	}
	completed, err := q.CompleteShipStructureImport(ctx, sqlc.CompleteShipStructureImportParams{ID: importID, ShipID: shipID})
	if err != nil {
		return s.confirmFailed(ctx, tx, shipID, importID, database.MapPgError(err))
	}
	if err := tx.Commit(ctx); err != nil {
		return s.confirmFailed(ctx, tx, shipID, importID, database.MapPgError(err))
	}
	return importResponse(completed), nil
}

func (s *service) confirmFailed(ctx context.Context, tx pgx.Tx, shipID, importID uuid.UUID, cause error) (ShipStructureImportResponse, error) {
	_ = tx.Rollback(ctx)
	s.recordConfirmError(ctx, shipID, importID, "Confirmation failed and was rolled back. The import can be retried.")
	appErr, ok := apperrors.As(cause)
	if ok && (appErr.Code == apperrors.CodeConflict || appErr.Code == apperrors.CodeIdempotencyConflict || appErr.Code == apperrors.CodeOverlapConflict) {
		return ShipStructureImportResponse{}, cause
	}
	if ok && appErr.HTTPStatus == 500 {
		return ShipStructureImportResponse{}, cause
	}
	return ShipStructureImportResponse{}, apperrors.Internal(cause)
}

func (s *service) recordConfirmError(ctx context.Context, shipID, importID uuid.UUID, message string) {
	if err := s.repo.setImportError(ctx, importID, shipID, message); err != nil {
		s.logger.ErrorContext(ctx, "failed to record import confirmation error", "ship_id", shipID, "import_id", importID, "error", err)
	}
}

func checkConfirmationConflicts(ctx context.Context, q *sqlc.Queries, shipID uuid.UUID, payload ImportPayload) error {
	levels, err := q.ListExistingDeckLevels(ctx, shipID)
	if err != nil {
		return database.MapPgError(err)
	}
	existingLevels := make(map[int32]bool, len(levels))
	for _, level := range levels {
		existingLevels[level] = true
	}
	deckLevels := make(map[int32]bool, len(payload.Decks))
	for _, deck := range payload.Decks {
		if existingLevels[deck.Level] || deckLevels[deck.Level] {
			return apperrors.Conflict(apperrors.CodeConflict, "Deck structure conflicts with existing records.")
		}
		deckLevels[deck.Level] = true
	}
	numbers, err := q.ListExistingRoomNumbers(ctx, shipID)
	if err != nil {
		return database.MapPgError(err)
	}
	existingNumbers := make(map[string]bool, len(numbers))
	for _, number := range numbers {
		existingNumbers[number] = true
	}
	roomNumbers := make(map[string]bool, len(payload.Rooms))
	for _, room := range payload.Rooms {
		if existingNumbers[room.Number] || roomNumbers[room.Number] {
			return apperrors.Conflict(apperrors.CodeConflict, "Room structure conflicts with existing records.")
		}
		roomNumbers[room.Number] = true
	}
	for _, area := range payload.Areas {
		if !deckLevels[area.DeckLevel] {
			return apperrors.Conflict(apperrors.CodeConflict, "An area references a deck that is not part of this import.")
		}
	}
	for _, room := range payload.Rooms {
		if !deckLevels[room.DeckLevel] {
			return apperrors.Conflict(apperrors.CodeConflict, "A room references a deck that is not part of this import.")
		}
	}
	return nil
}

func importNotFound() error {
	return apperrors.NotFound(apperrors.CodeNotFound, "Import not found.")
}

func isNotFound(err error) bool {
	appErr, ok := apperrors.As(err)
	return ok && appErr.Code == apperrors.CodeNotFound
}

func isIdempotencyConflict(err error) bool {
	appErr, ok := apperrors.As(err)
	return ok && appErr.Code == apperrors.CodeIdempotencyConflict
}

func stringPtr(value string) *string { return &value }
