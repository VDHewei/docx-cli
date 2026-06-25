package xlsxlib

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestReplaceAllByBytes_RealXLSX_NoMatchKeepsContentAndStructure(t *testing.T) {
	for _, fixtureName := range xlsxFixtureNames(t) {
		t.Run(fixtureName, func(t *testing.T) {
			data := readXlsxFixtureBytes(t, fixtureName)
			before := openXlsxFixtureBytes(t, data)
			defer before.Close()
			beforeTexts := xlsxTextList(before)
			beforeFormulas := xlsxFormulaSnapshot(t, before)

			xlsx, result, err := ReplaceAllByBytes(data, []ReplacementRule{{Old: "__XLSX_CLI_NO_SUCH_TEXT__", New: "unused"}}, ReplaceOptions{Workers: 2})
			if err != nil {
				t.Fatalf("ReplaceAllByBytes 失败: %v", err)
			}
			defer xlsx.Close()
			if result.TotalReplacements != 0 {
				t.Fatalf("无匹配内容时不应发生替换，实际 %d", result.TotalReplacements)
			}
			if afterTexts := xlsxTextList(xlsx); !reflect.DeepEqual(afterTexts, beforeTexts) {
				t.Fatalf("无匹配内容时单元格文本不应变化")
			}
			if afterFormulas := xlsxFormulaSnapshot(t, xlsx); !reflect.DeepEqual(afterFormulas, beforeFormulas) {
				t.Fatalf("无匹配内容时公式不应变化")
			}

			beforePath := saveXlsxForTest(t, before, "nomatch-before")
			afterPath := saveXlsxForTest(t, xlsx, "nomatch-after")
			if xlsxFileSize(t, afterPath) != xlsxFileSize(t, beforePath) {
				t.Fatalf("无匹配内容时保存后的工作簿大小应保持不变")
			}
		})
	}
}

func TestReplaceAll_RealXLSX_ReplacesTextOnlyAndPreservesWorkbookSemantics(t *testing.T) {
	for _, fixtureName := range xlsxFixtureNames(t) {
		t.Run(fixtureName, func(t *testing.T) {
			xlsx := openXlsxFixture(t, fixtureName)
			defer xlsx.Close()
			oldText := pickXlsxReplacementText(t, xlsx)
			newText := "XLSX_CLI_REAL_DATA_REPLACEMENT"
			beforeText := strings.Join(xlsxTextList(xlsx), "")
			beforeFormulas := xlsxFormulaSnapshot(t, xlsx)
			beforeStyles := xlsxStyleSnapshot(t, xlsx)
			beforeLayout := xlsxLayoutSnapshot(t, xlsx)

			result := ReplaceAll(xlsx, []ReplacementRule{{Old: oldText, New: newText}}, ReplaceOptions{Workers: 2})
			if result.TotalReplacements == 0 {
				t.Fatalf("期望真实工作簿至少发生 1 次替换")
			}
			afterText := strings.Join(xlsxTextList(xlsx), "")
			if strings.Count(afterText, oldText) >= strings.Count(beforeText, oldText) {
				t.Fatalf("替换后旧文本数量未减少: %q", oldText)
			}
			if !strings.Contains(afterText, newText) {
				t.Fatalf("替换后未找到新文本 %q", newText)
			}
			if afterFormulas := xlsxFormulaSnapshot(t, xlsx); !reflect.DeepEqual(afterFormulas, beforeFormulas) {
				t.Fatalf("替换字符串内容时公式不应变化")
			}
			if afterStyles := xlsxStyleSnapshot(t, xlsx); !reflect.DeepEqual(afterStyles, beforeStyles) {
				t.Fatalf("替换字符串内容时单元格样式不应变化")
			}
			if afterLayout := xlsxLayoutSnapshot(t, xlsx); !reflect.DeepEqual(afterLayout, beforeLayout) {
				t.Fatalf("替换字符串内容时工作表布局不应变化")
			}
		})
	}
}

func TestReplaceAll_RealXLSX_DoesNotReplaceFormulaResults(t *testing.T) {
	for _, fixtureName := range xlsxFixtureNames(t) {
		t.Run(fixtureName, func(t *testing.T) {
			xlsx := openXlsxFixture(t, fixtureName)
			defer xlsx.Close()
			beforeFormulas := xlsxFormulaSnapshot(t, xlsx)

			_ = ReplaceAll(xlsx, []ReplacementRule{{Old: "0", New: "9999"}}, ReplaceOptions{Workers: 2})
			if afterFormulas := xlsxFormulaSnapshot(t, xlsx); !reflect.DeepEqual(afterFormulas, beforeFormulas) {
				t.Fatalf("公式结果命中替换规则时，不应覆盖公式本身")
			}
			for cell, formula := range beforeFormulas {
				sheet, ref, _ := strings.Cut(cell, "!")
				afterFormula, err := xlsx.GetCellFormula(sheet, ref)
				if err != nil {
					t.Fatalf("读取公式失败 %s: %v", cell, err)
				}
				if afterFormula != formula {
					t.Fatalf("公式单元格 %s 被修改: got %q, want %q", cell, afterFormula, formula)
				}
			}
		})
	}
}

func TestReplaceAllByBytes_RealXLSX_SavedSizeTracksLongerAndShorterContent(t *testing.T) {
	for _, fixtureName := range xlsxFixtureNames(t) {
		t.Run(fixtureName, func(t *testing.T) {
			data := readXlsxFixtureBytes(t, fixtureName)
			fixture := openXlsxFixtureBytes(t, data)
			oldText := pickXlsxReplacementText(t, fixture)
			_ = fixture.Close()

			longXLSX, longResult, err := ReplaceAllByBytes(data, []ReplacementRule{{Old: oldText, New: longXlsxReplacement()}}, ReplaceOptions{Workers: 2})
			if err != nil {
				t.Fatalf("长文本 ReplaceAllByBytes 失败: %v", err)
			}
			defer longXLSX.Close()
			if longResult.TotalReplacements == 0 {
				t.Fatalf("长文本替换应至少发生 1 次")
			}
			longPath := saveXlsxForTest(t, longXLSX, "long")
			if got, wantMin := xlsxFileSize(t, longPath), int64(len(data)); got < wantMin {
				t.Fatalf("替换后总体字符内容更长时，新工作簿大小应不小于旧工作簿: got %d, want >= %d", got, wantMin)
			}

			shortXLSX, shortResult, err := ReplaceAllByBytes(data, []ReplacementRule{{Old: oldText, New: ""}}, ReplaceOptions{Workers: 2})
			if err != nil {
				t.Fatalf("短文本 ReplaceAllByBytes 失败: %v", err)
			}
			defer shortXLSX.Close()
			if shortResult.TotalReplacements == 0 {
				t.Fatalf("短文本替换应至少发生 1 次")
			}
			shortPath := saveXlsxForTest(t, shortXLSX, "short")
			if got, wantMax := xlsxFileSize(t, shortPath), int64(len(data)); got >= wantMax {
				t.Fatalf("替换后总体字符内容更短时，新工作簿大小应小于旧工作簿: got %d, want < %d", got, wantMax)
			}
		})
	}
}

func TestReplaceAllBytesPreservePackage_RealXLSX_NoMatchReturnsOriginalBytes(t *testing.T) {
	for _, fixtureName := range xlsxFixtureNames(t) {
		t.Run(fixtureName, func(t *testing.T) {
			data := readXlsxFixtureBytes(t, fixtureName)
			output, result, err := ReplaceAllBytesPreservePackage(data, []ReplacementRule{{Old: "__XLSX_CLI_NO_SUCH_TEXT__", New: "unused"}}, ReplaceOptions{Workers: 2})
			if err != nil {
				t.Fatalf("ReplaceAllBytesPreservePackage 失败: %v", err)
			}
			if result.TotalReplacements != 0 {
				t.Fatalf("无匹配内容时不应发生替换，实际 %d", result.TotalReplacements)
			}
			if !bytes.Equal(output, data) {
				t.Fatalf("无匹配内容时应返回原始 XLSX 字节")
			}
			_ = saveBytesForTest(t, output, "xlsx", "preserve-nomatch", ".xlsx")
		})
	}
}

func TestReplaceAllBytesPreservePackage_RealXLSX_ReplacesOnlyTextEntries(t *testing.T) {
	for _, fixtureName := range xlsxFixtureNames(t) {
		t.Run(fixtureName, func(t *testing.T) {
			data := readXlsxFixtureBytes(t, fixtureName)
			beforePath := xlsxFixturePath(fixtureName)
			before := openXlsxFixtureBytes(t, data)
			defer before.Close()
			oldText := pickXlsxReplacementText(t, before)
			newText := "XLSX_CLI_PRESERVE_PACKAGE_REPLACEMENT"
			beforeText := strings.Join(xlsxTextList(before), "")
			beforeFormulas := xlsxFormulaSnapshot(t, before)
			beforeStyles := xlsxStyleSnapshot(t, before)
			beforeLayout := xlsxLayoutSnapshot(t, before)

			output, result, err := ReplaceAllBytesPreservePackage(data, []ReplacementRule{{Old: oldText, New: newText}}, ReplaceOptions{Workers: 2})
			if err != nil {
				t.Fatalf("ReplaceAllBytesPreservePackage 失败: %v", err)
			}
			if result.TotalReplacements == 0 {
				t.Fatalf("期望保真替换至少发生 1 次")
			}
			afterPath := saveBytesForTest(t, output, "xlsx", "preserve-replace", ".xlsx")
			assertZipEntriesEqual(t, beforePath, afterPath, func(name string) bool {
				return !isXlsxReplaceableTextXML(name)
			})

			after := openXlsxFixtureBytes(t, output)
			defer after.Close()
			afterText := strings.Join(xlsxTextList(after), "")
			if strings.Count(afterText, oldText) >= strings.Count(beforeText, oldText) {
				t.Fatalf("替换后旧文本数量未减少: %q", oldText)
			}
			if !strings.Contains(afterText, newText) {
				t.Fatalf("替换后未找到新文本 %q", newText)
			}
			if afterFormulas := xlsxFormulaSnapshot(t, after); !reflect.DeepEqual(afterFormulas, beforeFormulas) {
				t.Fatalf("保真替换字符串内容时公式不应变化")
			}
			if afterStyles := xlsxStyleSnapshot(t, after); !reflect.DeepEqual(afterStyles, beforeStyles) {
				t.Fatalf("保真替换字符串内容时单元格样式不应变化")
			}
			if afterLayout := xlsxLayoutSnapshot(t, after); !reflect.DeepEqual(afterLayout, beforeLayout) {
				t.Fatalf("保真替换字符串内容时工作表布局不应变化")
			}
		})
	}
}

func TestReplaceAllBytesPreservePackage_RealXLSX_DoesNotReplaceFormulaResults(t *testing.T) {
	for _, fixtureName := range xlsxFixtureNames(t) {
		t.Run(fixtureName, func(t *testing.T) {
			data := readXlsxFixtureBytes(t, fixtureName)
			before := openXlsxFixtureBytes(t, data)
			defer before.Close()
			beforeFormulas := xlsxFormulaSnapshot(t, before)

			output, _, err := ReplaceAllBytesPreservePackage(data, []ReplacementRule{{Old: "0", New: "9999"}}, ReplaceOptions{Workers: 2})
			if err != nil {
				t.Fatalf("ReplaceAllBytesPreservePackage 失败: %v", err)
			}
			after := openXlsxFixtureBytes(t, output)
			defer after.Close()
			if afterFormulas := xlsxFormulaSnapshot(t, after); !reflect.DeepEqual(afterFormulas, beforeFormulas) {
				t.Fatalf("公式结果命中替换规则时，不应覆盖公式本身")
			}
		})
	}
}

func TestReplaceAllBytesPreservePackage_RealXLSX_SavedSizeTracksLongerAndShorterContent(t *testing.T) {
	for _, fixtureName := range xlsxFixtureNames(t) {
		t.Run(fixtureName, func(t *testing.T) {
			data := readXlsxFixtureBytes(t, fixtureName)
			fixture := openXlsxFixtureBytes(t, data)
			oldText := pickXlsxReplacementText(t, fixture)
			_ = fixture.Close()

			longOutput, longResult, err := ReplaceAllBytesPreservePackage(data, []ReplacementRule{{Old: oldText, New: longXlsxReplacement()}}, ReplaceOptions{Workers: 2})
			if err != nil {
				t.Fatalf("长文本 ReplaceAllBytesPreservePackage 失败: %v", err)
			}
			if longResult.TotalReplacements == 0 {
				t.Fatalf("长文本替换应至少发生 1 次")
			}
			_ = saveBytesForTest(t, longOutput, "xlsx", "preserve-long", ".xlsx")
			if len(longOutput) < len(data) {
				t.Fatalf("替换后总体字符内容更长时，新工作簿大小应不小于旧工作簿: got %d, want >= %d", len(longOutput), len(data))
			}

			shortOutput, shortResult, err := ReplaceAllBytesPreservePackage(data, []ReplacementRule{{Old: oldText, New: ""}}, ReplaceOptions{Workers: 2})
			if err != nil {
				t.Fatalf("短文本 ReplaceAllBytesPreservePackage 失败: %v", err)
			}
			if shortResult.TotalReplacements == 0 {
				t.Fatalf("短文本替换应至少发生 1 次")
			}
			_ = saveBytesForTest(t, shortOutput, "xlsx", "preserve-short", ".xlsx")
			if len(shortOutput) >= len(data) {
				t.Fatalf("替换后总体字符内容更短时，新工作簿大小应小于旧工作簿: got %d, want < %d", len(shortOutput), len(data))
			}
		})
	}
}

func xlsxFixtureNames(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join("..", "..", "test", "data", "*.xlsx"))
	if err != nil {
		t.Fatalf("查找真实 XLSX fixture 失败: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("test/data 下未找到 XLSX fixture")
	}
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, filepath.Base(match))
	}
	sort.Strings(names)
	return names
}

func xlsxFixturePath(name string) string {
	return filepath.Join("..", "..", "test", "data", name)
}

func readXlsxFixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	path := xlsxFixturePath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取真实 XLSX fixture 失败: %v", err)
	}
	return data
}

func openXlsxFixture(t *testing.T, name string) *excelize.File {
	t.Helper()
	return openXlsxFixtureBytes(t, readXlsxFixtureBytes(t, name))
}

func openXlsxFixtureBytes(t *testing.T, data []byte) *excelize.File {
	t.Helper()
	xlsx, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("打开真实 XLSX fixture 失败: %v", err)
	}
	return xlsx
}

func saveXlsxForTest(t *testing.T, xlsx *excelize.File, label string) string {
	t.Helper()
	outputDir := filepath.Join("..", "..", "test", "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("创建 XLSX 输出目录失败: %v", err)
	}
	path := filepath.Join(outputDir, "xlsxlib-"+sanitizeOutputName(t.Name())+"-"+label+".xlsx")
	if err := xlsx.SaveAs(path); err != nil {
		t.Fatalf("保存 XLSX 失败: %v", err)
	}
	return path
}

func saveBytesForTest(t *testing.T, data []byte, prefix, label, ext string) string {
	t.Helper()
	outputDir := filepath.Join("..", "..", "test", "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("创建输出目录失败: %v", err)
	}
	path := filepath.Join(outputDir, prefix+"-"+sanitizeOutputName(t.Name())+"-"+label+ext)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("保存输出文件失败: %v", err)
	}
	return path
}

func sanitizeOutputName(name string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	return replacer.Replace(name)
}

func xlsxFileSize(t *testing.T, path string) int64 {
	t.Helper()
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatalf("读取 XLSX 大小失败: %v", err)
	}
	return stat.Size()
}

func xlsxTextList(xlsx *excelize.File) []string {
	texts := ExtractAllText(xlsx)
	result := make([]string, 0, len(texts))
	for _, text := range texts {
		result = append(result, text.Text)
	}
	return result
}

func pickXlsxReplacementText(t *testing.T, xlsx *excelize.File) string {
	t.Helper()
	for _, text := range ExtractAllText(xlsx) {
		candidate := strings.TrimSpace(text.Text)
		if strings.Contains(candidate, "${") && len([]rune(candidate)) >= 8 {
			return text.Text
		}
	}
	for _, text := range ExtractAllText(xlsx) {
		candidate := strings.TrimSpace(text.Text)
		if len([]rune(candidate)) >= 8 && !strings.Contains(candidate, "\n") {
			return text.Text
		}
	}
	t.Fatalf("真实 XLSX fixture 中未找到可替换文本")
	return ""
}

func xlsxFormulaSnapshot(t *testing.T, xlsx *excelize.File) map[string]string {
	t.Helper()
	result := map[string]string{}
	for _, sheet := range xlsx.GetSheetList() {
		rows, err := xlsx.GetRows(sheet)
		if err != nil {
			t.Fatalf("读取工作表失败 %s: %v", sheet, err)
		}
		for rowIdx, row := range rows {
			for colIdx := range row {
				cell, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				if err != nil {
					t.Fatalf("计算单元格坐标失败: %v", err)
				}
				formula, err := xlsx.GetCellFormula(sheet, cell)
				if err != nil {
					t.Fatalf("读取公式失败 %s!%s: %v", sheet, cell, err)
				}
				if formula != "" {
					result[sheet+"!"+cell] = formula
				}
			}
		}
	}
	return result
}

func xlsxStyleSnapshot(t *testing.T, xlsx *excelize.File) map[string]int {
	t.Helper()
	result := map[string]int{}
	for _, sheet := range xlsx.GetSheetList() {
		rows, err := xlsx.GetRows(sheet)
		if err != nil {
			t.Fatalf("读取工作表失败 %s: %v", sheet, err)
		}
		for rowIdx, row := range rows {
			for colIdx, value := range row {
				if value == "" {
					continue
				}
				cell, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				if err != nil {
					t.Fatalf("计算单元格坐标失败: %v", err)
				}
				styleID, err := xlsx.GetCellStyle(sheet, cell)
				if err != nil {
					t.Fatalf("读取样式失败 %s!%s: %v", sheet, cell, err)
				}
				result[sheet+"!"+cell] = styleID
			}
		}
	}
	return result
}

func xlsxLayoutSnapshot(t *testing.T, xlsx *excelize.File) map[string]string {
	t.Helper()
	result := map[string]string{}
	for _, sheet := range xlsx.GetSheetList() {
		rows, err := xlsx.GetRows(sheet)
		if err != nil {
			t.Fatalf("读取工作表失败 %s: %v", sheet, err)
		}
		result[sheet+":rows"] = fmt.Sprint(len(rows))
		cols, err := xlsx.GetCols(sheet)
		if err != nil {
			t.Fatalf("读取列失败 %s: %v", sheet, err)
		}
		result[sheet+":cols"] = fmt.Sprint(len(cols))
		mergeCells, err := xlsx.GetMergeCells(sheet)
		if err != nil {
			t.Fatalf("读取合并单元格失败 %s: %v", sheet, err)
		}
		refs := make([]string, 0, len(mergeCells))
		for _, mergeCell := range mergeCells {
			refs = append(refs, mergeCell.GetStartAxis()+":"+mergeCell.GetEndAxis())
		}
		result[sheet+":merges"] = strings.Join(refs, "|")
	}
	return result
}

func longXlsxReplacement() string {
	var b strings.Builder
	b.WriteString("XLSX_CLI_LONG_REPLACEMENT_")
	for i := 0; i < 12000; i++ {
		fmt.Fprintf(&b, "%04d_%08x;", i, uint32(i*2654435761))
	}
	return b.String()
}

func assertZipEntriesEqual(t *testing.T, beforePath, afterPath string, include func(string) bool) {
	t.Helper()
	before := readZipFiles(t, beforePath)
	after := readZipFiles(t, afterPath)
	for name, beforeContent := range before {
		if !include(name) {
			continue
		}
		afterContent, ok := after[name]
		if !ok {
			t.Fatalf("XLSX 条目缺失: %s", name)
		}
		if !bytes.Equal(beforeContent, afterContent) {
			t.Fatalf("XLSX 条目不应变化: %s", name)
		}
	}
}

func readZipFiles(t *testing.T, path string) map[string][]byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 ZIP 文件失败: %v", err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("打开 ZIP 文件失败: %v", err)
	}
	files := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("打开 ZIP 条目失败 %s: %v", file.Name, err)
		}
		content, err := io.ReadAll(rc)
		if err != nil {
			_ = rc.Close()
			t.Fatalf("读取 ZIP 条目失败 %s: %v", file.Name, err)
		}
		_ = rc.Close()
		files[file.Name] = content
	}
	return files
}
