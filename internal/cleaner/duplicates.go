package cleaner

import (
	"crypto/sha256"
	"encoding/hex"
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
	Hash      string   // SHA-256 полного файла
	Size      int64    // размер каждого файла
	Paths     []string // пути к дубликатам (≥2)
	WasteSize int64    // Size * (len(Paths)-1) — экономия при удалении лишних
}

// DuplicateScanOptions — параметры сканирования дубликатов.
type DuplicateScanOptions struct {
	Roots      []string // корневые папки для сканирования
	MinSize    int64    // минимальный размер файла (по умолчанию 1 КБ)
	MaxSize    int64    // максимальный размер (0 = без ограничения)
	Extensions []string // только эти расширения (пусто = все)
	Timeout    time.Duration
}

// DuplicateScanResult — результат сканирования дубликатов.
type DuplicateScanResult struct {
	Groups     []DuplicateGroup
	TotalWaste int64
	TotalFiles int
	ScannedFiles int
	Duration   time.Duration
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

	for _, root := range opts.Roots {
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

	for _, cand := range candidates {
		wg.Add(1)
		go func(c hashGroup) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fullGroups := make(map[string][]string)
			for _, p := range c.paths {
				h := fullHash(p)
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
					})
				}
			}
			mu.Unlock()
		}(cand)
	}
	wg.Wait()

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
		Groups:       groups,
		TotalWaste:   totalWaste,
		TotalFiles:   totalFiles,
		ScannedFiles: scanned,
		Duration:     time.Since(start),
	}, nil
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

// skipDir — папки, которые не нужно обходить при поиске дубликатов.
func skipDir(name string) bool {
	skip := map[string]bool{
		"$recycle.bin": true, "system volume information": true,
		"$windows.~bt": true, "$windows.~ws": true,
		"windows": true, "winsxs": true, ".git": true,
		"node_modules": true, "__pycache__": true, ".cache": true,
		"appdata": true, "recovery": true,
	}
	return skip[name]
}
