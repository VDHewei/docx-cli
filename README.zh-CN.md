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
├── assets/           # 静态资源
│   └── logo.svg
├── tests/            # 测试文件
│   └── template_RISC.docx
├── go.mod
├── README.zh-CN.md
├── README.zh-TW.md
├── README.en.md
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
