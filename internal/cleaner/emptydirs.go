package cleaner

import (
	"os"
	"path/filepath"
	"strings"
)

// EmptyDirCandidate — пустая или почти пустая папка.
type EmptyDirCandidate struct {
	Path  string
	Depth int // глубина от корня сканирования
}

// EmptyDirScanOptions — параметры сканирования пустых папок.
type EmptyDirScanOptions struct {
	Roots       []string // корневые папки для сканирования
	MaxDepth    int      // максимальная глубина рекурсии (0 = без ограничения)
	IgnoreJunk  bool     // считать папки с только Thumbs.db/desktop.ini пустыми
}

// EmptyDirResult — результат сканирования.
type EmptyDirResult struct {
	Dirs  []EmptyDirCandidate
	Total int
}

// ScanEmptyDirs ищет пустые папки в указанных корнях.
// Безопасные папки (Desktop, Documents, Downloads и т.п.) пропускаются.
func ScanEmptyDirs(opts EmptyDirScanOptions) (*EmptyDirResult, error) {
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 10
	}

	junkFiles := map[string]bool{
		"thumbs.db": true, "desktop.ini": true, ".ds_store": true,
		"folder.jpg": true, "albumartsmall.jpg": true, "icon.ico": true,
	}
	if !opts.IgnoreJunk {
		junkFiles = nil
	}

	var out []EmptyDirCandidate

	for _, root := range opts.Roots {
		scanEmptyDirsRecursive(root, root, 0, opts.MaxDepth, junkFiles, &out)
	}

	return &EmptyDirResult{
		Dirs:  out,
		Total: len(out),
	}, nil
}

func scanEmptyDirsRecursive(root, current string, depth, maxDepth int, junkFiles map[string]bool, out *[]EmptyDirCandidate) {
	if depth > maxDepth {
		return
	}

	entries, err := os.ReadDir(current)
	if err != nil {
		return
	}

	// Пропускаем защищённые папки на первом уровне
	if depth == 1 {
		name := strings.ToLower(filepath.Base(current))
		if protectedEmptyDir(name) {
			return
		}
	}

	// Сначала рекурсивно обходим подпапки
	for _, e := range entries {
		if e.IsDir() {
			sub := filepath.Join(current, e.Name())
			scanEmptyDirsRecursive(root, sub, depth+1, maxDepth, junkFiles, out)
		}
	}

	// После рекурсии перечитываем — подпапки могли быть добавлены в список
	// Проверяем, пуста ли текущая папка
	if current == root {
		return // не добавляем сам корень
	}

	if isDirEmpty(current, junkFiles) {
		*out = append(*out, EmptyDirCandidate{
			Path:  current,
			Depth: depth,
		})
	}
}

// isDirEmpty проверяет, что папка пуста или содержит только junk-файлы.
func isDirEmpty(dir string, junkFiles map[string]bool) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			// Подпапка — если она не пуста, считаем текущую непустой
			sub := filepath.Join(dir, e.Name())
			if !isDirEmpty(sub, junkFiles) {
				return false
			}
			continue
		}
		name := strings.ToLower(e.Name())
		if junkFiles != nil && junkFiles[name] {
			continue // junk-файл, игнорируем
		}
		return false // реальный файл
	}
	return true
}

// protectedEmptyDir — папки, которые нельзя предлагать к удалению даже если пусты.
func protectedEmptyDir(name string) bool {
	protected := map[string]bool{
		"desktop": true, "documents": true, "downloads": true,
		"music": true, "pictures": true, "videos": true,
		"templates": true, "favorites": true, "links": true,
		"contacts": true, "searches": true, "saved games": true,
		"3d objects": true, "onedrive": true, "appdata": true,
		"local": true, "locallow": true, "roaming": true,
		"microsoft": true, "windows": true, "temp": true,
		"start menu": true, "programs": true, "startup": true,
		".git": true, ".vscode": true, ".ssh": true, ".config": true,
		"node_modules": true,
	}
	return protected[name]
}

// DeleteEmptyDir удаляет пустую папку безопасно.
func DeleteEmptyDir(path string) error {
	// Перепроверяем что папка пуста
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	// Удаляем junk-файлы сначала
	junk := map[string]bool{
		"thumbs.db": true, "desktop.ini": true, ".ds_store": true,
		"folder.jpg": true, "albumartsmall.jpg": true, "icon.ico": true,
	}
	for _, e := range entries {
		if !e.IsDir() && junk[strings.ToLower(e.Name())] {
			_ = os.Remove(filepath.Join(path, e.Name()))
		}
	}
	return os.RemoveAll(path)
}
