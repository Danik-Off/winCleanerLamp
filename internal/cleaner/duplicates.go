package cleaner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// DuplicateGroup — группа файлов-дубликатов с одинаковым содержимым.
type DuplicateGroup struct {
	Hash      string   `json:"hash"`      // SHA-256 полного файла
	Size      int64    `json:"size"`      // размер каждого файла
	Paths     []string `json:"paths"`     // пути к дубликатам (≥2)
	WasteSize int64    `json:"wasteSize"` // Size * (len(Paths)-1) — экономия при удалении лишних
	// RiskFlag непусто, если группа содержит файлы вне пользовательских папок
	// (Program Files, Windows) и/или исполняемые/библиотечные файлы —
	// совпадение содержимого НЕ означает, что файл можно безопасно удалить:
	// это может быть общий рантайм/DLL, используемый несколькими программами.
	RiskFlag string `json:"riskFlag,omitempty"`
}

// DuplicateScanOptions — параметры сканирования дубликатов.
type DuplicateScanOptions struct {
	Roots      []string // корневые папки для сканирования
	MinSize    int64    // минимальный размер файла (по умолчанию 1 КБ)
	MaxSize    int64    // максимальный размер (0 = без ограничения)
	Extensions []string // только эти расширения (пусто = все)
	Timeout    time.Duration
	// AllowSystemDirs разрешает сканировать корни внутри системных/установочных
	// каталогов (Program Files, Windows, ProgramData). По умолчанию false —
	// такие корни пропускаются с предупреждением, чтобы не предлагать
	// "дубликаты" общих DLL/рантаймов разных программ к удалению.
	AllowSystemDirs bool
}

// DuplicateScanResult — результат сканирования дубликатов.
type DuplicateScanResult struct {
	Groups          []DuplicateGroup `json:"groups"`
	TotalWaste      int64            `json:"totalWaste"`
	TotalFiles      int              `json:"totalFiles"`
	ScannedFiles    int              `json:"scannedFiles"`
	DurationSeconds float64          `json:"durationSeconds"`
	// SkippedRoots — корни, пропущенные как системные (см. AllowSystemDirs).
	SkippedRoots []string `json:"skippedRoots,omitempty"`
}

// ScanDuplicates ищет дубликаты файлов по алгоритму:
// 1. Группировка по размеру
// 2. Частичный хэш (первые 4 КБ) для предварительной фильтрации
// 3. Полный SHA-256 для подтверждения
func ScanDuplicates(opts DuplicateScanOptions) (*DuplicateScanResult, error) {
	start := time.Now()

	if opts.MinSize <= 0 {
		opts.MinSize = 1024 // 1 КБ по умолчанию
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Minute
	}

	deadline := time.After(opts.Timeout)

	// Шаг 1: Сбор файлов и группировка по размеру
	sizeGroups := make(map[int64][]string)
	var scanned int
	extFilter := make(map[string]bool)
	for _, ext := range opts.Extensions {
		extFilter[strings.ToLower(strings.TrimPrefix(ext, "."))] = true
	}

	var skippedRoots []string
	for _, root := range opts.Roots {
		if !opts.AllowSystemDirs && isSystemDirRoot(root) {
			skippedRoots = append(skippedRoots, root)
			continue
		}
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // пропускаем недоступные
			}
			// Проверяем таймаут каждые 1000 файлов
			if scanned%1000 == 0 {
				select {
				case <-deadline:
					return filepath.SkipAll
				default:
				}
			}
			if info.IsDir() {
				// Пропускаем системные/скрытые папки
				low := strings.ToLower(info.Name())
				if skipDir(low) {
					return filepath.SkipDir
				}
				return nil
			}
			size := info.Size()
			if size < opts.MinSize {
				return nil
			}
			if opts.MaxSize > 0 && size > opts.MaxSize {
				return nil
			}
			if len(extFilter) > 0 {
				ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
				if !extFilter[ext] {
					return nil
				}
			}
			sizeGroups[size] = append(sizeGroups[size], path)
			scanned++
			return nil
		})
	}

	// Убираем группы с одним файлом
	for size, paths := range sizeGroups {
		if len(paths) < 2 {
			delete(sizeGroups, size)
		}
	}

	// Шаг 2: Частичный хэш (первые 4 КБ) для каждой группы
	type hashGroup struct {
		hash  string
		paths []string
		size  int64
	}
	var candidates []hashGroup

	for size, paths := range sizeGroups {
		partialGroups := make(map[string][]string)
		for _, p := range paths {
			h := partialHash(p, 4096)
			if h != "" {
				partialGroups[h] = append(partialGroups[h], p)
			}
		}
		for h, ps := range partialGroups {
			if len(ps) >= 2 {
				candidates = append(candidates, hashGroup{hash: h, paths: ps, size: size})
			}
		}
	}

	// Шаг 3: Полный SHA-256 (параллельно)
	var mu sync.Mutex
	var wg sync.WaitGroup
	var groups []DuplicateGroup
	sem := make(chan struct{}, 4) // ограничиваем параллелизм IO

	// Инкрементальный кэш хэшей — на повторном скане того же диска не
	// перечитываем и не хэшируем файлы, которые не изменились (совпадают
	// размер и время модификации). На больших деревьях (десятки/сотни ГБ)
	// это превращает "минуты" в "секунды" при повторных запусках.
	hashCache := loadDupHashCache()
	var cacheMu sync.Mutex

	for _, cand := range candidates {
		wg.Add(1)
		go func(c hashGroup) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fullGroups := make(map[string][]string)
			for _, p := range c.paths {
				h := fullHashCached(p, c.size, hashCache, &cacheMu)
				if h != "" {
					fullGroups[h] = append(fullGroups[h], p)
				}
			}
			mu.Lock()
			for h, ps := range fullGroups {
				if len(ps) >= 2 {
					waste := c.size * int64(len(ps)-1)
					groups = append(groups, DuplicateGroup{
						Hash:      h,
						Size:      c.size,
						Paths:     ps,
						WasteSize: waste,
						RiskFlag:  duplicateGroupRiskFlag(ps),
					})
				}
			}
			mu.Unlock()
		}(cand)
	}
	wg.Wait()
	saveDupHashCache(hashCache)

	// Сортировка по WasteSize (убывание)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].WasteSize > groups[j].WasteSize
	})

	var totalWaste int64
	var totalFiles int
	for _, g := range groups {
		totalWaste += g.WasteSize
		totalFiles += len(g.Paths)
	}

	return &DuplicateScanResult{
		Groups:          groups,
		TotalWaste:      totalWaste,
		TotalFiles:      totalFiles,
		ScannedFiles:    scanned,
		DurationSeconds: time.Since(start).Seconds(),
		SkippedRoots:    skippedRoots,
	}, nil
}

// isSystemDirRoot проверяет, что корень сканирования лежит внутри
// системного/установочного каталога (Program Files, Windows, ProgramData) —
// такие корни по умолчанию исключаются из поиска дубликатов, см.
// DuplicateScanOptions.AllowSystemDirs.
func isSystemDirRoot(root string) bool {
	abs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	low := strings.ToLower(filepath.Clean(abs))
	systemRoots := []string{
		`c:\windows`,
		`c:\program files`,
		`c:\program files (x86)`,
		`c:\programdata`,
	}
	for _, s := range systemRoots {
		if low == s || strings.HasPrefix(low, s+`\`) {
			return true
		}
	}
	return false
}

// riskyDuplicateExt — расширения, для которых "дубликат" может оказаться
// разделяемым между программами файлом (рантайм, библиотека, драйвер), а не
// просто лишней копией.
var riskyDuplicateExt = map[string]bool{
	".exe": true, ".dll": true, ".sys": true, ".ocx": true, ".msi": true,
}

// duplicateGroupRiskFlag возвращает непустое предупреждение, если группа
// дубликатов содержит файлы вне пользовательских папок и/или исполняемые
// или библиотечные файлы — такие "дубликаты" не стоит удалять не глядя.
func duplicateGroupRiskFlag(paths []string) string {
	for _, p := range paths {
		low := strings.ToLower(p)
		ext := strings.ToLower(filepath.Ext(p))
		if isSystemDirRoot(filepath.Dir(low)) {
			return "файл находится в системной/установочной папке — удаление может затронуть другую программу"
		}
		if riskyDuplicateExt[ext] {
			return "исполняемый или библиотечный файл — совпадение содержимого не гарантирует, что он не используется другой программой"
		}
	}
	return ""
}

// partialHash вычисляет SHA-256 от первых n байт файла.
func partialHash(path string, n int) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	h := sha256.New()
	buf := make([]byte, n)
	nr, err := f.Read(buf)
	if err != nil && nr == 0 {
		return ""
	}
	h.Write(buf[:nr])
	return hex.EncodeToString(h.Sum(nil))
}

// fullHash вычисляет SHA-256 всего файла блоками по 64 КБ.
func fullHash(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ─── Инкрементальный кэш хэшей (path → {size, modTime, hash}) ───

type dupHashCacheEntry struct {
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"modTime"`
	Hash     string    `json:"hash"`
	LastSeen time.Time `json:"lastSeen"`
}

type dupHashCache struct {
	Entries map[string]dupHashCacheEntry `json:"entries"`
}

// maxDupHashCacheEntries — верхняя граница размера кэша; при превышении
// вытесняются самые давно не встречавшиеся записи (простой LRU по LastSeen).
const maxDupHashCacheEntries = 150000

func dupHashCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "dup-hash-cache.json"
	}
	dir := filepath.Join(home, "AppData", "Local", "winCleanerLamp")
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "dup-hash-cache.json")
}

func loadDupHashCache() *dupHashCache {
	c := &dupHashCache{Entries: map[string]dupHashCacheEntry{}}
	data, err := os.ReadFile(dupHashCachePath())
	if err != nil {
		return c
	}
	_ = json.Unmarshal(data, c)
	if c.Entries == nil {
		c.Entries = map[string]dupHashCacheEntry{}
	}
	return c
}

func saveDupHashCache(c *dupHashCache) {
	if len(c.Entries) > maxDupHashCacheEntries {
		type kv struct {
			key string
			t   time.Time
		}
		list := make([]kv, 0, len(c.Entries))
		for k, v := range c.Entries {
			list = append(list, kv{k, v.LastSeen})
		}
		sort.Slice(list, func(i, j int) bool { return list[i].t.Before(list[j].t) })
		drop := len(list) - maxDupHashCacheEntries
		for i := 0; i < drop; i++ {
			delete(c.Entries, list[i].key)
		}
	}
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	path := dupHashCachePath()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

// fullHashCached возвращает SHA-256 файла, используя кэш при совпадении
// размера и времени модификации — иначе считает заново и обновляет кэш.
func fullHashCached(path string, size int64, cache *dupHashCache, mu *sync.Mutex) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	key := strings.ToLower(path)

	mu.Lock()
	entry, ok := cache.Entries[key]
	mu.Unlock()
	if ok && entry.Size == size && entry.ModTime.Equal(info.ModTime()) {
		mu.Lock()
		entry.LastSeen = time.Now()
		cache.Entries[key] = entry
		mu.Unlock()
		return entry.Hash
	}

	h := fullHash(path)
	if h == "" {
		return ""
	}
	mu.Lock()
	cache.Entries[key] = dupHashCacheEntry{Size: size, ModTime: info.ModTime(), Hash: h, LastSeen: time.Now()}
	mu.Unlock()
	return h
}

// skipDir — папки, которые не нужно обходить при поиске дубликатов.
func skipDir(name string) bool {
	skip := map[string]bool{
		"$recycle.bin": true, "system volume information": true,
		"$windows.~bt": true, "$windows.~ws": true,
		"windows": true, "windows.old": true, "winsxs": true, ".git": true,
		"node_modules": true, "__pycache__": true, ".cache": true,
		"appdata": true, "recovery": true,
		"program files": true, "program files (x86)": true,
	}
	return skip[name]
}
