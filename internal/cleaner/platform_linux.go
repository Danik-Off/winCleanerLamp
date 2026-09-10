//go:build linux

package cleaner

// Мелкие платформенные константы Linux-ядра.
const (
	// В Linux $VAR/${VAR} — штатный способ записи путей, и XDG-переменные
	// (XDG_CACHE_HOME, XDG_DATA_HOME) задаются именно так.
	expandDollarVars = true
)
