//go:build windows

package cleaner

// Мелкие платформенные константы Windows-ядра.
const (
	// expandDollarVars=false: в Windows доллар — обычный символ в имени папки
	// (C:\$WINDOWS.~BT, C:\$GetCurrent, C:\$Recycle.Bin), а не начало
	// переменной окружения. См. ExpandPath.
	expandDollarVars = false
)
