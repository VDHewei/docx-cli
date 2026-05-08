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
	"github.com/VDHewei/docx-cli/pkg/i18n"
	"github.com/VDHewei/docx-cli/pkg/xlsxlib"
	"github.com/gomutex/godocx"
	"github.com/xuri/excelize/v2"
)

// Config 命令行配置
type Config struct {
	InputFile        string
	OutputFile       string
	Replacements     []ReplacementRule
	NoHeaders        bool
	NoFooters        bool
	SkipSheets       []string
	Verbose          bool
	ExtractOnly      bool
	ToTS             bool
	Workers          int
	LangSettingsFile string
	Help             bool
	Version          bool
}

// ReplacementRule is a unified replacement rule used by the CLI layer.
type ReplacementRule struct {
	Old string `json:"old"`
	New string `json:"new"`
}

var Version = "v0.2.3"

// fileType returns the file type based on extension.
func fileType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".docx":
		return "docx"
	case ".xlsx":
		return "xlsx"
	default:
		return ""
	}
}

func main() {
	cfg := parseFlags()

	// Initialize i18n (must happen before any T() calls)
	generated, err := i18n.Init(cfg.LangSettingsFile)
	if err != nil {
		log.Fatal(i18n.T(i18n.ErrI18nInit, map[string]interface{}{"Error": err.Error()}))
	}
	if generated {
		fmt.Println(i18n.T(i18n.InfoLangFileGenerated, map[string]interface{}{"File": cfg.LangSettingsFile}))
		return
	}

	if cfg.Version {
		printVersion()
		return
	}
	if cfg.Help {
		printHelp()
		return
	}

	if cfg.InputFile == "" {
		log.Fatal(i18n.T(i18n.ErrInputRequired))
	}

	if _, err := os.Stat(cfg.InputFile); err != nil {
		log.Fatal(i18n.T(i18n.ErrFileNotFound, map[string]interface{}{"File": cfg.InputFile}))
	}

	ft := fileType(cfg.InputFile)
	if ft == "" {
		log.Fatal(i18n.T(i18n.ErrUnsupportedFileType))
	}

	switch ft {
	case "docx":
		processDocx(cfg)
	case "xlsx":
		processXlsx(cfg)
	}
}

func processDocx(cfg *Config) {
	rootDoc, err := godocx.OpenDocument(cfg.InputFile)
	if err != nil {
		log.Fatal(i18n.T(i18n.ErrOpenDocument, map[string]interface{}{"Error": err.Error()}))
	}
	if rootDoc.Document == nil {
		log.Fatal(i18n.T(i18n.ErrInvalidDocStructure))
	}

	if cfg.Verbose {
		fmt.Println(i18n.T(i18n.VerboseInputFile, map[string]interface{}{"File": cfg.InputFile, "Type": "DOCX"}))
		if cfg.ExtractOnly {
			fmt.Println(i18n.T(i18n.VerboseModeExtract))
		} else if cfg.ToTS {
			fmt.Println(i18n.T(i18n.VerboseModeToTS))
		} else {
			fmt.Println(i18n.T(i18n.VerboseReplaceCount, map[string]interface{}{"Count": len(cfg.Replacements)}))
			fmt.Println(i18n.T(i18n.VerboseWorkers, map[string]interface{}{"Count": cfg.Workers}))
		}
		if cfg.NoHeaders {
			fmt.Println(i18n.T(i18n.VerboseSkipHeaders))
		}
		if cfg.NoFooters {
			fmt.Println(i18n.T(i18n.VerboseSkipFooters))
		}
		fmt.Println()
	}

	if cfg.ExtractOnly {
		allTexts := docxlib.ExtractAllText(rootDoc)
		fmt.Println(i18n.T(i18n.InfoExtractedTextCount, map[string]interface{}{"Count": len(allTexts)}))
		fmt.Println()
		for _, dt := range allTexts {
			loc := dt.Location
			locInfo := loc.Kind
			switch loc.Kind {
			case "table":
				locInfo = fmt.Sprintf("table[%d,%d,%d]", loc.TableIdx, loc.RowIdx, loc.CellIdx)
			case "body":
				locInfo = fmt.Sprintf("body[%d]", loc.ParaIndex)
			}
			fmt.Printf("  [%s] %s\n", locInfo, dt.Text)
		}
		return
	}

	if cfg.ToTS {
		tsCode := docxlib.ToTypeScript(rootDoc, "")
		outFile := cfg.OutputFile
		if outFile == "" {
			outFile = "docx_template.ts"
		}
		if err := os.WriteFile(outFile, []byte(tsCode), 0644); err != nil {
			log.Fatal(i18n.T(i18n.ErrWriteTSFile, map[string]interface{}{"Error": err.Error()}))
		}
		fmt.Println(i18n.T(i18n.InfoTSFileGenerated, map[string]interface{}{"File": outFile}))
		return
	}

	if len(cfg.Replacements) == 0 {
		log.Fatal(i18n.T(i18n.ErrNoReplacementRules))
	}

	docxRules := toDocxRules(cfg.Replacements)
	allTexts := docxlib.ExtractAllText(rootDoc)
	result := docxlib.ReplaceAll(rootDoc, docxRules, docxlib.ReplaceOptions{
		SkipHeaders: cfg.NoHeaders,
		SkipFooters: cfg.NoFooters,
		Workers:     cfg.Workers,
	})

	if cfg.Verbose {
		fmt.Println(i18n.T(i18n.VerboseReplaceDone, map[string]interface{}{"Count": result.TotalReplacements}))
		fmt.Println(i18n.T(i18n.VerboseParagraphs, map[string]interface{}{"Count": result.ParagraphsProcessed}))
		fmt.Println(i18n.T(i18n.VerboseCells, map[string]interface{}{"Count": result.CellsProcessed}))
		fmt.Println(i18n.T(i18n.VerboseHeaders, map[string]interface{}{"Count": result.HeadersProcessed}))
		fmt.Println(i18n.T(i18n.VerboseFooters, map[string]interface{}{"Count": result.FootersProcessed}))
	}

	outputFile := resolveOutputFile(cfg.InputFile, cfg.OutputFile)

	if !confirmOverwrite(outputFile) {
		return
	}

	ensureOutputDir(outputFile)

	if err := rootDoc.SaveTo(outputFile); err != nil {
		log.Fatal(i18n.T(i18n.ErrSaveDocument, map[string]interface{}{"Error": err.Error()}))
	}

	fmt.Println(i18n.T(i18n.InfoSuccessDocx))
	fmt.Println(i18n.T(i18n.InfoExtractedTextSegCount, map[string]interface{}{"Count": len(allTexts)}))
	fmt.Println(i18n.T(i18n.InfoTotalReplacements, map[string]interface{}{"Count": result.TotalReplacements}))
	fmt.Println(i18n.T(i18n.InfoProcessedParagraphs, map[string]interface{}{"Count": result.ParagraphsProcessed}))
	if result.CellsProcessed > 0 {
		fmt.Println(i18n.T(i18n.InfoProcessedCells, map[string]interface{}{"Count": result.CellsProcessed}))
	}
	if result.HeadersProcessed > 0 {
		fmt.Println(i18n.T(i18n.InfoProcessedHeaders, map[string]interface{}{"Count": result.HeadersProcessed}))
	}
	if result.FootersProcessed > 0 {
		fmt.Println(i18n.T(i18n.InfoProcessedFooters, map[string]interface{}{"Count": result.FootersProcessed}))
	}
	fmt.Println(i18n.T(i18n.InfoOutputFile, map[string]interface{}{"File": outputFile}))
}

func processXlsx(cfg *Config) {
	f, err := excelize.OpenFile(cfg.InputFile)
	if err != nil {
		log.Fatal(i18n.T(i18n.ErrOpenSpreadsheet, map[string]interface{}{"Error": err.Error()}))
	}
	defer f.Close()

	if cfg.Verbose {
		fmt.Println(i18n.T(i18n.VerboseInputFile, map[string]interface{}{"File": cfg.InputFile, "Type": "XLSX"}))
		if cfg.ExtractOnly {
			fmt.Println(i18n.T(i18n.VerboseModeExtract))
		} else {
			fmt.Println(i18n.T(i18n.VerboseReplaceCount, map[string]interface{}{"Count": len(cfg.Replacements)}))
			fmt.Println(i18n.T(i18n.VerboseWorkers, map[string]interface{}{"Count": cfg.Workers}))
		}
		if len(cfg.SkipSheets) > 0 {
			fmt.Println(i18n.T(i18n.VerboseSkipSheets, map[string]interface{}{"Sheets": strings.Join(cfg.SkipSheets, ", ")}))
		}
		fmt.Println()
	}

	if cfg.ExtractOnly {
		allTexts := xlsxlib.ExtractAllText(f)
		fmt.Println(i18n.T(i18n.InfoExtractedCellCount, map[string]interface{}{"Count": len(allTexts)}))
		fmt.Println()
		for _, ct := range allTexts {
			fmt.Printf("  [%s!%s] %s\n", ct.Location.Sheet, ct.Location.Cell, ct.Text)
		}
		return
	}

	if cfg.ToTS {
		log.Fatal(i18n.T(i18n.ErrTTSOnlyDocx))
	}

	if len(cfg.Replacements) == 0 {
		log.Fatal(i18n.T(i18n.ErrNoReplacementRules))
	}

	xlsxRules := toXlsxRules(cfg.Replacements)
	allTexts := xlsxlib.ExtractAllText(f)
	result := xlsxlib.ReplaceAll(f, xlsxRules, xlsxlib.ReplaceOptions{
		Workers:    cfg.Workers,
		SkipSheets: cfg.SkipSheets,
	})

	if cfg.Verbose {
		fmt.Println(i18n.T(i18n.VerboseReplaceDone, map[string]interface{}{"Count": result.TotalReplacements}))
		fmt.Println(i18n.T(i18n.VerboseCells, map[string]interface{}{"Count": result.CellsProcessed}))
		fmt.Println(i18n.T(i18n.VerboseSheets, map[string]interface{}{"Count": result.SheetsProcessed}))
	}

	outputFile := resolveOutputFile(cfg.InputFile, cfg.OutputFile)

	if !confirmOverwrite(outputFile) {
		return
	}

	ensureOutputDir(outputFile)

	if err := f.SaveAs(outputFile); err != nil {
		log.Fatal(i18n.T(i18n.ErrSaveSpreadsheet, map[string]interface{}{"Error": err.Error()}))
	}

	fmt.Println(i18n.T(i18n.InfoSuccessXlsx))
	fmt.Println(i18n.T(i18n.InfoExtractedCellTextCount, map[string]interface{}{"Count": len(allTexts)}))
	fmt.Println(i18n.T(i18n.InfoTotalReplacements, map[string]interface{}{"Count": result.TotalReplacements}))
	fmt.Println(i18n.T(i18n.InfoProcessedCells, map[string]interface{}{"Count": result.CellsProcessed}))
	fmt.Println(i18n.T(i18n.InfoProcessedSheets, map[string]interface{}{"Count": result.SheetsProcessed}))
	fmt.Println(i18n.T(i18n.InfoOutputFile, map[string]interface{}{"File": outputFile}))
}

func toDocxRules(rules []ReplacementRule) []docxlib.ReplacementRule {
	result := make([]docxlib.ReplacementRule, len(rules))
	for i, r := range rules {
		result[i] = docxlib.ReplacementRule{Old: r.Old, New: r.New}
	}
	return result
}

func toXlsxRules(rules []ReplacementRule) []xlsxlib.ReplacementRule {
	result := make([]xlsxlib.ReplacementRule, len(rules))
	for i, r := range rules {
		result[i] = xlsxlib.ReplacementRule{Old: r.Old, New: r.New}
	}
	return result
}

// parseFlags 解析命令行参数
func parseFlags() *Config {
	cfg := &Config{Workers: 0}

	// Pre-scan os.Args for -r/--replace values before flagSet.Parse consumes them,
	// because Go's flag package only retains the last value for repeated flags.
	cfg.Replacements = extractReplaceRules(os.Args[1:])

	flagSet := flag.NewFlagSet("docx-find-replace", flag.ContinueOnError)
	flagSet.Usage = func() {} // Suppress default usage; help is printed explicitly after i18n init

	flagSet.String("i", "", "")
	flagSet.String("input", "", "")
	flagSet.String("o", "", "")
	flagSet.String("output", "", "")
	flagSet.String("r", "", "")
	flagSet.String("replace", "", "")
	flagSet.String("f", "", "")
	flagSet.String("replace-file", "", "")
	flagSet.Bool("no-headers", false, "")
	flagSet.Bool("no-footers", false, "")
	flagSet.String("skip-sheets", "", "")
	flagSet.Bool("v", false, "")
	flagSet.Bool("V", false, "")
	flagSet.Bool("version", false, "")
	flagSet.Bool("verbose", false, "")
	flagSet.Bool("extract", false, "")
	flagSet.Bool("to-ts", false, "")
	flagSet.Int("workers", 0, "")
	flagSet.Bool("h", false, "")
	flagSet.Bool("help", false, "")
	langSettingsFile := flagSet.String("lang-settings-file", "", "")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	// Input/output
	cfg.InputFile = lookupFlag(flagSet, "i", "input")
	cfg.OutputFile = lookupFlag(flagSet, "o", "output")

	// Load replacement rules from file
	if f := lookupFlag(flagSet, "f", "replace-file"); f != "" {
		cfg.Replacements = append(cfg.Replacements, loadReplacementsFromFile(f)...)
	}

	cfg.NoHeaders = lookupBoolFlag(flagSet, "no-headers")
	cfg.NoFooters = lookupBoolFlag(flagSet, "no-footers")
	if v := lookupFlag(flagSet, "skip-sheets"); v != "" {
		cfg.SkipSheets = strings.Split(v, ",")
		for i := range cfg.SkipSheets {
			cfg.SkipSheets[i] = strings.TrimSpace(cfg.SkipSheets[i])
		}
	}
	cfg.Verbose = lookupBoolFlag(flagSet, "v") || lookupBoolFlag(flagSet, "verbose")
	cfg.ExtractOnly = lookupBoolFlag(flagSet, "extract")
	cfg.ToTS = lookupBoolFlag(flagSet, "to-ts")
	cfg.Workers = lookupIntFlag(flagSet, "workers")
	cfg.Help = lookupBoolFlag(flagSet, "h") || lookupBoolFlag(flagSet, "help")
	cfg.Version = lookupBoolFlag(flagSet, "V") || lookupBoolFlag(flagSet, "version")
	cfg.LangSettingsFile = *langSettingsFile

	return cfg
}

// extractReplaceRules scans args for all -r/--replace occurrences and returns
// parsed replacement rules. This is needed because Go's flag package only
// retains the last value for repeated flags.
func extractReplaceRules(args []string) []ReplacementRule {
	var rules []ReplacementRule
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-r=") {
			if rep := parseReplacement(strings.TrimPrefix(arg, "-r=")); rep != nil {
				rules = append(rules, *rep)
			}
		} else if strings.HasPrefix(arg, "--replace=") {
			if rep := parseReplacement(strings.TrimPrefix(arg, "--replace=")); rep != nil {
				rules = append(rules, *rep)
			}
		} else if arg == "-r" || arg == "--replace" {
			if i+1 < len(args) {
				i++
				if rep := parseReplacement(args[i]); rep != nil {
					rules = append(rules, *rep)
				}
			}
		}
	}
	return rules
}

// lookupFlag returns the non-empty value of the first matched flag name.
func lookupFlag(fs *flag.FlagSet, names ...string) string {
	for _, name := range names {
		f := fs.Lookup(name)
		if f != nil && f.Value.String() != "" {
			return f.Value.String()
		}
	}
	return ""
}

func lookupBoolFlag(fs *flag.FlagSet, name string) bool {
	f := fs.Lookup(name)
	if f == nil {
		return false
	}
	return f.Value.String() == "true"
}

func lookupIntFlag(fs *flag.FlagSet, name string) int {
	f := fs.Lookup(name)
	if f == nil {
		return 0
	}
	var n int
	fmt.Sscanf(f.Value.String(), "%d", &n)
	return n
}

func parseReplacement(repStr string) *ReplacementRule {
	parts := strings.SplitN(repStr, "=", 2)
	if len(parts) != 2 {
		log.Print(i18n.T(i18n.WarnInvalidReplaceFormat, map[string]interface{}{"Format": repStr}))
		return nil
	}
	return &ReplacementRule{Old: parts[0], New: parts[1]}
}

func loadReplacementsFromFile(filename string) []ReplacementRule {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(i18n.T(i18n.ErrOpenReplaceFile, map[string]interface{}{"Error": err.Error()}))
	}
	defer file.Close()

	var replacements []ReplacementRule
	if err := json.NewDecoder(file).Decode(&replacements); err != nil {
		log.Fatal(i18n.T(i18n.ErrParseReplaceFile, map[string]interface{}{"Error": err.Error()}))
	}
	return replacements
}

func printHelp() {
	fmt.Printf(i18n.T(i18n.HelpHeader)+"\n\n", Version)

	fmt.Printf("%s\n", i18n.T(i18n.HelpUsage))
	fmt.Println("  docx-find-replace -i <input> [options]")
	fmt.Println()

	fmt.Printf("%s\n", i18n.T(i18n.HelpSupportedTypes))
	fmt.Printf("  .docx  - %s\n", i18n.T(i18n.HelpDocxDesc))
	fmt.Printf("  .xlsx  - %s\n", i18n.T(i18n.HelpXlsxDesc))
	fmt.Println()

	fmt.Printf("%s\n", i18n.T(i18n.HelpOptions))
	fmt.Printf("  -i, --input <file>           %s\n", i18n.T(i18n.FlagInputShort))
	fmt.Printf("  -o, --output <file>          %s\n", i18n.T(i18n.FlagOutputShort))
	fmt.Printf("  -r, --replace <old=new>      %s\n", i18n.T(i18n.FlagReplaceShort))
	fmt.Printf("  -f, --replace-file <file>    %s\n", i18n.T(i18n.FlagReplaceFileShort))
	fmt.Printf("      --extract                %s\n", i18n.T(i18n.FlagExtract))
	fmt.Printf("      --to-ts                  %s\n", i18n.T(i18n.FlagToTS))
	fmt.Printf("      --no-headers             %s\n", i18n.T(i18n.FlagNoHeaders))
	fmt.Printf("      --no-footers             %s\n", i18n.T(i18n.FlagNoFooters))
	fmt.Printf("      --skip-sheets <patterns> %s\n", i18n.T(i18n.FlagSkipSheets))
	fmt.Printf("                               %s\n", i18n.T(i18n.FlagSkipSheetsDetail))
	fmt.Printf("      --workers <n>            %s\n", i18n.T(i18n.FlagWorkers))
	fmt.Printf("      --lang-settings-file <file> %s\n", i18n.T(i18n.FlagLangSettingsFile))
	fmt.Printf("  -v, --verbose                %s\n", i18n.T(i18n.FlagVerboseShort))
	fmt.Printf("  -V, --version                %s\n", i18n.T(i18n.FlagVersionShort))
	fmt.Printf("  -h, --help                   %s\n", i18n.T(i18n.FlagHelpShort))
	fmt.Println()

	fmt.Printf("%s\n", i18n.T(i18n.HelpExamples))
	fmt.Println("  # DOCX --extract")
	fmt.Println("  docx-find-replace -i input.docx --extract")
	fmt.Println()
	fmt.Println("  # DOCX replace")
	fmt.Println("  docx-find-replace -i input.docx -r \"Company A=Company B\"")
	fmt.Println()
	fmt.Println("  # XLSX --extract")
	fmt.Println("  docx-find-replace -i input.xlsx --extract")
	fmt.Println()
	fmt.Println("  # XLSX replace")
	fmt.Println("  docx-find-replace -i input.xlsx -r \"Old Name=New Name\"")
	fmt.Println()
	fmt.Println("  # XLSX --skip-sheets")
	fmt.Println("  docx-find-replace -i input.xlsx -r \"old=new\" --skip-sheets \"Sheet1,Sheet2\"")
	fmt.Println("  docx-find-replace -i input.xlsx -r \"old=new\" --skip-sheets \"!Summary\"")
	fmt.Println("  docx-find-replace -i input.xlsx -r \"old=new\" --skip-sheets \"*.Data\"")
	fmt.Println("  docx-find-replace -i input.xlsx -r \"old=new\" --skip-sheets \"@regexp:^Config\"")
	fmt.Println()
	fmt.Println("  # --lang-settings-file")
	fmt.Println("  docx-find-replace --lang-settings-file custom_ja.toml  # generate example")
	fmt.Println("  docx-find-replace -i input.docx -r \"old=new\" --lang-settings-file custom_ja.toml  # use custom")
	fmt.Println()
	fmt.Println("  # --to-ts")
	fmt.Println("  docx-find-replace -i input.docx --to-ts -o output.ts")
	fmt.Println()

	fmt.Printf("%s\n", i18n.T(i18n.HelpReplaceFileFmt))
	fmt.Println("  [")
	fmt.Println("    {\"old\": \"TODO\", \"new\": \"TODO\"}")
	fmt.Println("  ]")
}

func printVersion() {
	fmt.Println(Version)
}

// resolveOutputFile returns the output file path, defaulting to input_replaced.ext.
func resolveOutputFile(inputFile, outputFile string) string {
	if outputFile != "" {
		return outputFile
	}
	ext := filepath.Ext(inputFile)
	base := inputFile[:len(inputFile)-len(ext)]
	return base + "_replaced" + ext
}

// confirmOverwrite checks if the output file already exists and prompts the user.
// Returns false if the user declines overwriting.
func confirmOverwrite(outputFile string) bool {
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		return true
	}
	fmt.Println(i18n.T(i18n.WarnOutputFileExists, map[string]interface{}{"File": outputFile}))
	fmt.Print(i18n.T(i18n.InfoOverwritePrompt))
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		// EOF or read error — treat as cancel
		fmt.Println()
		fmt.Println(i18n.T(i18n.InfoOperationCancelled))
		return false
	}
	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println(i18n.T(i18n.InfoOperationCancelled))
		return false
	}
	return true
}

// ensureOutputDir creates the parent directory of the output file if needed.
func ensureOutputDir(outputFile string) {
	outputDir := filepath.Dir(outputFile)
	if outputDir == "." {
		return
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatal(i18n.T(i18n.ErrCreateOutputDir, map[string]interface{}{"Error": err.Error()}))
	}
}
