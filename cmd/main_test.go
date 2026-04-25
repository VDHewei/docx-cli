package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VDHewei/docx-cli/pkg/docxlib"
	"github.com/gomutex/godocx"
)

func TestParseReplacement(t *testing.T) {
	tests := []struct {
		input    string
		wantOld  string
		wantNew  string
		wantNil  bool
	}{
		{"foo=bar", "foo", "bar", false},
		{"foo=bar=baz", "foo", "bar=baz", false},
		{"foobar", "", "", true},
	}

	for _, tt := range tests {
		got := parseReplacement(tt.input)
		if tt.wantNil {
			if got != nil {
				t.Errorf("parseReplacement(%q) expected nil", tt.input)
			}
			continue
		}
		if got == nil {
			t.Errorf("parseReplacement(%q) unexpected nil", tt.input)
			continue
		}
		if got.Old != tt.wantOld || got.New != tt.wantNew {
			t.Errorf("parseReplacement(%q) = {%q, %q}, want {%q, %q}",
				tt.input, got.Old, got.New, tt.wantOld, tt.wantNew)
		}
	}
}

func TestDocxlibIntegration_BodyAndTable(t *testing.T) {
	doc, err := godocx.NewDocument()
	if err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	doc.AddParagraph("Hello World")

	table := doc.AddTable()
	row := table.AddRow()
	cell := row.AddCell()
	cell.AddParagraph("Table Cell Old")

	rules := []docxlib.ReplacementRule{{Old: "Hello", New: "Hi"}, {Old: "Old", New: "New"}}
	result := docxlib.ReplaceAll(doc, rules, docxlib.ReplaceOptions{Workers: 2})

	if result.TotalReplacements != 2 {
		t.Errorf("期望替换 2 次，实际 %d", result.TotalReplacements)
	}

	texts := docxlib.ExtractAllText(doc)
	foundBody := false
	foundTable := false
	for _, dt := range texts {
		if dt.Text == "Hi World" {
			foundBody = true
		}
		if dt.Text == "Table Cell New" {
			foundTable = true
		}
	}
	if !foundBody {
		t.Error("正文中未找到 'Hi World'")
	}
	if !foundTable {
		t.Error("表格中未找到 'Table Cell New'")
	}
}

func TestDocxlibIntegration_SaveAndReopen(t *testing.T) {
	doc, err := godocx.NewDocument()
	if err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}
	doc.AddParagraph("Persistent Text")

	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "docx_cli_test.docx")
	defer os.Remove(tmpFile)

	if err := doc.SaveTo(tmpFile); err != nil {
		t.Fatalf("保存失败: %v", err)
	}

	reopened, err := godocx.OpenDocument(tmpFile)
	if err != nil {
		t.Fatalf("重新打开失败: %v", err)
	}

	texts := docxlib.ExtractAllText(reopened)
	if len(texts) == 0 || !strings.Contains(texts[0].Text, "Persistent") {
		t.Errorf("重新打开后文本不匹配: %+v", texts)
	}
}

func TestIntegration_TemplateRISC_Extract(t *testing.T) {
	templatePath := filepath.Join("..", "tests", "template_RISC.docx")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Skip("未找到 template_RISC.docx")
	}

	doc, err := godocx.OpenDocument(templatePath)
	if err != nil {
		t.Fatalf("打开模板失败: %v", err)
	}

	texts := docxlib.ExtractAllText(doc)
	if len(texts) == 0 {
		t.Fatal("从模板中未提取到任何文本")
	}

	t.Logf("从 template_RISC.docx 提取到 %d 段文本", len(texts))
}

func TestIntegration_TemplateRISC_Replace(t *testing.T) {
	templatePath := filepath.Join("..", "tests", "template_RISC.docx")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Skip("未找到 template_RISC.docx")
	}

	doc, err := godocx.OpenDocument(templatePath)
	if err != nil {
		t.Fatalf("打开模板失败: %v", err)
	}

	textsBefore := docxlib.ExtractAllText(doc)
	if len(textsBefore) == 0 {
		t.Skip("模板中没有可提取文本")
	}

	oldText := textsBefore[0].Text
	if oldText == "" {
		t.Skip("第一段文本为空")
	}
	newText := oldText + "_TEST_REPLACED"

	rules := []docxlib.ReplacementRule{{Old: oldText, New: newText}}
	result := docxlib.ReplaceAll(doc, rules, docxlib.ReplaceOptions{Workers: 4})
	if result.TotalReplacements == 0 {
		t.Errorf("期望至少发生 1 次替换，实际 %d", result.TotalReplacements)
	}

	textsAfter := docxlib.ExtractAllText(doc)
	found := false
	for _, dt := range textsAfter {
		if dt.Text == newText {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("替换后未找到 '%s'", newText)
	}
}

func TestReplacedDocOpens(t *testing.T) {
	templatePath := filepath.Join("..", "tests", "template_RISC.docx")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Skip("未找到 template_RISC.docx")
	}

	// 1. 打开原文档
	doc, err := godocx.OpenDocument(templatePath)
	if err != nil {
		t.Fatalf("打开原文档失败: %v", err)
	}

	// 2. 执行替换
	rules := []docxlib.ReplacementRule{{Old: "Contract", New: "Agreement"}}
	result := docxlib.ReplaceAll(doc, rules, docxlib.ReplaceOptions{Workers: 4})
	if result.TotalReplacements == 0 {
		t.Log("警告: 未发生任何替换，但继续验证文档可打开")
	}

	// 3. 保存到临时文件
	tmpDir := os.TempDir()
	replacedPath := filepath.Join(tmpDir, "template_RISC_replaced_test.docx")
	defer os.Remove(replacedPath)

	if err := doc.SaveTo(replacedPath); err != nil {
		t.Fatalf("保存替换后的文档失败: %v", err)
	}

	// 4. 重新打开验证
	reopened, err := godocx.OpenDocument(replacedPath)
	if err != nil {
		t.Fatalf("重新打开替换后的文档失败: %v", err)
	}

	// 5. 验证内容
	texts := docxlib.ExtractAllText(reopened)
	if len(texts) == 0 {
		t.Fatal("替换后的文档重新打开后无法提取到文本")
	}

	// 检查替换是否生效
	found := false
	for _, dt := range texts {
		if strings.Contains(dt.Text, "Agreement") {
			found = true
			break
		}
	}
	if !found && result.TotalReplacements > 0 {
		t.Errorf("替换后重新打开文档，未找到 'Agreement'")
	}

	t.Logf("替换后文档验证通过: 提取到 %d 段文本", len(texts))
}
