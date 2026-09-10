//go:build darwin

package cleaner

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// isShortcutCandidate — в macOS «битым ярлыком» бывает две вещи:
// символическая ссылка в никуда и .app-бандл, внутри которого нет
// исполняемого файла (так выглядит приложение, часть которого удалили
// вручную).
//
// Файлы-псевдонимы Finder (alias) и .webloc сюда не входят: первые — это
// двоичный формат, который без Carbon/Cocoa не прочитать, вторые указывают
// на URL, а не на файл. Проверить их нечем, и записывать их в «проверено»
// было бы неправдой.
func isShortcutCandidate(p string, d os.DirEntry) bool {
	if d.Type()&fs.ModeSymlink != 0 {
		return true
	}
	return d.IsDir() && strings.EqualFold(filepath.Ext(p), ".app")
}

// DefaultShortcutRoots — рабочий стол, папки приложений и Dock-ярлыки
// пользователя.
func DefaultShortcutRoots() []string {
	roots := []string{
		sub(userHomeDir(), "Desktop"),
		sub(userHomeDir(), "Documents"),
	}
	roots = append(roots, applicationDirs()...)
	return nonEmpty(roots)
}

// resolveShortcutTargets проверяет цель каждого кандидата.
func resolveShortcutTargets(files []string) ([]BrokenShortcut, error) {
	var broken []BrokenShortcut
	for _, f := range files {
		info, err := os.Lstat(f)
		if err != nil {
			continue
		}

		if info.Mode()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(f)
			if err != nil {
				continue
			}
			resolved := target
			if !filepath.IsAbs(resolved) {
				resolved = filepath.Join(filepath.Dir(f), resolved)
			}
			if _, err := os.Stat(resolved); err == nil {
				continue
			}
			broken = append(broken, BrokenShortcut{
				Path:       f,
				TargetPath: target,
				Reason:     "символическая ссылка указывает на несуществующий путь",
			})
			continue
		}

		if strings.EqualFold(filepath.Ext(f), ".app") {
			exe := appBundleExecutable(f)
			if exe == "" {
				// Info.plist не читается или не содержит CFBundleExecutable —
				// делать вывод о «битости» не на чем.
				continue
			}
			if _, err := os.Stat(exe); err == nil {
				continue
			}
			broken = append(broken, BrokenShortcut{
				Path:       f,
				TargetPath: exe,
				Reason:     "в бандле приложения нет исполняемого файла",
			})
		}
	}
	return broken, nil
}
