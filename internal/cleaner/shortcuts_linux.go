//go:build linux

package cleaner

import (
	"os"
	"path/filepath"
	"strings"
)

// isShortcutCandidate — ярлыком в Linux считается файл .desktop.
func isShortcutCandidate(p string, d os.DirEntry) bool {
	return !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".desktop")
}

// DefaultShortcutRoots — стандартные места, где лежат ярлыки приложений:
// рабочий стол, меню приложений (пользовательское и системное), автозапуск.
func DefaultShortcutRoots() []string {
	roots := []string{
		sub(userHomeDir(), "Desktop"),
		sub(userHomeDir(), "Рабочий стол"),
		userAutostartDir(),
	}
	roots = append(roots, desktopApplicationDirs()...)
	return nonEmpty(roots)
}

// resolveShortcutTargets проверяет цель каждого .desktop: путь из TryExec или
// первое слово Exec. Отдельного процесса на файл (как WScript.Shell в
// Windows) здесь не нужно — формат текстовый и разбирается напрямую.
//
// Ярлыки, которым нечего проверять (Type=Link/Directory, запуск через
// D-Bus-активацию), пропускаются: у них нет исполняемого файла, и считать их
// битыми было бы ложной тревогой.
func resolveShortcutTargets(files []string) ([]BrokenShortcut, error) {
	var broken []BrokenShortcut
	for _, f := range files {
		keys := readDesktopEntry(f)
		if keys == nil {
			continue
		}
		if t := keys["Type"]; t != "" && t != "Application" {
			continue
		}
		if keys["DBusActivatable"] == "true" && keys["TryExec"] == "" {
			continue
		}
		target := desktopExecPath(keys)
		if target == "" || desktopTargetExists(target) {
			continue
		}
		broken = append(broken, BrokenShortcut{
			Path:       f,
			TargetPath: target,
			Reason:     "цель ярлыка не найдена (нет файла и нет команды в PATH)",
		})
	}
	return broken, nil
}
