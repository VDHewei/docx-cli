package xlsxlib

import (
	"bytes"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/xuri/excelize/v2"
)

// ReplaceOptions controls replacement behavior.
type ReplaceOptions struct {
	Workers    int      // Number of concurrent workers; 0 or negative means runtime.NumCPU().
	SkipSheets []string // Sheet name patterns to skip during replacement. Supports four pattern types:
	//   - exact match (case-insensitive): "Sheet1" matches sheets named "Sheet1", "SHEET1", etc.
	//   - negative substring match (prefix "!"): "!Summary" matches sheets whose name does NOT contain "Summary".
	//   - suffix match (prefix "*."): "*.Data" matches sheets whose name ends with ".Data".
	//   - regex match (prefix "@regexp:"): "@regexp:^Sheet\\d+$" matches sheets via a regular expression.
}

// XlsxOptions is a type alias for excelize.Options, used for passing options when opening XLSX files.
type XlsxOptions = excelize.Options

// MatchHandler is a function type that returns true if the given sheet name matches a skip pattern.
type MatchHandler func(string) bool

// OpenReader is a package-level variable that holds the function used to open an XLSX from an io.Reader.
// It defaults to excelize.OpenReader and can be replaced in tests for mocking.
var OpenReader = excelize.OpenReader

// CheckSkip returns true if the given sheet name matches any of the SkipSheets patterns.
// A matched sheet will be skipped during replacement.
func (o ReplaceOptions) CheckSkip(sheet string) bool {
	var filters []MatchHandler
	for _, vs := range o.SkipSheets {
		filters = append(filters, resolveMatcher(vs))
	}
	for _, filter := range filters {
		if filter(sheet) {
			return true
		}
	}
	return false
}

// resolveMatcher converts a pattern string into a MatchHandler function.
// The pattern syntax supports four modes:
//   - "!" prefix: negative substring match — matches sheets whose name does NOT contain the rest.
//   - "*." prefix: suffix match — matches sheets whose name ends with the rest.
//   - "@regexp:" prefix: regular expression match — matches sheets via the given regex.
//   - default: case-insensitive exact match.
func resolveMatcher(pattern string) MatchHandler {
	if pattern == "" {
		return func(v string) bool {
			return v == ""
		}
	}
	switch {
	case strings.HasPrefix(pattern, "!"):
		return func(s string) bool {
			return !strings.Contains(s, pattern[1:])
		}
	case strings.HasPrefix(pattern, "*."):
		return func(s string) bool {
			return strings.HasSuffix(s, pattern[2:])
		}
	case strings.HasPrefix(pattern, "@regexp:"):
		var reg = regexp.MustCompile(pattern[8:])
		return func(s string) bool {
			return reg.MatchString(s)
		}
	default:
		return func(s string) bool {
			return strings.EqualFold(s, pattern)
		}
	}
}

// ReplaceAll performs concurrent find-and-replace across all sheets in the spreadsheet.
// It preserves the original cell style (font, alignment, borders, fill, number format, width, height).
func ReplaceAll(f *excelize.File, rules []ReplacementRule, opts ReplaceOptions) ReplaceResult {
	if opts.Workers <= 0 {
		opts.Workers = runtime.NumCPU()
	}

	result := ReplaceResult{}

	if f == nil {
		return result
	}

	// Collect all cells that need to be checked.
	type cellWork struct {
		sheet string
		cell  string
	}

	var jobs []cellWork
	for _, sheet := range f.GetSheetList() {
		if opts.CheckSkip(sheet) {
			continue
		}
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}
		for rowIdx, row := range rows {
			for colIdx, cellValue := range row {
				if cellValue == "" {
					continue
				}
				// Check if any rule matches this cell
				matched := false
				for _, rule := range rules {
					if rule.Old == "" {
						continue
					}
					if strings.Contains(cellValue, rule.Old) {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
				cellName, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				if err != nil {
					continue
				}
				jobs = append(jobs, cellWork{sheet: sheet, cell: cellName})
			}
		}
	}

	// Process with worker pool
	jobsCh := make(chan cellWork, len(jobs))
	var mu sync.Mutex
	var wg sync.WaitGroup

	processedSheets := make(map[string]bool)

	for w := 0; w < opts.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobsCh {
				count := replaceCell(f, job.sheet, job.cell, rules)
				if count > 0 {
					mu.Lock()
					result.TotalReplacements += count
					result.CellsProcessed++
					processedSheets[job.sheet] = true
					mu.Unlock()
				}
			}
		}()
	}

	for _, job := range jobs {
		jobsCh <- job
	}
	close(jobsCh)
	wg.Wait()

	result.SheetsProcessed = len(processedSheets)
	return result
}

// replaceCell replaces text in a single cell, preserving its style.
func replaceCell(f *excelize.File, sheet, cell string, rules []ReplacementRule) int {
	formula, err := f.GetCellFormula(sheet, cell)
	if err == nil && formula != "" {
		return 0
	}

	cellValue, err := f.GetCellValue(sheet, cell)
	if err != nil || cellValue == "" {
		return 0
	}

	original := cellValue
	modified := original
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

	if modified == original || count == 0 {
		return 0
	}

	// Preserve the original cell style
	styleID, err := f.GetCellStyle(sheet, cell)
	if err != nil {
		styleID = 0
	}

	// Set the new value
	if err := f.SetCellValue(sheet, cell, modified); err != nil {
		return 0
	}

	// Re-apply the preserved style
	if styleID > 0 {
		_ = f.SetCellStyle(sheet, cell, cell, styleID)
	}

	return count
}

// ReplaceAllByBytes performs concurrent find-and-replace across all sheets in the spreadsheet.
// It preserves the original cell style (font, alignment, borders, fill, number format, width, height).
func ReplaceAllByBytes(data []byte, rules []ReplacementRule, opts ReplaceOptions, xlsxOpts ...XlsxOptions) (*excelize.File,
	ReplaceResult, error) {
	xlsx, err := excelize.OpenReader(bytes.NewReader(data), xlsxOpts...)
	if err != nil {
		return nil, ReplaceResult{}, err
	}
	return xlsx, ReplaceAll(xlsx, rules, opts), nil
}
