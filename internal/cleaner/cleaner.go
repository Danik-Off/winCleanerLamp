package cleaner

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
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
	// RebootPending — файлы, занятые другим процессом прямо сейчас: не
	// удалось удалить сразу, но удаление запланировано на следующую
	// перезагрузку (MoveFileEx/MOVEFILE_DELAY_UNTIL_REBOOT). Уже учтены в
	// Bytes/Files как "будут освобождены", но это отдельный счётчик для
	// точного сообщения пользователю.
	RebootPending int
	// Записи о найденных файлах (для последующей записи в конфиг)
	Records []JunkRecord
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

	// Спец-действия (Корзина, DNS-кеш, журналы и т.п.) реализует само ядро:
	// specials_windows.go / specials_linux.go / specials_darwin.go.
	if t.Special != SpecialNone {
		if special, handled := processSpecial(t, opts); handled {
			return special
		}
		r.Skipped = true
		r.SkippedReason = "действие не поддерживается этой ОС: " + string(t.Special)
		return r
	}

	for _, raw := range t.Paths {
		p := ExpandPath(raw)
		if p == "" {
			continue
		}
		if ok, reason := IsPathSafeToDelete(p); !ok {
			r.Errors = append(r.Errors, fmt.Sprintf("небезопасный путь пропущен: %s (%s)", p, reason))
			continue
		}

		info, err := os.Lstat(p)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				if opts.Verbose {
					opts.log("  [skip] не существует: %s", p)
				}
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
				if opts.Verbose {
					opts.log("  [would delete] %s (%s)", p, human(sz))
				}
			} else {
				if err := os.Remove(p); err != nil {
					r.Errors = append(r.Errors, fmt.Sprintf("remove %s: %v", p, err))
				} else {
					r.Bytes += sz
					r.Files++
					if opts.Verbose {
						opts.log("  [deleted] %s (%s)", p, human(sz))
					}
				}
			}
			continue
		}

		processDir(p, t, opts, &r)
	}

	return r
}

// processDir обрабатывает один каталог категории. Категории, у которых внутри
// корня нужно чистить не всё подряд, а конкретные подпапки профилей
// (Firefox, JetBrains, Skype в Windows; профили браузеров в Linux/macOS),
// обрабатывает своё ядро — см. platformProcessDir в specials_<os>.go.
func processDir(root string, t Target, opts Options, r *Report) {
	if platformProcessDir(root, t, opts, r) {
		return
	}
	walkAndDelete(root, t, opts, r, t.KeepRoot)
}

// processProfileSubdirs — общий помощник для ядер: для каждого подкаталога
// root чистит перечисленные относительные подпути (профили браузеров, версии
// IDE и т.п.). Относительные пути записываются через "/" и приводятся к
// разделителю текущей ОС.
func processProfileSubdirs(root string, relPaths []string, t Target, opts Options, r *Report) {
	forEachSubdir(root, r, func(profile string) {
		for _, rel := range relPaths {
			p := filepath.Join(profile, filepath.FromSlash(rel))
			if _, err := os.Stat(p); err == nil {
				walkAndDelete(p, t, opts, r, true)
			}
		}
	})
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
		if ok, _ := IsPathSafeToDelete(path); !ok {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		if t.forbidden(path) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			dirs = append(dirs, item{path, info})
			return nil
		}
		// Файл
		if !isDeletableFile(info) {
			return nil
		}
		if tooYoung(info, t, opts) {
			return nil
		}
		sz := info.Size()
		if opts.DryRun {
			r.Bytes += sz
			r.Files++
			// Записываем в Records
			r.Records = append(r.Records, JunkRecord{
				Path:       path,
				CategoryID: t.ID,
				Size:       sz,
			})
			return nil
		}
		scheduledReboot := false
		if err := os.Remove(path); err != nil {
			// пытаемся снять read-only
			_ = os.Chmod(path, 0o666)
			if err2 := os.Remove(path); err2 != nil {
				// Файл занят другим процессом прямо сейчас (открытый лог,
				// DLL используемой программы и т.п.) — вместо того чтобы
				// просто пропустить его с ошибкой, планируем удаление на
				// следующую перезагрузку тем же способом, что используют
				// установщики Windows (MoveFileEx/MOVEFILE_DELAY_UNTIL_REBOOT).
				if err3 := scheduleDeleteOnReboot(path); err3 != nil {
					r.Errors = append(r.Errors, fmt.Sprintf("remove %s: %v", path, err2))
					return nil
				}
				scheduledReboot = true
			}
		}
		r.Bytes += sz
		r.Files++
		if scheduledReboot {
			r.RebootPending++
		}
		// Записываем в Records
		r.Records = append(r.Records, JunkRecord{
			Path:       path,
			CategoryID: t.ID,
			Size:       sz,
		})
		if opts.Verbose {
			if scheduledReboot {
				opts.log("  [удалится при перезагрузке] %s (%s)", path, human(sz))
			} else {
				opts.log("  [deleted] %s (%s)", path, human(sz))
			}
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

// forbidden — путь попал под ForbidSubstrings категории. Так категории
// защищают отдельные вложенные пути от удаления: в Linux это, например,
// сокеты и блокировки внутри /tmp (.X11-unix, .ICE-unix), в macOS —
// служебные подпапки внутри каталогов TMPDIR.
func (t Target) forbidden(path string) bool {
	if len(t.ForbidSubstrings) == 0 {
		return false
	}
	low := strings.ToLower(filepath.ToSlash(path))
	for _, s := range t.ForbidSubstrings {
		if s == "" {
			continue
		}
		if strings.Contains(low, strings.ToLower(filepath.ToSlash(s))) {
			return true
		}
	}
	return false
}

// isDeletableFile — обычный файл или символическая ссылка. Сокеты, каналы и
// файлы устройств пропускаются: в Linux и macOS они лежат вперемешку с
// мусором в /tmp и /private/var/folders и принадлежат запущенным программам,
// а их размер на диске всё равно нулевой.
func isDeletableFile(info fs.FileInfo) bool {
	mode := info.Mode()
	return mode.IsRegular() || mode&fs.ModeSymlink != 0
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

// ---- helpers ----

func dirSize(root string) (int64, int) {
	var total int64
	var n int
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
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
