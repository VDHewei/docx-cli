package xlsxlib

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"regexp"
	"strings"
)

var xlsxTextElementRE = regexp.MustCompile(`(?s)(<((?:[A-Za-z0-9_]+:)?t)\b[^>]*>)(.*?)(</(?:[A-Za-z0-9_]+:)?t>)`)

// ReplaceAllBytesPreservePackage replaces text directly inside the XLSX OOXML
// package and returns the updated package bytes. If no replacement is made, the
// original bytes are returned unchanged.
func ReplaceAllBytesPreservePackage(data []byte, rules []ReplacementRule, opts ReplaceOptions) ([]byte, ReplaceResult, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, ReplaceResult{}, err
	}

	modifiedFiles := make(map[string][]byte)
	result := ReplaceResult{}
	processedSheets := make(map[string]bool)
	for _, file := range reader.File {
		if !isXlsxReplaceableTextXML(file.Name) {
			continue
		}

		content, err := readZipEntry(file)
		if err != nil {
			return nil, ReplaceResult{}, err
		}
		modified, count := replaceXlsxXMLTextContent(content, rules)
		if count == 0 {
			continue
		}

		modifiedFiles[file.Name] = modified
		result.TotalReplacements += count
		result.CellsProcessed += count
		if strings.HasPrefix(file.Name, "xl/worksheets/") {
			processedSheets[file.Name] = true
		}
	}

	if result.TotalReplacements == 0 {
		return append([]byte(nil), data...), result, nil
	}

	result.SheetsProcessed = len(processedSheets)
	output, err := rewriteZipPackage(reader.File, modifiedFiles)
	if err != nil {
		return nil, ReplaceResult{}, err
	}
	return output, result, nil
}

func isXlsxReplaceableTextXML(name string) bool {
	return name == "xl/sharedStrings.xml" ||
		(strings.HasPrefix(name, "xl/worksheets/") && strings.HasSuffix(name, ".xml"))
}

func replaceXlsxXMLTextContent(content []byte, rules []ReplacementRule) ([]byte, int) {
	total := 0
	modified := xlsxTextElementRE.ReplaceAllFunc(content, func(match []byte) []byte {
		parts := xlsxTextElementRE.FindSubmatch(match)
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

func xmlUnescapeString(s string) (string, error) {
	var value string
	if err := xml.Unmarshal([]byte("<x>"+s+"</x>"), &value); err != nil {
		return "", err
	}
	return value, nil
}

func readZipEntry(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func rewriteZipPackage(files []*zip.File, modifiedFiles map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for _, file := range files {
		modified, ok := modifiedFiles[file.Name]
		if !ok {
			if err := writer.Copy(file); err != nil {
				_ = writer.Close()
				return nil, err
			}
			continue
		}

		header := file.FileHeader
		header.CRC32 = 0
		header.CompressedSize = 0
		header.UncompressedSize = 0
		header.CompressedSize64 = 0
		header.UncompressedSize64 = 0
		entryWriter, err := writer.CreateHeader(&header)
		if err != nil {
			_ = writer.Close()
			return nil, err
		}
		if _, err := entryWriter.Write(modified); err != nil {
			_ = writer.Close()
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
