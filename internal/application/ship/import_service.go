package ship

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/audit"
	"github.com/ren0503/tycoon/internal/application/auth"
	"github.com/ren0503/tycoon/internal/platform/errors"
)

const staleImportAfter = 10 * time.Minute

func (s *service) UploadStructureImport(ctx context.Context, p auth.Principal, shipID uuid.UUID, fileName string, data []byte, idempotencyKey string) (ImportPreview, error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		return ImportPreview{}, importNotFound()
	}
	digest := sha256.Sum256(data)
	fileSHA := hex.EncodeToString(digest[:])

	if idempotencyKey != "" {
		existing, err := s.repo.GetShipStructureImportByIdempotencyKey(ctx, GetShipStructureImportByIdempotencyKeyParams{ShipID: shipID, IdempotencyKey: &idempotencyKey})
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
	created, err := s.repo.CreateShipStructureImport(ctx, CreateShipStructureImportParams{
		ID: id, ShipID: shipID, IdempotencyKey: key, Status: "validating",
		FileName: fileName, FileSizeBytes: int64(len(data)), FileSha256: fileSHA,
		ValidationErrors: json.RawMessage("[]"), CreatedBy: createdBy,
	})
	if err != nil {
		if idempotencyKey != "" && isIdempotencyConflict(err) {
			existing, getErr := s.repo.GetShipStructureImportByIdempotencyKey(ctx, GetShipStructureImportByIdempotencyKeyParams{ShipID: shipID, IdempotencyKey: &idempotencyKey})
			if getErr == nil {
				return s.replayImport(ctx, existing, fileSHA)
			}
		}
		return ImportPreview{}, err
	}

	existingLevels, err := s.repo.ExistingDeckLevels(ctx, shipID)
	if err != nil {
		return ImportPreview{}, err
	}
	existingNumbers, err := s.repo.ExistingRoomNumbers(ctx, shipID)
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
	updated, err := s.repo.UpdateShipStructureImportValidation(ctx, UpdateShipStructureImportValidationParams{
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

func (s *service) replayImport(ctx context.Context, existing ShipStructureImport, fileSHA string) (ImportPreview, error) {
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

func (s *service) loadImport(ctx context.Context, p auth.Principal, shipID, importID uuid.UUID) (ShipStructureImport, error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		return ShipStructureImport{}, importNotFound()
	}
	row, err := s.repo.GetShipStructureImport(ctx, GetShipStructureImportParams{ID: importID, ShipID: shipID})
	if err != nil {
		if isNotFound(err) {
			return ShipStructureImport{}, importNotFound()
		}
		return ShipStructureImport{}, err
	}
	return s.recoverStaleImport(ctx, row)
}

func (s *service) recoverStaleImport(ctx context.Context, row ShipStructureImport) (ShipStructureImport, error) {
	if (row.Status == "validating" || row.Status == "uploaded") && time.Since(row.UpdatedAt) > staleImportAfter {
		message := "Validation was interrupted. Upload the workbook again to retry."
		recovered, err := s.repo.FailShipStructureImport(ctx, FailShipStructureImportParams{ID: row.ID, ShipID: row.ShipID, ErrorMessage: &message})
		if err != nil {
			if isNotFound(err) {
				if fresh, getErr := s.repo.GetShipStructureImport(ctx, GetShipStructureImportParams{ID: row.ID, ShipID: row.ShipID}); getErr == nil {
					return fresh, nil
				}
			}
			return ShipStructureImport{}, err
		}
		return recovered, nil
	}
	return row, nil
}

func (s *service) ConfirmStructureImport(ctx context.Context, p auth.Principal, shipID, importID uuid.UUID) (ShipStructureImportResponse, error) {
	if _, err := s.getShipScoped(ctx, p, shipID); err != nil {
		return ShipStructureImportResponse{}, importNotFound()
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return ShipStructureImportResponse{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	exists, err := tx.ShipExists(ctx, ShipExistsParams{ID: shipID, TenantID: nullUUID(p.TenantID)})
	if err != nil {
		return ShipStructureImportResponse{}, err
	}
	if !exists {
		return ShipStructureImportResponse{}, importNotFound()
	}

	row, err := tx.GetShipStructureImportForUpdate(ctx, GetShipStructureImportForUpdateParams{ID: importID, ShipID: shipID})
	if err != nil {
		if isNotFound(err) {
			return ShipStructureImportResponse{}, importNotFound()
		}
		return ShipStructureImportResponse{}, err
	}
	if (row.Status == "validating" || row.Status == "uploaded") && time.Since(row.UpdatedAt) > staleImportAfter {
		failed, failErr := tx.FailShipStructureImport(ctx, FailShipStructureImportParams{
			ID: importID, ShipID: shipID, ErrorMessage: stringPtr("Validation was interrupted. Upload the workbook again to retry."),
		})
		if failErr != nil {
			return ShipStructureImportResponse{}, failErr
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return ShipStructureImportResponse{}, commitErr
		}
		_ = failed
		return ShipStructureImportResponse{}, apperrors.Conflict(apperrors.CodeConflict,
			"The import is no longer in a confirmable state.")
	}
	if row.Status == "completed" {
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return ShipStructureImportResponse{}, commitErr
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
	if err := checkConfirmationConflicts(ctx, tx, shipID, payload); err != nil {
		_ = tx.Rollback(ctx)
		s.recordConfirmError(ctx, shipID, importID, "The ship structure changed since validation. Upload the workbook again.")
		return ShipStructureImportResponse{}, err
	}
	affected, err := tx.SetShipStructureImportProcessing(ctx, SetShipStructureImportProcessingParams{ID: importID, ShipID: shipID})
	if err != nil {
		return ShipStructureImportResponse{}, err
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
		if _, insertErr := tx.CreateDeck(ctx, CreateDeckParams{ID: id, ShipID: shipID, Level: deck.Level, Name: deck.Name}); insertErr != nil {
			return s.confirmFailed(ctx, tx, shipID, importID, insertErr)
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
		if _, insertErr := tx.CreateArea(ctx, CreateAreaParams{
			ID: id, ShipID: shipID, DeckID: deckID, Name: area.Name, AreaType: area.AreaType,
			Capacity: area.Capacity, Layout: area.Layout,
		}); insertErr != nil {
			return s.confirmFailed(ctx, tx, shipID, importID, insertErr)
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
		if _, insertErr := tx.CreateRoom(ctx, CreateRoomParams{
			ID: id, ShipID: shipID, DeckID: deckID, Number: room.Number, RoomType: room.RoomType,
			Capacity: room.Capacity, IsBookable: room.IsBookable, Layout: room.Layout,
		}); insertErr != nil {
			return s.confirmFailed(ctx, tx, shipID, importID, insertErr)
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
	if err := tx.AppendAudit(ctx, audit.Entry{
		Source: "cloud", ActorType: actorType, ActorID: actorID,
		Action: "ship.structure_import.confirmed", Entity: &entity, EntityID: &entityID,
		ShipID: &ship, Details: details,
	}); err != nil {
		return s.confirmFailed(ctx, tx, shipID, importID, err)
	}
	completed, err := tx.CompleteShipStructureImport(ctx, CompleteShipStructureImportParams{ID: importID, ShipID: shipID})
	if err != nil {
		return s.confirmFailed(ctx, tx, shipID, importID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return s.confirmFailed(ctx, tx, shipID, importID, err)
	}
	return importResponse(completed), nil
}

func (s *service) confirmFailed(ctx context.Context, tx Tx, shipID, importID uuid.UUID, cause error) (ShipStructureImportResponse, error) {
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
	if err := s.repo.SetShipStructureImportError(ctx, SetShipStructureImportErrorParams{ID: importID, ShipID: shipID, ErrorMessage: &message}); err != nil {
		s.logger.ErrorContext(ctx, "failed to record import confirmation error", "ship_id", shipID, "import_id", importID, "error", err)
	}
}

func checkConfirmationConflicts(ctx context.Context, tx Tx, shipID uuid.UUID, payload ImportPayload) error {
	levels, err := tx.ListExistingDeckLevels(ctx, shipID)
	if err != nil {
		return err
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
	numbers, err := tx.ListExistingRoomNumbers(ctx, shipID)
	if err != nil {
		return err
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
