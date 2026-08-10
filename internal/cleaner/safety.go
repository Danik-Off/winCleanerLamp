package cleaner

import (
	"os"
	"path/filepath"
	"strings"
)

// forbiddenPathPrefixes — каталоги, удаление которых (включая любые вложенные
// пути) всегда запрещено. Раньше этот список дублировался в cleaner.go,
// emptydirs.go и gui/electron/main.ts с разной (и местами более слабой)
// логикой сравнения — теперь это единственный источник истины.
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
// которые сама программа целенаправленно чистит (см. targets.go: font-cache).
var pathSafetyExceptions = map[string]bool{
	`c:\windows\system32\fntcache.dat`: true,
}

// IsPathSafeToDelete — единая проверка безопасности пути перед удалением.
// Используется всеми операциями удаления (обычные категории, --delete-path,
// --delete-dir, OrphanCleaner). Возвращает (true, "") если путь можно
// удалять, иначе (false, причина).
func IsPathSafeToDelete(p string) (bool, string) {
	if strings.TrimSpace(p) == "" {
		return false, "пустой путь"
	}

	abs, err := filepath.Abs(p)
	if err != nil {
		return false, "не удалось определить абсолютный путь"
	}
	abs = filepath.Clean(abs)

	if ok, reason := checkStaticPathSafety(abs); !ok {
		return false, reason
	}

	// Если путь — символическая ссылка/junction, ведущая наружу разрешённой
	// зоны, EvalSymlinks вернёт итоговый физический путь — проверяем и его.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		resolved = filepath.Clean(resolved)
		if !strings.EqualFold(resolved, abs) {
			if ok, reason := checkStaticPathSafety(resolved); !ok {
				return false, reason + " (через символическую ссылку/junction)"
			}
		}
	}

	return true, ""
}

// checkStaticPathSafety — проверки по самой строке пути, без обращения к ФС
// (кроме UserHomeDir, который не трогает диск).
func checkStaticPathSafety(abs string) (bool, string) {
	low := strings.ToLower(abs)

	// UNC-пути (\\server\share) и расширенные (\\?\...) — вне зоны ответственности.
	if strings.HasPrefix(low, `\\`) {
		return false, "UNC/расширенные пути не поддерживаются"
	}

	// Корень диска ("C:\", "D:\").
	if len(abs) <= 3 {
		return false, "нельзя удалить корень диска"
	}

	if pathSafetyExceptions[low] {
		return true, ""
	}

	for _, f := range forbiddenPathPrefixes {
		if low == f || strings.HasPrefix(low, f+`\`) {
			return false, "путь входит в защищённый системный каталог: " + f
		}
	}

	// Домашняя папка пользователя целиком (не вложенные пути в ней — они
	// как раз и есть легитимные цели очистки, например AppData).
	if home, err := os.UserHomeDir(); err == nil {
		if strings.EqualFold(abs, filepath.Clean(home)) {
			return false, "нельзя удалить домашнюю папку пользователя целиком"
		}
	}

	return true, ""
}
