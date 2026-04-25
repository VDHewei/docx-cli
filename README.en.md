<p align="center">
  <img src="assets/logo.svg" alt="docx-cli logo" width="120" height="120">
</p>

<h1 align="center">docx-cli</h1>

<p align="center">
  <a href="README.zh-CN.md">简体中文</a> |
  <a href="README.zh-TW.md">繁體中文</a> |
  <a href="README.en.md">English</a>
</p>

---

`docx-cli` is a Go-based command-line tool for find-and-replace, text extraction, and TypeScript code generation from DOCX documents.

### Features

- **Find & Replace**: Replace text in body paragraphs, table cells, headers, and footers
- **Concurrent Processing**: Multi-worker concurrent replacement for better performance
- **Text Extraction**: Extract all text content with location information
- **Style Preservation**: Retains original text styles (font, bold, italic, color, size, etc.)
- **TypeScript Conversion**: Convert DOCX documents to TypeScript source code for [docx.js](https://www.npmjs.com/package/docx)
- **Image Support**: Extract embedded images and embed them as Base64 in generated TypeScript

### Installation

```bash
go install github.com/VDHewei/docx-cli/cmd@latest
```

Or build from source:

```bash
git clone https://github.com/VDHewei/docx-cli.git
cd docx-cli
go build -o docx-find-replace.exe ./cmd
```

### Usage

#### Find & Replace

```bash
# Basic replacement
docx-find-replace -i input.docx -r "Company A=Company B"

# Multiple rules
docx-find-replace -i input.docx -r "Hello=Hi" -r "World=Earth"

# JSON rule file
docx-find-replace -i input.docx -f replacements.json

# Skip headers/footers
docx-find-replace -i input.docx -r "old=new" --no-headers --no-footers

# Specify worker count
docx-find-replace -i input.docx -r "old=new" --workers 8
```

#### Extract Text

```bash
docx-find-replace -i input.docx --extract
```

#### Convert to TypeScript

```bash
# Default output: docx_template.ts
docx-find-replace -i input.docx --to-ts

# Custom output file
docx-find-replace -i input.docx --to-ts -o my_template.ts
```

The generated TypeScript file requires the [docx](https://www.npmjs.com/package/docx) npm package:

```bash
npm install docx
# or
bun add docx
```

### Project Structure

```
docx-cli/
├── cmd/              # CLI entrypoint
│   ├── main.go
│   └── main_test.go
├── pkg/docxlib/      # Core library
│   ├── types.go      # Common types
│   ├── extract.go    # Text extraction
│   ├── replace.go    # Replacement logic
│   ├── to_ts.go      # TypeScript conversion
│   └── unsafe.go     # Unsafe reflection helpers
├── assets/           # Static assets
│   └── logo.svg
├── tests/            # Test files
│   └── template_RISC.docx
├── go.mod
├── README.zh-CN.md
├── README.zh-TW.md
├── README.en.md
└── LICENSE
```

### Testing

```bash
go test ./...
```

### Acknowledgements

This project uses the following open-source projects:

- [gomutex/godocx](https://github.com/gomutex/godocx) — Go DOCX parsing library, [MIT License](https://opensource.org/licenses/MIT)
- [docx](https://github.com/dolanmiu/docx) (docx.js) — TypeScript DOCX generation library, [MIT License](https://opensource.org/licenses/MIT)

### License

This project is licensed under the [Apache License 2.0](LICENSE).
