package cleaner

import (
	"os"
	"path/filepath"
	"strings"
)

// BrokenShortcut — ярлык, чья цель не найдена на диске.
type BrokenShortcut struct {
	Path       string `json:"path"`
	TargetPath string `json:"targetPath,omitempty"`
	Reason     string `json:"reason"`
}

// ShortcutScanResult — результат поиска битых ярлыков.
type ShortcutScanResult struct {
	Broken  []BrokenShortcut `json:"broken"`
	Scanned int              `json:"scanned"`
}

// ScanBrokenShortcuts ищет ярлыки, чья цель не существует на диске.
// Что считается ярлыком и как разрешается его цель, зависит от ОС:
// .lnk через WScript.Shell в Windows, .desktop (ключ Exec) в Linux,
// .app-бандлы и символические ссылки в macOS — см. isShortcutCandidate и
// resolveShortcutTargets в shortcuts_<os>.go.
func ScanBrokenShortcuts(roots []string) (*ShortcutScanResult, error) {
	if len(roots) == 0 {
		roots = DefaultShortcutRoots()
	}

	var files []string
	seen := map[string]bool{}
	for _, root := range roots {
		if root == "" {
			continue
		}
		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !isShortcutCandidate(p, d) {
				return nil
			}
			key := strings.ToLower(filepath.Clean(p))
			if !seen[key] {
				seen[key] = true
				files = append(files, p)
			}
			// Ярлык-каталог (.app в macOS) — цельный объект: внутрь него
			// заходить не нужно.
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		})
	}

	result := &ShortcutScanResult{Scanned: len(files)}
	if len(files) == 0 {
		return result, nil
	}

	broken, err := resolveShortcutTargets(files)
	if err != nil {
		return result, err
	}
	result.Broken = broken
	return result, nil
}
