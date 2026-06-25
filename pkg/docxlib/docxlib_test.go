package docxlib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gomutex/godocx"
	"github.com/gomutex/godocx/wml/ctypes"
)

func TestExtractAllText_BodyParagraphs(t *testing.T) {
	doc, err := godocx.NewDocument()
	if err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	doc.AddParagraph("Hello World")
	doc.AddParagraph("Second paragraph")

	texts := ExtractAllText(doc)
	if len(texts) != 2 {
		t.Fatalf("期望提取 2 段文本，实际 %d", len(texts))
	}
	if texts[0].Text != "Hello World" {
		t.Errorf("期望 'Hello World'，实际 '%s'", texts[0].Text)
	}
	if texts[1].Text != "Second paragraph" {
		t.Errorf("期望 'Second paragraph'，实际 '%s'", texts[1].Text)
	}
	if texts[0].Location.Kind != "body" {
		t.Errorf("期望 Kind='body'，实际 '%s'", texts[0].Location.Kind)
	}
}

func TestExtractAllText_TableCells(t *testing.T) {
	doc, err := godocx.NewDocument()
	if err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	table := doc.AddTable()
	row := table.AddRow()
	cell := row.AddCell()
	cell.AddParagraph("Cell A1")
	cell = row.AddCell()
	cell.AddParagraph("Cell B1")

	texts := ExtractAllText(doc)

	var found []string
	for _, dt := range texts {
		if dt.Location.Kind == "table" {
			found = append(found, dt.Text)
		}
	}
	if len(found) != 2 {
		t.Fatalf("期望提取 2 个单元格文本，实际 %d", len(found))
	}
	if found[0] != "Cell A1" {
		t.Errorf("期望 'Cell A1'，实际 '%s'", found[0])
	}
	if found[1] != "Cell B1" {
		t.Errorf("期望 'Cell B1'，实际 '%s'", found[1])
	}
}

func TestReplaceAll_BodyParagraphs(t *testing.T) {
	doc, err := godocx.NewDocument()
	if err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	doc.AddParagraph("Hello World")
	doc.AddParagraph("Hello Go")

	rules := []ReplacementRule{{Old: "Hello", New: "Hi"}}
	result := ReplaceAll(doc, rules, ReplaceOptions{Workers: 2})

	if result.TotalReplacements != 2 {
		t.Errorf("期望替换 2 次，实际 %d", result.TotalReplacements)
	}
	if result.ParagraphsProcessed != 2 {
		t.Errorf("期望处理 2 个段落，实际 %d", result.ParagraphsProcessed)
	}

	texts := ExtractAllText(doc)
	if len(texts) != 2 {
		t.Fatalf("期望 2 段文本，实际 %d", len(texts))
	}
	if texts[0].Text != "Hi World" {
		t.Errorf("段落1期望 'Hi World'，实际 '%s'", texts[0].Text)
	}
	if texts[1].Text != "Hi Go" {
		t.Errorf("段落2期望 'Hi Go'，实际 '%s'", texts[1].Text)
	}
}

func TestReplaceAll_TableCells(t *testing.T) {
	doc, err := godocx.NewDocument()
	if err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	table := doc.AddTable()
	row := table.AddRow()
	cell := row.AddCell()
	cell.AddParagraph("Old Value")
	cell = row.AddCell()
	cell.AddParagraph("Another Old")

	rules := []ReplacementRule{{Old: "Old", New: "New"}}
	result := ReplaceAll(doc, rules, ReplaceOptions{Workers: 2})

	if result.TotalReplacements != 2 {
		t.Errorf("期望替换 2 次，实际 %d", result.TotalReplacements)
	}
	if result.CellsProcessed != 2 {
		t.Errorf("期望处理 2 个单元格，实际 %d", result.CellsProcessed)
	}

	texts := ExtractAllText(doc)
	var found []string
	for _, dt := range texts {
		if dt.Location.Kind == "table" {
			found = append(found, dt.Text)
		}
	}
	if len(found) != 2 {
		t.Fatalf("期望 2 个单元格文本，实际 %d", len(found))
	}
	if !strings.Contains(found[0], "New") {
		t.Errorf("单元格1应包含 'New'，实际 '%s'", found[0])
	}
	if !strings.Contains(found[1], "New") {
		t.Errorf("单元格2应包含 'New'，实际 '%s'", found[1])
	}
}

func TestReplaceAll_SkipHeadersFooters(t *testing.T) {
	doc, err := godocx.NewDocument()
	if err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}
	doc.AddParagraph("Body text")

	// No real headers/footers in a new document, but ensure options are respected.
	rules := []ReplacementRule{{Old: "Body", New: "Content"}}
	result := ReplaceAll(doc, rules, ReplaceOptions{SkipHeaders: true, SkipFooters: true, Workers: 1})
	if result.TotalReplacements != 1 {
		t.Errorf("期望替换 1 次，实际 %d", result.TotalReplacements)
	}
}

func TestReplaceAll_PreservesStyle(t *testing.T) {
	doc, err := godocx.NewDocument()
	if err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	para := doc.AddParagraph("Hello World")
	// Replace the default run with a styled run
	ct := para.GetCT()
	ct.Children = nil
	styledRun := &ctypes.Run{
		Property: &ctypes.RunProperty{
			Bold:   ctypes.OnOffFromBool(true),
			Italic: ctypes.OnOffFromBool(true),
		},
		Children: []ctypes.RunChild{
			{Text: ctypes.TextFromString("Hello World")},
		},
	}
	ct.Children = append(ct.Children, ctypes.ParagraphChild{Run: styledRun})

	rules := []ReplacementRule{{Old: "Hello", New: "Goodbye"}}
	result := ReplaceAll(doc, rules, ReplaceOptions{Workers: 1})

	if result.TotalReplacements != 1 {
		t.Fatalf("期望替换 1 次，实际 %d", result.TotalReplacements)
	}

	// Verify text changed
	ctAfter := para.GetCT()
	if len(ctAfter.Children) == 0 || ctAfter.Children[0].Run == nil {
		t.Fatal("替换后段落结构无效")
	}
	runAfter := ctAfter.Children[0].Run
	if len(runAfter.Children) == 0 || runAfter.Children[0].Text == nil {
		t.Fatal("替换后 Run 结构无效")
	}
	if runAfter.Children[0].Text.Text != "Goodbye World" {
		t.Errorf("文本期望 'Goodbye World'，实际 '%s'", runAfter.Children[0].Text.Text)
	}

	// Verify style preserved
	if runAfter.Property == nil {
		t.Fatal("样式未保留: Property 为 nil")
	}
	if runAfter.Property.Bold == nil || *runAfter.Property.Bold.Val != "true" {
		t.Error("粗体样式未保留")
	}
	if runAfter.Property.Italic == nil || *runAfter.Property.Italic.Val != "true" {
		t.Error("斜体样式未保留")
	}
}

func TestIntegration_ExtractTemplateRISC(t *testing.T) {
	templatePath := filepath.Join("..", "..", "test", "data", "template_RISC.docx")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Skip("跳过集成测试: 未找到 template_RISC.docx")
	}

	doc, err := godocx.OpenDocument(templatePath)
	if err != nil {
		t.Fatalf("打开模板文档失败: %v", err)
	}

	texts := ExtractAllText(doc)
	if len(texts) == 0 {
		t.Error("期望从 template_RISC.docx 中提取到文本，但未找到")
	}

	t.Logf("从 template_RISC.docx 中提取到 %d 段文本", len(texts))
	for i, dt := range texts {
		if i >= 10 {
			break
		}
		t.Logf("  [%s] %s", dt.Location.Kind, dt.Text)
	}
}

func TestIntegration_ReplaceTemplateRISC(t *testing.T) {
	templatePath := filepath.Join("..", "..", "test", "data", "template_RISC.docx")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Skip("跳过集成测试: 未找到 template_RISC.docx")
	}

	doc, err := godocx.OpenDocument(templatePath)
	if err != nil {
		t.Fatalf("打开模板文档失败: %v", err)
	}

	// Gather a sample of existing text to craft a replacement rule.
	textsBefore := ExtractAllText(doc)
	if len(textsBefore) == 0 {
		t.Skip("模板文档中没有可提取的文本，跳过替换测试")
	}

	// Use the first piece of text as old value, append "_REPLACED".
	oldText := textsBefore[0].Text
	if oldText == "" {
		t.Skip("第一段文本为空，跳过")
	}
	newText := oldText + "_REPLACED"

	rules := []ReplacementRule{{Old: oldText, New: newText}}
	result := ReplaceAll(doc, rules, ReplaceOptions{Workers: 4})

	t.Logf("替换结果: %+v", result)

	if result.TotalReplacements == 0 {
		t.Errorf("期望至少发生 1 次替换，实际 %d", result.TotalReplacements)
	}

	// Verify replacement by re-extracting.
	textsAfter := ExtractAllText(doc)
	found := false
	for _, dt := range textsAfter {
		if dt.Text == newText {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("替换后未找到期望的文本 '%s'", newText)
	}
}
