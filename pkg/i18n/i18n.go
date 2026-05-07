package i18n

import (
	"embed"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	go_i18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// bundle is the singleton i18n bundle, initialized by Init.
var bundle *go_i18n.Bundle

// localizer is the active localizer after Init.
var localizer *go_i18n.Localizer

// supportedTags defines the languages this application supports.
var supportedTags = []language.Tag{
	language.MustParse("zh"),
	language.MustParse("zh-TW"),
	language.English,
}

var matcher = language.NewMatcher(supportedTags)

// Init initializes the i18n system. It loads built-in locale files,
// detects the system locale, and optionally loads a custom TOML file.
//
// If langSettingsFile is non-empty and the file does not exist,
// it generates an example TOML at that path and returns generated=true.
// If the file exists, it loads it as a custom locale override.
func Init(langSettingsFile string) (generated bool, err error) {
	bundle = go_i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	// Load embedded built-in locales
	if err := loadEmbeddedLocales(LocaleFS); err != nil {
		return false, fmt.Errorf("load embedded locales: %w", err)
	}

	// Detect system locale and create default localizer
	tag := detectSystemLocale()
	localizer = go_i18n.NewLocalizer(bundle, tag.String())

	// Handle custom language settings file
	if langSettingsFile != "" {
		if _, statErr := os.Stat(langSettingsFile); os.IsNotExist(statErr) {
			// File does not exist: generate example and return
			if genErr := GenerateExampleTOML(langSettingsFile); genErr != nil {
				return false, fmt.Errorf("generate example TOML: %w", genErr)
			}
			return true, nil
		}
		// File exists: load as custom locale override
		if loadErr := loadCustomTOML(langSettingsFile); loadErr != nil {
			return false, fmt.Errorf("load custom TOML %s: %w", langSettingsFile, loadErr)
		}
	}

	return false, nil
}

// T translates a message identified by key, applying optional template data.
// If the key is not found, it falls back to the key itself.
func T(key string, data ...map[string]interface{}) string {
	if localizer == nil {
		return key
	}
	msg, err := localizer.Localize(&go_i18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: firstMap(data),
	})
	if err != nil {
		// Fallback: try English
		enLocalizer := go_i18n.NewLocalizer(bundle, "en")
		msg, err = enLocalizer.Localize(&go_i18n.LocalizeConfig{
			MessageID:    key,
			TemplateData: firstMap(data),
		})
		if err != nil {
			return key
		}
	}
	return msg
}

func firstMap(maps []map[string]interface{}) map[string]interface{} {
	if len(maps) > 0 {
		return maps[0]
	}
	return nil
}

// detectSystemLocale returns the best-match language tag from supportedTags
// based on the system's locale settings.
func detectSystemLocale() language.Tag {
	// Try environment variables (Unix/macOS, also set on some Windows terminals)
	for _, env := range []string{"LANG", "LC_ALL", "LC_MESSAGES"} {
		if val := os.Getenv(env); val != "" {
			if tag := parseLocaleString(val); tag != language.Und {
				return tag
			}
		}
	}

	// Windows: syscall to kernel32.GetUserDefaultUILanguage
	if runtime.GOOS == "windows" {
		if tag := detectWindowsLocale(); tag != language.Und {
			return tag
		}
	}

	// Fallback to English
	return language.English
}

// parseLocaleString normalizes a locale string (e.g. "zh_CN.UTF-8", "zh-TW")
// and matches it against the supported tags.
func parseLocaleString(s string) language.Tag {
	// Strip encoding suffix: "zh_CN.UTF-8" -> "zh_CN"
	s = strings.Split(s, ".")[0]
	// Normalize separator: "zh_CN" -> "zh-CN"
	s = strings.ReplaceAll(s, "_", "-")

	tag, err := language.Parse(s)
	if err != nil {
		return language.Und
	}

	// Use matcher to find the best supported tag
	matched, _, _ := matcher.Match(tag)
	return matched
}

// loadEmbeddedLocales reads all embedded TOML files and loads them into the bundle.
func loadEmbeddedLocales(fs embed.FS) error {
	entries, err := fs.ReadDir("locales")
	if err != nil {
		return fmt.Errorf("read embedded locales dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := fs.ReadFile("locales/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read embedded locale %s: %w", entry.Name(), err)
		}
		if _, err := bundle.ParseMessageFileBytes(data, entry.Name()); err != nil {
			return fmt.Errorf("parse embedded locale %s: %w", entry.Name(), err)
		}
	}
	return nil
}

// loadCustomTOML reads a custom TOML file, extracts the [meta].language field,
// strips the [meta] section, and loads the rest into the bundle.
func loadCustomTOML(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Step 1: Parse [meta] to get the language tag
	var meta struct {
		Meta struct {
			Language string `toml:"language"`
		} `toml:"meta"`
	}
	if err := toml.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("parse meta: %w", err)
	}
	if meta.Meta.Language == "" {
		return fmt.Errorf("[meta] section with 'language' field is required")
	}

	langTag, err := language.Parse(meta.Meta.Language)
	if err != nil {
		return fmt.Errorf("invalid language tag %q: %w", meta.Meta.Language, err)
	}

	// Step 2: Remove [meta] section from the raw TOML map and re-serialize
	var raw map[string]interface{}
	if err := toml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse TOML: %w", err)
	}
	delete(raw, "meta")

	cleanedData, err := toml.Marshal(&raw)
	if err != nil {
		return fmt.Errorf("re-serialize TOML: %w", err)
	}

	// Step 3: Load into bundle using synthesized filename (go-i18n determines language from filename)
	synthesizedName := "active." + meta.Meta.Language + ".toml"
	if _, err := bundle.ParseMessageFileBytes(cleanedData, synthesizedName); err != nil {
		return fmt.Errorf("load custom translations for %s: %w", meta.Meta.Language, err)
	}

	// Update the localizer to prefer the custom language
	localizer = go_i18n.NewLocalizer(bundle, langTag.String())

	return nil
}
