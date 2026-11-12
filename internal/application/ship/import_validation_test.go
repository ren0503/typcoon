package ship

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestTemplateValidatesAsReady(t *testing.T) {
	data, err := buildTemplate()
	if err != nil {
		t.Fatalf("build template: %v", err)
	}
	result := validateWorkbook(data, "ship-structure-import-template.xlsx", map[int32]bool{}, map[string]bool{})
	if len(result.Findings) != 0 {
		t.Fatalf("template findings = %#v", result.Findings)
	}
	if result.Counts.Decks.Total != 2 || result.Counts.Areas.Total != 2 || result.Counts.Rooms.Total != 2 {
		t.Fatalf("unexpected template counts: %#v", result.Counts)
	}
	if len(result.Payload.Decks) != 2 || len(result.Payload.Areas) != 2 || len(result.Payload.Rooms) != 2 {
		t.Fatalf("unexpected payload: %#v", result.Payload)
	}
}

func TestWorkbookValidationReportsDuplicatesConflictsAndReferences(t *testing.T) {
	data := workbook(t, map[string][][]any{
		"Decks": {
			{"level", "name"},
			{1, "Main"},
			{1, "Duplicate"},
			{3, "Existing"},
		},
		"Areas": {
			{"deck_level", "name", "area_type", "capacity", "layout"},
			{2, "Missing Deck", "lobby", 10, ""},
			{1, "Bad Type", "not-a-type", -1, `{`},
		},
		"Rooms": {
			{"deck_level", "number", "room_type", "capacity", "is_bookable", "layout"},
			{1, "101", "inside", 2, "maybe", ""},
			{1, "101", "inside", 2, "true", ""},
			{3, "102", "inside", 2, "true", ""},
		},
	})
	result := validateWorkbook(data, "structure.xlsx", map[int32]bool{3: true}, map[string]bool{"102": true})
	if len(result.Findings) == 0 {
		t.Fatal("expected findings")
	}
	if !hasFinding(result.Findings, "duplicate", "level") {
		t.Fatal("missing duplicate deck finding")
	}
	if !hasFinding(result.Findings, "conflict", "level") {
		t.Fatal("missing existing deck conflict")
	}
	if !hasFinding(result.Findings, "invalid_reference", "deck_level") {
		t.Fatal("missing deck reference finding")
	}
	if !hasFinding(result.Findings, "invalid_boolean", "is_bookable") {
		t.Fatal("missing boolean finding")
	}
	if result.Counts.Decks.Invalid != 3 || result.Counts.Areas.Invalid != 2 || result.Counts.Rooms.Invalid != 3 {
		t.Fatalf("unexpected counts: %#v", result.Counts)
	}
}

func TestWorkbookValidationRejectsMacros(t *testing.T) {
	data := workbookEntries(t, map[string][]byte{
		"[Content_Types].xml": []byte(`<Types/>`),
		"xl/vbaProject.bin":   []byte("macro"),
	})
	result := validateWorkbook(data, "macro.xlsx", nil, nil)
	if !hasFinding(result.Findings, "macro_enabled", "") {
		t.Fatalf("expected macro finding, got %#v", result.Findings)
	}
}

func hasFinding(findings []ValidationFinding, code, column string) bool {
	for _, finding := range findings {
		if finding.Code == code && finding.Column == column {
			return true
		}
	}
	return false
}

func workbook(t *testing.T, sheets map[string][][]any) []byte {
	t.Helper()
	file := excelize.NewFile()
	defer func() { _ = file.Close() }()
	first := true
	order := []string{"Decks", "Areas", "Rooms"}
	used := map[string]bool{}
	for _, sheet := range order {
		rows, ok := sheets[sheet]
		if !ok {
			continue
		}
		used[sheet] = true
		if first {
			if err := file.SetSheetName("Sheet1", sheet); err != nil {
				t.Fatalf("rename sheet: %v", err)
			}
			first = false
		} else if _, err := file.NewSheet(sheet); err != nil {
			t.Fatalf("new sheet: %v", err)
		}
		writeRows(t, file, sheet, rows)
	}
	for sheet, rows := range sheets {
		if used[sheet] {
			continue
		}
		if first {
			if err := file.SetSheetName("Sheet1", sheet); err != nil {
				t.Fatalf("rename sheet: %v", err)
			}
			first = false
		} else if _, err := file.NewSheet(sheet); err != nil {
			t.Fatalf("new sheet: %v", err)
		}
		writeRows(t, file, sheet, rows)
	}
	buffer, err := file.WriteToBuffer()
	if err != nil {
		t.Fatalf("write workbook: %v", err)
	}
	return buffer.Bytes()
}

func writeRows(t *testing.T, file *excelize.File, sheet string, rows [][]any) {
	t.Helper()
	for index, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, index+1)
		if err != nil {
			t.Fatalf("cell name: %v", err)
		}
		if err := file.SetSheetRow(sheet, cell, &row); err != nil {
			t.Fatalf("set row: %v", err)
		}
	}
}

func workbookEntries(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, data := range entries {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := file.Write(data); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buffer.Bytes()
}

func TestWorkbookValidationRejectsUnsupportedFiles(t *testing.T) {
	if result := validateWorkbook([]byte("not a zip"), "structure.xls", nil, nil); !hasFinding(result.Findings, "unsupported_format", "") {
		t.Fatalf("expected unsupported extension finding, got %#v", result.Findings)
	}
	if result := validateWorkbook([]byte{0xD0, 0xCF, 0x11, 0xE0}, "structure.xlsx", nil, nil); !hasFinding(result.Findings, "unsupported_format", "") {
		t.Fatalf("expected OLE rejection, got %#v", result.Findings)
	}
}

func TestWorkbookValidationReportsMissingSheetsAndColumns(t *testing.T) {
	missingSheets := workbook(t, map[string][][]any{
		"Decks": {{"level", "name"}, {1, "Main"}},
	})
	result := validateWorkbook(missingSheets, "structure.xlsx", nil, nil)
	for _, sheet := range []string{"Areas", "Rooms"} {
		found := false
		for _, finding := range result.Findings {
			if finding.Sheet == sheet && finding.Code == "missing_sheet" {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected missing sheet finding for %s, got %#v", sheet, result.Findings)
		}
	}

	missingColumns := workbook(t, map[string][][]any{
		"Decks": {{"level"}, {1}},
		"Areas": {{"deck_level", "name", "area_type", "capacity", "layout"}, {1, "Lobby", "lobby", "", ""}},
		"Rooms": {{"deck_level", "number", "room_type", "capacity", "is_bookable", "layout"}, {1, "101", "inside", 2, "true", ""}},
	})
	result = validateWorkbook(missingColumns, "structure.xlsx", nil, nil)
	if !hasFinding(result.Findings, "missing_column", "name") {
		t.Fatalf("expected missing deck name column, got %#v", result.Findings)
	}
	if result.Counts.Decks.Total != 1 || result.Counts.Decks.Valid != 0 {
		t.Fatalf("unexpected deck counts: %#v", result.Counts.Decks)
	}
}

func TestWorkbookValidationRequiresImportableRecords(t *testing.T) {
	data := workbook(t, map[string][][]any{
		"Decks": {{"level", "name"}},
		"Areas": {{"deck_level", "name", "area_type", "capacity", "layout"}},
		"Rooms": {{"deck_level", "number", "room_type", "capacity", "is_bookable", "layout"}},
	})
	result := validateWorkbook(data, "structure.xlsx", nil, nil)
	if !hasFinding(result.Findings, "no_importable_records", "") {
		t.Fatalf("expected no-records finding, got %#v", result.Findings)
	}
}

func TestWorkbookValidationAcceptsBlankRowsAndDefaultsBookableTrue(t *testing.T) {
	data := workbook(t, map[string][][]any{
		"Decks": {{"level", "name"}, {}, {0, "Main"}},
		"Areas": {{"deck_level", "name", "area_type", "capacity", "layout"}, {}, {0, "Lobby", "lobby", "", ""}},
		"Rooms": {{"deck_level", "number", "room_type", "capacity", "is_bookable", "layout"}, {}, {0, "001", "inside", 2, "", ""}},
	})
	result := validateWorkbook(data, "structure.xlsx", nil, nil)
	if len(result.Findings) != 0 {
		t.Fatalf("unexpected findings: %#v", result.Findings)
	}
	if len(result.Payload.Rooms) != 1 || !result.Payload.Rooms[0].IsBookable {
		t.Fatalf("room default is_bookable not applied: %#v", result.Payload.Rooms)
	}
}
