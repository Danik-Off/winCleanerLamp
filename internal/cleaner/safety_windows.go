//go:build windows

package cleaner

import "strings"

// forbiddenPathPrefixes — каталоги Windows, удаление которых (включая любые
// вложенные пути) всегда запрещено.
//
// Внутри C:\Windows намеренно запрещены только заведомо опасные подкаталоги
// (System32, WinSxS и т.п.), а не весь C:\Windows целиком — часть встроенных
// категорий (prefetch, windows-temp, cbs-logs, panther, windows-logs и др.)
// легитимно чистят другие подпапки C:\Windows.
var forbiddenPathPrefixes = []string{
	`c:\windows\system32`,
	`c:\windows\syswow64`,
	`c:\windows\winsxs`,
	`c:\windows\installer`,
	`c:\windows\servicing`,
	`c:\windows\boot`,
	`c:\program files`,
	`c:\program files (x86)`,
	`c:\programdata\microsoft\windows\start menu`,
	`c:\users\default`,
	`c:\users\public`,
	`c:\users\all users`,
	`c:\perflogs`,
}

// pathSafetyExceptions — точечные, явно оправданные исключения из
// forbiddenPathPrefixes для отдельных файлов внутри защищённых каталогов,
// которые сама программа целенаправленно чистит (см. targets_windows.go:
// font-cache).
var pathSafetyExceptions = map[string]bool{
	`c:\windows\system32\fntcache.dat`: true,
}

// platformPathSafety — windows-специфичные проверки: UNC-пути и корень диска.
func platformPathSafety(abs, low string) (bool, string) {
	// UNC-пути (\\server\share) и расширенные (\\?\...) — вне зоны ответственности.
	if strings.HasPrefix(low, `\\`) {
		return false, "UNC/расширенные пути не поддерживаются"
	}

	// Корень диска ("C:\", "D:\").
	if len(abs) <= 3 {
		return false, "нельзя удалить корень диска"
	}

	return true, ""
}
