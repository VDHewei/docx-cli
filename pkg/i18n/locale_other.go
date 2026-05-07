//go:build !windows

package i18n

import "golang.org/x/text/language"

// detectWindowsLocale is a no-op on non-Windows platforms.
func detectWindowsLocale() language.Tag {
	return language.Und
}
