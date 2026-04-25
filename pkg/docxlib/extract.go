package docxlib

import (
	"bytes"
	"encoding/xml"
	"strings"
	"sync"

	"github.com/gomutex/godocx/docx"
	"github.com/gomutex/godocx/wml/ctypes"
)

// ExtractAllText extracts all text from the document along with its location.
// It scans body paragraphs, table cells, headers and footers.
func ExtractAllText(rootDoc *docx.RootDoc) []DocText {
	var result []DocText
	var mu sync.Mutex

	if rootDoc == nil || rootDoc.Document == nil || rootDoc.Document.Body == nil {
		return result
	}

	body := rootDoc.Document.Body

	// Process body paragraphs and tables
	for i, child := range body.Children {
		if child.Para != nil {
			texts := extractParagraphTexts(child.Para.GetCT(), TextLocation{
				Kind:      "body",
				ParaIndex: i,
				TableIdx:  -1,
				RowIdx:    -1,
				CellIdx:   -1,
			})
			mu.Lock()
			result = append(result, texts...)
			mu.Unlock()
		} else if child.Table != nil {
			tableCT := unsafeGetTableCT(child.Table)
			texts := extractTableTexts(tableCT, i)
			mu.Lock()
			result = append(result, texts...)
			mu.Unlock()
		}
	}

	// Process headers and footers via FileMap XML
	rootDoc.FileMap.Range(func(key, value any) bool {
		path := key.(string)
		content := value.([]byte)

		isHeader := strings.Contains(path, "word/header") && strings.HasSuffix(path, ".xml")
		isFooter := strings.Contains(path, "word/footer") && strings.HasSuffix(path, ".xml")

		if isHeader || isFooter {
			kind := "header"
			if isFooter {
				kind = "footer"
			}
			texts := extractXMLTexts(content, kind, path)
			mu.Lock()
			result = append(result, texts...)
			mu.Unlock()
		}
		return true
	})

	return result
}

func extractParagraphTexts(para *ctypes.Paragraph, loc TextLocation) []DocText {
	var result []DocText
	for runIdx, child := range para.Children {
		if child.Run == nil {
			continue
		}
		for textIdx, runChild := range child.Run.Children {
			if runChild.Text != nil && runChild.Text.Text != "" {
				locCopy := loc
				locCopy.RunIdx = runIdx
				locCopy.TextIdx = textIdx
				result = append(result, DocText{
					Text:     runChild.Text.Text,
					Location: locCopy,
				})
			}
		}
	}
	return result
}

func extractTableTexts(tableCT *ctypes.Table, tableIdx int) []DocText {
	var result []DocText
	for rowIdx, rowContent := range tableCT.RowContents {
		if rowContent.Row == nil {
			continue
		}
		for cellIdx, cellContent := range rowContent.Row.Contents {
			if cellContent.Cell == nil {
				continue
			}
			for _, block := range cellContent.Cell.Contents {
				if block.Paragraph != nil {
					loc := TextLocation{
						Kind:     "table",
						TableIdx: tableIdx,
						RowIdx:   rowIdx,
						CellIdx:  cellIdx,
					}
					result = append(result, extractParagraphTexts(block.Paragraph, loc)...)
				}
			}
		}
	}
	return result
}

func extractXMLTexts(content []byte, kind, path string) []DocText {
	var result []DocText
	decoder := xml.NewDecoder(bytes.NewReader(content))
	var currentText strings.Builder
	inText := false
	textSeq := 0

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch elem := token.(type) {
		case xml.StartElement:
			if elem.Name.Local == "t" {
				inText = true
				currentText.Reset()
			}
		case xml.CharData:
			if inText {
				currentText.Write(elem)
			}
		case xml.EndElement:
			if elem.Name.Local == "t" && inText {
				inText = false
				result = append(result, DocText{
					Text: currentText.String(),
					Location: TextLocation{
						Kind:      kind,
						ParaIndex: textSeq,
						TableIdx:  -1,
						RowIdx:    -1,
						CellIdx:   -1,
					},
				})
				textSeq++
			}
		}
	}

	return result
}
