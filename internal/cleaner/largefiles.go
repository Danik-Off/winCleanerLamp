package cleaner

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LargeFile — один найденный крупный файл.
type LargeFile struct {
	Path          string `json:"path"`
	SizeBytes     int64  `json:"sizeBytes"`
	SizeFormatted string `json:"sizeFormatted"`
	ModTime       string `json:"modTime"`
	// InSystemDir — файл лежит внутри Program Files/Windows/ProgramData —
	// как и для дубликатов, такие файлы показываются, но требуют большей
	// осторожности при удалении (могут принадлежать установленной программе).
	InSystemDir bool `json:"inSystemDir,omitempty"`
}

// LargeFilesResult — результат поиска крупных файлов.
type LargeFilesResult struct {
	Files        []LargeFile `json:"files"`
	ScannedFiles int         `json:"scannedFiles"`
	TotalBytes   int64       `json:"totalBytes"`
	// SkippedRoots — корни, пропущенные как системные (см. LargeFilesOptions.AllowSystemDirs).
	SkippedRoots []string `json:"skippedRoots,omitempty"`
}

// LargeFilesOptions — параметры поиска крупных файлов.
type LargeFilesOptions struct {
	Roots        []string      // корневые папки для сканирования
	TopN         int           // сколько вернуть, отсортировано по убыванию размера (по умолчанию 200)
	MinSizeBytes int64         // порог отсечения (по умолчанию 100 МБ)
	Timeout      time.Duration // по умолчанию 5 минут
	// AllowSystemDirs разрешает сканировать корни внутри системных/установочных
	// каталогов — по умолчанию такие корни пропускаются (см. duplicates.go:
	// isSystemDirRoot), чтобы не подсовывать в первую очередь файлы, которые
	// нельзя трогать без риска сломать установленную программу.
	AllowSystemDirs bool
}

// DefaultLargeFileRoots — папки по умолчанию для поиска крупных файлов, если
// пользователь не указал свои: весь профиль пользователя (Downloads,
// Desktop, Documents, Videos и т.п. уже внутри него).
func DefaultLargeFileRoots() []string {
	var roots []string
	if home := ExpandPath(`%USERPROFILE%`); home != "" {
		roots = append(roots, home)
	}
	return roots
}

// ScanLargeFiles ищет самые крупные файлы в указанных папках — аналог
// "анализатора диска" (WizTree/TreeSize): не привязан к заранее известным
// категориям мусора, находит то, что реально занимает место (видео, ISO,
// установщики, старые бэкапы, образы виртуальных машин), и оставляет решение
// об удалении пользователю. Ничего не удаляется автоматически.
func ScanLargeFiles(opts LargeFilesOptions) (*LargeFilesResult, error) {
	topN := opts.TopN
	if topN <= 0 {
		topN = 200
	}
	minSize := opts.MinSizeBytes
	if minSize <= 0 {
		minSize = 100 * 1024 * 1024 // 100 МБ
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	deadline := time.After(timeout)

	result := &LargeFilesResult{}
	var candidates []LargeFile
	var scanned int

	for _, root := range opts.Roots {
		expanded := ExpandPath(root)
		if expanded == "" {
			continue
		}
		if !opts.AllowSystemDirs && isSystemDirRoot(expanded) {
			result.SkippedRoots = append(result.SkippedRoots, root)
			continue
		}
		inSystem := isSystemDirRoot(expanded)

		_ = filepath.Walk(expanded, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // пропускаем недоступные файлы/папки
			}
			if scanned%2000 == 0 {
				select {
				case <-deadline:
					return filepath.SkipAll
				default:
				}
			}
			if info.IsDir() {
				if skipDir(strings.ToLower(info.Name())) {
					return filepath.SkipDir
				}
				return nil
			}
			scanned++
			size := info.Size()
			if size < minSize {
				return nil
			}
			if ok, _ := IsPathSafeToDelete(path); !ok {
				return nil
			}
			candidates = append(candidates, LargeFile{
				Path:          path,
				SizeBytes:     size,
				SizeFormatted: human(size),
				ModTime:       info.ModTime().Format(time.RFC3339),
				InSystemDir:   inSystem,
			})
			return nil
		})
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].SizeBytes > candidates[j].SizeBytes })
	if len(candidates) > topN {
		candidates = candidates[:topN]
	}
	for _, c := range candidates {
		result.TotalBytes += c.SizeBytes
	}
	result.Files = candidates
	result.ScannedFiles = scanned
	return result, nil
}
