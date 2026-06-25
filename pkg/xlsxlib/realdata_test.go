package xlsxlib

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestReplaceAllByBytes_RealXLSX_NoMatchKeepsContentAndStructure(t *testing.T) {
	data := readXlsxFixtureBytes(t, "template_SD.xlsx")
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

	beforePath := saveXlsxForTest(t, before)
	afterPath := saveXlsxForTest(t, xlsx)
	if xlsxFileSize(t, afterPath) != xlsxFileSize(t, beforePath) {
		t.Fatalf("无匹配内容时保存后的工作簿大小应保持不变")
	}
}

func TestReplaceAll_RealXLSX_ReplacesTextOnlyAndPreservesWorkbookSemantics(t *testing.T) {
	xlsx := openXlsxFixture(t, "template_SD.xlsx")
	defer xlsx.Close()
	oldText := pickXlsxReplacementText(t, xlsx)
	newText := "XLSX_CLI_REAL_DATA_REPLACEMENT"
	beforeFormulas := xlsxFormulaSnapshot(t, xlsx)
	beforeStyles := xlsxStyleSnapshot(t, xlsx)
	beforeLayout := xlsxLayoutSnapshot(t, xlsx)

	result := ReplaceAll(xlsx, []ReplacementRule{{Old: oldText, New: newText}}, ReplaceOptions{Workers: 2})
	if result.TotalReplacements == 0 {
		t.Fatalf("期望真实工作簿至少发生 1 次替换")
	}
	afterText := strings.Join(xlsxTextList(xlsx), "")
	if strings.Contains(afterText, oldText) {
		t.Fatalf("替换后不应再包含旧文本 %q", oldText)
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
}

func TestReplaceAll_RealXLSX_DoesNotReplaceFormulaResults(t *testing.T) {
	xlsx := openXlsxFixture(t, "template_SD.xlsx")
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
}

func TestReplaceAllByBytes_RealXLSX_SavedSizeTracksLongerAndShorterContent(t *testing.T) {
	data := readXlsxFixtureBytes(t, "template_SD.xlsx")
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
	longPath := saveXlsxForTest(t, longXLSX)
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
	shortPath := saveXlsxForTest(t, shortXLSX)
	if got, wantMax := xlsxFileSize(t, shortPath), int64(len(data)); got >= wantMax {
		t.Fatalf("替换后总体字符内容更短时，新工作簿大小应小于旧工作簿: got %d, want < %d", got, wantMax)
	}
}

func readXlsxFixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "test", "data", name)
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

func saveXlsxForTest(t *testing.T, xlsx *excelize.File) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "out.xlsx")
	if err := xlsx.SaveAs(path); err != nil {
		t.Fatalf("保存 XLSX 失败: %v", err)
	}
	return path
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
	for i := 0; i < 3000; i++ {
		fmt.Fprintf(&b, "%04d_%08x;", i, uint32(i*2654435761))
	}
	return b.String()
}
