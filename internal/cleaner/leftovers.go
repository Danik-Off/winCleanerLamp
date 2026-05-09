package cleaner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// LeftoverType — тип найденного остатка.
type LeftoverType string

const (
	LeftoverFolder   LeftoverType = "folder"   // папка в AppData/ProgramData/Program Files
	LeftoverRegistry LeftoverType = "registry" // ключ реестра без программы-владельца
	LeftoverEmpty    LeftoverType = "empty"    // пустая или почти пустая папка
)

// LeftoverCandidate — папка/ключ, похожие на остаток от удалённой программы.
type LeftoverCandidate struct {
	Path           string
	Size           int64
	Files          int
	Reason         string       // почему помечена (например, "нет в Uninstall registry")
	Type           LeftoverType // тип остатка
	OrphanMatch    string       // имя программы из orphaned_apps.json (если совпало)
	CacheHit       bool         // это кеш из orphan DB (cachePaths)
	InstalledMatch bool         // программа установлена в системе
}

// InstalledProgram — запись об установленной программе.
type InstalledProgram struct {
	DisplayName     string `json:"displayName"`
	Publisher       string `json:"publisher,omitempty"`
	InstallLocation string `json:"installLocation,omitempty"`
	InOrphanDB      bool   `json:"inOrphanDB"`
}

// LeftoversResult — полный результат сканирования остатков.
type LeftoversResult struct {
	Candidates []LeftoverCandidate
	Installed  []InstalledProgram
}

// LeftoverScanOptions — параметры сканирования.
type LeftoverScanOptions struct {
	OrphanCfg *OrphanConfig // orphaned_apps.json (может быть nil)
	LogFile   string        // файл для логирования новых находок (если не пусто)
}

// ScanLeftovers ищет остатки удалённых программ.
// ВНИМАНИЕ: это эвристика — возвращается только для просмотра, не удаляется автоматически.
func ScanLeftovers() ([]LeftoverCandidate, error) {
	r, err := ScanLeftoversEx(LeftoverScanOptions{})
	if err != nil {
		return nil, err
	}
	return r.Candidates, nil
}

// ScanLeftoversEx — расширенное сканирование остатков с поддержкой orphaned_apps.json.
func ScanLeftoversEx(opts LeftoverScanOptions) (*LeftoversResult, error) {
	installed := installedProgramNames()
	installPaths := installedProgramPaths()
	whitelist := knownSystemFolders()

	// Собираем orphan lookup из JSON
	orphanPathIndex := buildOrphanPathIndex(opts.OrphanCfg)

	var mu sync.Mutex
	var wg sync.WaitGroup
	var out []LeftoverCandidate

	// 1. Ассоциативный поиск: AppData + ProgramData
	appDataRoots := []string{
		ExpandPath(`%APPDATA%`),
		ExpandPath(`%LOCALAPPDATA%`),
		`C:\ProgramData`,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		results := scanAppDataLeftovers(appDataRoots, installed, whitelist)
		mu.Lock()
		out = append(out, results...)
		mu.Unlock()
	}()

	// 2. Program Files: папки без записи в Uninstall
	progRoots := []string{
		`C:\Program Files`,
		`C:\Program Files (x86)`,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		results := scanProgramFilesLeftovers(progRoots, installed, installPaths, whitelist)
		mu.Lock()
		out = append(out, results...)
		mu.Unlock()
	}()

	// 3. Реестр: ключи HKCU\Software без программ
	wg.Add(1)
	go func() {
		defer wg.Done()
		results := scanRegistryLeftovers(installed, whitelist)
		mu.Lock()
		out = append(out, results...)
		mu.Unlock()
	}()

	// 4. Пустые папки
	wg.Add(1)
	go func() {
		defer wg.Done()
		results := scanEmptyFolders(appDataRoots)
		mu.Lock()
		out = append(out, results...)
		mu.Unlock()
	}()

	// 5. Кеш из orphan.json (cachePaths установленных программ)
	if opts.OrphanCfg != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results := scanOrphanCachePaths(opts.OrphanCfg)
			mu.Lock()
			out = append(out, results...)
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Помечаем кандидатов по orphan DB
	for i := range out {
		if out[i].Type == LeftoverFolder || out[i].Type == LeftoverRegistry {
			if match, ok := orphanPathIndex[strings.ToLower(filepath.Clean(out[i].Path))]; ok {
				out[i].OrphanMatch = match.displayName
				out[i].CacheHit = match.isCache
				out[i].Reason = fmt.Sprintf("известно из orphan DB: %s", match.displayName)
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return typeOrder(out[i].Type) < typeOrder(out[j].Type)
		}
		return out[i].Size > out[j].Size
	})

	// Логирование новых находок
	if opts.LogFile != "" {
		logUnknownLeftovers(out, opts.LogFile)
	}

	// Список установленных программ
	programs := GetInstalledPrograms(opts.OrphanCfg)

	return &LeftoversResult{
		Candidates: out,
		Installed:  programs,
	}, nil
}

// orphanPathMatch — соответствие пути записи в orphan DB.
type orphanPathMatch struct {
	displayName string
	isCache     bool
}

// buildOrphanPathIndex собирает индекс путей из orphan DB для быстрого lookup.
func buildOrphanPathIndex(cfg *OrphanConfig) map[string]orphanPathMatch {
	idx := make(map[string]orphanPathMatch)
	if cfg == nil {
		return idx
	}
	for _, app := range cfg.Apps {
		for _, p := range app.InstallPaths {
			if exp := ExpandPath(p); exp != "" {
				idx[strings.ToLower(filepath.Clean(exp))] = orphanPathMatch{app.DisplayName, false}
			}
		}
		for _, p := range app.AdditionalPaths {
			if exp := ExpandPath(p); exp != "" {
				idx[strings.ToLower(filepath.Clean(exp))] = orphanPathMatch{app.DisplayName, false}
			}
		}
		for _, p := range app.CachePaths {
			if exp := ExpandPath(p); exp != "" {
				idx[strings.ToLower(filepath.Clean(exp))] = orphanPathMatch{app.DisplayName, true}
			}
		}
	}
	return idx
}

// scanOrphanCachePaths сканирует cachePaths из orphan DB и возвращает найденные.
func scanOrphanCachePaths(cfg *OrphanConfig) []LeftoverCandidate {
	var out []LeftoverCandidate
	if cfg == nil {
		return out
	}
	for _, app := range cfg.Apps {
		for _, raw := range app.CachePaths {
			p := ExpandPath(raw)
			if p == "" {
				continue
			}
			info, err := os.Stat(p)
			if err != nil {
				continue
			}
			var size int64
			var files int
			if info.IsDir() {
				size, files = dirSizeWithTimeout(p, 2*time.Second)
			} else {
				size = info.Size()
				files = 1
			}
			if size == 0 && files == 0 {
				continue
			}
			out = append(out, LeftoverCandidate{
				Path:        p,
				Size:        size,
				Files:       files,
				Reason:      fmt.Sprintf("кеш %s (orphan DB)", app.DisplayName),
				Type:        LeftoverFolder,
				OrphanMatch: app.DisplayName,
				CacheHit:    true,
			})
		}
	}
	return out
}

// GetInstalledPrograms возвращает список установленных программ из реестра.
func GetInstalledPrograms(orphanCfg *OrphanConfig) []InstalledProgram {
	orphanNames := make(map[string]bool)
	if orphanCfg != nil {
		for _, app := range orphanCfg.Apps {
			orphanNames[strings.ToLower(app.DisplayName)] = true
		}
	}

	keys := []string{
		`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`HKLM\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
		`HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
	}
	seen := make(map[string]bool)
	var programs []InstalledProgram
	for _, k := range keys {
		out, err := exec.Command("reg", "query", k, "/s").Output()
		if err != nil {
			continue
		}
		var curName, curPublisher, curLocation string
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				if curName != "" && !seen[strings.ToLower(curName)] {
					seen[strings.ToLower(curName)] = true
					programs = append(programs, InstalledProgram{
						DisplayName:     curName,
						Publisher:       curPublisher,
						InstallLocation: curLocation,
						InOrphanDB:      orphanNames[strings.ToLower(curName)],
					})
				}
				curName, curPublisher, curLocation = "", "", ""
				continue
			}
			if strings.HasPrefix(line, "DisplayName") {
				if idx := strings.Index(line, "REG_SZ"); idx >= 0 {
					curName = strings.TrimSpace(line[idx+len("REG_SZ"):])
				}
			}
			if strings.HasPrefix(line, "Publisher") {
				if idx := strings.Index(line, "REG_SZ"); idx >= 0 {
					curPublisher = strings.TrimSpace(line[idx+len("REG_SZ"):])
				}
			}
			if strings.HasPrefix(line, "InstallLocation") {
				if idx := strings.Index(line, "REG_SZ"); idx >= 0 {
					curLocation = strings.TrimSpace(line[idx+len("REG_SZ"):])
				}
			}
		}
		// flush last entry
		if curName != "" && !seen[strings.ToLower(curName)] {
			seen[strings.ToLower(curName)] = true
			programs = append(programs, InstalledProgram{
				DisplayName:     curName,
				Publisher:       curPublisher,
				InstallLocation: curLocation,
				InOrphanDB:      orphanNames[strings.ToLower(curName)],
			})
		}
	}
	sort.Slice(programs, func(i, j int) bool {
		return strings.ToLower(programs[i].DisplayName) < strings.ToLower(programs[j].DisplayName)
	})
	return programs
}

// logUnknownLeftovers записывает находки, которых нет в orphan DB, в лог-файл.
func logUnknownLeftovers(candidates []LeftoverCandidate, logFile string) {
	type logEntry struct {
		Path   string `json:"path"`
		Size   int64  `json:"size_bytes"`
		Files  int    `json:"files"`
		Reason string `json:"reason"`
		Type   string `json:"type"`
	}
	var unknowns []logEntry
	for _, c := range candidates {
		if c.OrphanMatch == "" && c.Type == LeftoverFolder {
			unknowns = append(unknowns, logEntry{
				Path:   c.Path,
				Size:   c.Size,
				Files:  c.Files,
				Reason: c.Reason,
				Type:   string(c.Type),
			})
		}
	}
	if len(unknowns) == 0 {
		return
	}
	data, err := json.MarshalIndent(unknowns, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(logFile), 0o755)
	_ = os.WriteFile(logFile, data, 0o644)
}

func typeOrder(t LeftoverType) int {
	switch t {
	case LeftoverFolder:
		return 0
	case LeftoverEmpty:
		return 1
	case LeftoverRegistry:
		return 2
	}
	return 3
}

// scanAppDataLeftovers — ассоциативный поиск в AppData/ProgramData.
func scanAppDataLeftovers(roots []string, installed map[string]bool, whitelist map[string]bool) []LeftoverCandidate {
	var out []LeftoverCandidate
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			low := strings.ToLower(name)
			if whitelist[low] {
				continue
			}
			if matchesInstalled(name, installed) {
				continue
			}
			full := filepath.Join(root, name)
			size, files := dirSizeWithTimeout(full, 2*time.Second)
			if size == 0 && files == 0 {
				continue
			}
			out = append(out, LeftoverCandidate{
				Path:   full,
				Size:   size,
				Files:  files,
				Reason: "нет в списке установленных программ",
				Type:   LeftoverFolder,
			})
		}
	}
	return out
}

// scanProgramFilesLeftovers — поиск папок в Program Files без записи в Uninstall.
func scanProgramFilesLeftovers(roots []string, installed map[string]bool, installPaths map[string]bool, whitelist map[string]bool) []LeftoverCandidate {
	var out []LeftoverCandidate
	pfWhitelist := programFilesWhitelist()
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			low := strings.ToLower(name)
			if whitelist[low] || pfWhitelist[low] {
				continue
			}
			full := filepath.Join(root, name)
			// Проверяем по путям установки из реестра
			if installPaths[strings.ToLower(full)] {
				continue
			}
			if matchesInstalled(name, installed) {
				continue
			}
			size, files := dirSizeWithTimeout(full, 2*time.Second)
			if size == 0 && files == 0 {
				continue
			}
			out = append(out, LeftoverCandidate{
				Path:   full,
				Size:   size,
				Files:  files,
				Reason: "нет в Uninstall реестре",
				Type:   LeftoverFolder,
			})
		}
	}
	return out
}

// scanRegistryLeftovers — ищет ключи HKCU\Software, которые не соответствуют установленным программам.
func scanRegistryLeftovers(installed map[string]bool, whitelist map[string]bool) []LeftoverCandidate {
	var out []LeftoverCandidate
	regRoots := []string{
		`HKCU\Software`,
	}
	for _, regRoot := range regRoots {
		cmdOut, err := exec.Command("reg", "query", regRoot).Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(cmdOut), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || !strings.Contains(strings.ToUpper(line), strings.ToUpper(regRoot)) {
				continue
			}
			parts := strings.Split(line, `\`)
			if len(parts) < 3 {
				continue
			}
			keyName := parts[len(parts)-1]
			low := strings.ToLower(strings.TrimSpace(keyName))
			if low == "" {
				continue
			}
			if whitelist[low] || registryWhitelist()[low] {
				continue
			}
			if matchesInstalled(keyName, installed) {
				continue
			}
			out = append(out, LeftoverCandidate{
				Path:   line,
				Size:   0,
				Files:  0,
				Reason: "ключ реестра без установленной программы",
				Type:   LeftoverRegistry,
			})
		}
	}
	return out
}

// scanEmptyFolders — ищет пустые или почти пустые папки (только служебные файлы).
func scanEmptyFolders(roots []string) []LeftoverCandidate {
	var out []LeftoverCandidate
	junkFiles := map[string]bool{
		"thumbs.db": true, "desktop.ini": true, ".ds_store": true,
		"folder.jpg": true, "albumartsmall.jpg": true,
	}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			full := filepath.Join(root, e.Name())
			if isDirEffectivelyEmpty(full, junkFiles) {
				out = append(out, LeftoverCandidate{
					Path:   full,
					Size:   0,
					Files:  0,
					Reason: "пустая папка",
					Type:   LeftoverEmpty,
				})
			}
		}
	}
	return out
}

// isDirEffectivelyEmpty проверяет, что папка пуста или содержит только служебные файлы.
func isDirEffectivelyEmpty(path string, junkFiles map[string]bool) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			sub := filepath.Join(path, e.Name())
			if !isDirEffectivelyEmpty(sub, junkFiles) {
				return false
			}
			continue
		}
		if !junkFiles[strings.ToLower(e.Name())] {
			return false
		}
	}
	return true
}

// dirSizeWithTimeout считает размер папки с таймаутом.
func dirSizeWithTimeout(path string, timeout time.Duration) (int64, int) {
	type result struct {
		size  int64
		files int
	}
	ch := make(chan result, 1)
	go func() {
		s, f := dirSize(path)
		ch <- result{s, f}
	}()
	select {
	case r := <-ch:
		return r.size, r.files
	case <-time.After(timeout):
		return -1, 0
	}
}

// installedProgramNames читает DisplayName из реестра (Uninstall и Publisher папки).
func installedProgramNames() map[string]bool {
	keys := []string{
		`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`HKLM\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
		`HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
	}
	result := map[string]bool{}
	for _, k := range keys {
		out, err := exec.Command("reg", "query", k, "/s", "/v", "DisplayName").Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			// формат: "    DisplayName    REG_SZ    My App Name"
			if !strings.HasPrefix(line, "DisplayName") {
				continue
			}
			idx := strings.Index(line, "REG_SZ")
			if idx < 0 {
				continue
			}
			name := strings.TrimSpace(line[idx+len("REG_SZ"):])
			if name == "" {
				continue
			}
			for _, tok := range tokenize(name) {
				result[tok] = true
			}
		}
	}
	// Также добавим вендоров из HKLM\SOFTWARE (первый уровень) — часто имя папки AppData совпадает с вендором.
	if out, err := exec.Command("reg", "query", `HKLM\SOFTWARE`).Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(strings.ToUpper(line), "HKEY_LOCAL_MACHINE\\SOFTWARE\\") {
				continue
			}
			parts := strings.Split(line, `\`)
			if len(parts) == 0 {
				continue
			}
			last := strings.ToLower(strings.TrimSpace(parts[len(parts)-1]))
			if last != "" {
				result[last] = true
			}
		}
	}
	return result
}

// installedProgramPaths читает InstallLocation из реестра Uninstall.
func installedProgramPaths() map[string]bool {
	keys := []string{
		`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`HKLM\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
		`HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
	}
	result := map[string]bool{}
	for _, k := range keys {
		out, err := exec.Command("reg", "query", k, "/s", "/v", "InstallLocation").Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "InstallLocation") {
				continue
			}
			idx := strings.Index(line, "REG_SZ")
			if idx < 0 {
				continue
			}
			loc := strings.TrimSpace(line[idx+len("REG_SZ"):])
			if loc == "" {
				continue
			}
			result[strings.ToLower(filepath.Clean(loc))] = true
		}
	}
	return result
}

// tokenize режет DisplayName на значимые токены (слова >2 символов, lowercase).
func tokenize(s string) []string {
	s = strings.ToLower(s)
	var out []string
	var b strings.Builder
	flush := func() {
		w := b.String()
		b.Reset()
		if len(w) >= 3 {
			out = append(out, w)
		}
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	// Плюс целая строка без пробелов
	out = append(out, strings.ReplaceAll(s, " ", ""))
	return out
}

// matchesInstalled: true если имя папки пересекается с любым токеном установленного ПО.
func matchesInstalled(folder string, installed map[string]bool) bool {
	toks := tokenize(folder)
	for _, t := range toks {
		if installed[t] {
			return true
		}
	}
	// также: подстрочная проверка
	low := strings.ToLower(folder)
	for k := range installed {
		if len(k) >= 4 && strings.Contains(low, k) {
			return true
		}
	}
	return false
}

// knownSystemFolders — имена папок в AppData/ProgramData, которые не являются остатками программ.
func knownSystemFolders() map[string]bool {
	list := []string{
		// общие системные / встроенные
		"microsoft", "windows", "windowsapps", "packages", "temp", "tmp",
		"d3dscache", "connecteddevicesplatform", "comms", "crashdumps",
		"virtualstore", "programs", "application data", "history", "desktop",
		"downloaded installations", "downloads", "diagnosis", "publishers",
		".default", "default", "default user", "public",
		// ProgramData системные
		"ssh", "regid.1991-06.com.microsoft", "usoshared",
		"package cache", "softwaredistrribution", "windows defender",
		"windows security health", "windowsholographicdevices",
		// популярные вендоры, у которых AppData-папка остаётся навсегда
		"adobe", "google", "mozilla", "apple", "realtek", "intel", "nvidia",
		"amd", "dell", "hp", "lenovo", "asus", "acer", "logitech", "razer",
		"oracle", "ibm", "sun", "jetbrains", "docker", "kubernetes",
		"notepad++", "vlc", "7-zip", "git", "github",
		// часто встречающиеся
		"local", "locallow", "roaming",
	}
	out := make(map[string]bool, len(list))
	for _, s := range list {
		out[s] = true
	}
	return out
}

// programFilesWhitelist — папки в Program Files, которые точно системные.
func programFilesWhitelist() map[string]bool {
	list := []string{
		"common files", "internet explorer", "microsoft update health tools",
		"windows defender", "windows defender advanced threat protection",
		"windows mail", "windows media player", "windows multimedia platform",
		"windows nt", "windows photo viewer", "windows portable devices",
		"windows security", "windows sidebar", "windowsapps",
		"windowspowershell", "microsoft.net", "msbuild", "reference assemblies",
		"dotnet", "iis", "iis express", "microsoft sdks", "microsoft sql server",
		"microsoft visual studio", "uninstall information",
		"windows kits", "nvidia corporation", "realtek", "intel",
		"amd", "dell", "hp", "lenovo",
	}
	out := make(map[string]bool, len(list))
	for _, s := range list {
		out[s] = true
	}
	return out
}

// registryWhitelist — ключи HKCU\Software, которые всегда присутствуют.
func registryWhitelist() map[string]bool {
	list := []string{
		"microsoft", "classes", "policies", "registeredapplications",
		"wine", "wow6432node", "defaultuserext", "im providers",
		"appdata", "intel", "nvidia corporation", "realtek",
		"khronos", "opengl", "amd", "google", "mozilla",
		"adobe", "apple", "apple computer, inc.", "apple inc.",
		"java", "javafx", "javasoft", "oracle",
	}
	out := make(map[string]bool, len(list))
	for _, s := range list {
		out[s] = true
	}
	return out
}
