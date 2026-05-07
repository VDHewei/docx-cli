//go:build windows

package i18n

import (
	"syscall"
)

import "golang.org/x/text/language"

// detectWindowsLocale uses kernel32.GetUserDefaultUILanguage to detect the
// Windows UI language and maps it to a supported language tag.
func detectWindowsLocale() language.Tag {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetUserDefaultUILanguage")
	langID, _, _ := proc.Call()

	// LANGID mapping: primary language + sublanguage
	// https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-lcid/
	primary := uintptr(langID) & 0xFF

	switch primary {
	case 0x04: // Chinese
		subLang := (uintptr(langID) >> 10) & 0x3F
		if subLang >= 0x02 { // zh-TW, zh-HK, zh-MO, zh-SG
			return parseLocaleString("zh-TW")
		}
		return parseLocaleString("zh")
	default:
		return parseLocaleString("en")
	}
}
