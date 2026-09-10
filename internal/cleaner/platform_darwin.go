//go:build darwin

package cleaner

// Мелкие платформенные константы macOS-ядра.
const (
	// В macOS $VAR/${VAR} — штатный способ записи путей.
	expandDollarVars = true
)
