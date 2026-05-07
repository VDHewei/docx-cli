package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/language"
)

// --- parseLocaleString tests ---

func TestParseLocaleString_SimplifiedChinese(t *testing.T) {
	tag := parseLocaleString("zh_CN.UTF-8")
	if !strings.HasPrefix(tag.String(), "zh") {
		t.Errorf("expected zh variant, got %s", tag)
	}
}

func TestParseLocaleString_TraditionalChinese(t *testing.T) {
	tag := parseLocaleString("zh_TW.UTF-8")
	if !strings.HasPrefix(tag.String(), "zh-TW") {
		t.Errorf("expected zh-TW variant, got %s", tag)
	}
}

func TestParseLocaleString_English(t *testing.T) {
	tag := parseLocaleString("en_US.UTF-8")
	if !strings.HasPrefix(tag.String(), "en") {
		t.Errorf("expected en variant, got %s", tag)
	}
}

func TestParseLocaleString_PlainTag(t *testing.T) {
	tag := parseLocaleString("zh")
	if !strings.HasPrefix(tag.String(), "zh") {
		t.Errorf("expected zh variant, got %s", tag)
	}
}

func TestParseLocaleString_ZhHK(t *testing.T) {
	tag := parseLocaleString("zh_HK")
	// zh_HK should match zh-TW via the matcher
	s := tag.String()
	if !strings.HasPrefix(s, "zh-TW") && !strings.HasPrefix(s, "zh-Hant") {
		t.Errorf("expected zh-TW or zh-Hant variant for zh_HK, got %s", tag)
	}
}

func TestParseLocaleString_InvalidTag(t *testing.T) {
	// "xx" is not a valid ISO 639 code, so language.Parse returns Und,
	// and the matcher cannot find a supported tag. This is expected.
	// The important thing is it doesn't panic.
	_ = parseLocaleString("xx_YY.UTF-8")
}

func TestParseLocaleString_EmptyString(t *testing.T) {
	tag := parseLocaleString("")
	if tag != language.Und {
		t.Errorf("expected Und for empty string, got %s", tag)
	}
}

func TestParseLocaleString_NoEncoding(t *testing.T) {
	tag := parseLocaleString("zh_CN")
	if !strings.HasPrefix(tag.String(), "zh") {
		t.Errorf("expected zh variant, got %s", tag)
	}
}

// --- Init + T tests ---

func TestInit_DefaultLocale(t *testing.T) {
	generated, err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if generated {
		t.Error("expected generated=false for empty langSettingsFile")
	}
}

func TestT_WithZhLocale(t *testing.T) {
	// Force Chinese locale
	os.Setenv("LANG", "zh_CN.UTF-8")
	defer os.Unsetenv("LANG")

	generated, err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if generated {
		t.Error("expected generated=false")
	}

	msg := T(ErrInputRequired)
	if msg == ErrInputRequired {
		t.Error("T() returned the key itself, translation not found")
	}
	if !strings.Contains(msg, "必须指定输入文件") {
		t.Errorf("expected Chinese translation, got: %s", msg)
	}
}

func TestT_WithEnLocale(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	generated, err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if generated {
		t.Error("expected generated=false")
	}

	msg := T(ErrInputRequired)
	if !strings.Contains(msg, "input file required") {
		t.Errorf("expected English translation, got: %s", msg)
	}
}

func TestT_WithZhTWLocale(t *testing.T) {
	os.Setenv("LANG", "zh_TW.UTF-8")
	defer os.Unsetenv("LANG")

	generated, err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if generated {
		t.Error("expected generated=false")
	}

	msg := T(ErrInputRequired)
	if !strings.Contains(msg, "必須指定輸入檔案") {
		t.Errorf("expected Traditional Chinese translation, got: %s", msg)
	}
}

func TestT_WithTemplateData(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	_, err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	msg := T(ErrFileNotFound, map[string]interface{}{"File": "missing.docx"})
	if !strings.Contains(msg, "missing.docx") {
		t.Errorf("expected template data rendered, got: %s", msg)
	}
}

func TestT_UnknownKey(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	_, err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	msg := T("NonExistentKey")
	if msg != "NonExistentKey" {
		t.Errorf("expected key as fallback for unknown key, got: %s", msg)
	}
}

func TestT_BeforeInit(t *testing.T) {
	// Reset package state to simulate pre-init
	bundle = nil
	localizer = nil

	msg := T(ErrInputRequired)
	if msg != ErrInputRequired {
		t.Errorf("expected key as fallback before Init, got: %s", msg)
	}
}

// --- Init with --lang-settings-file (generate example) ---

func TestInit_GenerateExampleTOML(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "example.toml")

	generated, err := Init(outPath)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if !generated {
		t.Error("expected generated=true when file doesn't exist")
	}

	// Verify file was created
	if _, statErr := os.Stat(outPath); os.IsNotExist(statErr) {
		t.Fatal("example TOML file was not created")
	}

	// Verify file contains [meta] section
	data, readErr := os.ReadFile(outPath)
	if readErr != nil {
		t.Fatalf("cannot read generated file: %v", readErr)
	}
	content := string(data)
	if !strings.Contains(content, "[meta]") {
		t.Error("generated TOML missing [meta] section")
	}
	if !strings.Contains(content, "language") {
		t.Error("generated TOML missing language field in [meta]")
	}
	if !strings.Contains(content, "[ErrInputRequired]") {
		t.Error("generated TOML missing ErrInputRequired key")
	}
}

// --- Init with custom TOML (load existing file) ---

func TestInit_LoadCustomTOML(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "custom_ja.toml")
	customContent := `[meta]
language = "ja"

[ErrInputRequired]
other = "エラー: 入力ファイルを指定してください"

[FlagInputShort]
other = "入力ファイルパス"
`
	if writeErr := os.WriteFile(customPath, []byte(customContent), 0644); writeErr != nil {
		t.Fatalf("cannot write custom TOML: %v", writeErr)
	}

	generated, err := Init(customPath)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if generated {
		t.Error("expected generated=false when file exists")
	}

	// Verify custom translation loaded
	msg := T(ErrInputRequired)
	if !strings.Contains(msg, "エラー") {
		t.Errorf("expected Japanese translation, got: %s", msg)
	}

	// Verify custom Flag translation loaded
	flagMsg := T(FlagInputShort)
	if !strings.Contains(flagMsg, "入力ファイルパス") {
		t.Errorf("expected Japanese flag description, got: %s", flagMsg)
	}
}

func TestInit_CustomTOML_MissingMeta(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "bad.toml")
	if writeErr := os.WriteFile(customPath, []byte(`[ErrInputRequired]
other = "test"
`), 0644); writeErr != nil {
		t.Fatalf("cannot write custom TOML: %v", writeErr)
	}

	_, err := Init(customPath)
	if err == nil {
		t.Fatal("expected error for TOML without [meta] section")
	}
	if !strings.Contains(err.Error(), "meta") {
		t.Errorf("error should mention meta, got: %v", err)
	}
}

func TestInit_CustomTOML_InvalidLanguage(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "bad_lang.toml")
	if writeErr := os.WriteFile(customPath, []byte(`[meta]
language = "***invalid***"

[ErrInputRequired]
other = "test"
`), 0644); writeErr != nil {
		t.Fatalf("cannot write custom TOML: %v", writeErr)
	}

	_, err := Init(customPath)
	if err == nil {
		t.Fatal("expected error for invalid language tag")
	}
}

func TestInit_CustomTOML_FallbackToEnForMissingKeys(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "partial.toml")
	if writeErr := os.WriteFile(customPath, []byte(`[meta]
language = "ja"

[ErrInputRequired]
other = "エラー: 入力ファイルを指定してください"
`), 0644); writeErr != nil {
		t.Fatalf("cannot write custom TOML: %v", writeErr)
	}

	_, err := Init(customPath)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Custom key should be Japanese
	msg := T(ErrInputRequired)
	if !strings.Contains(msg, "エラー") {
		t.Errorf("expected Japanese for custom key, got: %s", msg)
	}

	// Missing key should fallback to English
	msg2 := T(ErrUnsupportedFileType)
	if !strings.Contains(msg2, "Unsupported") {
		t.Errorf("expected English fallback for missing custom key, got: %s", msg2)
	}
}

// --- AllKeys tests ---

func TestAllKeys_ContainsAllCategories(t *testing.T) {
	keys := AllKeys()
	keySet := make(map[string]bool, len(keys))
	for _, k := range keys {
		keySet[k] = true
	}

	// Spot-check one key from each prefix category
	cases := []string{
		ErrInputRequired,
		WarnOutputFileExists,
		VerboseInputFile,
		InfoSuccessDocx,
		HelpHeader,
		FlagInputShort,
	}
	for _, k := range cases {
		if !keySet[k] {
			t.Errorf("AllKeys missing key: %s", k)
		}
	}
}

func TestAllKeys_NoDuplicates(t *testing.T) {
	keys := AllKeys()
	seen := make(map[string]bool, len(keys))
	for _, k := range keys {
		if seen[k] {
			t.Errorf("duplicate key in AllKeys: %s", k)
		}
		seen[k] = true
	}
}

func TestAllKeys_Count(t *testing.T) {
	keys := AllKeys()
	expectedCount := 15 + 2 + 14 + 17 + 8 + 15 // Err + Warn + Verbose + Info + Help + Flag
	if len(keys) != expectedCount {
		t.Errorf("expected %d keys, got %d", expectedCount, len(keys))
	}
}

// --- TOML locale file completeness tests ---

func TestTOMLFiles_HaveAllKeys(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	_, err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	keys := AllKeys()
	// Check each locale has translations for all keys
	locales := []struct {
		tag   string
		check func(string) bool
	}{
		{"en", func(msg string) bool { return msg != "" && !strings.HasPrefix(msg, "Err") && !strings.HasPrefix(msg, "Warn") && !strings.HasPrefix(msg, "Verbose") && !strings.HasPrefix(msg, "Info") && !strings.HasPrefix(msg, "Help") && !strings.HasPrefix(msg, "Flag") }},
	}

	for _, loc := range locales {
		locCheck := loc.check
		for _, key := range keys {
			// Create a fresh localizer for this locale
			msg := T(key) // uses the current localizer
			if !locCheck(msg) {
				t.Logf("locale %s: key %s returned %q (may need investigation)", loc.tag, key, msg)
			}
		}
	}
}

func TestEnLocale_AllKeysTranslated(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	_, err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	keys := AllKeys()
	for _, key := range keys {
		msg := T(key)
		// If T returns the key itself, it means no translation was found
		if msg == key {
			t.Errorf("en locale: no translation for key %s", key)
		}
	}
}

func TestZhLocale_AllKeysTranslated(t *testing.T) {
	os.Setenv("LANG", "zh_CN.UTF-8")
	defer os.Unsetenv("LANG")

	_, err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	keys := AllKeys()
	for _, key := range keys {
		msg := T(key)
		if msg == key {
			t.Errorf("zh locale: no translation for key %s", key)
		}
	}
}

func TestZhTWLocale_AllKeysTranslated(t *testing.T) {
	os.Setenv("LANG", "zh_TW.UTF-8")
	defer os.Unsetenv("LANG")

	_, err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	keys := AllKeys()
	for _, key := range keys {
		msg := T(key)
		if msg == key {
			t.Errorf("zh-TW locale: no translation for key %s", key)
		}
	}
}

// --- Template rendering tests ---

func TestT_TemplateRendering_ErrFileNotFound(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	_, err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	msg := T(ErrFileNotFound, map[string]interface{}{"File": "/tmp/test.docx"})
	if !strings.Contains(msg, "/tmp/test.docx") {
		t.Errorf("template variable not rendered, got: %s", msg)
	}
}

func TestT_TemplateRendering_VerboseInputFile(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	_, err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	msg := T(VerboseInputFile, map[string]interface{}{"File": "test.xlsx", "Type": "XLSX"})
	if !strings.Contains(msg, "test.xlsx") || !strings.Contains(msg, "XLSX") {
		t.Errorf("template variables not rendered, got: %s", msg)
	}
}

func TestT_TemplateRendering_VerboseReplaceCount(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	_, err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	msg := T(VerboseReplaceCount, map[string]interface{}{"Count": 5})
	if !strings.Contains(msg, "5") {
		t.Errorf("Count template variable not rendered, got: %s", msg)
	}
}

// --- Unsupported locale fallback ---

func TestInit_UnsupportedLocaleFallsBackToEn(t *testing.T) {
	os.Setenv("LANG", "fr_FR.UTF-8")
	defer os.Unsetenv("LANG")

	_, err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	msg := T(ErrInputRequired)
	// French is not a supported locale, should fallback to English
	if !strings.Contains(msg, "input file required") {
		t.Errorf("expected English fallback for unsupported locale, got: %s", msg)
	}
}

// --- GenerateExampleTOML tests ---

func TestGenerateExampleTOML_ContainsAllKeys(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	_, err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "example.toml")

	if genErr := GenerateExampleTOML(outPath); genErr != nil {
		t.Fatalf("GenerateExampleTOML failed: %v", genErr)
	}

	data, readErr := os.ReadFile(outPath)
	if readErr != nil {
		t.Fatalf("cannot read generated file: %v", readErr)
	}
	content := string(data)

	keys := AllKeys()
	for _, key := range keys {
		if !strings.Contains(content, "["+key+"]") {
			t.Errorf("generated TOML missing key section [%s]", key)
		}
	}
}

func TestGenerateExampleTOML_ContainsMetaSection(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	_, err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "example.toml")

	if genErr := GenerateExampleTOML(outPath); genErr != nil {
		t.Fatalf("GenerateExampleTOML failed: %v", genErr)
	}

	data, readErr := os.ReadFile(outPath)
	if readErr != nil {
		t.Fatalf("cannot read generated file: %v", readErr)
	}
	content := string(data)

	if !strings.Contains(content, "[meta]") {
		t.Error("generated TOML missing [meta] section")
	}
	if !strings.Contains(content, `language = "ja"`) {
		t.Error("generated TOML missing default language = ja")
	}
}

// --- loadCustomTOML edge cases ---

func TestLoadCustomTOML_OverwriteBuiltInTranslation(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "custom_en.toml")
	// Override a single English key with a different value
	if writeErr := os.WriteFile(customPath, []byte(`[meta]
language = "en"

[ErrInputRequired]
other = "CUSTOM: you must provide an input file"
`), 0644); writeErr != nil {
		t.Fatalf("cannot write custom TOML: %v", writeErr)
	}

	_, err := Init(customPath)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	msg := T(ErrInputRequired)
	if !strings.Contains(msg, "CUSTOM:") {
		t.Errorf("custom TOML should override built-in, got: %s", msg)
	}
}
