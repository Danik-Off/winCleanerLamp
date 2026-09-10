package cleaner

import (
	"encoding/json"
	"fmt"
	"os"
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
	Path        string       `json:"path"`
	Size        int64        `json:"size"`
	Files       int          `json:"files"`
	Reason      string       `json:"reason"` // почему помечена (например, "нет в Uninstall registry")
	Type        LeftoverType `json:"type"`
	OrphanMatch string       `json:"orphanMatch,omitempty"` // имя программы из orphaned_apps.json (если совпало)
	CacheHit    bool         `json:"cacheHit"`              // это кеш из orphan DB (cachePaths)
	// InstalledMatch — программа, которой принадлежит запись, СЕЙЧАС установлена
	// в системе. false = программа удалена — приоритетный кандидат на очистку.
	// Значимо только когда OrphanMatch непусто.
	InstalledMatch bool `json:"installedMatch"`
	// LikelyUserData — путь похож на пользовательские данные (см. isLikelyUserDataPath
	// в orphan.go) — предупреждение для UI, не блокирует показ/выбор.
	LikelyUserData bool `json:"likelyUserData"`
}

// InstalledProgram — запись об установленной программе.
type InstalledProgram struct {
	DisplayName     string `json:"displayName"`
	Publisher       string `json:"publisher,omitempty"`
	InstallLocation string `json:"installLocation,omitempty"`
	InOrphanDB      bool   `json:"inOrphanDB"`
	// UninstallString — команда деинсталляции из реестра Uninstall (та же,
	// что использует "Программы и компоненты"). Пусто, если у записи нет
	// UninstallString (бывает у некоторых компонентов без собственного
	// деинсталлятора).
	UninstallString string `json:"uninstallString,omitempty"`
}

// LeftoversResult — полный результат сканирования остатков.
type LeftoversResult struct {
	Candidates []LeftoverCandidate `json:"candidates"`
	Installed  []InstalledProgram  `json:"installed"`
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

	// 1. Ассоциативный поиск в пользовательских каталогах данных:
	// AppData/ProgramData в Windows, ~/.config + ~/.local/share в Linux,
	// ~/Library/Application Support в macOS (см. leftoverUserDataRoots).
	appDataRoots := leftoverUserDataRoots()
	wg.Add(1)
	go func() {
		defer wg.Done()
		results := scanAppDataLeftovers(appDataRoots, installed, whitelist)
		mu.Lock()
		out = append(out, results...)
		mu.Unlock()
	}()

	// 2. Каталоги установленных программ: папки, которым не соответствует ни
	// одна известная системе программа (Program Files в Windows, /opt и
	// ~/.local/share/applications в Linux, /Applications в macOS).
	progRoots := leftoverProgramRoots()
	wg.Add(1)
	go func() {
		defer wg.Done()
		results := scanProgramFilesLeftovers(progRoots, installed, installPaths, whitelist)
		mu.Lock()
		out = append(out, results...)
		mu.Unlock()
	}()

	// 3. Записи вне файлов: ключи HKCU\Software без программ в Windows,
	// осиротевшие .desktop и юниты автозапуска в Linux/macOS.
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

	// Ключевая проверка по запросу пользователя: для каждой записи с
	// известным приложением (OrphanMatch) — установлена ли программа СЕЙЧАС.
	// Это позволяет в первую очередь показывать/предлагать к удалению именно
	// остатки уже УДАЛЁННЫХ программ, а не просто делить на "известно/неизвестно".
	//
	// LikelyUserData — та же эвристика, что и в OrphanCleaner (orphan.go),
	// но здесь применяется КО ВСЕМ кандидатам, включая unknownFolders —
	// у "неизвестных" папок нет привязки к программе, но их последний
	// сегмент пути всё равно может выглядеть как сохранения/проекты/фото,
	// и на это стоит явно указать пользователю перед удалением.
	for i := range out {
		if out[i].OrphanMatch != "" {
			out[i].InstalledMatch = matchesInstalled(out[i].OrphanMatch, installed)
		}
		if out[i].Type == LeftoverFolder {
			out[i].LikelyUserData = isLikelyUserDataPath(out[i].Path)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return typeOrder(out[i].Type) < typeOrder(out[j].Type)
		}
		// Внутри одной группы — сначала остатки уже удалённых программ.
		if out[i].OrphanMatch != "" && out[j].OrphanMatch != "" && out[i].InstalledMatch != out[j].InstalledMatch {
			return !out[i].InstalledMatch
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

// nonEmpty убирает из списка путей пустые строки — ExpandPath возвращает "",
// если переменной окружения нет (например, %PROGRAMFILES(X86)% на 32-битной
// системе или windows-переменная в Linux-ядре).
func nonEmpty(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
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
