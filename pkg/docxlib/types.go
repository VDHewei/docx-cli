// Package docxlib provides find-and-replace utilities for DOCX documents.
// It works around godocx API limitations using reflection to access table,
// header and footer contents.
package docxlib

// ReplacementRule defines a single find-and-replace rule.
type ReplacementRule struct {
	Old string `json:"old"`
	New string `json:"new"`
}

// TextLocation describes where a piece of text was found in the document.
type TextLocation struct {
	Kind      string // "body", "table", "header", "footer"
	ParaIndex int    // index within the container (body / text sequence in XML)
	TableIdx  int    // -1 if not in a table
	RowIdx    int    // -1 if not in a table
	CellIdx   int    // -1 if not in a table
	RunIdx    int    // index of run within paragraph
	TextIdx   int    // index of text element within run
}

// DocText represents a piece of text found in the document, along with its location.
type DocText struct {
	Text     string
	Location TextLocation
}

// ReplaceResult holds the outcome of a replacement operation.
type ReplaceResult struct {
	TotalReplacements   int
	ParagraphsProcessed int
	CellsProcessed      int
	HeadersProcessed    int
	FootersProcessed    int
}
