package cleaner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ─── orphaned_apps.json schema ───

// OrphanApp описывает одну запись в orphaned_apps.json.
type OrphanApp struct {
	DisplayName     string   `json:"displayName"`
	Publisher       string   `json:"publisher,omitempty"`
	InstallPaths    []string `json:"installPaths"`
	AdditionalPaths []string `json:"additionalPaths,omitempty"`
	RegistryKeys    []string `json:"registryKeys,omitempty"`
	CachePaths      []string `json:"cachePaths,omitempty"`
	Notes           string   `json:"notes,omitempty"`
}

// OrphanConfigPath возвращает путь к orphaned_apps.json рядом с исполняемым файлом.
func OrphanConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "orphaned_apps.json"
	}
	return filepath.Join(filepath.Dir(exe), "orphaned_apps.json")
}

// CacheTargetsFromOrphanConfig создаёт Target-ы из cachePaths всех записей orphaned_apps.json.
// Эти таргеты можно добавить в обычный scan/clean.
func CacheTargetsFromOrphanConfig(cfg *OrphanConfig) []Target {
	if cfg == nil {
		return nil
	}
	var targets []Target
	for _, app := range cfg.Apps {
		if len(app.CachePaths) == 0 {
			continue
		}
		targets = append(targets, Target{
			ID:          "orphan-cache-" + sanitizeID(app.DisplayName),
			Name:        fmt.Sprintf("Кеш %s", app.DisplayName),
			Description: fmt.Sprintf("Кеш-файлы программы %s (безопасно).", app.DisplayName),
			Paths:       app.CachePaths,
			KeepRoot:    true,
		})
	}
	return targets
}

// sanitizeID превращает displayName в id-совместимую строку.
func sanitizeID(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, name)
	if len(name) > 30 {
		name = name[:30]
	}
	return name
}

// OrphanConfig — массив записей orphaned_apps.json.
type OrphanConfig struct {
	Apps []OrphanApp `json:"apps"`
}

// LoadOrphanConfig загружает orphaned_apps.json.
func LoadOrphanConfig(path string) (*OrphanConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("чтение %s: %w", path, err)
	}
	var cfg OrphanConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		// Попробуем как просто массив
		var apps []OrphanApp
		if err2 := json.Unmarshal(data, &apps); err2 != nil {
			return nil, fmt.Errorf("парсинг %s: %w", path, err)
		}
		cfg.Apps = apps
	}
	return &cfg, nil
}

// SaveOrphanConfig сохраняет orphaned_apps.json.
func SaveOrphanConfig(cfg *OrphanConfig, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o755)
	return os.WriteFile(path, data, 0o644)
}

// ─── Scan: проверка записей из JSON ───

// OrphanScanResult — результат сканирования одной записи.
type OrphanScanResult struct {
	App          OrphanApp
	FoundPaths   []OrphanFoundPath // подтверждённые пути на диске
	FoundRegKeys []string          // подтверждённые ключи реестра
	TotalSize    int64
	TotalFiles   int
}

// OrphanFoundPath — найденный путь с его размером.
type OrphanFoundPath struct {
	Path   string
	Size   int64
	Files  int
	Exists bool
}

// OrphanScan проверяет каждую запись из orphaned_apps.json.
// Возвращает только те, у которых найдены остатки на диске.
func OrphanScan(cfg *OrphanConfig, verbose bool) []OrphanScanResult {
	var mu sync.Mutex
	var wg sync.WaitGroup
	var results []OrphanScanResult

	for _, app := range cfg.Apps {
		wg.Add(1)
		go func(a OrphanApp) {
			defer wg.Done()
			r := scanOneOrphan(a)
			if len(r.FoundPaths) > 0 || len(r.FoundRegKeys) > 0 {
				mu.Lock()
				results = append(results, r)
				mu.Unlock()
			}
		}(app)
	}
	wg.Wait()
	return results
}

func scanOneOrphan(app OrphanApp) OrphanScanResult {
	r := OrphanScanResult{App: app}

	// Проверяем installPaths + additionalPaths + cachePaths
	allPaths := make([]string, 0, len(app.InstallPaths)+len(app.AdditionalPaths)+len(app.CachePaths))
	allPaths = append(allPaths, app.InstallPaths...)
	allPaths = append(allPaths, app.AdditionalPaths...)
	allPaths = append(allPaths, app.CachePaths...)

	for _, raw := range allPaths {
		p := ExpandPath(raw)
		if p == "" {
			continue
		}
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		fp := OrphanFoundPath{Path: p, Exists: true}
		if info.IsDir() {
			fp.Size, fp.Files = dirSizeWithTimeout(p, 3*time.Second)
		} else {
			fp.Size = info.Size()
			fp.Files = 1
		}
		r.FoundPaths = append(r.FoundPaths, fp)
		if fp.Size > 0 {
			r.TotalSize += fp.Size
		}
		r.TotalFiles += fp.Files
	}

	// Проверяем ключи реестра
	for _, key := range app.RegistryKeys {
		if registryKeyExists(key) {
			r.FoundRegKeys = append(r.FoundRegKeys, key)
		}
	}

	return r
}

func registryKeyExists(key string) bool {
	err := exec.Command("reg", "query", key, "/ve").Run()
	if err != nil {
		// Пробуем без /ve — может быть просто ключ без значения по умолчанию
		err = exec.Command("reg", "query", key).Run()
	}
	return err == nil
}

// ─── Discover: поиск неизвестных папок ───

// DiscoverOptions — параметры для команды discover.
type DiscoverOptions struct {
	Roots      []string      // корневые папки для сканирования
	OrphanCfg  *OrphanConfig // текущий orphaned_apps.json (для исключения)
	JSONOutput bool
	OutputFile string
}

// DiscoverResult — одна найденная подозрительная папка.
type DiscoverResult struct {
	Path          string `json:"path"`
	SizeMB        int64  `json:"size_mb"`
	SizeBytes     int64  `json:"size_bytes"`
	HasExecutable bool   `json:"has_executable"`
}

// DefaultDiscoverRoots — корневые папки по умолчанию.
func DefaultDiscoverRoots() []string {
	roots := []string{
		`C:\Program Files`,
		`C:\Program Files (x86)`,
		`C:\ProgramData`,
	}
	if appdata := ExpandPath(`%APPDATA%`); appdata != "" {
		roots = append(roots, appdata)
	}
	if localAppdata := ExpandPath(`%LOCALAPPDATA%`); localAppdata != "" {
		roots = append(roots, localAppdata)
	}
	return roots
}

// OrphanDiscover ищет неизвестные папки в стандартных расположениях.
func OrphanDiscover(opts DiscoverOptions) ([]DiscoverResult, error) {
	roots := opts.Roots
	if len(roots) == 0 {
		roots = DefaultDiscoverRoots()
	}

	// 1. Собираем множество известных путей
	knownPaths := buildKnownPaths(opts.OrphanCfg)
	whitelist := discoverWhitelist()
	installed := installedProgramNames()
	installPaths := installedProgramPaths()

	var mu sync.Mutex
	var wg sync.WaitGroup
	var results []DiscoverResult

	for _, root := range roots {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			found := discoverInRoot(r, knownPaths, whitelist, installed, installPaths)
			mu.Lock()
			results = append(results, found...)
			mu.Unlock()
		}(root)
	}
	wg.Wait()

	// Сохраняем в файл если указано
	if opts.OutputFile != "" {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return results, fmt.Errorf("JSON marshal: %w", err)
		}
		if err := os.WriteFile(opts.OutputFile, data, 0o644); err != nil {
			return results, fmt.Errorf("запись в %s: %w", opts.OutputFile, err)
		}
	}

	return results, nil
}

func buildKnownPaths(cfg *OrphanConfig) map[string]bool {
	known := make(map[string]bool)
	if cfg != nil {
		for _, app := range cfg.Apps {
			for _, p := range app.InstallPaths {
				expanded := ExpandPath(p)
				if expanded != "" {
					known[strings.ToLower(filepath.Clean(expanded))] = true
				}
			}
			for _, p := range app.AdditionalPaths {
				expanded := ExpandPath(p)
				if expanded != "" {
					known[strings.ToLower(filepath.Clean(expanded))] = true
				}
			}
			for _, p := range app.CachePaths {
				expanded := ExpandPath(p)
				if expanded != "" {
					known[strings.ToLower(filepath.Clean(expanded))] = true
				}
			}
		}
	}
	return known
}

func discoverInRoot(root string, knownPaths map[string]bool, whitelist map[string]bool, installed map[string]bool, installPaths map[string]bool) []DiscoverResult {
	var results []DiscoverResult

	entries, err := os.ReadDir(root)
	if err != nil {
		return results
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		low := strings.ToLower(name)

		// Пропускаем по белому списку
		if whitelist[low] {
			continue
		}
		// Пропускаем GUID-папки (начинаются с {)
		if strings.HasPrefix(name, "{") {
			continue
		}

		full := filepath.Join(root, name)
		fullLow := strings.ToLower(filepath.Clean(full))

		// Пропускаем если в known paths из JSON
		if knownPaths[fullLow] {
			continue
		}
		// Пропускаем если в install paths из реестра
		if installPaths[fullLow] {
			continue
		}
		// Пропускаем если совпадает с установленными программами
		if matchesInstalled(name, installed) {
			continue
		}

		// Считаем размер
		size, _ := dirSizeWithTimeout(full, 2*time.Second)
		if size == 0 {
			continue
		}

		hasExe := dirHasExecutable(full)

		results = append(results, DiscoverResult{
			Path:          full,
			SizeMB:        size / (1024 * 1024),
			SizeBytes:     size,
			HasExecutable: hasExe,
		})
	}

	return results
}

// dirHasExecutable проверяет, содержит ли папка .exe файлы (до 2 уровней вглубь).
func dirHasExecutable(root string) bool {
	depth := 0
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if found {
			return filepath.SkipAll
		}
		// Ограничиваем глубину
		rel, _ := filepath.Rel(root, path)
		depth = strings.Count(rel, string(filepath.Separator))
		if d.IsDir() && depth > 2 {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".exe") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// discoverWhitelist — папки, которые точно не мусор (системные/вендорские).
func discoverWhitelist() map[string]bool {
	list := []string{
		// Windows / системные
		"windowsapps", "microsoft", "common files", "internet explorer",
		"windows defender", "windows defender advanced threat protection",
		"windows mail", "windows media player", "windows multimedia platform",
		"windows nt", "windows photo viewer", "windows portable devices",
		"windows security", "windows sidebar", "windowspowershell",
		"microsoft.net", "msbuild", "reference assemblies", "dotnet",
		"iis", "iis express", "microsoft sdks", "microsoft sql server",
		"microsoft visual studio", "uninstall information", "windows kits",
		"package cache", "softwaredistrribution", "microsoft update health tools",
		// Вендоры
		"nvidia corporation", "nvidia", "realtek", "intel", "amd",
		"dell", "hp", "lenovo", "asus", "acer", "logitech", "razer",
		"google", "mozilla", "adobe", "apple", "oracle", "jetbrains",
		// AppData системные
		"microsoft", "windows", "packages", "temp", "tmp",
		"d3dscache", "connecteddevicesplatform", "comms", "crashdumps",
		"virtualstore", "programs", "application data", "history",
		"desktop", "downloads", "diagnosis", "publishers",
		".default", "default", "default user", "public",
		"ssh", "regid.1991-06.com.microsoft", "usoshared",
		"windowsholographicdevices",
		"local", "locallow", "roaming",
	}
	out := make(map[string]bool, len(list))
	for _, s := range list {
		out[s] = true
	}
	return out
}

// ─── Clean: удаление мусора для указанных программ ───

// OrphanCleanOptions — параметры очистки.
type OrphanCleanOptions struct {
	CacheOnly bool   // удалять только кеш (cachePaths) — безопасная операция
	Force     bool   // не запрашивать подтверждение
	Recycle   bool   // в корзину вместо удаления (через PowerShell)
	ExportReg string // экспорт реестра перед удалением
	Verbose   bool
	Logger    func(string)
}

// OrphanCleanResult — результат очистки одной программы.
type OrphanCleanResult struct {
	App          OrphanApp
	DeletedPaths []string
	DeletedKeys  []string
	FreedBytes   int64
	FreedFiles   int
	Errors       []string
}

// OrphanClean удаляет остатки указанных программ.
func OrphanClean(cfg *OrphanConfig, names []string, opts OrphanCleanOptions) []OrphanCleanResult {
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[strings.ToLower(n)] = true
	}

	var results []OrphanCleanResult
	for _, app := range cfg.Apps {
		if !nameSet[strings.ToLower(app.DisplayName)] {
			continue
		}
		r := cleanOneOrphan(app, opts)
		results = append(results, r)
	}
	return results
}

func cleanOneOrphan(app OrphanApp, opts OrphanCleanOptions) OrphanCleanResult {
	r := OrphanCleanResult{App: app}

	log := func(format string, a ...any) {
		if opts.Logger != nil {
			opts.Logger(fmt.Sprintf(format, a...))
		}
	}

	var pathsToClean []string

	if opts.CacheOnly {
		// Только кеш — безопасная операция
		pathsToClean = app.CachePaths
		log("  [кеш] Удаление только кеш-файлов для %s (безопасно)", app.DisplayName)
	} else {
		// Полная очистка — предупреждение
		log("  ⚠ ВНИМАНИЕ: полная очистка %s!", app.DisplayName)
		log("    Будут удалены настройки, профили и данные программы.")
		log("    Для безопасного удаления только кеша используйте --orphan-cache-only.")

		// Экспорт реестра перед удалением
		if opts.ExportReg != "" && len(app.RegistryKeys) > 0 {
			for _, key := range app.RegistryKeys {
				outFile := filepath.Join(opts.ExportReg,
					strings.ReplaceAll(strings.ReplaceAll(key, `\`, "_"), "/", "_")+".reg")
				cmd := exec.Command("reg", "export", key, outFile, "/y")
				if err := cmd.Run(); err != nil {
					r.Errors = append(r.Errors, fmt.Sprintf("экспорт реестра %s: %v", key, err))
				} else {
					log("  Экспортирован: %s → %s", key, outFile)
				}
			}
		}

		pathsToClean = make([]string, 0, len(app.InstallPaths)+len(app.AdditionalPaths)+len(app.CachePaths))
		pathsToClean = append(pathsToClean, app.InstallPaths...)
		pathsToClean = append(pathsToClean, app.AdditionalPaths...)
		pathsToClean = append(pathsToClean, app.CachePaths...)
	}

	for _, raw := range pathsToClean {
		p := ExpandPath(raw)
		if p == "" {
			continue
		}
		if !safePath(p) {
			r.Errors = append(r.Errors, fmt.Sprintf("небезопасный путь: %s", p))
			continue
		}

		info, err := os.Stat(p)
		if err != nil {
			continue
		}

		if opts.Recycle {
			if err := moveToRecycle(p); err != nil {
				r.Errors = append(r.Errors, fmt.Sprintf("корзина %s: %v", p, err))
				continue
			}
		} else {
			if info.IsDir() {
				if err := os.RemoveAll(p); err != nil {
					r.Errors = append(r.Errors, fmt.Sprintf("rmdir %s: %v", p, err))
					continue
				}
			} else {
				if err := os.Remove(p); err != nil {
					r.Errors = append(r.Errors, fmt.Sprintf("rm %s: %v", p, err))
					continue
				}
			}
		}

		if info.IsDir() {
			size, files := dirSize(p)
			r.FreedBytes += size
			r.FreedFiles += files
		} else {
			r.FreedBytes += info.Size()
			r.FreedFiles++
		}
		r.DeletedPaths = append(r.DeletedPaths, p)
		if opts.Verbose {
			log("  Удалён: %s", p)
		}
	}

	// Удаляем ключи реестра (только при полной очистке)
	if opts.CacheOnly {
		return r
	}
	for _, key := range app.RegistryKeys {
		if !registryKeyExists(key) {
			continue
		}
		cmd := exec.Command("reg", "delete", key, "/f")
		if err := cmd.Run(); err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("reg delete %s: %v", key, err))
			continue
		}
		r.DeletedKeys = append(r.DeletedKeys, key)
		if opts.Verbose {
			log("  Ключ реестра удалён: %s", key)
		}
	}

	return r
}

// moveToRecycle перемещает файл/папку в корзину через PowerShell.
func moveToRecycle(path string) error {
	psCmd := fmt.Sprintf(`Add-Type -AssemblyName Microsoft.VisualBasic; [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteFile('%s', 'OnlyErrorDialogs', 'SendToRecycleBin')`,
		strings.ReplaceAll(path, "'", "''"))
	cmd := exec.Command("powershell", "-NoProfile", "-Command", psCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ─── Info: подробная информация ───

// OrphanInfo возвращает детальную информацию по программе из результатов scan.
func OrphanInfo(scanResults []OrphanScanResult, displayName string) *OrphanScanResult {
	low := strings.ToLower(displayName)
	for _, r := range scanResults {
		if strings.ToLower(r.App.DisplayName) == low {
			return &r
		}
	}
	return nil
}
