package docxlib

import (
	"bytes"
	"encoding/xml"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/gomutex/godocx/docx"
	"github.com/gomutex/godocx/packager"
	"github.com/gomutex/godocx/wml/ctypes"
)

var wordTextElementRE = regexp.MustCompile(`(?s)(<((?:(?:w|a):)?t)\b[^>]*>)(.*?)(</(?:(?:w|a):)?t>)`)

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

	var textRefs []*ctypes.Text
	var chunks []string
	for childIdx := range u.para.Children {
		run := u.para.Children[childIdx].Run
		if run == nil {
			continue
		}
		for runChildIdx := range run.Children {
			runChild := &run.Children[runChildIdx]
			if runChild.Text != nil {
				textRefs = append(textRefs, runChild.Text)
				chunks = append(chunks, runChild.Text.Text)
			}
		}
	}

	original := strings.Join(chunks, "")
	if original == "" {
		return 0
	}

	replacedChunks, count, changed := replaceTextChunks(chunks, rules)
	if changed {
		for i, textRef := range textRefs {
			textRef.Text = replacedChunks[i]
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

	content := val.([]byte)
	modified, count := replaceXMLTextContent(content, rules)
	if count > 0 {
		u.rootDoc.FileMap.Store(u.path, modified)
	}

	return count
}

func applyReplacementRules(text string, rules []ReplacementRule) (string, int) {
	modified := text
	count := 0
	for _, rule := range rules {
		if rule.Old == "" {
			continue
		}
		if strings.Contains(modified, rule.Old) {
			n := strings.Count(modified, rule.Old)
			modified = strings.ReplaceAll(modified, rule.Old, rule.New)
			count += n
		}
	}
	return modified, count
}

func replaceTextChunks(chunks []string, rules []ReplacementRule) ([]string, int, bool) {
	original := strings.Join(chunks, "")
	modified, count := applyReplacementRules(original, rules)
	if modified == original || count == 0 {
		return chunks, count, false
	}

	return distributeText(chunks, modified), count, true
}

func distributeText(originalChunks []string, text string) []string {
	result := make([]string, len(originalChunks))
	remaining := []rune(text)
	for i, chunk := range originalChunks {
		if i == len(originalChunks)-1 {
			result[i] = string(remaining)
			break
		}
		width := len([]rune(chunk))
		if width >= len(remaining) {
			result[i] = string(remaining)
			remaining = nil
			continue
		}
		result[i] = string(remaining[:width])
		remaining = remaining[width:]
	}
	return result
}

func replaceXMLTextContent(content []byte, rules []ReplacementRule) ([]byte, int) {
	total := 0
	modified := wordTextElementRE.ReplaceAllFunc(content, func(match []byte) []byte {
		parts := wordTextElementRE.FindSubmatch(match)
		if len(parts) != 5 {
			return match
		}

		text, err := xmlUnescapeString(string(parts[3]))
		if err != nil {
			return match
		}

		replaced, count := applyReplacementRules(text, rules)
		if count == 0 {
			return match
		}
		total += count

		var escaped bytes.Buffer
		if err := xml.EscapeText(&escaped, []byte(replaced)); err != nil {
			return match
		}

		out := make([]byte, 0, len(parts[1])+escaped.Len()+len(parts[4]))
		out = append(out, parts[1]...)
		out = append(out, escaped.Bytes()...)
		out = append(out, parts[4]...)
		return out
	})

	return modified, total
}

func xmlUnescapeString(s string) (string, error) {
	var value string
	if err := xml.Unmarshal([]byte("<x>"+s+"</x>"), &value); err != nil {
		return "", err
	}
	return value, nil
}
