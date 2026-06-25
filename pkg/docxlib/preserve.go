package docxlib

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
)

// ReplaceAllBytesPreservePackage replaces text directly inside the DOCX OOXML
// package and returns the updated package bytes. If no replacement is made, the
// original bytes are returned unchanged.
func ReplaceAllBytesPreservePackage(data []byte, rules []ReplacementRule, opts ReplaceOptions) ([]byte, ReplaceResult, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, ReplaceResult{}, err
	}

	modifiedFiles := make(map[string][]byte)
	result := ReplaceResult{}
	for _, file := range reader.File {
		kind, ok := docxReplaceableXMLKind(file.Name, opts)
		if !ok {
			continue
		}

		content, err := readZipEntry(file)
		if err != nil {
			return nil, ReplaceResult{}, err
		}
		modified, count := replaceXMLTextContent(content, rules)
		if count == 0 {
			continue
		}

		modifiedFiles[file.Name] = modified
		result.TotalReplacements += count
		switch kind {
		case "body":
			result.ParagraphsProcessed++
		case "header":
			result.HeadersProcessed++
		case "footer":
			result.FootersProcessed++
		}
	}

	if result.TotalReplacements == 0 {
		return append([]byte(nil), data...), result, nil
	}

	output, err := rewriteZipPackage(reader.File, modifiedFiles)
	if err != nil {
		return nil, ReplaceResult{}, err
	}
	return output, result, nil
}

func docxReplaceableXMLKind(name string, opts ReplaceOptions) (string, bool) {
	if name == "word/document.xml" {
		return "body", true
	}
	if strings.HasPrefix(name, "word/header") && strings.HasSuffix(name, ".xml") {
		return "header", !opts.SkipHeaders
	}
	if strings.HasPrefix(name, "word/footer") && strings.HasSuffix(name, ".xml") {
		return "footer", !opts.SkipFooters
	}
	return "", false
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
