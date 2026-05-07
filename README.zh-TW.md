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

`docx-cli` 是一個基於 Go 的命令列工具，用於對 DOCX 文件和 XLSX 電子表格進行查找替換、文字提取，以及將 DOCX 轉換為 TypeScript 程式碼。

### 功能特性

- **全文查找替換**：支援正文段落、表格儲存格、頁首、頁尾中的文字替換（DOCX）；支援多工作表儲存格文字替換，保持原儲存格樣式、字體、寬高（XLSX）
- **並發處理**：支援多 Worker 並發替換，提升處理速度
- **文字提取**：提取 DOCX/XLSX 中所有文字內容並顯示位置資訊
- **樣式保留**：DOCX 替換時保留原始文字樣式（字體、粗體、斜體、顏色、大小等）；XLSX 替換時保留儲存格樣式（字體、對齊、邊框、填充、數字格式、欄寬、行高）
- **TypeScript 轉換**：將 DOCX 文件轉換為 `docx.js` 可用的 TypeScript 原始碼
- **圖片支援**：提取內嵌圖片並以 Base64 形式嵌入生成的 TypeScript
- **國際化 (i18n)**：內建簡體中文 (zh)、繁體中文 (zh-TW)、英文 (en) 支援；自動偵測系統語言；支援透過 TOML 檔案擴充自訂語言

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
# DOCX 基本替換
docx-find-replace -i input.docx -r "Company A=Company B"

# DOCX 多條替換規則
docx-find-replace -i input.docx -r "Hello=Hi" -r "World=Earth"

# XLSX 基本替換（保持原儲存格樣式、字體、寬高）
docx-find-replace -i input.xlsx -r "Company A=Company B"

# XLSX 多條替換規則
docx-find-replace -i input.xlsx -r "Hello=Hi" -r "World=Earth"

# 使用 JSON 規則文件
docx-find-replace -i input.docx -f replacements.json
docx-find-replace -i input.xlsx -f replacements.json

# 跳過頁首頁尾（僅 DOCX）
docx-find-replace -i input.docx -r "old=new" --no-headers --no-footers

# 跳過指定工作表（僅 XLSX，支援模式比對）
docx-find-replace -i input.xlsx -r "old=new" --skip-sheets "Sheet1,Sheet2"  # 精確比對（逗號分隔）
docx-find-replace -i input.xlsx -r "old=new" --skip-sheets "!Summary"       # 跳過名稱不包含 "Summary" 的工作表
docx-find-replace -i input.xlsx -r "old=new" --skip-sheets "*.Data"         # 跳過名稱以 ".Data" 結尾的工作表
docx-find-replace -i input.xlsx -r "old=new" --skip-sheets "@regexp:^Cfg"   # 跳過名稱符合正則表示式的工作表

# 指定並發 Worker 數
docx-find-replace -i input.docx -r "old=new" --workers 8
docx-find-replace -i input.xlsx -r "old=new" --workers 8
```

#### 國際化 (i18n)

CLI 自動偵測系統語言並顯示對應語言的輸出（zh、zh-TW 或 en）。不支援的語言預設回退為英文。

```bash
# 透過環境變數指定語言
LANG=zh docx-find-replace -i input.docx --extract       # 簡體中文
LANG=zh-TW docx-find-replace -i input.xlsx --extract    # 繁體中文
LANG=en docx-find-replace -i input.docx -r "old=new"    # 英文
```

可透過自訂 TOML 檔案擴充或覆蓋翻譯：

```bash
# 生成包含所有可翻譯鍵的範例 TOML 檔案
docx-find-replace --lang-settings-file custom_ja.toml

# 使用自訂 TOML 檔案執行
docx-find-replace -i input.docx -r "old=new" --lang-settings-file custom_ja.toml
```

自訂 TOML 檔案格式透過 `[meta]` 宣告目標語言：

```toml
[meta]
language = "ja"    # BCP 47 語言標籤

[ErrInputRequired]
other = "エラー: 入力ファイルを指定してください (-i または --input)"

[FlagInputShort]
other = "入力ファイルパス (.docx または .xlsx)"
```

#### 提取文字

```bash
# DOCX
docx-find-replace -i input.docx --extract

# XLSX
docx-find-replace -i input.xlsx --extract
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
├── pkg/docxlib/      # DOCX 核心庫
│   ├── types.go      # 公共類型
│   ├── extract.go    # 文字提取
│   ├── replace.go    # 替換邏輯
│   ├── to_ts.go      # TypeScript 轉換
│   └── unsafe.go     # unsafe 反射輔助
├── pkg/xlsxlib/      # XLSX 核心庫
│   ├── types.go      # 公共類型
│   ├── extract.go    # 文字提取
│   ├── replace.go    # 替換邏輯
│   └── xlsxlib_test.go
├── pkg/i18n/         # 國際化
│   ├── i18n.go       # 核心: Bundle, Localizer, T()
│   ├── keys.go       # 訊息鍵常量
│   ├── embed.go      # 嵌入語言檔案
│   ├── gen.go        # 範例 TOML 生成器
│   └── locales/      # 內建翻譯 (zh, zh-TW, en)
├── assets/           # 靜態資源
│   └── logo.svg
├── tests/            # 測試文件
│   └── template_RISC.docx
├── go.mod
├── README.zh-CN.md
├── README.zh-TW.md
├── README.en.md
└── LICENSE
```

### 測試

```bash
go test ./...
```

### 致謝

本專案使用了以下開源專案：

- [gomutex/godocx](https://github.com/gomutex/godocx) — Go DOCX 解析庫，[MIT License](https://opensource.org/licenses/MIT)
- [xuri/excelize](https://github.com/xuri/excelize) — Go XLSX 解析庫，[BSD-3 License](https://opensource.org/licenses/BSD-3-Clause)
- [nicksnyder/go-i18n](https://github.com/nicksnyder/go-i18n) — Go 國際化庫，[MIT License](https://opensource.org/licenses/MIT)
- [docx](https://github.com/dolanmiu/docx) (docx.js) — TypeScript DOCX 生成庫，[MIT License](https://opensource.org/licenses/MIT)

### 開源協議

本專案採用 [Apache License 2.0](LICENSE)。
