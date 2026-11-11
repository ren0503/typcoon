package ship

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/database/sqlc"
)

// ValidationFinding is one safe, row-level workbook finding.
type ValidationFinding struct {
	Sheet   string `json:"sheet"`
	Row     int    `json:"row"`
	Column  string `json:"column"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ImportEntityCounts summarizes total, valid, and invalid rows for one sheet.
type ImportEntityCounts struct {
	Total   int32 `json:"total"`
	Valid   int32 `json:"valid"`
	Invalid int32 `json:"invalid"`
}

// ImportCounts summarizes all required sheets.
type ImportCounts struct {
	Decks ImportEntityCounts `json:"decks"`
	Areas ImportEntityCounts `json:"areas"`
	Rooms ImportEntityCounts `json:"rooms"`
}

// ImportedDeck is one normalized valid deck row.
type ImportedDeck struct {
	Level int32  `json:"level"`
	Name  string `json:"name"`
}

// ImportedArea is one normalized valid area row.
type ImportedArea struct {
	DeckLevel int32           `json:"deck_level"`
	Name      string          `json:"name"`
	AreaType  string          `json:"area_type"`
	Capacity  *int32          `json:"capacity"`
	Layout    json.RawMessage `json:"layout"`
}

// ImportedRoom is one normalized valid room row.
type ImportedRoom struct {
	DeckLevel  int32           `json:"deck_level"`
	Number     string          `json:"number"`
	RoomType   string          `json:"room_type"`
	Capacity   int16           `json:"capacity"`
	IsBookable bool            `json:"is_bookable"`
	Layout     json.RawMessage `json:"layout"`
}

// ImportRecords contains all normalized valid rows.
type ImportRecords struct {
	Decks []ImportedDeck `json:"decks"`
	Areas []ImportedArea `json:"areas"`
	Rooms []ImportedRoom `json:"rooms"`
}

// ImportPayload is persisted in validated_payload.
type ImportPayload struct {
	Decks []ImportedDeck `json:"decks"`
	Areas []ImportedArea `json:"areas"`
	Rooms []ImportedRoom `json:"rooms"`
}

// ImportFile is safe display metadata for the uploaded workbook.
type ImportFile struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

// ImportTimestamps contains import lifecycle timestamps.
type ImportTimestamps struct {
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

// ShipStructureImportResponse is the public import status representation.
type ShipStructureImportResponse struct {
	ID           uuid.UUID        `json:"id"`
	ShipID       uuid.UUID        `json:"ship_id"`
	Status       string           `json:"status"`
	File         ImportFile       `json:"file"`
	Counts       ImportCounts     `json:"counts"`
	CanConfirm   bool             `json:"can_confirm"`
	ErrorSummary *string          `json:"error_summary"`
	Timestamps   ImportTimestamps `json:"timestamps"`
}

// ImportPreview is the upload/preview response.
type ImportPreview struct {
	Import  ShipStructureImportResponse `json:"import"`
	Records ImportRecords               `json:"records"`
	Errors  []ValidationFinding         `json:"errors"`
}

func importResponse(row sqlc.ShipStructureImport) ShipStructureImportResponse {
	return ShipStructureImportResponse{
		ID:     row.ID,
		ShipID: row.ShipID,
		Status: row.Status,
		File: ImportFile{
			Name:      row.FileName,
			SizeBytes: row.FileSizeBytes,
			SHA256:    row.FileSha256,
		},
		Counts: ImportCounts{
			Decks: ImportEntityCounts{Total: row.DecksTotal, Valid: row.DecksValid, Invalid: row.DecksInvalid},
			Areas: ImportEntityCounts{Total: row.AreasTotal, Valid: row.AreasValid, Invalid: row.AreasInvalid},
			Rooms: ImportEntityCounts{Total: row.RoomsTotal, Valid: row.RoomsValid, Invalid: row.RoomsInvalid},
		},
		CanConfirm:   row.Status == "ready",
		ErrorSummary: row.ErrorMessage,
		Timestamps: ImportTimestamps{
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
			CompletedAt: row.CompletedAt,
		},
	}
}

func importPreview(row sqlc.ShipStructureImport) ImportPreview {
	preview := ImportPreview{
		Import:  importResponse(row),
		Records: ImportRecords{Decks: []ImportedDeck{}, Areas: []ImportedArea{}, Rooms: []ImportedRoom{}},
		Errors:  decodeFindings(row.ValidationErrors),
	}
	if len(row.ValidatedPayload) > 0 {
		var payload ImportPayload
		if err := json.Unmarshal(row.ValidatedPayload, &payload); err == nil {
			if payload.Decks != nil {
				preview.Records.Decks = payload.Decks
			}
			if payload.Areas != nil {
				preview.Records.Areas = payload.Areas
			}
			if payload.Rooms != nil {
				preview.Records.Rooms = payload.Rooms
			}
		}
	}
	return preview
}

func decodeFindings(raw json.RawMessage) []ValidationFinding {
	findings := []ValidationFinding{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &findings)
	}
	if findings == nil {
		findings = []ValidationFinding{}
	}
	return findings
}
