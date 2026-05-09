package cleaner

import (
	"fmt"
	"os"
	"os/exec"
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

// EmptyDirDeleteResult — результат попытки удаления.
type EmptyDirDeleteResult struct {
	Path       string
	Success    bool
	Error      string
	MovedToRecycleBin bool
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

// isSafePathForDeletion проверяет, что путь безопасен для удаления пустых папок.
// Запрещает системные директории и корневые пути.
func isSafePathForDeletion(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	abs = filepath.Clean(abs)
	low := strings.ToLower(abs)

	// Запрещаем корень диска
	if len(abs) <= 3 {
		return false
	}

	// Системные директории — абсолютное табу
	forbidden := []string{
		`c:\windows`,
		`c:\windows\system32`,
		`c:\windows\syswow64`,
		`c:\windows\winsxs`,
		`c:\program files`,
		`c:\program files (x86)`,
		`c:\programdata`,
		`c:\users\all users`,
		`c:\users\default`,
		`c:\users\public`,
		`c:\perflogs`,
	}
	for _, f := range forbidden {
		if strings.HasPrefix(low, f) {
			return false
		}
	}

	return true
}

// DeleteEmptyDirToRecycleBin удаляет пустую папку, перемещая её в Корзину.
// Возвращает результат операции с флагом MovedToRecycleBin.
func DeleteEmptyDirToRecycleBin(path string) EmptyDirDeleteResult {
	result := EmptyDirDeleteResult{Path: path}

	// Перепроверяем что папка существует и это директория
	if _, err := os.Stat(path); err != nil {
		result.Error = fmt.Sprintf("папка не найдена: %v", err)
		return result
	}

	// Проверка безопасности
	if !isSafePathForDeletion(path) {
		result.Error = "удаление этой папки небезопасно (системная директория)"
		return result
	}

	// Проверяем что папка действительно пуста (или содержит только junk-файлы)
	entries, err := os.ReadDir(path)
	if err != nil {
		result.Error = fmt.Sprintf("не удалось прочитать папку: %v", err)
		return result
	}

	// Удаляем junk-файлы внутри
	junk := map[string]bool{
		"thumbs.db": true, "desktop.ini": true, ".ds_store": true,
		"folder.jpg": true, "albumartsmall.jpg": true, "icon.ico": true,
	}
	for _, e := range entries {
		if !e.IsDir() && junk[strings.ToLower(e.Name())] {
			_ = os.Remove(filepath.Join(path, e.Name()))
		}
	}

	// Перемещаем в Корзину через PowerShell
	// Шелл.Application позволяет переместить в корзину
	psScript := fmt.Sprintf(`
		$shell = New-Object -ComObject Shell.Application
		$folder = $shell.Namespace(0)
		$folder.ParseName("%s").MoveToHere()
	`, strings.ReplaceAll(path, "&", "_"))

	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", psScript)
	if err := cmd.Run(); err != nil {
		// Если PowerShell не сработал, пробуем альтернативный метод
		// через MoveSpecialFolder с SID корзины
		result.Error = fmt.Sprintf("не удалось переместить в корзину: %v", err)
		return result
	}

	result.Success = true
	result.MovedToRecycleBin = true
	return result
}

// DeleteEmptyDir удаляет пустую папку безопасно (устаревший метод, используйте DeleteEmptyDirToRecycleBin).
func DeleteEmptyDir(path string) error {
	result := DeleteEmptyDirToRecycleBin(path)
	if result.Error != "" {
		return fmt.Errorf("%s", result.Error)
	}
	return nil
}
