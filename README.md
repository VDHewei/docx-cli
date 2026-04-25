# docx-cli

[简体中文](#简体中文) | [繁體中文](#繁體中文) | [English](#english)

---

## 简体中文

`docx-cli` 是一个基于 Go 的命令行工具，用于对 DOCX 文档进行查找替换、文本提取以及转换为 TypeScript 代码。

### 功能特性

- **全文查找替换**：支持正文段落、表格单元格、页眉、页脚中的文本替换
- **并发处理**：支持多 Worker 并发替换，提升处理速度
- **文本提取**：提取 DOCX 中所有文本内容并显示位置信息
- **样式保留**：替换时保留原始文本样式（字体、粗体、斜体、颜色、大小等）
- **TypeScript 转换**：将 DOCX 文档转换为 `docx.js` 可用的 TypeScript 源代码
- **图片支持**：提取内嵌图片并以 Base64 形式嵌入生成的 TypeScript

### 安装

```bash
go install github.com/VDHewei/docx-cli/cmd@latest
```

或从源码构建：

```bash
git clone https://github.com/VDHewei/docx-cli.git
cd docx-cli
go build -o docx-find-replace.exe ./cmd
```

### 用法

#### 查找替换

```bash
# 基本替换
docx-find-replace -i input.docx -r "Company A=Company B"

# 多条替换规则
docx-find-replace -i input.docx -r "Hello=Hi" -r "World=Earth"

# 使用 JSON 规则文件
docx-find-replace -i input.docx -f replacements.json

# 跳过页眉页脚
docx-find-replace -i input.docx -r "old=new" --no-headers --no-footers

# 指定并发 Worker 数
docx-find-replace -i input.docx -r "old=new" --workers 8
```

#### 提取文本

```bash
docx-find-replace -i input.docx --extract
```

#### 转换为 TypeScript

```bash
# 默认输出 docx_template.ts
docx-find-replace -i input.docx --to-ts

# 指定输出文件名
docx-find-replace -i input.docx --to-ts -o my_template.ts
```

生成的 TypeScript 文件依赖 [docx](https://www.npmjs.com/package/docx) npm 包：

```bash
npm install docx
# 或
bun add docx
```

### 项目结构

```
docx-cli/
├── cmd/              # CLI 入口
│   ├── main.go
│   └── main_test.go
├── pkg/docxlib/      # 核心库
│   ├── types.go      # 公共类型
│   ├── extract.go    # 文本提取
│   ├── replace.go    # 替换逻辑
│   ├── to_ts.go      # TypeScript 转换
│   └── unsafe.go     # unsafe 反射辅助
├── tests/            # 测试文件
│   └── template_RISC.docx
├── go.mod
├── README.md
└── LICENSE
```

### 测试

```bash
go test ./...
```

### 致谢

本项目使用了以下开源项目：

- [gomutex/godocx](https://github.com/gomutex/godocx) — Go DOCX 解析库，[MIT License](https://opensource.org/licenses/MIT)
- [docx](https://github.com/dolanmiu/docx) (docx.js) — TypeScript DOCX 生成库，[MIT License](https://opensource.org/licenses/MIT)

### 开源协议

本项目采用 [Apache License 2.0](LICENSE)。

---

## 繁體中文

`docx-cli` 是一個基於 Go 的命令列工具，用於對 DOCX 文件進行查找替換、文字提取以及轉換為 TypeScript 程式碼。

### 功能特性

- **全文查找替換**：支援正文段落、表格儲存格、頁首、頁尾中的文字替換
- **並發處理**：支援多 Worker 並發替換，提升處理速度
- **文字提取**：提取 DOCX 中所有文字內容並顯示位置資訊
- **樣式保留**：替換時保留原始文字樣式（字體、粗體、斜體、顏色、大小等）
- **TypeScript 轉換**：將 DOCX 文件轉換為 `docx.js` 可用的 TypeScript 原始碼
- **圖片支援**：提取內嵌圖片並以 Base64 形式嵌入生成的 TypeScript

### 安裝

```bash
go install github.com/VDHewei/docx-cli/cmd@latest
```

或從源碼建置：

```bash
git clone https://github.com/VDHewei/docx-cli.git
cd docx-cli
go build -o docx-find-replace.exe ./cmd
```

### 用法

#### 查找替換

```bash
# 基本替換
docx-find-replace -i input.docx -r "Company A=Company B"

# 多條替換規則
docx-find-replace -i input.docx -r "Hello=Hi" -r "World=Earth"

# 使用 JSON 規則文件
docx-find-replace -i input.docx -f replacements.json

# 跳過頁首頁尾
docx-find-replace -i input.docx -r "old=new" --no-headers --no-footers

# 指定並發 Worker 數
docx-find-replace -i input.docx -r "old=new" --workers 8
```

#### 提取文字

```bash
docx-find-replace -i input.docx --extract
```

#### 轉換為 TypeScript

```bash
# 預設輸出 docx_template.ts
docx-find-replace -i input.docx --to-ts

# 指定輸出文件名稱
docx-find-replace -i input.docx --to-ts -o my_template.ts
```

生成的 TypeScript 文件依賴 [docx](https://www.npmjs.com/package/docx) npm 套件：

```bash
npm install docx
# 或
bun add docx
```

### 專案結構

```
docx-cli/
├── cmd/              # CLI 入口
│   ├── main.go
│   └── main_test.go
├── pkg/docxlib/      # 核心庫
│   ├── types.go      # 公共類型
│   ├── extract.go    # 文字提取
│   ├── replace.go    # 替換邏輯
│   ├── to_ts.go      # TypeScript 轉換
│   └── unsafe.go     # unsafe 反射輔助
├── tests/            # 測試文件
│   └── template_RISC.docx
├── go.mod
├── README.md
└── LICENSE
```

### 測試

```bash
go test ./...
```

### 開源協議

本專案採用 [Apache License 2.0](LICENSE)。

---

## English

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
├── tests/            # Test files
│   └── template_RISC.docx
├── go.mod
├── README.md
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
