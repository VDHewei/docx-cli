package docxlib

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/gomutex/godocx"
	"github.com/gomutex/godocx/docx"
)

func TestReplaceAllByBytes_RealDocx_NoMatchKeepsContentAndStructure(t *testing.T) {
	for _, fixtureName := range docxFixtureNames(t) {
		t.Run(fixtureName, func(t *testing.T) {
			data := readDocxFixtureBytes(t, fixtureName)
			beforeDoc, beforeResult, err := ReplaceAllByBytes(data, nil, ReplaceOptions{Workers: 2})
			if err != nil {
				t.Fatalf("创建 ReplaceAllByBytes 基线失败: %v", err)
			}
			if beforeResult.TotalReplacements != 0 {
				t.Fatalf("无规则基线不应发生替换，实际 %d", beforeResult.TotalReplacements)
			}
			beforeTexts := docxTextList(beforeDoc)

			doc, result, err := ReplaceAllByBytes(data, []ReplacementRule{{Old: "__DOCX_CLI_NO_SUCH_TEXT__", New: "unused"}}, ReplaceOptions{Workers: 2})
			if err != nil {
				t.Fatalf("ReplaceAllByBytes 失败: %v", err)
			}
			if result.TotalReplacements != 0 {
				t.Fatalf("无匹配内容时不应发生替换，实际 %d", result.TotalReplacements)
			}
			if afterTexts := docxTextList(doc); !reflect.DeepEqual(afterTexts, beforeTexts) {
				t.Fatalf("无匹配内容时文本不应变化")
			}

			beforePath := saveDocxForTest(t, beforeDoc)
			afterPath := saveDocxForTest(t, doc)
			assertDocxStructureEqual(t, beforePath, afterPath)
		})
	}
}

func TestReplaceAll_RealDocx_ReplacesTextOnlyAndPreservesStructure(t *testing.T) {
	for _, fixtureName := range docxFixtureNames(t) {
		t.Run(fixtureName, func(t *testing.T) {
			doc := openDocxFixture(t, fixtureName)
			oldText := pickDocxReplacementText(t, doc)
			newText := "DOCX CLI REAL DATA REPLACEMENT"

			beforeText := strings.Join(docxTextList(doc), "")
			beforePath := saveDocxForTest(t, openDocxFixture(t, fixtureName))
			result := ReplaceAll(doc, []ReplacementRule{{Old: oldText, New: newText}}, ReplaceOptions{Workers: 2})
			if result.TotalReplacements == 0 {
				t.Fatalf("期望真实文档至少发生 1 次替换")
			}

			afterText := strings.Join(docxTextList(doc), "")
			if strings.Count(afterText, oldText) >= strings.Count(beforeText, oldText) {
				t.Fatalf("替换后旧文本数量未减少: %q", oldText)
			}
			if !strings.Contains(afterText, newText) {
				t.Fatalf("替换后未找到新文本 %q", newText)
			}

			afterPath := saveDocxForTest(t, doc)
			assertDocxStructureEqual(t, beforePath, afterPath)
			assertDocxEntriesEqual(t, beforePath, afterPath, func(name string) bool {
				return (strings.HasPrefix(name, "word/header") || strings.HasPrefix(name, "word/footer")) && strings.HasSuffix(name, ".xml")
			})
		})
	}
}

func TestReplaceAllByBytes_RealDocx_SavedSizeTracksLongerAndShorterContent(t *testing.T) {
	for _, fixtureName := range docxFixtureNames(t) {
		t.Run(fixtureName, func(t *testing.T) {
			data := readDocxFixtureBytes(t, fixtureName)
			oldText := pickDocxReplacementText(t, openDocxFixture(t, fixtureName))

			longDoc, longResult, err := ReplaceAllByBytes(data, []ReplacementRule{{Old: oldText, New: longDocxReplacement()}}, ReplaceOptions{Workers: 2})
			if err != nil {
				t.Fatalf("长文本 ReplaceAllByBytes 失败: %v", err)
			}
			if longResult.TotalReplacements == 0 {
				t.Fatalf("长文本替换应至少发生 1 次")
			}
			longPath := saveDocxForTest(t, longDoc)
			if got, wantMin := docxFileSize(t, longPath), int64(len(data)); got < wantMin {
				t.Fatalf("替换后总体字符内容更长时，新文档大小应不小于旧文档: got %d, want >= %d", got, wantMin)
			}

			shortDoc, shortResult, err := ReplaceAllByBytes(data, []ReplacementRule{{Old: oldText, New: ""}}, ReplaceOptions{Workers: 2})
			if err != nil {
				t.Fatalf("短文本 ReplaceAllByBytes 失败: %v", err)
			}
			if shortResult.TotalReplacements == 0 {
				t.Fatalf("短文本替换应至少发生 1 次")
			}
			shortPath := saveDocxForTest(t, shortDoc)
			if got, wantMax := docxFileSize(t, shortPath), int64(len(data)); got >= wantMax {
				t.Fatalf("替换后总体字符内容更短时，新文档大小应小于旧文档: got %d, want < %d", got, wantMax)
			}
		})
	}
}

func docxFixtureNames(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join("..", "..", "test", "data", "*.docx"))
	if err != nil {
		t.Fatalf("查找真实 DOCX fixture 失败: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("test/data 下未找到 DOCX fixture")
	}
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, filepath.Base(match))
	}
	sort.Strings(names)
	return names
}

func readDocxFixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "test", "data", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取真实 DOCX fixture 失败: %v", err)
	}
	return data
}

func openDocxFixture(t *testing.T, name string) *docx.RootDoc {
	t.Helper()
	path := filepath.Join("..", "..", "test", "data", name)
	doc, err := godocx.OpenDocument(path)
	if err != nil {
		t.Fatalf("打开真实 DOCX fixture 失败: %v", err)
	}
	return doc
}

func saveDocxForTest(t *testing.T, doc *docx.RootDoc) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "out.docx")
	if err := doc.SaveTo(path); err != nil {
		t.Fatalf("保存 DOCX 失败: %v", err)
	}
	return path
}

func docxFileSize(t *testing.T, path string) int64 {
	t.Helper()
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatalf("读取 DOCX 大小失败: %v", err)
	}
	return stat.Size()
}

func docxTextList(doc *docx.RootDoc) []string {
	texts := ExtractAllText(doc)
	result := make([]string, 0, len(texts))
	for _, text := range texts {
		result = append(result, text.Text)
	}
	return result
}

func pickDocxReplacementText(t *testing.T, doc *docx.RootDoc) string {
	t.Helper()
	for _, text := range ExtractAllText(doc) {
		candidate := strings.TrimSpace(text.Text)
		if len([]rune(candidate)) >= 8 && !strings.Contains(candidate, "\n") {
			return text.Text
		}
	}
	t.Fatalf("真实 DOCX fixture 中未找到可替换文本")
	return ""
}

func longDocxReplacement() string {
	var b strings.Builder
	b.WriteString("DOCX_CLI_LONG_REPLACEMENT_")
	for i := 0; i < 12000; i++ {
		fmt.Fprintf(&b, "%04d_%08x;", i, uint32(i*2654435761))
	}
	return b.String()
}

func assertDocxStructureEqual(t *testing.T, beforePath, afterPath string) {
	t.Helper()
	before := docxStructureSnapshot(t, beforePath)
	after := docxStructureSnapshot(t, afterPath)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("DOCX 非文本结构发生变化")
	}
}

func docxStructureSnapshot(t *testing.T, path string) map[string]string {
	t.Helper()
	files := readZipFiles(t, path)
	snapshot := make(map[string]string, len(files))
	for name, content := range files {
		if strings.HasPrefix(name, "docProps/") {
			continue
		}
		if strings.HasPrefix(name, "word/") && strings.HasSuffix(name, ".xml") {
			content = canonicalWordXMLWithoutText(t, content)
		}
		snapshot[name] = string(content)
	}
	return snapshot
}

func canonicalWordXMLWithoutText(t *testing.T, content []byte) []byte {
	t.Helper()
	decoder := xml.NewDecoder(bytes.NewReader(content))
	var out strings.Builder
	inText := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("解析 DOCX XML 失败: %v", err)
		}
		switch elem := token.(type) {
		case xml.StartElement:
			out.WriteString("<")
			out.WriteString(xmlNameKey(elem.Name))
			attrs := make([]string, 0, len(elem.Attr))
			for _, attr := range elem.Attr {
				attrs = append(attrs, xmlNameKey(attr.Name)+"="+attr.Value)
			}
			sort.Strings(attrs)
			for _, attr := range attrs {
				out.WriteString(" ")
				out.WriteString(attr)
			}
			out.WriteString(">")
			if elem.Name.Local == "t" {
				inText++
			}
		case xml.EndElement:
			if elem.Name.Local == "t" && inText > 0 {
				inText--
			}
			out.WriteString("</")
			out.WriteString(xmlNameKey(elem.Name))
			out.WriteString(">")
		case xml.CharData:
			if inText > 0 {
				out.WriteString("__TEXT__")
			} else {
				out.WriteString(string(elem))
			}
		}
	}
	return []byte(out.String())
}

func xmlNameKey(name xml.Name) string {
	if name.Space == "" {
		return name.Local
	}
	return name.Space + ":" + name.Local
}

func assertDocxEntriesEqual(t *testing.T, beforePath, afterPath string, include func(string) bool) {
	t.Helper()
	before := readZipFiles(t, beforePath)
	after := readZipFiles(t, afterPath)
	for name, beforeContent := range before {
		if !include(name) {
			continue
		}
		afterContent, ok := after[name]
		if !ok {
			t.Fatalf("DOCX 条目缺失: %s", name)
		}
		if !bytes.Equal(beforeContent, afterContent) {
			t.Fatalf("DOCX 条目不应变化: %s", name)
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
