// Package xlsxlib provides find-and-replace utilities for XLSX spreadsheets.
// It preserves original cell styles (font, alignment, borders, fill, number format, etc.)
// when performing text replacements.
package xlsxlib

// ReplacementRule defines a single find-and-replace rule.
type ReplacementRule struct {
	Old string `json:"old"`
	New string `json:"new"`
}

// CellLocation describes where a piece of text was found in the spreadsheet.
type CellLocation struct {
	Sheet string // sheet name
	Cell  string // cell reference (e.g. "A1")
}

// CellText represents a piece of text found in a cell, along with its location.
type CellText struct {
	Text     string
	Location CellLocation
}

// ReplaceResult holds the outcome of a replacement operation.
type ReplaceResult struct {
	TotalReplacements int
	CellsProcessed    int
	SheetsProcessed   int
}
