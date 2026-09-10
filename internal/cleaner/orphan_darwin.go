//go:build darwin

package cleaner

import (
	"os"
	"path/filepath"
	"strings"
)

// registryKeyExists — в macOS реестра нет. Поле registryKeys в
// orphaned_apps.json заполнено windows-путями, поэтому здесь всегда false.
func registryKeyExists(_ string) bool { return false }

// DefaultDiscoverRoots — где искать неизвестные каталоги: пользовательские
// каталоги настроек, данных и кешей плюс папки приложений.
func DefaultDiscoverRoots() []string {
	roots := []string{
		userAppSupportDir(),
		userCacheDir(),
		userLogsDir(),
		sub(userLibraryDir(), "Containers"),
		sub(userLibraryDir(), "Preferences"),
	}
	roots = append(roots, applicationDirs()...)
	return nonEmpty(roots)
}

// dirHasExecutable проверяет, есть ли в каталоге исполняемый файл (до 2
// уровней вглубь). Для macOS признаком «здесь программа» служит и сам
// .app-бандл: внутри него исполняемый файл лежит глубже, в Contents/MacOS.
func dirHasExecutable(root string) bool {
	if strings.EqualFold(filepath.Ext(root), ".app") {
		return true
	}
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if found {
			return filepath.SkipAll
		}
		rel, _ := filepath.Rel(root, path)
		depth := strings.Count(rel, string(filepath.Separator))
		if d.IsDir() {
			if strings.EqualFold(filepath.Ext(d.Name()), ".app") {
				found = true
				return filepath.SkipAll
			}
			if depth > 2 {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		if info.Mode().Perm()&0o111 != 0 || isDarwinBinaryFile(d.Name()) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// isDarwinBinaryFile — файлы, которые означают «здесь установлена
// программа», даже если бит x снят.
func isDarwinBinaryFile(name string) bool {
	low := strings.ToLower(name)
	for _, ext := range []string{".dylib", ".framework", ".kext", ".jar"} {
		if strings.HasSuffix(low, ext) {
			return true
		}
	}
	return false
}

// discoverWhitelist — каталоги, которые точно не мусор.
func discoverWhitelist() map[string]bool {
	out := knownSystemFolders()
	for _, s := range []string{
		"com.apple", "apple", "google", "mozilla", "microsoft", "jetbrains",
		"adobe", "homebrew", "utilities", "xcode", "safari", "mail",
	} {
		out[s] = true
	}
	for s := range programFilesWhitelist() {
		out[s] = true
	}
	return out
}
