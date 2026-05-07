package i18n

// Error message keys (Err prefix) — used in log.Fatal / log.Fatalf
const (
	ErrInputRequired       = "ErrInputRequired"
	ErrFileNotFound        = "ErrFileNotFound"
	ErrUnsupportedFileType = "ErrUnsupportedFileType"
	ErrOpenDocument        = "ErrOpenDocument"
	ErrInvalidDocStructure = "ErrInvalidDocStructure"
	ErrWriteTSFile         = "ErrWriteTSFile"
	ErrNoReplacementRules  = "ErrNoReplacementRules"
	ErrOpenSpreadsheet     = "ErrOpenSpreadsheet"
	ErrTTSOnlyDocx         = "ErrTTSOnlyDocx"
	ErrOpenReplaceFile     = "ErrOpenReplaceFile"
	ErrParseReplaceFile    = "ErrParseReplaceFile"
	ErrCreateOutputDir     = "ErrCreateOutputDir"
	ErrSaveDocument        = "ErrSaveDocument"
	ErrSaveSpreadsheet     = "ErrSaveSpreadsheet"
	ErrI18nInit            = "ErrI18nInit"
)

// Warning message keys (Warn prefix) — used in log.Printf
const (
	WarnInvalidReplaceFormat = "WarnInvalidReplaceFormat"
	WarnOutputFileExists     = "WarnOutputFileExists"
)

// Verbose output keys (Verbose prefix) — guarded by cfg.Verbose
const (
	VerboseInputFile    = "VerboseInputFile"
	VerboseModeExtract  = "VerboseModeExtract"
	VerboseModeToTS     = "VerboseModeToTS"
	VerboseReplaceCount = "VerboseReplaceCount"
	VerboseWorkers      = "VerboseWorkers"
	VerboseSkipHeaders  = "VerboseSkipHeaders"
	VerboseSkipFooters  = "VerboseSkipFooters"
	VerboseSkipSheets   = "VerboseSkipSheets"
	VerboseReplaceDone  = "VerboseReplaceDone"
	VerboseParagraphs   = "VerboseParagraphs"
	VerboseCells        = "VerboseCells"
	VerboseHeaders      = "VerboseHeaders"
	VerboseFooters      = "VerboseFooters"
	VerboseSheets       = "VerboseSheets"
)

// Info output keys (Info prefix) — normal fmt output
const (
	InfoExtractedTextCount     = "InfoExtractedTextCount"
	InfoExtractedCellCount     = "InfoExtractedCellCount"
	InfoExtractedTextSegCount  = "InfoExtractedTextSegCount"
	InfoExtractedCellTextCount = "InfoExtractedCellTextCount"
	InfoTSFileGenerated        = "InfoTSFileGenerated"
	InfoOverwritePrompt        = "InfoOverwritePrompt"
	InfoOperationCancelled     = "InfoOperationCancelled"
	InfoSuccessDocx            = "InfoSuccessDocx"
	InfoSuccessXlsx            = "InfoSuccessXlsx"
	InfoTotalReplacements      = "InfoTotalReplacements"
	InfoProcessedParagraphs    = "InfoProcessedParagraphs"
	InfoProcessedCells         = "InfoProcessedCells"
	InfoProcessedHeaders       = "InfoProcessedHeaders"
	InfoProcessedFooters       = "InfoProcessedFooters"
	InfoProcessedSheets        = "InfoProcessedSheets"
	InfoOutputFile             = "InfoOutputFile"
	InfoLangFileGenerated      = "InfoLangFileGenerated"
)

// Help section keys (Help prefix) — used in printHelp()
const (
	HelpHeader         = "HelpHeader"
	HelpUsage          = "HelpUsage"
	HelpSupportedTypes = "HelpSupportedTypes"
	HelpDocxDesc       = "HelpDocxDesc"
	HelpXlsxDesc       = "HelpXlsxDesc"
	HelpOptions        = "HelpOptions"
	HelpExamples       = "HelpExamples"
	HelpReplaceFileFmt = "HelpReplaceFileFmt"
)

// Flag description keys (Flag prefix) — help text flag descriptions
const (
	FlagInputShort       = "FlagInputShort"
	FlagOutputShort      = "FlagOutputShort"
	FlagReplaceShort     = "FlagReplaceShort"
	FlagReplaceFileShort = "FlagReplaceFileShort"
	FlagNoHeaders        = "FlagNoHeaders"
	FlagNoFooters        = "FlagNoFooters"
	FlagSkipSheets       = "FlagSkipSheets"
	FlagSkipSheetsDetail = "FlagSkipSheetsDetail"
	FlagVerboseShort     = "FlagVerboseShort"
	FlagVersionShort     = "FlagVersionShort"
	FlagExtract          = "FlagExtract"
	FlagToTS             = "FlagToTS"
	FlagWorkers          = "FlagWorkers"
	FlagHelpShort        = "FlagHelpShort"
	FlagLangSettingsFile = "FlagLangSettingsFile"
)

// AllKeys returns all message key constants for iteration (used in gen.go).
func AllKeys() []string {
	return []string{
		ErrInputRequired, ErrFileNotFound, ErrUnsupportedFileType,
		ErrOpenDocument, ErrInvalidDocStructure, ErrWriteTSFile,
		ErrNoReplacementRules, ErrOpenSpreadsheet, ErrTTSOnlyDocx,
		ErrOpenReplaceFile, ErrParseReplaceFile, ErrCreateOutputDir,
		ErrSaveDocument, ErrSaveSpreadsheet, ErrI18nInit,
		WarnInvalidReplaceFormat, WarnOutputFileExists,
		VerboseInputFile, VerboseModeExtract, VerboseModeToTS,
		VerboseReplaceCount, VerboseWorkers, VerboseSkipHeaders,
		VerboseSkipFooters, VerboseSkipSheets, VerboseReplaceDone,
		VerboseParagraphs, VerboseCells, VerboseHeaders,
		VerboseFooters, VerboseSheets,
		InfoExtractedTextCount, InfoExtractedCellCount,
		InfoExtractedTextSegCount, InfoExtractedCellTextCount,
		InfoTSFileGenerated, InfoOverwritePrompt,
		InfoOperationCancelled, InfoSuccessDocx, InfoSuccessXlsx,
		InfoTotalReplacements, InfoProcessedParagraphs,
		InfoProcessedCells, InfoProcessedHeaders,
		InfoProcessedFooters, InfoProcessedSheets,
		InfoOutputFile, InfoLangFileGenerated,
		HelpHeader, HelpUsage, HelpSupportedTypes,
		HelpDocxDesc, HelpXlsxDesc, HelpOptions,
		HelpExamples, HelpReplaceFileFmt,
		FlagInputShort, FlagOutputShort, FlagReplaceShort,
		FlagReplaceFileShort, FlagNoHeaders, FlagNoFooters,
		FlagSkipSheets, FlagSkipSheetsDetail,
		FlagVerboseShort, FlagVersionShort, FlagExtract,
		FlagToTS, FlagWorkers, FlagHelpShort, FlagLangSettingsFile,
	}
}
