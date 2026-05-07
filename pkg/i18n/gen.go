package i18n

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	go_i18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

// GenerateExampleTOML writes a starter TOML file at the given path.
// It contains a [meta] section with language = "ja" as a placeholder,
// and all known message keys with their English translations as starting points.
func GenerateExampleTOML(outputPath string) error {
	// Build the example as a map
	example := make(map[string]interface{})

	// [meta] section
	example["meta"] = map[string]string{
		"language": "ja",
	}

	// Get English translations as the base for the example
	enLocalizer := go_i18n.NewLocalizer(bundle, "en")

	// Add all message keys
	keys := AllKeys()
	sort.Strings(keys)
	for _, key := range keys {
		// Try to get the English translation as a starting point
		enMsg, err := enLocalizer.Localize(&go_i18n.LocalizeConfig{
			MessageID: key,
		})
		if err != nil || enMsg == "" {
			enMsg = fmt.Sprintf("TODO: translate %s", key)
		}
		example[key] = map[string]string{
			"other": enMsg,
		}
	}

	// Write with header comment
	var sb strings.Builder
	sb.WriteString("# Language settings file for docx-cli\n")
	sb.WriteString("# Change the language below to your BCP 47 language tag (e.g. \"ja\", \"ko\", \"fr\", \"de\")\n")
	sb.WriteString("# Then translate each 'other' field to your language\n")
	sb.WriteString("#\n")
	sb.WriteString("# Pattern syntax for --skip-sheets:\n")
	sb.WriteString("#   exact match (case-insensitive): \"Sheet1\"\n")
	sb.WriteString("#   negative substring (prefix \"!\"): \"!Summary\"\n")
	sb.WriteString("#   suffix match (prefix \"*.\"): \"*.Data\"\n")
	sb.WriteString("#   regex match (prefix \"@regexp:\"): \"@regexp:^Sheet\\\\d+$\"\n")
	sb.WriteString("\n")

	// Write [meta] section first
	metaData, _ := toml.Marshal(map[string]interface{}{"meta": example["meta"]})
	sb.Write(metaData)
	sb.WriteString("\n")

	// Write remaining keys (sorted)
	delete(example, "meta")
	data, err := toml.Marshal(&example)
	if err != nil {
		return fmt.Errorf("cannot encode example TOML: %w", err)
	}
	sb.Write(data)

	if err := os.WriteFile(outputPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("cannot write example TOML: %w", err)
	}

	return nil
}
