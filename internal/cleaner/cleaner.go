package cleaner

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Report — результат сканирования/очистки одной цели.
type Report struct {
	Target        Target
	Bytes         int64
	Files         int
	Errors        []string
	Skipped       bool
	SkippedReason string
}

// Options управляют поведением сканера/чистильщика.
type Options struct {
	DryRun  bool          // только посчитать размер, ничего не удалять
	Verbose bool          // логировать каждое действие
	MinAge  time.Duration // глобальный минимальный возраст файла (0 = не применять)
	Logger  func(string)
}

func (o *Options) log(format string, a ...any) {
	if o == nil || o.Logger == nil {
		return
	}
	o.Logger(fmt.Sprintf(format, a...))
}

// Process обрабатывает одну цель (scan или clean в зависимости от opts.DryRun).
func Process(t Target, opts Options) Report {
	r := Report{Target: t}

	// Спец-действия
	switch t.Special {
	case SpecialRecycleBin:
		return processRecycleBin(t, opts)
	case SpecialDNSCache:
		return processDNSCache(t, opts)
	case SpecialThumbnailCache:
		return processThumbnailCache(t, opts)
	case SpecialEventLogs:
		return processEventLogs(t, opts)
	}

	for _, raw := range t.Paths {
		p := ExpandPath(raw)
		if p == "" {
			continue
		}
		if !safePath(p) {
			r.Errors = append(r.Errors, fmt.Sprintf("небезопасный путь пропущен: %s", p))
			continue
		}

		info, err := os.Lstat(p)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				opts.log("  [skip] не существует: %s", p)
				continue
			}
			r.Errors = append(r.Errors, fmt.Sprintf("stat %s: %v", p, err))
			continue
		}

		// Отдельный файл (например, MEMORY.DMP)
		if !info.IsDir() {
			if tooYoung(info, t, opts) {
				continue
			}
			sz := info.Size()
			if opts.DryRun {
				r.Bytes += sz
				r.Files++
				opts.log("  [would delete] %s (%s)", p, human(sz))
			} else {
				if err := os.Remove(p); err != nil {
					r.Errors = append(r.Errors, fmt.Sprintf("remove %s: %v", p, err))
				} else {
					r.Bytes += sz
					r.Files++
					opts.log("  [deleted] %s (%s)", p, human(sz))
				}
			}
			continue
		}

		processDir(p, t, opts, &r)
	}

	return r
}

func processDir(root string, t Target, opts Options, r *Report) {
	switch t.ID {
	case "firefox-cache":
		// только cache2 во всех профилях
		forEachSubdir(root, r, func(profile string) {
			p := filepath.Join(profile, "cache2")
			if _, err := os.Stat(p); err == nil {
				walkAndDelete(p, t, opts, r, true)
			}
		})
		return
	case "jetbrains-logs":
		// %LOCALAPPDATA%\JetBrains\<IDE>\{log,caches}
		forEachSubdir(root, r, func(ide string) {
			for _, sub := range []string{"log", "caches"} {
				p := filepath.Join(ide, sub)
				if _, err := os.Stat(p); err == nil {
					walkAndDelete(p, t, opts, r, true)
				}
			}
		})
		return
	case "office-cache":
		// %LOCALAPPDATA%\Microsoft\Office\<ver>\OfficeFileCache
		forEachSubdir(root, r, func(ver string) {
			p := filepath.Join(ver, "OfficeFileCache")
			if _, err := os.Stat(p); err == nil {
				walkAndDelete(p, t, opts, r, true)
			}
		})
		return
	case "teams-new-cache":
		// %LOCALAPPDATA%\Packages\MSTeams_*\LocalCache\Microsoft\MSTeams\{Cache,GPUCache,Code Cache}
		entries, err := os.ReadDir(root)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				r.Errors = append(r.Errors, err.Error())
			}
			return
		}
		for _, e := range entries {
			name := e.Name()
			if !e.IsDir() || !strings.HasPrefix(name, "MSTeams_") {
				continue
			}
			base := filepath.Join(root, name, "LocalCache", "Microsoft", "MSTeams")
			for _, sub := range []string{"Cache", "GPUCache", "Code Cache", "tmp"} {
				p := filepath.Join(base, sub)
				if _, err := os.Stat(p); err == nil {
					walkAndDelete(p, t, opts, r, true)
				}
			}
		}
		return
	}

	walkAndDelete(root, t, opts, r, t.KeepRoot)
}

// forEachSubdir вызывает fn для каждого поддиректория root.
func forEachSubdir(root string, r *Report, fn func(string)) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			r.Errors = append(r.Errors, fmt.Sprintf("readdir %s: %v", root, err))
		}
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		fn(filepath.Join(root, e.Name()))
	}
}

// walkAndDelete обходит дерево и удаляет/считает файлы.
// Если keepRoot=true, сам root не удаляется.
func walkAndDelete(root string, t Target, opts Options, r *Report, keepRoot bool) {
	// Собираем список для обработки в пост-порядке (чтобы удалять вложенные первее).
	type item struct {
		path string
		info fs.FileInfo
	}
	var dirs []item

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// пропускаем недоступные файлы/папки
			r.Errors = append(r.Errors, fmt.Sprintf("walk %s: %v", path, err))
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !safePath(path) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		if d.IsDir() {
			dirs = append(dirs, item{path, info})
			return nil
		}
		// Файл
		if tooYoung(info, t, opts) {
			return nil
		}
		sz := info.Size()
		if opts.DryRun {
			r.Bytes += sz
			r.Files++
			return nil
		}
		if err := os.Remove(path); err != nil {
			// пытаемся снять read-only
			_ = os.Chmod(path, 0o666)
			if err2 := os.Remove(path); err2 != nil {
				r.Errors = append(r.Errors, fmt.Sprintf("remove %s: %v", path, err2))
				return nil
			}
		}
		r.Bytes += sz
		r.Files++
		if opts.Verbose {
			opts.log("  [deleted] %s (%s)", path, human(sz))
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		r.Errors = append(r.Errors, fmt.Sprintf("walk %s: %v", root, err))
	}

	if opts.DryRun {
		return
	}

	// Удаляем пустые директории в обратном порядке
	for i := len(dirs) - 1; i >= 0; i-- {
		p := dirs[i].path
		if keepRoot && p == root {
			continue
		}
		_ = os.Remove(p) // удалится только если пусто
	}
}

func tooYoung(info fs.FileInfo, t Target, opts Options) bool {
	var minAge time.Duration
	if t.MinAgeHours > 0 {
		minAge = time.Duration(t.MinAgeHours) * time.Hour
	}
	if opts.MinAge > minAge {
		minAge = opts.MinAge
	}
	if minAge == 0 {
		return false
	}
	return time.Since(info.ModTime()) < minAge
}

// safePath — защита от удаления чего-то критичного.
// Мы запрещаем корень диска, системные каталоги, домашнюю папку пользователя целиком и т.д.
func safePath(p string) bool {
	if p == "" {
		return false
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return false
	}
	abs = filepath.Clean(abs)
	low := strings.ToLower(abs)

	// Корни дисков
	if len(abs) <= 3 { // "C:\"
		return false
	}

	forbidden := []string{
		`c:\windows\system32`,
		`c:\windows\syswow64`,
		`c:\windows\winsxs`,
		`c:\program files`,
		`c:\program files (x86)`,
		`c:\programdata\microsoft\windows\start menu`,
		`c:\users\default`,
		`c:\users\public`,
	}
	for _, f := range forbidden {
		if low == f {
			return false
		}
	}

	// Не позволяем удалять корень профиля пользователя.
	if home, err := os.UserHomeDir(); err == nil {
		if strings.EqualFold(abs, filepath.Clean(home)) {
			return false
		}
	}

	return true
}

// ---- Special actions ----

func processRecycleBin(t Target, opts Options) Report {
	r := Report{Target: t}
	if opts.DryRun {
		// Попробуем посчитать размер $Recycle.Bin на всех дисках
		drives := listDrives()
		for _, d := range drives {
			p := filepath.Join(d, `$Recycle.Bin`)
			size, files := dirSize(p)
			r.Bytes += size
			r.Files += files
		}
		return r
	}
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "Clear-RecycleBin -Force -ErrorAction SilentlyContinue")
	if out, err := cmd.CombinedOutput(); err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("Clear-RecycleBin: %v: %s", err, strings.TrimSpace(string(out))))
	}
	return r
}

func processDNSCache(t Target, opts Options) Report {
	r := Report{Target: t}
	if opts.DryRun {
		r.Skipped = true
		r.SkippedReason = "DNS-кеш — действие, не имеет размера"
		return r
	}
	cmd := exec.Command("ipconfig", "/flushdns")
	if out, err := cmd.CombinedOutput(); err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("ipconfig /flushdns: %v: %s", err, strings.TrimSpace(string(out))))
	}
	return r
}

func processEventLogs(t Target, opts Options) Report {
	r := Report{Target: t}
	if opts.DryRun {
		size, files := dirSize(`C:\Windows\System32\winevt\Logs`)
		r.Bytes = size
		r.Files = files
		return r
	}
	// Перечисляем журналы через wevtutil el и чистим каждый через wevtutil cl.
	out, err := exec.Command("wevtutil", "el").Output()
	if err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("wevtutil el: %v", err))
		return r
	}
	for _, line := range strings.Split(string(out), "\n") {
		log := strings.TrimSpace(line)
		if log == "" {
			continue
		}
		if err := exec.Command("wevtutil", "cl", log).Run(); err != nil {
			// многие журналы нельзя очищать (Analytical/Debug) — пропускаем молча
			if opts.Verbose {
				opts.log("  [skip evtx] %s: %v", log, err)
			}
			continue
		}
		r.Files++
	}
	return r
}

func processThumbnailCache(t Target, opts Options) Report {
	r := Report{Target: t}
	root := ExpandPath(`%LOCALAPPDATA%\Microsoft\Windows\Explorer`)
	entries, err := os.ReadDir(root)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			r.Errors = append(r.Errors, err.Error())
		}
		return r
	}
	for _, e := range entries {
		n := strings.ToLower(e.Name())
		if !strings.HasPrefix(n, "thumbcache_") && !strings.HasPrefix(n, "iconcache_") {
			continue
		}
		p := filepath.Join(root, e.Name())
		info, err := e.Info()
		if err != nil {
			continue
		}
		sz := info.Size()
		if opts.DryRun {
			r.Bytes += sz
			r.Files++
			continue
		}
		if err := os.Remove(p); err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("remove %s: %v", p, err))
			continue
		}
		r.Bytes += sz
		r.Files++
	}
	return r
}

// ---- helpers ----

func listDrives() []string {
	var drives []string
	for c := 'A'; c <= 'Z'; c++ {
		d := string(c) + `:\`
		if _, err := os.Stat(d); err == nil {
			drives = append(drives, d)
		}
	}
	return drives
}

func dirSize(root string) (int64, int) {
	var total int64
	var n int
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		total += info.Size()
		n++
		return nil
	})
	return total, n
}

// Human-readable size.
func human(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// Human экспортируется для CLI.
func Human(b int64) string { return human(b) }
