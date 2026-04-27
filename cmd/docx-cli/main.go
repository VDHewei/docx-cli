package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/VDHewei/docx-cli/pkg/docxlib"
	"github.com/gomutex/godocx"
)

// Config 命令行配置
type Config struct {
	InputFile    string
	OutputFile   string
	Replacements []docxlib.ReplacementRule
	NoHeaders    bool
	NoFooters    bool
	Verbose      bool
	ExtractOnly  bool
	ToTS         bool
	Workers      int
	Help         bool
	Version      bool
}

var Version = "v0.1.6"

func main() {
	cfg := parseFlags()

	if cfg.Version {
		printVersion()
		return
	}
	if cfg.Help {
		printHelp()
		return
	}

	if cfg.InputFile == "" {
		log.Fatal("错误: 必须指定输入文件 (-i 或 --input)")
	}

	if _, err := os.Stat(cfg.InputFile); os.IsNotExist(err) {
		log.Fatalf("输入文件不存在: %s", cfg.InputFile)
	}

	rootDoc, err := godocx.OpenDocument(cfg.InputFile)
	if err != nil {
		log.Fatalf("无法打开文档: %v", err)
	}
	if rootDoc.Document == nil {
		log.Fatal("文档结构无效")
	}

	if cfg.Verbose {
		fmt.Printf("输入文件: %s\n", cfg.InputFile)
		if cfg.ExtractOnly {
			fmt.Println("模式: 仅提取文本")
		} else if cfg.ToTS {
			fmt.Println("模式: 转换为 TypeScript")
		} else {
			fmt.Printf("替换规则数量: %d\n", len(cfg.Replacements))
			fmt.Printf("并发 workers: %d\n", cfg.Workers)
		}
		if cfg.NoHeaders {
			fmt.Println("跳过页眉: 是")
		}
		if cfg.NoFooters {
			fmt.Println("跳过页脚: 是")
		}
		fmt.Println()
	}

	// 1. 仅提取文本
	if cfg.ExtractOnly {
		allTexts := docxlib.ExtractAllText(rootDoc)
		fmt.Printf("共提取到 %d 段文本:\n\n", len(allTexts))
		for _, dt := range allTexts {
			loc := dt.Location
			locInfo := loc.Kind
			if loc.Kind == "table" {
				locInfo = fmt.Sprintf("table[%d,%d,%d]", loc.TableIdx, loc.RowIdx, loc.CellIdx)
			} else if loc.Kind == "body" {
				locInfo = fmt.Sprintf("body[%d]", loc.ParaIndex)
			}
			fmt.Printf("  [%s] %s\n", locInfo, dt.Text)
		}
		return
	}

	// 2. 转换为 TypeScript
	if cfg.ToTS {
		tsCode := docxlib.ToTypeScript(rootDoc, "")
		outFile := cfg.OutputFile
		if outFile == "" {
			outFile = "docx_template.ts"
		}
		if err := os.WriteFile(outFile, []byte(tsCode), 0644); err != nil {
			log.Fatalf("写入 TS 文件失败: %v", err)
		}
		fmt.Printf("TypeScript 文件已生成: %s\n", outFile)
		return
	}

	if len(cfg.Replacements) == 0 {
		log.Fatal("错误: 必须至少指定一个替换规则 (-r 或 -f)，或使用 --extract 提取文本")
	}

	// 3. 执行并发替换
	allTexts := docxlib.ExtractAllText(rootDoc)
	result := docxlib.ReplaceAll(rootDoc, cfg.Replacements, docxlib.ReplaceOptions{
		SkipHeaders: cfg.NoHeaders,
		SkipFooters: cfg.NoFooters,
		Workers:     cfg.Workers,
	})

	if cfg.Verbose {
		fmt.Printf("替换完成: 总计 %d 次替换\n", result.TotalReplacements)
		fmt.Printf("  段落: %d\n", result.ParagraphsProcessed)
		fmt.Printf("  单元格: %d\n", result.CellsProcessed)
		fmt.Printf("  页眉: %d\n", result.HeadersProcessed)
		fmt.Printf("  页脚: %d\n", result.FootersProcessed)
	}

	// 确定输出文件名
	outputFile := cfg.OutputFile
	if outputFile == "" {
		ext := filepath.Ext(cfg.InputFile)
		base := cfg.InputFile[:len(cfg.InputFile)-len(ext)]
		outputFile = base + "_replaced" + ext
	}

	// 检查输出文件是否已存在
	if _, err := os.Stat(outputFile); err == nil {
		fmt.Printf("警告: 输出文件已存在: %s\n", outputFile)
		fmt.Print("是否覆盖? (y/N): ")
		var response string
		_, _ = fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("操作已取消")
			return
		}
	}

	// 保存文档
	outputDir := filepath.Dir(outputFile)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Fatalf("无法创建输出目录: %v", err)
		}
	}

	if err := rootDoc.SaveTo(outputFile); err != nil {
		log.Fatalf("保存文档失败: %v", err)
	}

	fmt.Printf("\n成功处理并保存文档！\n")
	fmt.Printf("  提取文本段数: %d\n", len(allTexts))
	fmt.Printf("  总共进行了 %d 次替换\n", result.TotalReplacements)
	fmt.Printf("  处理了 %d 个段落\n", result.ParagraphsProcessed)
	if result.CellsProcessed > 0 {
		fmt.Printf("  处理了 %d 个单元格\n", result.CellsProcessed)
	}
	if result.HeadersProcessed > 0 {
		fmt.Printf("  处理了 %d 个页眉\n", result.HeadersProcessed)
	}
	if result.FootersProcessed > 0 {
		fmt.Printf("  处理了 %d 个页脚\n", result.FootersProcessed)
	}
	fmt.Printf("  输出文件: %s\n", outputFile)
}

// parseFlags 解析命令行参数
func parseFlags() *Config {
	cfg := &Config{Workers: 0} // 0 表示使用 runtime.NumCPU()

	flagSet := flag.NewFlagSet("docx-find-replace", flag.ContinueOnError)

	input := flagSet.String("i", "", "输入 DOCX 文件路径")
	inputLong := flagSet.String("input", "", "输入 DOCX 文件路径")
	output := flagSet.String("o", "", "输出文件路径 (DOCX 默认: <input>_replaced.docx, TS 默认: docx_template.ts)")
	outputLong := flagSet.String("output", "", "输出文件路径")
	replace := flagSet.String("r", "", "替换文本 old=new (可多次使用)")
	replaceLong := flagSet.String("replace", "", "替换文本 old=value (可多次使用)")
	replaceFile := flagSet.String("f", "", "JSON 替换规则文件")
	replaceFileLong := flagSet.String("replace-file", "", "JSON 替换规则文件")
	noHeaders := flagSet.Bool("no-headers", false, "跳过页眉部分")
	noFooters := flagSet.Bool("no-footers", false, "跳过页脚部分")
	verbose := flagSet.Bool("v", false, "显示详细处理信息")
	version := flagSet.Bool("V", false, "显示版本号")
	versionLong := flagSet.Bool("version", false, "显示版本号")
	verboseLong := flagSet.Bool("verbose", false, "显示详细处理信息")
	extract := flagSet.Bool("extract", false, "仅提取文档中的所有文本")
	toTS := flagSet.Bool("to-ts", false, "将 DOCX 转换为 TypeScript 源码")
	workers := flagSet.Int("workers", 0, "并发 worker 数量 (默认: CPU 核心数)")
	help := flagSet.Bool("h", false, "显示帮助信息")
	helpLong := flagSet.Bool("help", false, "显示帮助信息")

	_ = flagSet.Parse(os.Args[1:])
	flagSet.Usage = printHelp
	// 输入/输出
	if *input != "" {
		cfg.InputFile = *input
	} else if *inputLong != "" {
		cfg.InputFile = *inputLong
	}
	if *output != "" {
		cfg.OutputFile = *output
	} else if *outputLong != "" {
		cfg.OutputFile = *outputLong
	}

	// 替换规则
	for _, repStr := range []string{*replace, *replaceLong} {
		if repStr != "" {
			if rep := parseReplacement(repStr); rep != nil {
				cfg.Replacements = append(cfg.Replacements, *rep)
			}
		}
	}

	// 解析位置参数中的替换选项
	args := flagSet.Args()
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-r=") {
			if rep := parseReplacement(strings.TrimPrefix(arg, "-r=")); rep != nil {
				cfg.Replacements = append(cfg.Replacements, *rep)
			}
		} else if strings.HasPrefix(arg, "--replace=") {
			if rep := parseReplacement(strings.TrimPrefix(arg, "--replace=")); rep != nil {
				cfg.Replacements = append(cfg.Replacements, *rep)
			}
		} else if arg == "-r" || arg == "--replace" {
			if i+1 < len(args) {
				i++
				if rep := parseReplacement(args[i]); rep != nil {
					cfg.Replacements = append(cfg.Replacements, *rep)
				}
			}
		}
	}

	// 从文件加载替换规则
	for _, f := range []string{*replaceFile, *replaceFileLong} {
		if f != "" {
			cfg.Replacements = append(cfg.Replacements, loadReplacementsFromFile(f)...)
		}
	}

	cfg.NoHeaders = *noHeaders
	cfg.NoFooters = *noFooters
	cfg.Verbose = *verbose || *verboseLong
	cfg.ExtractOnly = *extract
	cfg.ToTS = *toTS
	cfg.Workers = *workers
	cfg.Help = *help || *helpLong
	cfg.Version = *version || *versionLong
	if cfg.Help {
		printHelp()
		os.Exit(0)
	}

	return cfg
}

func parseReplacement(repStr string) *docxlib.ReplacementRule {
	parts := strings.SplitN(repStr, "=", 2)
	if len(parts) != 2 {
		log.Printf("警告: 无效的替换格式 '%s'，忽略", repStr)
		return nil
	}
	return &docxlib.ReplacementRule{Old: parts[0], New: parts[1]}
}

func loadReplacementsFromFile(filename string) []docxlib.ReplacementRule {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("无法打开替换文件: %v", err)
	}
	defer file.Close()

	var replacements []docxlib.ReplacementRule
	if err := json.NewDecoder(file).Decode(&replacements); err != nil {
		log.Fatalf("无法解析替换文件: %v", err)
	}
	return replacements
}

func printHelp() {
	helpText := `docx-find-replace (%s)- 在 DOCX 文件中查找和替换文本

用法:
  docx-find-replace -i <input> [选项]

选项:
  -i, --input <file>         输入 DOCX 文件路径
  -o, --output <file>        输出文件路径 (DOCX 默认: <input>_replaced.docx, TS 默认: docx_template.ts)
  -r, --replace <old=new>    替换文本 (可多次使用)
  -f, --replace-file <file>  JSON 文件，包含替换规则
      --extract              仅提取文档中的所有文本，不执行替换
      --to-ts                将 DOCX 转换为 TypeScript 源码
      --no-headers           跳过页眉部分
      --no-footers           跳过页脚部分
      --workers <n>          并发 worker 数量 (默认: CPU 核心数)
  -v, --verbose              显示详细处理信息
  -h, --help                 显示此帮助信息

示例:
  # 提取文档中所有文本
  docx-find-replace -i input.docx --extract

  # 替换文本
  docx-find-replace -i input.docx -r "Company A=Company B"

  # 转换为 TypeScript
  docx-find-replace -i input.docx --to-ts -o output.ts

替换文件格式 (JSON):
  [
    {"old": "要查找的文本", "new": "替换后的文本"}
  ]
`
	fmt.Print(fmt.Sprintf(helpText, Version))
}

func printVersion() {
	fmt.Println(Version)
}
