package cleaner

import (
	"os"
	"path/filepath"
	"strings"
)

// Единая проверка безопасности пути перед удалением — общая для всех ядер.
// Списки защищённых каталогов и правила «корня» у каждой ОС свои и живут в
// safety_windows.go / safety_linux.go / safety_darwin.go:
//
//	forbiddenPathPrefixes — каталоги (с вложенными путями), удаление которых
//	                        всегда запрещено, в нижнем регистре;
//	pathSafetyExceptions  — точечные исключения-файлы внутри них;
//	platformPathSafety    — проверки, специфичные для ОС (корень диска и UNC
//	                        в Windows, "/" и системные точки монтирования
//	                        в Linux/macOS).
//
// Раньше этот список дублировался в cleaner.go, emptydirs.go и
// gui/electron/main.ts с разной (и местами более слабой) логикой сравнения —
// теперь это единственный источник истины.

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

	if ok, reason := platformPathSafety(abs, low); !ok {
		return false, reason
	}

	if pathSafetyExceptions[low] {
		return true, ""
	}

	sep := string(filepath.Separator)
	for _, f := range forbiddenPathPrefixes {
		if low == f || strings.HasPrefix(low, f+sep) {
			return false, "путь входит в защищённый системный каталог: " + f
		}
	}

	// Домашняя папка пользователя целиком (не вложенные пути в ней — они
	// как раз и есть легитимные цели очистки, например AppData / ~/.cache).
	if home, err := os.UserHomeDir(); err == nil {
		if strings.EqualFold(abs, filepath.Clean(home)) {
			return false, "нельзя удалить домашнюю папку пользователя целиком"
		}
	}

	return true, ""
}
