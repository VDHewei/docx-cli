package xlsxlib

import (
	"bytes"
	"runtime"
	"strings"
	"sync"

	"github.com/xuri/excelize/v2"
)

// ReplaceOptions controls replacement behavior.
type ReplaceOptions struct {
	Workers    int      // Number of concurrent workers; 0 or negative means runtime.NumCPU().
	SkipSheets []string // skip scan text worksheet
}

type XlsxOptions = excelize.Options

var OpenReader = excelize.OpenReader

func (o ReplaceOptions) CheckSkip(sheet string) bool {
	for _, vs := range o.SkipSheets {
		if vs == sheet {
			return true
		}
	}
	return false
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
	cellValue, err := f.GetCellValue(sheet, cell)
	if err != nil || cellValue == "" {
		return 0
	}

	original := cellValue
	modified := original
	count := 0

	for _, rule := range rules {
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
