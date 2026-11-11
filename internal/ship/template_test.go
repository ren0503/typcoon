package ship

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestTemplateSheetsAndHeaders(t *testing.T) {
	data, err := buildTemplate()
	if err != nil {
		t.Fatalf("build template: %v", err)
	}
	file, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("open template: %v", err)
	}
	defer func() { _ = file.Close() }()

	sheets := map[string]bool{}
	for _, name := range file.GetSheetList() {
		sheets[name] = true
	}
	for _, name := range []string{"Decks", "Areas", "Rooms", "Reference"} {
		if !sheets[name] {
			t.Fatalf("template is missing sheet %s", name)
		}
	}
	for sheet, want := range map[string][]string{
		"Decks": deckColumns,
		"Areas": areaColumns,
		"Rooms": roomColumns,
	} {
		rows, err := file.GetRows(sheet)
		if err != nil || len(rows) == 0 {
			t.Fatalf("read %s: %v", sheet, err)
		}
		if strings.Join(rows[0], ",") != strings.Join(want, ",") {
			t.Fatalf("%s headers = %#v, want %#v", sheet, rows[0], want)
		}
	}
}

func TestTemplateReferenceDocumentsValidationRules(t *testing.T) {
	data, err := buildTemplate()
	if err != nil {
		t.Fatalf("build template: %v", err)
	}
	file, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("open template: %v", err)
	}
	defer func() { _ = file.Close() }()

	rows, err := file.GetRows("Reference")
	if err != nil {
		t.Fatalf("read reference: %v", err)
	}
	var reference strings.Builder
	for _, row := range rows {
		reference.WriteString(strings.Join(row, " "))
		reference.WriteString("\n")
	}
	text := reference.String()
	for areaType := range areaTypes {
		if !strings.Contains(text, areaType) {
			t.Fatalf("reference sheet does not document area type %q", areaType)
		}
	}
	for roomType := range roomTypes {
		if !strings.Contains(text, roomType) {
			t.Fatalf("reference sheet does not document room type %q", roomType)
		}
	}
	for _, rule := range []string{"deck_level", "is_bookable", "10 MiB", "5,000", "true/false", "yes/no", "1/0"} {
		if !strings.Contains(text, rule) {
			t.Fatalf("reference sheet does not document %q", rule)
		}
	}
}
