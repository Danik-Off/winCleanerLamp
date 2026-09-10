//go:build linux

package cleaner

import (
	"os"
	"path/filepath"
)

// xdgDir возвращает каталог по XDG Base Directory Specification: значение
// переменной, если она задана абсолютным путём, иначе fallback внутри
// домашнего каталога. Именно так и положено искать пути в Linux — жёстко
// зашитый ~/.cache ломается у пользователей с переопределённым XDG_CACHE_HOME.
func xdgDir(env, fallback string) string {
	if v := os.Getenv(env); filepath.IsAbs(v) {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, fallback)
}

func userCacheDir() string { return xdgDir("XDG_CACHE_HOME", ".cache") }
func userDataDir() string  { return xdgDir("XDG_DATA_HOME", ".local/share") }
func userConfigDir() string {
	return xdgDir("XDG_CONFIG_HOME", ".config")
}
func userStateDir() string { return xdgDir("XDG_STATE_HOME", ".local/state") }

// appStateDir — каталог файлов состояния ядра (junk.json, кеш хэшей).
func appStateDir() string {
	if d := userStateDir(); d != "" {
		return filepath.Join(d, "cleanerlamp")
	}
	return ""
}

// systemDirRoots — системные каталоги (в нижнем регистре): поиск дубликатов
// и крупных файлов по умолчанию их пропускает.
var systemDirRoots = []string{
	"/usr", "/bin", "/sbin", "/lib", "/lib32", "/lib64",
	"/opt", "/etc", "/var", "/boot", "/proc", "/sys", "/run", "/snap",
}

// skipDirPlatform — каталоги, которые не обходятся при поиске дубликатов.
var skipDirPlatform = map[string]bool{
	"proc": true, "sys": true, "dev": true, "run": true,
	"snap": true, ".snapshots": true, "lost+found": true,
	".var": true, "flatpak": true,
}

// protectedEmptyDirPlatform — каталоги, которые не предлагаются к удалению,
// даже если пусты: пустой ~/.config/autostart или ~/.local/bin — норма.
var protectedEmptyDirPlatform = map[string]bool{
	"autostart": true, "applications": true, "systemd": true,
	"bin": true, "lib": true, "share": true, "state": true,
	"icons": true, "fonts": true, "themes": true, "mime": true,
	".steam": true, ".var": true,
}

// riskyDuplicateExt — расширения, для которых совпадение содержимого не
// означает, что копия лишняя: разделяемые библиотеки, модули ядра и пакеты.
var riskyDuplicateExt = map[string]bool{
	".so": true, ".ko": true, ".a": true, ".deb": true, ".rpm": true,
	".appimage": true, ".run": true, ".bin": true,
}
