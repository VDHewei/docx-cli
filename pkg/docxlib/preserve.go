package docxlib

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"hash/crc32"
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
