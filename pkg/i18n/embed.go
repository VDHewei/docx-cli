package i18n

import "embed"

//go:embed locales/active.zh.toml locales/active.zh-TW.toml locales/active.en.toml
var LocaleFS embed.FS
