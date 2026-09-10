//go:build darwin

package cleaner

import (
	"os"
	"path/filepath"
)

// В macOS всё пользовательское хозяйство лежит в ~/Library с фиксированными
// именами подкаталогов (Caches, Logs, Application Support, Preferences) —
// XDG-переменные здесь не используются.

func userLibraryDir() string { return sub(userHomeDir(), "Library") }

func userCacheDir() string { return sub(userLibraryDir(), "Caches") }

func userLogsDir() string { return sub(userLibraryDir(), "Logs") }

func userAppSupportDir() string { return sub(userLibraryDir(), "Application Support") }

// appStateDir — каталог файлов состояния ядра (junk.json, кеш хэшей).
func appStateDir() string {
	if d := userAppSupportDir(); d != "" {
		return filepath.Join(d, "CleanerLamp")
	}
	return ""
}

// userTempDir — каталог TMPDIR текущего пользователя
// (/private/var/folders/<xx>/<...>/T). У каждого пользователя он свой, и
// узнать его можно только из окружения — фиксированного пути не существует.
func userTempDir() string {
	if d := os.Getenv("TMPDIR"); filepath.IsAbs(d) {
		return filepath.Clean(d)
	}
	return ""
}

// systemDirRoots — системные каталоги (в нижнем регистре): поиск дубликатов
// и крупных файлов по умолчанию их пропускает.
var systemDirRoots = []string{
	"/system", "/usr", "/bin", "/sbin", "/applications", "/library",
	"/private/var/db", "/private/var/vm", "/volumes", "/cores",
}

// skipDirPlatform — каталоги, которые не обходятся при поиске дубликатов:
// внутри бандлов и системных библиотек «дубликаты» — это нормальная
// структура приложения, а не лишние копии.
var skipDirPlatform = map[string]bool{
	"library": true, "applications": true, "system": true,
	".trash": true, ".trashes": true, ".timemachine": true,
	".spotlight-v100": true, ".fseventsd": true, ".documentrevisions-v100": true,
	"derivedddata": true, "deriveddata": true,
}

// protectedEmptyDirPlatform — каталоги, которые не предлагаются к удалению,
// даже если пусты.
var protectedEmptyDirPlatform = map[string]bool{
	"library": true, "applications": true, "movies": true, "sites": true,
	"caches": true, "logs": true, "preferences": true,
	"application support": true, "launchagents": true, "launchdaemons": true,
	"icloud drive": true, "mobile documents": true,
}

// riskyDuplicateExt — расширения, для которых совпадение содержимого не
// означает, что копия лишняя: части бандлов, библиотеки и расширения ядра.
var riskyDuplicateExt = map[string]bool{
	".dylib": true, ".so": true, ".kext": true, ".bundle": true,
	".framework": true, ".pkg": true, ".a": true,
}
