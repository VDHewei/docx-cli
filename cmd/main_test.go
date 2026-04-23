package main

import (
	"os"
	"testing"

	"github.com/gomutex/godocx"
)

func TestPerformReplacements(t *testing.T) {
	// 创建测试文档
	doc, err := godocx.NewDocument()
	if err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	doc.AddParagraph("Hello World")
	doc.AddParagraph("Test Document")
	doc.AddParagraph("Hello Everyone")

	// 直接在内存中操作（不保存到文件）
	replacements := []Replacement{
		{Old: "Hello", New: "Goodbye"},
	}
	stats := performReplacements(doc, replacements, false, false, false)

	// 验证结果
	if stats.totalReplacements != 2 {
		t.Errorf("期望替换 2 次，实际 %d 次", stats.totalReplacements)
	}

	if stats.paragraphsProcessed != 2 {
		t.Errorf("期望处理 2 个段落，实际 %d 个", stats.paragraphsProcessed)
	}

	// 验证文本内容
	paragraphs := doc.Document.Body.Children
	if len(paragraphs) != 3 {
		t.Errorf("期望 3 个段落，实际 %d 个", len(paragraphs))
	}

	// 检查第一个段落
	text1 := getParagraphText(paragraphs[0].Para)
	if text1 != "Goodbye World" {
		t.Errorf("段落1期望 'Goodbye World'，实际 '%s'", text1)
	}

	// 检查第三个段落
	text3 := getParagraphText(paragraphs[2].Para)
	if text3 != "Goodbye Everyone" {
		t.Errorf("段落3期望 'Goodbye Everyone'，实际 '%s'", text3)
	}

	t.Log("替换功能测试通过")
}

func TestPerformReplacementsMultiple(t *testing.T) {
	// 创建文档
	doc, err := godocx.NewDocument()
	if err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	doc.AddParagraph("Hello World")
	doc.AddParagraph("ABC Corporation")

	// 多个替换规则
	replacements := []Replacement{
		{Old: "Hello", New: "Goodbye"},
		{Old: "ABC Corporation", New: "XYZ Company"},
	}
	stats := performReplacements(doc, replacements, false, false, false)

	// 验证
	if stats.totalReplacements != 2 {
		t.Errorf("期望 2 次替换，实际 %d 次", stats.totalReplacements)
	}

	paragraphs := doc.Document.Body.Children
	text0 := getParagraphText(paragraphs[0].Para)
	text1 := getParagraphText(paragraphs[1].Para)

	if text0 != "Goodbye World" {
		t.Errorf("段落0期望 'Goodbye World'，实际 '%s'", text0)
	}
	if text1 != "XYZ Company" {
		t.Errorf("段落1期望 'XYZ Company'，实际 '%s'", text1)
	}

	t.Log("多规则替换测试通过")
}

func TestSaveDocument(t *testing.T) {
	// 创建文档
	doc, err := godocx.NewDocument()
	if err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	doc.AddParagraph("Test Content")

	// 使用临时目录
	tmpDir := os.TempDir()
	tmpFile := tmpDir + "/docx_test_output.docx"
	
	if err := doc.SaveTo(tmpFile); err != nil {
		t.Fatalf("保存文档失败: %v", err)
	}
	defer os.Remove(tmpFile)

	// 重新打开验证
	rootDoc, err := godocx.OpenDocument(tmpFile)
	if err != nil {
		t.Fatalf("打开文档失败: %v", err)
	}

	if rootDoc.Document == nil || rootDoc.Document.Body == nil {
		t.Error("文档结构无效")
	}

	if len(rootDoc.Document.Body.Children) != 1 {
		t.Errorf("期望 1 个段落，实际 %d 个", len(rootDoc.Document.Body.Children))
	}

	t.Log("保存功能测试通过")
}

func TestGetParagraphText(t *testing.T) {
	// 创建文档
	doc, err := godocx.NewDocument()
	if err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	doc.AddParagraph("Test Paragraph Content")

	// 获取段落文本
	if len(doc.Document.Body.Children) == 0 {
		t.Fatal("没有段落")
	}

	text := getParagraphText(doc.Document.Body.Children[0].Para)
	if text != "Test Paragraph Content" {
		t.Errorf("期望 'Test Paragraph Content'，实际 '%s'", text)
	}

	t.Log("获取段落文本测试通过")
}