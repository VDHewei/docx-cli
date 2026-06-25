package xlsxlib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestExtractAllText_BasicCells(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetCellValue("Sheet1", "A1", "Hello World")
	f.SetCellValue("Sheet1", "B2", "Second Cell")

	texts := ExtractAllText(f)
	if len(texts) < 2 {
		t.Fatalf("期望至少提取 2 个单元格，实际 %d", len(texts))
	}

	foundHello := false
	foundSecond := false
	for _, ct := range texts {
		if ct.Text == "Hello World" {
			foundHello = true
			if ct.Location.Sheet != "Sheet1" {
				t.Errorf("期望 Sheet='Sheet1'，实际 '%s'", ct.Location.Sheet)
			}
			if ct.Location.Cell != "A1" {
				t.Errorf("期望 Cell='A1'，实际 '%s'", ct.Location.Cell)
			}
		}
		if ct.Text == "Second Cell" {
			foundSecond = true
		}
	}
	if !foundHello {
		t.Error("未找到 'Hello World'")
	}
	if !foundSecond {
		t.Error("未找到 'Second Cell'")
	}
}

func TestExtractAllText_MultipleSheets(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetCellValue("Sheet1", "A1", "Sheet1 Data")
	f.NewSheet("Sheet2")
	f.SetCellValue("Sheet2", "A1", "Sheet2 Data")

	texts := ExtractAllText(f)
	foundSheet1 := false
	foundSheet2 := false
	for _, ct := range texts {
		if ct.Text == "Sheet1 Data" && ct.Location.Sheet == "Sheet1" {
			foundSheet1 = true
		}
		if ct.Text == "Sheet2 Data" && ct.Location.Sheet == "Sheet2" {
			foundSheet2 = true
		}
	}
	if !foundSheet1 {
		t.Error("未在 Sheet1 中找到 'Sheet1 Data'")
	}
	if !foundSheet2 {
		t.Error("未在 Sheet2 中找到 'Sheet2 Data'")
	}
}

func TestExtractAllText_NilFile(t *testing.T) {
	texts := ExtractAllText(nil)
	if len(texts) != 0 {
		t.Errorf("期望 0 个文本，实际 %d", len(texts))
	}
}

func TestFindText(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetCellValue("Sheet1", "A1", "Hello World")
	f.SetCellValue("Sheet1", "A2", "Goodbye World")
	f.SetCellValue("Sheet1", "A3", "Hello Again")

	results := FindText(f, "Hello")
	if len(results) != 2 {
		t.Fatalf("期望找到 2 个单元格，实际 %d", len(results))
	}
	for _, r := range results {
		if !strings.Contains(r.Text, "Hello") {
			t.Errorf("结果不包含 'Hello': %s", r.Text)
		}
	}
}

func TestReplaceAll_BasicReplace(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetCellValue("Sheet1", "A1", "Hello World")
	f.SetCellValue("Sheet1", "A2", "Hello Go")

	rules := []ReplacementRule{{Old: "Hello", New: "Hi"}}
	result := ReplaceAll(f, rules, ReplaceOptions{Workers: 2})

	if result.TotalReplacements != 2 {
		t.Errorf("期望替换 2 次，实际 %d", result.TotalReplacements)
	}
	if result.CellsProcessed != 2 {
		t.Errorf("期望处理 2 个单元格，实际 %d", result.CellsProcessed)
	}

	val1, _ := f.GetCellValue("Sheet1", "A1")
	val2, _ := f.GetCellValue("Sheet1", "A2")
	if val1 != "Hi World" {
		t.Errorf("A1 期望 'Hi World'，实际 '%s'", val1)
	}
	if val2 != "Hi Go" {
		t.Errorf("A2 期望 'Hi Go'，实际 '%s'", val2)
	}
}

func TestReplaceAll_MultipleRules(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetCellValue("Sheet1", "A1", "Old Company Name")

	rules := []ReplacementRule{
		{Old: "Old", New: "New"},
		{Old: "Company", New: "Corp"},
	}
	result := ReplaceAll(f, rules, ReplaceOptions{Workers: 1})

	if result.TotalReplacements != 2 {
		t.Errorf("期望替换 2 次，实际 %d", result.TotalReplacements)
	}

	val, _ := f.GetCellValue("Sheet1", "A1")
	if val != "New Corp Name" {
		t.Errorf("期望 'New Corp Name'，实际 '%s'", val)
	}
}

func TestReplaceAll_NoMatch(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetCellValue("Sheet1", "A1", "Hello World")

	rules := []ReplacementRule{{Old: "NotFound", New: "Replaced"}}
	result := ReplaceAll(f, rules, ReplaceOptions{Workers: 1})

	if result.TotalReplacements != 0 {
		t.Errorf("期望 0 次替换，实际 %d", result.TotalReplacements)
	}

	val, _ := f.GetCellValue("Sheet1", "A1")
	if val != "Hello World" {
		t.Errorf("不应修改未匹配的单元格，实际 '%s'", val)
	}
}

func TestReplaceAll_NilFile(t *testing.T) {
	rules := []ReplacementRule{{Old: "a", New: "b"}}
	result := ReplaceAll(nil, rules, ReplaceOptions{Workers: 1})
	if result.TotalReplacements != 0 {
		t.Errorf("期望 0 次替换，实际 %d", result.TotalReplacements)
	}
}

func TestReplaceAll_PreservesStyle(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	// Create a styled cell
	styleID, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Italic: true,
			Family: "Arial",
			Size:   14,
			Color:  "FF0000",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#FFFF00"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	if err != nil {
		t.Fatalf("创建样式失败: %v", err)
	}

	f.SetCellValue("Sheet1", "A1", "Hello World")
	if err := f.SetCellStyle("Sheet1", "A1", "A1", styleID); err != nil {
		t.Fatalf("设置样式失败: %v", err)
	}

	// Record style before replacement
	styleBefore, err := f.GetCellStyle("Sheet1", "A1")
	if err != nil {
		t.Fatalf("获取样式失败: %v", err)
	}

	// Perform replacement
	rules := []ReplacementRule{{Old: "Hello", New: "Hi"}}
	result := ReplaceAll(f, rules, ReplaceOptions{Workers: 1})

	if result.TotalReplacements != 1 {
		t.Fatalf("期望替换 1 次，实际 %d", result.TotalReplacements)
	}

	// Verify text changed
	val, _ := f.GetCellValue("Sheet1", "A1")
	if val != "Hi World" {
		t.Errorf("期望 'Hi World'，实际 '%s'", val)
	}

	// Verify style preserved
	styleAfter, err := f.GetCellStyle("Sheet1", "A1")
	if err != nil {
		t.Fatalf("获取替换后样式失败: %v", err)
	}
	if styleAfter != styleBefore {
		t.Errorf("样式未保留: 替换前 styleID=%d，替换后 styleID=%d", styleBefore, styleAfter)
	}

	// Verify style details
	styleDetail, err := f.GetStyle(styleAfter)
	if err != nil {
		t.Fatalf("获取样式详情失败: %v", err)
	}
	if styleDetail.Font == nil || !styleDetail.Font.Bold || !styleDetail.Font.Italic {
		t.Error("字体样式(Bold/Italic)未保留")
	}
	if styleDetail.Font.Family != "Arial" {
		t.Errorf("字体族未保留: 期望 'Arial'，实际 '%s'", styleDetail.Font.Family)
	}
	if styleDetail.Font.Size != 14 {
		t.Errorf("字体大小未保留: 期望 14，实际 %f", styleDetail.Font.Size)
	}
}

func TestReplaceAll_PreservesColumnWidthAndRowHeight(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetCellValue("Sheet1", "A1", "Hello World")
	f.SetColWidth("Sheet1", "A", "A", 30)
	f.SetRowHeight("Sheet1", 1, 40)

	rules := []ReplacementRule{{Old: "Hello", New: "Hi"}}
	_ = ReplaceAll(f, rules, ReplaceOptions{Workers: 1})

	// Verify column width preserved
	colWidth, err := f.GetColWidth("Sheet1", "A")
	if err != nil {
		t.Fatalf("获取列宽失败: %v", err)
	}
	if colWidth != 30 {
		t.Errorf("列宽未保留: 期望 30，实际 %f", colWidth)
	}

	// Verify row height preserved
	rowHeight, err := f.GetRowHeight("Sheet1", 1)
	if err != nil {
		t.Fatalf("获取行高失败: %v", err)
	}
	if rowHeight != 40 {
		t.Errorf("行高未保留: 期望 40，实际 %f", rowHeight)
	}
}

func TestReplaceAll_MultipleSheets(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetCellValue("Sheet1", "A1", "Hello World")
	f.NewSheet("Sheet2")
	f.SetCellValue("Sheet2", "A1", "Hello Again")

	rules := []ReplacementRule{{Old: "Hello", New: "Hi"}}
	result := ReplaceAll(f, rules, ReplaceOptions{Workers: 2})

	if result.TotalReplacements != 2 {
		t.Errorf("期望替换 2 次，实际 %d", result.TotalReplacements)
	}
	if result.SheetsProcessed != 2 {
		t.Errorf("期望处理 2 个工作表，实际 %d", result.SheetsProcessed)
	}

	val1, _ := f.GetCellValue("Sheet1", "A1")
	val2, _ := f.GetCellValue("Sheet2", "A1")
	if val1 != "Hi World" {
		t.Errorf("Sheet1 A1 期望 'Hi World'，实际 '%s'", val1)
	}
	if val2 != "Hi Again" {
		t.Errorf("Sheet2 A1 期望 'Hi Again'，实际 '%s'", val2)
	}
}

func TestIntegration_SaveAndReopen(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetCellValue("Sheet1", "A1", "Hello World")

	// Add a style
	styleID, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12},
	})
	f.SetCellStyle("Sheet1", "A1", "A1", styleID)

	// Replace
	rules := []ReplacementRule{{Old: "Hello", New: "Goodbye"}}
	_ = ReplaceAll(f, rules, ReplaceOptions{Workers: 1})

	// Save
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "xlsxlib_test.xlsx")
	defer os.Remove(tmpFile)

	if err := f.SaveAs(tmpFile); err != nil {
		t.Fatalf("保存失败: %v", err)
	}

	// Reopen and verify
	f2, err := excelize.OpenFile(tmpFile)
	if err != nil {
		t.Fatalf("重新打开失败: %v", err)
	}
	defer f2.Close()

	val, err := f2.GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("读取单元格失败: %v", err)
	}
	if val != "Goodbye World" {
		t.Errorf("期望 'Goodbye World'，实际 '%s'", val)
	}
}

// --- resolveMatcher tests ---

func TestResolveMatcher_EmptyPattern(t *testing.T) {
	handler := resolveMatcher("")
	if !handler("") {
		t.Error("空模式应匹配空字符串")
	}
	if handler("Sheet1") {
		t.Error("空模式不应匹配非空字符串")
	}
}

func TestResolveMatcher_ExactMatch(t *testing.T) {
	handler := resolveMatcher("Sheet1")
	if !handler("Sheet1") {
		t.Error("应匹配 'Sheet1'")
	}
	if !handler("SHEET1") {
		t.Error("精确匹配应不区分大小写")
	}
	if handler("Sheet2") {
		t.Error("不应匹配 'Sheet2'")
	}
}

func TestResolveMatcher_NegativeSubstring(t *testing.T) {
	handler := resolveMatcher("!Summary")
	if !handler("Data") {
		t.Error("不含 'Summary' 的名称应匹配")
	}
	if !handler("DetailSheet") {
		t.Error("不含 'Summary' 的名称应匹配")
	}
	if handler("Summary") {
		t.Error("包含 'Summary' 的名称不应匹配否定模式")
	}
	if handler("SummaryReport") {
		t.Error("包含 'Summary' 的名称不应匹配否定模式")
	}
}

func TestResolveMatcher_SuffixMatch(t *testing.T) {
	handler := resolveMatcher("*.Data")
	if !handler("Sales.Data") {
		t.Error("以 '.Data' 结尾的名称应匹配")
	}
	if handler("DataSheet") {
		t.Error("不以 '.Data' 结尾的名称不应匹配")
	}
}

func TestResolveMatcher_RegexpMatch(t *testing.T) {
	handler := resolveMatcher("@regexp:^Sheet\\d+$")
	if !handler("Sheet1") {
		t.Error("应匹配 'Sheet1'")
	}
	if !handler("Sheet99") {
		t.Error("应匹配 'Sheet99'")
	}
	if handler("SheetA") {
		t.Error("不应匹配 'SheetA'")
	}
	if handler("MySheet1") {
		t.Error("不应匹配 'MySheet1'（不以 Sheet 开头）")
	}
}

// --- CheckSkip tests ---

func TestCheckSkip_NoPatterns(t *testing.T) {
	opts := ReplaceOptions{SkipSheets: nil}
	if opts.CheckSkip("Sheet1") {
		t.Error("无模式时不应跳过任何工作表")
	}
}

func TestCheckSkip_ExactMatch(t *testing.T) {
	opts := ReplaceOptions{SkipSheets: []string{"Sheet1"}}
	if !opts.CheckSkip("Sheet1") {
		t.Error("应跳过 'Sheet1'")
	}
	if opts.CheckSkip("Sheet2") {
		t.Error("不应跳过 'Sheet2'")
	}
}

func TestCheckSkip_MultiplePatterns(t *testing.T) {
	opts := ReplaceOptions{SkipSheets: []string{"Sheet1", "!Data"}}
	if !opts.CheckSkip("Sheet1") {
		t.Error("应跳过 'Sheet1'（精确匹配）")
	}
	if !opts.CheckSkip("Report") {
		t.Error("应跳过 'Report'（不含 'Data'）")
	}
	if opts.CheckSkip("DataSummary") {
		t.Error("不应跳过包含 'Data' 的工作表")
	}
}

// --- ReplaceAll with SkipSheets ---

func TestReplaceAll_SkipSheets(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetCellValue("Sheet1", "A1", "Hello World")
	f.NewSheet("Sheet2")
	f.SetCellValue("Sheet2", "A1", "Hello Again")

	rules := []ReplacementRule{{Old: "Hello", New: "Hi"}}
	result := ReplaceAll(f, rules, ReplaceOptions{
		Workers:    1,
		SkipSheets: []string{"Sheet2"},
	})

	if result.TotalReplacements != 1 {
		t.Errorf("期望替换 1 次（跳过 Sheet2），实际 %d", result.TotalReplacements)
	}
	if result.SheetsProcessed != 1 {
		t.Errorf("期望处理 1 个工作表，实际 %d", result.SheetsProcessed)
	}

	val1, _ := f.GetCellValue("Sheet1", "A1")
	val2, _ := f.GetCellValue("Sheet2", "A1")
	if val1 != "Hi World" {
		t.Errorf("Sheet1 A1 期望 'Hi World'，实际 '%s'", val1)
	}
	if val2 != "Hello Again" {
		t.Errorf("Sheet2 A1 应未被修改，实际 '%s'", val2)
	}
}

func TestReplaceAll_SkipSheets_NegativePattern(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetCellValue("Sheet1", "A1", "Hello World")
	f.NewSheet("Summary")
	f.SetCellValue("Summary", "A1", "Hello Summary")

	rules := []ReplacementRule{{Old: "Hello", New: "Hi"}}
	result := ReplaceAll(f, rules, ReplaceOptions{
		Workers:    1,
		SkipSheets: []string{"!Summary"}, // skip sheets NOT containing "Summary"
	})

	if result.TotalReplacements != 1 {
		t.Errorf("期望替换 1 次（仅处理 Summary），实际 %d", result.TotalReplacements)
	}

	val1, _ := f.GetCellValue("Sheet1", "A1")
	val2, _ := f.GetCellValue("Summary", "A1")
	if val1 != "Hello World" {
		t.Errorf("Sheet1 应被跳过，实际 '%s'", val1)
	}
	if val2 != "Hi Summary" {
		t.Errorf("Summary A1 期望 'Hi Summary'，实际 '%s'", val2)
	}
}

func TestReplaceAll_SkipSheets_SuffixPattern(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetCellValue("Sheet1", "A1", "Hello World")
	f.NewSheet("Sales.Data")
	f.SetCellValue("Sales.Data", "A1", "Hello Data")

	rules := []ReplacementRule{{Old: "Hello", New: "Hi"}}
	result := ReplaceAll(f, rules, ReplaceOptions{
		Workers:    1,
		SkipSheets: []string{"*.Data"}, // skip sheets ending with ".Data"
	})

	if result.TotalReplacements != 1 {
		t.Errorf("期望替换 1 次（跳过 *.Data），实际 %d", result.TotalReplacements)
	}

	val1, _ := f.GetCellValue("Sheet1", "A1")
	val2, _ := f.GetCellValue("Sales.Data", "A1")
	if val1 != "Hi World" {
		t.Errorf("Sheet1 A1 期望 'Hi World'，实际 '%s'", val1)
	}
	if val2 != "Hello Data" {
		t.Errorf("Sales.Data A1 应被跳过，实际 '%s'", val2)
	}
}

func TestReplaceAll_SkipSheets_RegexpPattern(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetCellValue("Sheet1", "A1", "Hello World")
	f.NewSheet("Config")
	f.SetCellValue("Config", "A1", "Hello Config")

	rules := []ReplacementRule{{Old: "Hello", New: "Hi"}}
	result := ReplaceAll(f, rules, ReplaceOptions{
		Workers:    1,
		SkipSheets: []string{"@regexp:^Sheet\\d+$"}, // skip sheets matching regex
	})

	if result.TotalReplacements != 1 {
		t.Errorf("期望替换 1 次（跳过正则匹配的工作表），实际 %d", result.TotalReplacements)
	}

	val1, _ := f.GetCellValue("Sheet1", "A1")
	val2, _ := f.GetCellValue("Config", "A1")
	if val1 != "Hello World" {
		t.Errorf("Sheet1 应被跳过，实际 '%s'", val1)
	}
	if val2 != "Hi Config" {
		t.Errorf("Config A1 期望 'Hi Config'，实际 '%s'", val2)
	}
}

// --- ReplaceAllByBytes tests ---

func TestReplaceAllByBytes_Basic(t *testing.T) {
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "Hello World")
	f.SetCellValue("Sheet1", "A2", "Hello Go")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("写入缓冲区失败: %v", err)
	}
	f.Close()

	rules := []ReplacementRule{{Old: "Hello", New: "Hi"}}
	xlsx, result, err := ReplaceAllByBytes(buf.Bytes(), rules, ReplaceOptions{Workers: 1})
	if err != nil {
		t.Fatalf("ReplaceAllByBytes 失败: %v", err)
	}
	defer xlsx.Close()

	if result.TotalReplacements != 2 {
		t.Errorf("期望替换 2 次，实际 %d", result.TotalReplacements)
	}
	if result.CellsProcessed != 2 {
		t.Errorf("期望处理 2 个单元格，实际 %d", result.CellsProcessed)
	}

	val1, _ := xlsx.GetCellValue("Sheet1", "A1")
	val2, _ := xlsx.GetCellValue("Sheet1", "A2")
	if val1 != "Hi World" {
		t.Errorf("A1 期望 'Hi World'，实际 '%s'", val1)
	}
	if val2 != "Hi Go" {
		t.Errorf("A2 期望 'Hi Go'，实际 '%s'", val2)
	}
}

func TestReplaceAllByBytes_InvalidData(t *testing.T) {
	_, _, err := ReplaceAllByBytes([]byte("not an xlsx"), nil, ReplaceOptions{})
	if err == nil {
		t.Error("期望返回错误，但得到 nil")
	}
}

func TestReplaceAllByBytes_WithSkipSheets(t *testing.T) {
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "Hello World")
	f.NewSheet("Sheet2")
	f.SetCellValue("Sheet2", "A1", "Hello Skip")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("写入缓冲区失败: %v", err)
	}
	f.Close()

	rules := []ReplacementRule{{Old: "Hello", New: "Hi"}}
	xlsx, result, err := ReplaceAllByBytes(buf.Bytes(), rules, ReplaceOptions{
		Workers:    1,
		SkipSheets: []string{"Sheet2"},
	})
	if err != nil {
		t.Fatalf("ReplaceAllByBytes 失败: %v", err)
	}
	defer xlsx.Close()

	if result.TotalReplacements != 1 {
		t.Errorf("期望替换 1 次（跳过 Sheet2），实际 %d", result.TotalReplacements)
	}

	val1, _ := xlsx.GetCellValue("Sheet1", "A1")
	val2, _ := xlsx.GetCellValue("Sheet2", "A1")
	if val1 != "Hi World" {
		t.Errorf("Sheet1 A1 期望 'Hi World'，实际 '%s'", val1)
	}
	if val2 != "Hello Skip" {
		t.Errorf("Sheet2 A1 应未被修改，实际 '%s'", val2)
	}
}

func TestReplaceAllByBytes_PreservesStyle(t *testing.T) {
	f := excelize.NewFile()
	styleID, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14},
	})
	f.SetCellValue("Sheet1", "A1", "Hello World")
	f.SetCellStyle("Sheet1", "A1", "A1", styleID)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("写入缓冲区失败: %v", err)
	}
	f.Close()

	rules := []ReplacementRule{{Old: "Hello", New: "Hi"}}
	xlsx, result, err := ReplaceAllByBytes(buf.Bytes(), rules, ReplaceOptions{Workers: 1})
	if err != nil {
		t.Fatalf("ReplaceAllByBytes 失败: %v", err)
	}
	defer xlsx.Close()

	if result.TotalReplacements != 1 {
		t.Fatalf("期望替换 1 次，实际 %d", result.TotalReplacements)
	}

	val, _ := xlsx.GetCellValue("Sheet1", "A1")
	if val != "Hi World" {
		t.Errorf("期望 'Hi World'，实际 '%s'", val)
	}

	styleAfter, _ := xlsx.GetCellStyle("Sheet1", "A1")
	styleDetail, _ := xlsx.GetStyle(styleAfter)
	if styleDetail.Font == nil || !styleDetail.Font.Bold {
		t.Error("粗体样式未保留")
	}
	if styleDetail.Font.Size != 14 {
		t.Errorf("字体大小未保留: 期望 14，实际 %f", styleDetail.Font.Size)
	}
}

func TestReplaceAllBytesPreservePackage_WithSkipSheets(t *testing.T) {
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "Hello Shared")
	f.NewSheet("Sheet2")
	f.SetCellValue("Sheet2", "A1", "Hello Shared")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("写入缓冲区失败: %v", err)
	}
	f.Close()

	output, result, err := ReplaceAllBytesPreservePackage(buf.Bytes(), []ReplacementRule{{Old: "Hello", New: "Hi"}}, ReplaceOptions{
		Workers:    1,
		SkipSheets: []string{"Sheet2"},
	})
	if err != nil {
		t.Fatalf("ReplaceAllBytesPreservePackage 失败: %v", err)
	}
	if result.TotalReplacements != 1 {
		t.Fatalf("期望替换 1 次，实际 %d", result.TotalReplacements)
	}

	xlsx, err := excelize.OpenReader(bytes.NewReader(output))
	if err != nil {
		t.Fatalf("打开保真替换结果失败: %v", err)
	}
	defer xlsx.Close()

	val1, _ := xlsx.GetCellValue("Sheet1", "A1")
	val2, _ := xlsx.GetCellValue("Sheet2", "A1")
	if val1 != "Hi Shared" {
		t.Errorf("Sheet1 A1 期望 'Hi Shared'，实际 '%s'", val1)
	}
	if val2 != "Hello Shared" {
		t.Errorf("Sheet2 A1 应未被修改，实际 '%s'", val2)
	}
}
