package xlsxlib

import (
	"strings"

	"github.com/xuri/excelize/v2"
)

// ExtractAllText extracts all text from the spreadsheet along with cell locations.
// It scans all sheets and all cells that contain string values or formula results.
func ExtractAllText(f *excelize.File) []CellText {
	var result []CellText

	if f == nil {
		return result
	}

	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}

		for rowIdx, row := range rows {
			for colIdx, cellValue := range row {
				if cellValue == "" {
					continue
				}
				cellName, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				if err != nil {
					continue
				}
				result = append(result, CellText{
					Text: cellValue,
					Location: CellLocation{
						Sheet: sheet,
						Cell:  cellName,
					},
				})
			}
		}
	}

	return result
}

// FindText extracts all cells whose text contains the given substring.
func FindText(f *excelize.File, substring string) []CellText {
	var result []CellText
	for _, ct := range ExtractAllText(f) {
		if strings.Contains(ct.Text, substring) {
			result = append(result, ct)
		}
	}
	return result
}
