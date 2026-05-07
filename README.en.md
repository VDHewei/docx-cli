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

`docx-cli` is a Go-based command-line tool for find-and-replace, text extraction, and TypeScript code generation from DOCX documents and XLSX spreadsheets.

### Features

- **Find & Replace**: Replace text in body paragraphs, table cells, headers, and footers (DOCX); replace text in cells across sheets while preserving styles (XLSX)
- **Concurrent Processing**: Multi-worker concurrent replacement for better performance
- **Text Extraction**: Extract all text content with location information
- **Style Preservation**: Retains original text styles (font, bold, italic, color, size, etc.) in DOCX; preserve cell styles (font, alignment, borders, fill, number format, width, height) in XLSX
- **TypeScript Conversion**: Convert DOCX documents to TypeScript source code for [docx.js](https://www.npmjs.com/package/docx)
- **Image Support**: Extract embedded images and embed them as Base64 in generated TypeScript
- **i18n**: Built-in support for Chinese (zh), Traditional Chinese (zh-TW), and English (en); auto-detects system locale; supports custom language extensions via TOML files

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
# DOCX basic replacement
docx-find-replace -i input.docx -r "Company A=Company B"

# DOCX multiple rules
docx-find-replace -i input.docx -r "Hello=Hi" -r "World=Earth"

# XLSX basic replacement (preserves cell styles, font, width, height)
docx-find-replace -i input.xlsx -r "Company A=Company B"

# XLSX multiple rules
docx-find-replace -i input.xlsx -r "Hello=Hi" -r "World=Earth"

# JSON rule file
docx-find-replace -i input.docx -f replacements.json
docx-find-replace -i input.xlsx -f replacements.json

# Skip headers/footers (DOCX only)
docx-find-replace -i input.docx -r "old=new" --no-headers --no-footers

# Skip specific sheets (XLSX only, supports pattern matching)
docx-find-replace -i input.xlsx -r "old=new" --skip-sheets "Sheet1,Sheet2"  # exact match (comma-separated)
docx-find-replace -i input.xlsx -r "old=new" --skip-sheets "!Summary"       # skip sheets NOT containing "Summary"
docx-find-replace -i input.xlsx -r "old=new" --skip-sheets "*.Data"         # skip sheets ending with ".Data"
docx-find-replace -i input.xlsx -r "old=new" --skip-sheets "@regexp:^Cfg"   # skip sheets matching regex

# Specify worker count
docx-find-replace -i input.docx -r "old=new" --workers 8
docx-find-replace -i input.xlsx -r "old=new" --workers 8
```

#### Internationalization (i18n)

The CLI auto-detects your system locale and displays output in the matching language (zh, zh-TW, or en). Unsupported locales fall back to English.

```bash
# Force a specific locale via environment variable
LANG=zh docx-find-replace -i input.docx --extract       # Simplified Chinese
LANG=zh-TW docx-find-replace -i input.xlsx --extract    # Traditional Chinese
LANG=en docx-find-replace -i input.docx -r "old=new"    # English
```

You can extend or override translations with a custom TOML file:

```bash
# Generate an example TOML with all translatable keys
docx-find-replace --lang-settings-file custom_ja.toml

# Use a custom TOML for another run
docx-find-replace -i input.docx -r "old=new" --lang-settings-file custom_ja.toml
```

The custom TOML file format declares the language inside `[meta]`:

```toml
[meta]
language = "ja"    # BCP 47 language tag

[ErrInputRequired]
other = "エラー: 入力ファイルを指定してください (-i または --input)"

[FlagInputShort]
other = "入力ファイルパス (.docx または .xlsx)"
```

#### Extract Text

```bash
# DOCX
docx-find-replace -i input.docx --extract

# XLSX
docx-find-replace -i input.xlsx --extract
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
├── pkg/docxlib/      # DOCX core library
│   ├── types.go      # Common types
│   ├── extract.go    # Text extraction
│   ├── replace.go    # Replacement logic
│   ├── to_ts.go      # TypeScript conversion
│   └── unsafe.go     # Unsafe reflection helpers
├── pkg/xlsxlib/      # XLSX core library
│   ├── types.go      # Common types
│   ├── extract.go    # Text extraction
│   ├── replace.go    # Replacement logic
│   └── xlsxlib_test.go
├── pkg/i18n/         # Internationalization
│   ├── i18n.go       # Core: Bundle, Localizer, T()
│   ├── keys.go       # Message key constants
│   ├── embed.go      # Embedded locale files
│   ├── gen.go        # Example TOML generator
│   └── locales/      # Built-in translations (zh, zh-TW, en)
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
- [xuri/excelize](https://github.com/xuri/excelize) — Go XLSX parsing library, [BSD-3 License](https://opensource.org/licenses/BSD-3-Clause)
- [nicksnyder/go-i18n](https://github.com/nicksnyder/go-i18n) — Go i18n library, [MIT License](https://opensource.org/licenses/MIT)
- [docx](https://github.com/dolanmiu/docx) (docx.js) — TypeScript DOCX generation library, [MIT License](https://opensource.org/licenses/MIT)

### License

This project is licensed under the [Apache License 2.0](LICENSE).
