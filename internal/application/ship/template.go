package ship

import (
	"context"

	"github.com/xuri/excelize/v2"
)

func buildTemplate() ([]byte, error) {
	file := excelize.NewFile()
	defer func() { _ = file.Close() }()

	if err := file.SetSheetName("Sheet1", "Decks"); err != nil {
		return nil, err
	}
	for _, sheet := range []string{"Areas", "Rooms", "Reference"} {
		if _, err := file.NewSheet(sheet); err != nil {
			return nil, err
		}
	}

	rows := map[string][][]any{
		"Decks": {
			{"level", "name"},
			{1, "Main Deck"},
			{2, "Upper Deck"},
		},
		"Areas": {
			{"deck_level", "name", "area_type", "capacity", "layout"},
			{1, "Main Lobby", "lobby", 40, `{"x":0,"y":0,"w":10,"h":8}`},
			{1, "Sunset Restaurant", "restaurant", 120, ""},
		},
		"Rooms": {
			{"deck_level", "number", "room_type", "capacity", "is_bookable", "layout"},
			{2, "201", "balcony", 2, "true", ""},
			{2, "202", "suite", 4, "TRUE", `{"x":12,"y":0,"w":6,"h":5}`},
		},
		"Reference": {
			{"Sheet", "Column", "Rule"},
			{"Decks", "level", "Whole number 0-2147483647; unique within the workbook and among non-deleted decks on the ship."},
			{"Decks", "name", "Required text; trimmed and non-blank."},
			{"Areas", "deck_level", "The level of a row in the Decks sheet, not a database id."},
			{"Areas", "name", "Required text; trimmed and non-blank."},
			{"Areas", "area_type", "One of: restaurant, lobby, open_deck, theatre, shop, corridor, stairwell, spa, pool, other."},
			{"Areas", "capacity", "Optional whole number greater than or equal to zero."},
			{"Areas", "layout", "Optional JSON object or value."},
			{"Rooms", "deck_level", "The level of a row in the Decks sheet, not a database id."},
			{"Rooms", "number", "Required text; unique within the workbook and among non-deleted rooms on the ship."},
			{"Rooms", "room_type", "One of: inside, oceanview, balcony, suite, crew, other."},
			{"Rooms", "capacity", "Required whole number from 1 to 32767."},
			{"Rooms", "is_bookable", "Optional boolean: true/false, yes/no, or 1/0. Blank defaults to true."},
			{"Rooms", "layout", "Optional JSON object or value."},
			{"Limits", "file", "Only .xlsx workbooks up to 10 MiB and 5,000 non-blank data rows per sheet are accepted."},
			{"Limits", "sheets", "Decks, Areas, and Rooms are required. Extra sheets, including Reference, are ignored."},
		},
	}
	for sheet, values := range rows {
		for index, row := range values {
			cell, err := excelize.CoordinatesToCellName(1, index+1)
			if err != nil {
				return nil, err
			}
			if err := file.SetSheetRow(sheet, cell, &row); err != nil {
				return nil, err
			}
		}
	}
	for sheet, widths := range map[string][]float64{
		"Decks":     {12, 28},
		"Areas":     {14, 28, 18, 12, 32},
		"Rooms":     {14, 14, 16, 12, 14, 32},
		"Reference": {14, 18, 100},
	} {
		for index, width := range widths {
			column, err := excelize.ColumnNumberToName(index + 1)
			if err != nil {
				return nil, err
			}
			_ = file.SetColWidth(sheet, column, column, width)
		}
	}
	buffer, err := file.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (s *service) StructureImportTemplate(_ context.Context) ([]byte, error) {
	return buildTemplate()
}
