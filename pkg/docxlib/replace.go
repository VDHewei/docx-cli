package docxlib

import (
	"runtime"
	"strings"
	"sync"

	"github.com/gomutex/godocx/docx"
	"github.com/gomutex/godocx/packager"
	"github.com/gomutex/godocx/wml/ctypes"
)

// ReplaceOptions controls replacement behavior.
type ReplaceOptions struct {
	SkipHeaders bool // If true, skip header replacement.
	SkipFooters bool // If true, skip footer replacement.
	Workers     int  // Number of concurrent workers; 0 or negative means runtime.NumCPU().
}

// replaceUnit represents a single unit of work for the worker pool.
type replaceUnit struct {
	kind    string // "body", "table", "header", "footer"
	para    *ctypes.Paragraph
	path    string // FileMap key for XML-based units
	rootDoc *docx.RootDoc
}

// ReplaceAll performs concurrent find-and-replace across the document.
// It supports both region-level concurrency (different paragraphs/cells/XML files
// are processed in parallel) and rule-level application (all rules are applied
// sequentially within each unit to avoid conflicts).
func ReplaceAll(rootDoc *docx.RootDoc, rules []ReplacementRule, opts ReplaceOptions) ReplaceResult {
	if opts.Workers <= 0 {
		opts.Workers = runtime.NumCPU()
	}

	result := ReplaceResult{}

	if rootDoc == nil || rootDoc.Document == nil || rootDoc.Document.Body == nil {
		return result
	}

	// Gather all replaceable units.
	var units []replaceUnit
	body := rootDoc.Document.Body

	// Body paragraphs and tables.
	for _, child := range body.Children {
		if child.Para != nil {
			units = append(units, replaceUnit{
				kind:    "body",
				para:    child.Para.GetCT(),
				rootDoc: rootDoc,
			})
		} else if child.Table != nil {
			tableCT := unsafeGetTableCT(child.Table)
			for _, rowContent := range tableCT.RowContents {
				if rowContent.Row == nil {
					continue
				}
				for _, cellContent := range rowContent.Row.Contents {
					if cellContent.Cell == nil {
						continue
					}
					for _, block := range cellContent.Cell.Contents {
						if block.Paragraph != nil {
							units = append(units, replaceUnit{
								kind:    "table",
								para:    block.Paragraph,
								rootDoc: rootDoc,
							})
						}
					}
				}
			}
		}
	}

	// Headers and footers via FileMap.
	if !opts.SkipHeaders || !opts.SkipFooters {
		rootDoc.FileMap.Range(func(key, value any) bool {
			path := key.(string)

			isHeader := strings.Contains(path, "word/header") && strings.HasSuffix(path, ".xml")
			isFooter := strings.Contains(path, "word/footer") && strings.HasSuffix(path, ".xml")

			if (isHeader && !opts.SkipHeaders) || (isFooter && !opts.SkipFooters) {
				kind := "header"
				if isFooter {
					kind = "footer"
				}
				units = append(units, replaceUnit{
					kind:    kind,
					path:    path,
					rootDoc: rootDoc,
				})
			}
			return true
		})
	}

	// Process units with a worker pool.
	jobs := make(chan replaceUnit, len(units))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for w := 0; w < opts.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for unit := range jobs {
				count := unit.apply(rules)
				mu.Lock()
				result.TotalReplacements += count
				switch unit.kind {
				case "body":
					if count > 0 {
						result.ParagraphsProcessed++
					}
				case "table":
					if count > 0 {
						result.CellsProcessed++
					}
				case "header":
					if count > 0 {
						result.HeadersProcessed++
					}
				case "footer":
					if count > 0 {
						result.FootersProcessed++
					}
				}
				mu.Unlock()
			}
		}()
	}

	for _, unit := range units {
		jobs <- unit
	}
	close(jobs)
	wg.Wait()

	return result
}

// ReplaceAllByBytes performs concurrent find-and-replace across the document.
// It supports both region-level concurrency (different paragraphs/cells/XML files
// are processed in parallel) and rule-level application (all rules are applied
// sequentially within each unit to avoid conflicts).
func ReplaceAllByBytes(data []byte, rules []ReplacementRule, opts ReplaceOptions) (*docx.RootDoc, ReplaceResult, error) {
	rootDoc, err := packager.Unpack(&data)
	if err != nil {
		return nil, ReplaceResult{}, err
	}
	return rootDoc, ReplaceAll(rootDoc, rules, opts), nil
}

// apply runs the replacement rules on this unit.
func (u *replaceUnit) apply(rules []ReplacementRule) int {
	if u.kind == "header" || u.kind == "footer" {
		return u.applyXML(rules)
	}
	return u.applyParagraph(rules)
}

// applyParagraph replaces text inside a ctypes.Paragraph.
func (u *replaceUnit) applyParagraph(rules []ReplacementRule) int {
	if u.para == nil {
		return 0
	}

	// Extract the full concatenated text of the paragraph.
	var text strings.Builder
	for _, child := range u.para.Children {
		if child.Run == nil {
			continue
		}
		for _, runChild := range child.Run.Children {
			if runChild.Text != nil {
				text.WriteString(runChild.Text.Text)
			}
		}
	}

	original := text.String()
	if original == "" {
		return 0
	}

	modified := original
	count := 0
	for _, rule := range rules {
		if strings.Contains(modified, rule.Old) {
			n := strings.Count(modified, rule.Old)
			modified = strings.ReplaceAll(modified, rule.Old, rule.New)
			count += n
		}
	}

	if modified != original {
		// Preserve the first run's Property (font, bold, italic, size, color, etc.)
		// and replace its text content, removing all other runs to avoid duplication.
		var preservedProp *ctypes.RunProperty
		for _, child := range u.para.Children {
			if child.Run != nil {
				preservedProp = child.Run.Property
				break
			}
		}

		newRun := &ctypes.Run{
			Property: preservedProp,
			Children: []ctypes.RunChild{
				{Text: ctypes.TextFromString(modified)},
			},
		}
		u.para.Children = []ctypes.ParagraphChild{
			{Run: newRun},
		}
	}

	return count
}

// applyXML replaces text inside a raw XML file stored in RootDoc.FileMap.
func (u *replaceUnit) applyXML(rules []ReplacementRule) int {
	if u.path == "" || u.rootDoc == nil {
		return 0
	}

	val, ok := u.rootDoc.FileMap.Load(u.path)
	if !ok {
		return 0
	}

	content := string(val.([]byte))
	original := content
	count := 0
	for _, rule := range rules {
		if strings.Contains(content, rule.Old) {
			n := strings.Count(content, rule.Old)
			content = strings.ReplaceAll(content, rule.Old, rule.New)
			count += n
		}
	}

	if content != original {
		u.rootDoc.FileMap.Store(u.path, []byte(content))
	}

	return count
}
