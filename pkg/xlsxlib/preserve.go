package xlsxlib

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"encoding/xml"
	"hash/crc32"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var xlsxTextElementRE = regexp.MustCompile(`(?s)(<((?:[A-Za-z0-9_]+:)?t)\b[^>]*>)(.*?)(</(?:[A-Za-z0-9_]+:)?t>)`)
var xlsxSharedStringRE = regexp.MustCompile(`(?s)<si\b[^>]*>.*?</si>`)
var xlsxSharedStringCellRE = regexp.MustCompile(`(?s)<c\b[^>]*\bt="s"[^>]*>.*?</c>`)
var xlsxCellValueRE = regexp.MustCompile(`(?s)(<v>)(\d+)(</v>)`)

// ReplaceAllBytesPreservePackage replaces text directly inside the XLSX OOXML
// package and returns the updated package bytes. If no replacement is made, the
// original bytes are returned unchanged.
func ReplaceAllBytesPreservePackage(data []byte, rules []ReplacementRule, opts ReplaceOptions) ([]byte, ReplaceResult, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, ReplaceResult{}, err
	}

	if len(opts.SkipSheets) > 0 {
		return replaceAllBytesPreservePackageWithSheetSkips(data, reader, rules, opts)
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

type sharedStringModification struct {
	content []byte
	count   int
}

func replaceAllBytesPreservePackageWithSheetSkips(data []byte, reader *zip.Reader, rules []ReplacementRule, opts ReplaceOptions) ([]byte, ReplaceResult, error) {
	skippedWorksheets, err := xlsxSkippedWorksheetPaths(reader.File, opts)
	if err != nil {
		return nil, ReplaceResult{}, err
	}

	contents := make(map[string][]byte)
	for _, file := range reader.File {
		if file.Name == "xl/sharedStrings.xml" || (strings.HasPrefix(file.Name, "xl/worksheets/") && strings.HasSuffix(file.Name, ".xml")) {
			content, err := readZipEntry(file)
			if err != nil {
				return nil, ReplaceResult{}, err
			}
			contents[file.Name] = content
		}
	}

	sharedMods := map[int]sharedStringModification{}
	if sharedStrings, ok := contents["xl/sharedStrings.xml"]; ok {
		for idx, si := range xlsxSharedStringRE.FindAll(sharedStrings, -1) {
			modified, count := replaceXlsxXMLTextContent(si, rules)
			if count > 0 {
				sharedMods[idx] = sharedStringModification{content: modified, count: count}
			}
		}
	}

	usedSharedMods := map[int]sharedStringModification{}
	for name, content := range contents {
		if !strings.HasPrefix(name, "xl/worksheets/") || skippedWorksheets[name] {
			continue
		}
		collectUsedSharedStringMods(content, sharedMods, usedSharedMods)
	}

	sharedIndexMap := map[int]int{}
	if len(usedSharedMods) > 0 {
		sharedStrings := contents["xl/sharedStrings.xml"]
		modifiedSharedStrings, indexMap := appendModifiedSharedStrings(sharedStrings, usedSharedMods)
		contents["xl/sharedStrings.xml"] = modifiedSharedStrings
		sharedIndexMap = indexMap
	}

	modifiedFiles := make(map[string][]byte)
	result := ReplaceResult{}
	processedSheets := map[string]bool{}

	for name, content := range contents {
		if name == "xl/sharedStrings.xml" {
			if len(usedSharedMods) > 0 {
				modifiedFiles[name] = content
			}
			continue
		}
		if skippedWorksheets[name] {
			continue
		}

		modified := content
		sharedCount := 0
		sharedCells := 0
		if len(sharedIndexMap) > 0 {
			modified, sharedCount, sharedCells = replaceSharedStringCellReferences(modified, sharedIndexMap, usedSharedMods)
		}
		modified, inlineCount := replaceXlsxXMLTextContent(modified, rules)
		if sharedCount == 0 && inlineCount == 0 {
			continue
		}

		modifiedFiles[name] = modified
		result.TotalReplacements += sharedCount + inlineCount
		result.CellsProcessed += sharedCells + inlineCount
		processedSheets[name] = true
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

func collectUsedSharedStringMods(content []byte, sharedMods, used map[int]sharedStringModification) {
	xlsxSharedStringCellRE.ReplaceAllFunc(content, func(cell []byte) []byte {
		idx, ok := sharedStringIndexFromCell(cell)
		if !ok {
			return cell
		}
		if mod, ok := sharedMods[idx]; ok {
			used[idx] = mod
		}
		return cell
	})
}

func appendModifiedSharedStrings(content []byte, mods map[int]sharedStringModification) ([]byte, map[int]int) {
	matches := xlsxSharedStringRE.FindAllIndex(content, -1)
	indexMap := make(map[int]int, len(mods))
	closeTag := bytes.LastIndex(content, []byte("</sst>"))
	if closeTag < 0 {
		return content, indexMap
	}

	var appended bytes.Buffer
	nextIndex := len(matches)
	for idx, mod := range mods {
		indexMap[idx] = nextIndex
		nextIndex++
		appended.Write(mod.content)
	}

	output := make([]byte, 0, len(content)+appended.Len())
	output = append(output, content[:closeTag]...)
	output = append(output, appended.Bytes()...)
	output = append(output, content[closeTag:]...)
	output = incrementXMLIntAttr(output, "count", len(mods))
	output = incrementXMLIntAttr(output, "uniqueCount", len(mods))
	return output, indexMap
}

func replaceSharedStringCellReferences(content []byte, indexMap map[int]int, mods map[int]sharedStringModification) ([]byte, int, int) {
	total := 0
	cells := 0
	modified := xlsxSharedStringCellRE.ReplaceAllFunc(content, func(cell []byte) []byte {
		idx, ok := sharedStringIndexFromCell(cell)
		if !ok {
			return cell
		}
		newIdx, ok := indexMap[idx]
		if !ok {
			return cell
		}

		replaced := replaceCellValueIndex(cell, newIdx)
		if bytes.Equal(replaced, cell) {
			return cell
		}
		total += mods[idx].count
		cells++
		return replaced
	})
	return modified, total, cells
}

func sharedStringIndexFromCell(cell []byte) (int, bool) {
	parts := xlsxCellValueRE.FindSubmatch(cell)
	if len(parts) != 4 {
		return 0, false
	}
	idx, err := strconv.Atoi(string(parts[2]))
	if err != nil {
		return 0, false
	}
	return idx, true
}

func replaceCellValueIndex(cell []byte, idx int) []byte {
	parts := xlsxCellValueRE.FindSubmatchIndex(cell)
	if len(parts) != 8 {
		return cell
	}
	value := []byte(strconv.Itoa(idx))
	output := make([]byte, 0, len(cell)-parts[5]+parts[4]+len(value))
	output = append(output, cell[:parts[4]]...)
	output = append(output, value...)
	output = append(output, cell[parts[5]:]...)
	return output
}

func incrementXMLIntAttr(content []byte, attr string, delta int) []byte {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(attr) + `="(\d+)"`)
	return re.ReplaceAllFunc(content, func(match []byte) []byte {
		parts := re.FindSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		value, err := strconv.Atoi(string(parts[1]))
		if err != nil {
			return match
		}
		return []byte(attr + `="` + strconv.Itoa(value+delta) + `"`)
	})
}

func xlsxSkippedWorksheetPaths(files []*zip.File, opts ReplaceOptions) (map[string]bool, error) {
	workbookRels := map[string]string{}
	workbookSheets := map[string]string{}

	for _, file := range files {
		switch file.Name {
		case "xl/_rels/workbook.xml.rels":
			content, err := readZipEntry(file)
			if err != nil {
				return nil, err
			}
			workbookRels, err = parseWorkbookRelationships(content)
			if err != nil {
				return nil, err
			}
		case "xl/workbook.xml":
			content, err := readZipEntry(file)
			if err != nil {
				return nil, err
			}
			workbookSheets, err = parseWorkbookSheets(content)
			if err != nil {
				return nil, err
			}
		}
	}

	skipped := map[string]bool{}
	for relID, sheetName := range workbookSheets {
		target, ok := workbookRels[relID]
		if !ok {
			continue
		}
		skipped[normalizeWorkbookRelationshipTarget(target)] = opts.CheckSkip(sheetName)
	}
	return skipped, nil
}

func parseWorkbookRelationships(content []byte) (map[string]string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(content))
	result := map[string]string{}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return result, nil
		}
		if err != nil {
			return nil, err
		}
		elem, ok := token.(xml.StartElement)
		if !ok || elem.Name.Local != "Relationship" {
			continue
		}
		var id, target string
		for _, attr := range elem.Attr {
			switch attr.Name.Local {
			case "Id":
				id = attr.Value
			case "Target":
				target = attr.Value
			}
		}
		if id != "" && target != "" {
			result[id] = target
		}
	}
}

func parseWorkbookSheets(content []byte) (map[string]string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(content))
	result := map[string]string{}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return result, nil
		}
		if err != nil {
			return nil, err
		}
		elem, ok := token.(xml.StartElement)
		if !ok || elem.Name.Local != "sheet" {
			continue
		}
		var name, relID string
		for _, attr := range elem.Attr {
			switch attr.Name.Local {
			case "name":
				name = attr.Value
			case "id":
				relID = attr.Value
			}
		}
		if relID != "" {
			result[relID] = name
		}
	}
}

func normalizeWorkbookRelationshipTarget(target string) string {
	target = strings.ReplaceAll(target, "\\", "/")
	if strings.HasPrefix(target, "/") {
		target = strings.TrimPrefix(target, "/")
		return path.Clean(target)
	}
	return path.Clean("xl/" + target)
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

		header, rawContent, err := rawZipHeaderAndContent(file, modified)
		if err != nil {
			_ = writer.Close()
			return nil, err
		}
		entryWriter, err := writer.CreateRaw(header)
		if err != nil {
			_ = writer.Close()
			return nil, err
		}
		if _, err := entryWriter.Write(rawContent); err != nil {
			_ = writer.Close()
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func rawZipHeaderAndContent(file *zip.File, content []byte) (*zip.FileHeader, []byte, error) {
	header := file.FileHeader
	rawContent := content
	if header.Method == zip.Deflate {
		var compressed bytes.Buffer
		deflater, err := flate.NewWriter(&compressed, flate.BestCompression)
		if err != nil {
			return nil, nil, err
		}
		if _, err := deflater.Write(content); err != nil {
			_ = deflater.Close()
			return nil, nil, err
		}
		if err := deflater.Close(); err != nil {
			return nil, nil, err
		}
		rawContent = compressed.Bytes()
	}

	header.Flags &^= 0x8
	header.CRC32 = crc32.ChecksumIEEE(content)
	header.CompressedSize = uint32(len(rawContent))
	header.UncompressedSize = uint32(len(content))
	header.CompressedSize64 = uint64(len(rawContent))
	header.UncompressedSize64 = uint64(len(content))
	return &header, rawContent, nil
}
