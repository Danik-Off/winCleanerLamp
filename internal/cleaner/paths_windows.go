//go:build windows

package cleaner

import (
	"os"
	"path/filepath"
)

// appStateDir — каталог, где ядро хранит свои файлы состояния
// (junk.json, кеш хэшей дубликатов): %LOCALAPPDATA%\winCleanerLamp.
func appStateDir() string {
	if local := ExpandPath(`%LOCALAPPDATA%`); local != "" {
		return filepath.Join(local, "winCleanerLamp")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "AppData", "Local", "winCleanerLamp")
}

// systemDirRoots — системные/установочные каталоги (в нижнем регистре):
// поиск дубликатов и крупных файлов помечает их отдельно и по умолчанию
// пропускает.
var systemDirRoots = []string{
	`c:\windows`,
	`c:\program files`,
	`c:\program files (x86)`,
	`c:\programdata`,
}

// skipDirPlatform — папки Windows, которые не обходятся при поиске дубликатов.
var skipDirPlatform = map[string]bool{
	"$recycle.bin": true, "system volume information": true,
	"$windows.~bt": true, "$windows.~ws": true,
	"windows": true, "windows.old": true, "winsxs": true,
	"appdata": true, "recovery": true,
	"program files": true, "program files (x86)": true,
}

// protectedEmptyDirPlatform — папки Windows, которые не предлагаются к
// удалению, даже если пусты.
var protectedEmptyDirPlatform = map[string]bool{
	"3d objects": true, "onedrive": true, "appdata": true,
	"local": true, "locallow": true, "roaming": true,
	"microsoft": true, "windows": true,
	"start menu": true, "programs": true, "startup": true,
}

// riskyDuplicateExt — расширения, для которых "дубликат" может оказаться
// разделяемым между программами файлом (рантайм, библиотека, драйвер), а не
// просто лишней копией.
var riskyDuplicateExt = map[string]bool{
	".exe": true, ".dll": true, ".sys": true, ".ocx": true, ".msi": true,
}
