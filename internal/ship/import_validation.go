package ship

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	maxWorkbookBytes           = 10 << 20
	maxArchiveBytes            = 64 << 20
	maxArchiveEntryBytes       = 32 << 20
	maxSheetDataRows           = 5000
	maxAreaCapacity            = math.MaxInt32
	maxRoomCapacity      int64 = 32767
)

var (
	deckColumns = []string{"level", "name"}
	areaColumns = []string{"deck_level", "name", "area_type", "capacity", "layout"}
	roomColumns = []string{"deck_level", "number", "room_type", "capacity", "is_bookable", "layout"}
)

type workbookValidation struct {
	Counts   ImportCounts
	Payload  ImportPayload
	Findings []ValidationFinding
	Summary  string
}

type sheetRow struct {
	row      int
	findings []ValidationFinding
}

func (r *sheetRow) add(sheet, column, code, message string) {
	r.findings = append(r.findings, ValidationFinding{Sheet: sheet, Row: r.row, Column: column, Code: code, Message: message})
}

func validateWorkbook(data []byte, fileName string, existingDeckLevels map[int32]bool, existingRoomNumbers map[string]bool) workbookValidation {
	result := workbookValidation{
		Payload:  ImportPayload{Decks: []ImportedDeck{}, Areas: []ImportedArea{}, Rooms: []ImportedRoom{}},
		Findings: []ValidationFinding{},
	}
	fileFindings := checkWorkbookFile(data, fileName)
	if len(fileFindings) > 0 {
		result.Findings = append(result.Findings, fileFindings...)
		result.Summary = "The workbook file could not be validated."
		return result
	}

	file, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{
		UnzipSizeLimit:    maxArchiveBytes,
		UnzipXMLSizeLimit: maxArchiveEntryBytes,
	})
	if err != nil {
		result.Findings = append(result.Findings, ValidationFinding{Code: "file_invalid", Message: "The workbook is not a readable .xlsx file."})
		result.Summary = "The workbook file could not be validated."
		return result
	}
	defer func() { _ = file.Close() }()

	sheets, sheetFindings := resolveSheets(file.GetSheetList())
	result.Findings = append(result.Findings, sheetFindings...)

	decks := parseDeckRows(file, sheets["Decks"], &result.Findings)
	areas := parseAreaRows(file, sheets["Areas"], &result.Findings)
	rooms := parseRoomRows(file, sheets["Rooms"], &result.Findings)

	markDeckConflicts(decks, existingDeckLevels)
	validDeckLevels := make(map[int32]bool)
	for _, row := range decks {
		if len(row.findings) == 0 {
			validDeckLevels[row.levelValue] = true
		}
	}
	markAreaReferences(areas, validDeckLevels)
	markRoomReferences(rooms, validDeckLevels)
	markRoomConflicts(rooms, existingRoomNumbers)

	decksTotal, decksValid := countRows(decks)
	areasTotal, areasValid := countRows(areas)
	roomsTotal, roomsValid := countRows(rooms)
	result.Counts = ImportCounts{
		Decks: ImportEntityCounts{Total: decksTotal, Valid: decksValid, Invalid: decksTotal - decksValid},
		Areas: ImportEntityCounts{Total: areasTotal, Valid: areasValid, Invalid: areasTotal - areasValid},
		Rooms: ImportEntityCounts{Total: roomsTotal, Valid: roomsValid, Invalid: roomsTotal - roomsValid},
	}
	for _, row := range decks {
		result.Findings = append(result.Findings, row.findings...)
		if len(row.findings) == 0 {
			result.Payload.Decks = append(result.Payload.Decks, ImportedDeck{Level: row.levelValue, Name: row.name})
		}
	}
	for _, row := range areas {
		result.Findings = append(result.Findings, row.findings...)
		if len(row.findings) == 0 {
			result.Payload.Areas = append(result.Payload.Areas, ImportedArea{
				DeckLevel: row.deckLevel, Name: row.name, AreaType: row.areaType,
				Capacity: row.capacity, Layout: row.layout,
			})
		}
	}
	for _, row := range rooms {
		result.Findings = append(result.Findings, row.findings...)
		if len(row.findings) == 0 {
			result.Payload.Rooms = append(result.Payload.Rooms, ImportedRoom{
				DeckLevel: row.deckLevel, Number: row.number, RoomType: row.roomType,
				Capacity: row.capacity, IsBookable: row.bookable, Layout: row.layout,
			})
		}
	}

	if decksTotal == 0 && (areasTotal > 0 || roomsTotal > 0) {
		result.Findings = append(result.Findings, ValidationFinding{
			Sheet: "Decks", Code: "missing_deck_records",
			Message: "Areas or rooms are present but the Decks sheet has no data rows.",
		})
	}
	if decksTotal+areasTotal+roomsTotal == 0 {
		result.Findings = append(result.Findings, ValidationFinding{
			Code: "no_importable_records", Message: "The workbook contains no non-blank data rows.",
		})
	}
	if len(result.Findings) > 0 {
		result.Summary = "Workbook validation failed; review the findings and upload a corrected file."
	}
	return result
}

func checkWorkbookFile(data []byte, fileName string) []ValidationFinding {
	var findings []ValidationFinding
	add := func(sheet, code, message string) {
		findings = append(findings, ValidationFinding{Sheet: sheet, Code: code, Message: message})
	}
	if int64(len(data)) > maxWorkbookBytes {
		add("", "file_too_large", "The uploaded workbook exceeds the 10 MiB limit.")
		return findings
	}
	if !strings.EqualFold(filepath.Ext(fileName), ".xlsx") {
		add("", "unsupported_format", "Only .xlsx workbooks are supported.")
		return findings
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		add("", "unsupported_format", "The uploaded file is not an .xlsx workbook.")
		return findings
	}
	var total uint64
	for _, entry := range reader.File {
		lower := strings.ToLower(entry.Name)
		if strings.HasSuffix(lower, "vbaproject.bin") {
			add("", "macro_enabled", "Macro-enabled workbooks are not supported.")
			return findings
		}
		if entry.UncompressedSize64 > maxArchiveEntryBytes {
			add("", "archive_too_large", "The workbook archive expands beyond the safety limit.")
			return findings
		}
		total += entry.UncompressedSize64
		if total > maxArchiveBytes {
			add("", "archive_too_large", "The workbook archive expands beyond the safety limit.")
			return findings
		}
	}
	return findings
}

func resolveSheets(names []string) (map[string]string, []ValidationFinding) {
	resolved := map[string]string{}
	counts := map[string]int{}
	required := []string{"Decks", "Areas", "Rooms"}
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		for _, want := range required {
			if strings.EqualFold(trimmed, want) {
				counts[want]++
				if counts[want] == 1 {
					resolved[want] = name
				}
			}
		}
	}
	var findings []ValidationFinding
	for _, want := range required {
		switch counts[want] {
		case 0:
			findings = append(findings, ValidationFinding{
				Sheet: want, Code: "missing_sheet", Message: "Required sheet " + want + " is missing.",
			})
		case 1:
		default:
			findings = append(findings, ValidationFinding{
				Sheet: want, Code: "duplicate_sheet", Message: "More than one sheet matches the required name " + want + ".",
			})
			delete(resolved, want)
		}
	}
	return resolved, findings
}

type deckCandidate struct {
	sheetRow
	levelValue int32
	name       string
	hasLevel   bool
}

type areaCandidate struct {
	sheetRow
	deckLevel int32
	name      string
	areaType  string
	capacity  *int32
	layout    json.RawMessage
}

type roomCandidate struct {
	sheetRow
	deckLevel int32
	number    string
	roomType  string
	capacity  int16
	bookable  bool
	layout    json.RawMessage
}

func parseDeckRows(file *excelize.File, sheet string, all *[]ValidationFinding) []*deckCandidate {
	rows, columns, ok := readSheet(file, sheet, deckColumns, all)
	if !ok {
		return nil
	}
	out := make([]*deckCandidate, 0, len(rows))
	for _, row := range rows {
		candidate := &deckCandidate{sheetRow: sheetRow{row: row.number}}
		rawLevel := cell(row.cells, columns, "level")
		if rawLevel == "" {
			candidate.add("Decks", "level", "required", "Deck level is required.")
		} else if level, err := parseWholeNumber(rawLevel); err != nil || level < 0 || level > math.MaxInt32 {
			candidate.add("Decks", "level", "invalid_type", "Deck level must be a whole number greater than or equal to zero.")
		} else {
			candidate.levelValue = int32(level)
			candidate.hasLevel = true
		}
		candidate.name = cell(row.cells, columns, "name")
		if candidate.name == "" {
			candidate.add("Decks", "name", "required", "Deck name is required.")
		}
		out = append(out, candidate)
	}
	return out
}

func parseAreaRows(file *excelize.File, sheet string, all *[]ValidationFinding) []*areaCandidate {
	rows, columns, ok := readSheet(file, sheet, areaColumns, all)
	if !ok {
		return nil
	}
	out := make([]*areaCandidate, 0, len(rows))
	for _, row := range rows {
		candidate := &areaCandidate{sheetRow: sheetRow{row: row.number}}
		candidate.deckLevel = requiredInt(row.cells, columns, "deck_level", "Areas", candidate)
		candidate.name = requiredText(row.cells, columns, "name", "Areas", "Area name", candidate)
		candidate.areaType = strings.TrimSpace(cell(row.cells, columns, "area_type"))
		if candidate.areaType == "" {
			candidate.add("Areas", "area_type", "required", "Area type is required.")
		} else if !areaTypes[candidate.areaType] {
			candidate.add("Areas", "area_type", "invalid_enum", "Area type is not recognised.")
		}
		if raw := strings.TrimSpace(cell(row.cells, columns, "capacity")); raw != "" {
			capacity, err := parseWholeNumber(raw)
			if err != nil || capacity < 0 || capacity > maxAreaCapacity {
				candidate.add("Areas", "capacity", "invalid_capacity", "Area capacity must be a whole number greater than or equal to zero.")
			} else {
				value := int32(capacity)
				candidate.capacity = &value
			}
		}
		candidate.layout = parseLayout(row.cells, columns, "layout", "Areas", candidate)
		out = append(out, candidate)
	}
	return out
}

func parseRoomRows(file *excelize.File, sheet string, all *[]ValidationFinding) []*roomCandidate {
	rows, columns, ok := readSheet(file, sheet, roomColumns, all)
	if !ok {
		return nil
	}
	out := make([]*roomCandidate, 0, len(rows))
	for _, row := range rows {
		candidate := &roomCandidate{sheetRow: sheetRow{row: row.number}}
		candidate.deckLevel = requiredInt(row.cells, columns, "deck_level", "Rooms", candidate)
		candidate.number = requiredText(row.cells, columns, "number", "Rooms", "Room number", candidate)
		candidate.roomType = strings.TrimSpace(cell(row.cells, columns, "room_type"))
		if candidate.roomType == "" {
			candidate.add("Rooms", "room_type", "required", "Room type is required.")
		} else if !roomTypes[candidate.roomType] {
			candidate.add("Rooms", "room_type", "invalid_enum", "Room type is not recognised.")
		}
		rawCapacity := strings.TrimSpace(cell(row.cells, columns, "capacity"))
		if rawCapacity == "" {
			candidate.add("Rooms", "capacity", "required", "Room capacity is required.")
		} else if capacity, err := parseWholeNumber(rawCapacity); err != nil || capacity <= 0 || capacity > maxRoomCapacity {
			candidate.add("Rooms", "capacity", "invalid_capacity", "Room capacity must be a whole number between 1 and 32767.")
		} else {
			candidate.capacity = int16(capacity)
		}
		candidate.bookable = true
		if raw := strings.TrimSpace(cell(row.cells, columns, "is_bookable")); raw != "" {
			parsed, ok := parseBool(raw)
			if !ok {
				candidate.add("Rooms", "is_bookable", "invalid_boolean", "is_bookable must be true/false, yes/no, or 1/0.")
			} else {
				candidate.bookable = parsed
			}
		}
		candidate.layout = parseLayout(row.cells, columns, "layout", "Rooms", candidate)
		out = append(out, candidate)
	}
	return out
}

func markDeckConflicts(rows []*deckCandidate, existing map[int32]bool) {
	byLevel := map[int32][]*deckCandidate{}
	for _, row := range rows {
		if row.hasLevel {
			byLevel[row.levelValue] = append(byLevel[row.levelValue], row)
		}
	}
	for level, group := range byLevel {
		if existing[level] {
			for _, row := range group {
				row.add("Decks", "level", "conflict", fmt.Sprintf("Deck level %d already exists on this ship.", level))
			}
		}
		if len(group) > 1 {
			for _, row := range group {
				row.add("Decks", "level", "duplicate", fmt.Sprintf("Deck level %d appears more than once in the workbook.", level))
			}
		}
	}
}

func markAreaReferences(rows []*areaCandidate, validDeckLevels map[int32]bool) {
	for _, row := range rows {
		if row.deckLevel == 0 && hasCode(row.findings, "deck_level") {
			continue
		}
		if !validDeckLevels[row.deckLevel] {
			row.add("Areas", "deck_level", "invalid_reference", "deck_level must reference a valid row in the Decks sheet.")
		}
	}
}

func markRoomReferences(rows []*roomCandidate, validDeckLevels map[int32]bool) {
	for _, row := range rows {
		if row.deckLevel == 0 && hasCode(row.findings, "deck_level") {
			continue
		}
		if !validDeckLevels[row.deckLevel] {
			row.add("Rooms", "deck_level", "invalid_reference", "deck_level must reference a valid row in the Decks sheet.")
		}
	}
}

func markRoomConflicts(rows []*roomCandidate, existing map[string]bool) {
	byNumber := map[string][]*roomCandidate{}
	for _, row := range rows {
		number := strings.TrimSpace(row.number)
		if number == "" {
			continue
		}
		byNumber[number] = append(byNumber[number], row)
		if existing[number] {
			row.add("Rooms", "number", "conflict", "Room number already exists on this ship.")
		}
	}
	for number, group := range byNumber {
		if len(group) > 1 {
			for _, row := range group {
				row.add("Rooms", "number", "duplicate", "Room number "+number+" appears more than once in the workbook.")
			}
		}
	}
}

func countRows[T any](rows []T) (int32, int32) {
	var total, valid int32
	for _, row := range rows {
		total++
		if !rowHasFindings(row) {
			valid++
		}
	}
	return total, valid
}

func rowHasFindings(row any) bool {
	switch candidate := row.(type) {
	case *deckCandidate:
		return len(candidate.findings) > 0
	case *areaCandidate:
		return len(candidate.findings) > 0
	case *roomCandidate:
		return len(candidate.findings) > 0
	}
	return true
}

func readSheet(file *excelize.File, sheet string, required []string, all *[]ValidationFinding) ([]parsedRow, map[string]int, bool) {
	if sheet == "" {
		return nil, nil, false
	}
	rows, err := file.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		*all = append(*all, ValidationFinding{Sheet: sheet, Code: "file_invalid", Message: "The workbook sheet could not be read."})
		return nil, nil, false
	}
	headers := make(map[string]int)
	seen := make(map[string]int)
	for index, raw := range rows[0] {
		key := strings.ToLower(strings.TrimSpace(raw))
		if key == "" {
			continue
		}
		seen[key]++
		if _, exists := headers[key]; !exists {
			headers[key] = index
		}
	}
	for _, column := range required {
		switch seen[column] {
		case 0:
			*all = append(*all, ValidationFinding{
				Sheet: sheet, Row: 1, Column: column, Code: "missing_column",
				Message: "Required column " + column + " is missing.",
			})
		case 1:
		default:
			*all = append(*all, ValidationFinding{
				Sheet: sheet, Row: 1, Column: column, Code: "duplicate_column",
				Message: "Required column " + column + " appears more than once.",
			})
		}
	}

	var out []parsedRow
	for index, row := range rows[1:] {
		if isBlank(row) {
			continue
		}
		out = append(out, parsedRow{number: index + 2, cells: row})
		if len(out) > maxSheetDataRows {
			*all = append(*all, ValidationFinding{
				Sheet: sheet, Code: "too_many_rows",
				Message: "A sheet may contain at most 5,000 non-blank data rows.",
			})
			return nil, headers, false
		}
	}
	return out, headers, true
}

type parsedRow struct {
	number int
	cells  []string
}

func cell(row []string, columns map[string]int, name string) string {
	index, ok := columns[name]
	if !ok || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func requiredInt(row []string, columns map[string]int, column, sheet string, candidate interface {
	add(sheet, column, code, message string)
}) int32 {
	raw := cell(row, columns, column)
	if raw == "" {
		candidate.add(sheet, column, "required", column+" is required.")
		return 0
	}
	value, err := parseWholeNumber(raw)
	if err != nil || value < 0 || value > math.MaxInt32 {
		candidate.add(sheet, column, "invalid_type", column+" must be a whole number greater than or equal to zero.")
		return 0
	}
	return int32(value)
}

func requiredText(row []string, columns map[string]int, column, sheet, label string, candidate interface {
	add(sheet, column, code, message string)
}) string {
	value := cell(row, columns, column)
	if value == "" {
		candidate.add(sheet, column, "required", label+" is required.")
	}
	return value
}

func parseLayout(row []string, columns map[string]int, column, sheet string, candidate interface {
	add(sheet, column, code, message string)
}) json.RawMessage {
	raw := cell(row, columns, column)
	if raw == "" {
		return nil
	}
	if !json.Valid([]byte(raw)) {
		candidate.add(sheet, column, "invalid_json", "layout must be valid JSON when present.")
		return nil
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(raw)); err != nil {
		candidate.add(sheet, column, "invalid_json", "layout must be valid JSON when present.")
		return nil
	}
	return json.RawMessage(compact.Bytes())
}

func parseWholeNumber(raw string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
}

func parseBool(raw string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "yes", "1":
		return true, true
	case "false", "no", "0":
		return false, true
	default:
		return false, false
	}
}

func isBlank(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func hasCode(findings []ValidationFinding, column string) bool {
	for _, finding := range findings {
		if finding.Column == column {
			return true
		}
	}
	return false
}
