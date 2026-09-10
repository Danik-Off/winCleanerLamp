//go:build darwin

package cleaner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// processSpecial выполняет нефайловые действия macOS-ядра: Корзина, DNS,
// QuickLook, Homebrew и локальные снимки Time Machine.
func processSpecial(t Target, opts Options) (Report, bool) {
	switch t.Special {
	case SpecialTrash:
		return processTrash(t, opts), true
	case SpecialDNSCache:
		return processDNSCache(t, opts), true
	case SpecialQuickLookCache:
		return processQuickLookCache(t, opts), true
	case SpecialBrewCleanup:
		return processBrewCleanup(t, opts), true
	case SpecialLocalSnapshots:
		return processLocalSnapshots(t, opts), true
	}
	return Report{Target: t}, false
}

// platformProcessDir — категории macOS, где внутри корня чистятся конкретные
// подпапки профилей.
func platformProcessDir(root string, t Target, opts Options, r *Report) bool {
	switch t.ID {
	case "firefox-cache":
		// ~/Library/Caches/Firefox/Profiles/<профиль>/cache2
		processProfileSubdirs(root, []string{"cache2"}, t, opts, r)
		return true
	case "jetbrains-logs":
		// ~/Library/{Logs,Caches}/JetBrains/<IDE>
		processProfileSubdirs(root, []string{"log", "caches", "logs"}, t, opts, r)
		return true
	}
	return false
}

// ─── Корзина ───

func processTrash(t Target, opts Options) Report {
	r := Report{Target: t}
	dir := trashDir()
	if dir == "" {
		r.Skipped = true
		r.SkippedReason = "не удалось определить ~/.Trash"
		return r
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return r
		}
		r.Errors = append(r.Errors, fmt.Sprintf("чтение %s: %v", dir, err))
		return r
	}
	for _, e := range entries {
		// .DS_Store в самой Корзине — служебный файл Finder, он же хранит
		// сведения для «Положить обратно»: не трогаем.
		if e.Name() == ".DS_Store" {
			continue
		}
		full := filepath.Join(dir, e.Name())
		size, files := dirSize(full)
		if !e.IsDir() {
			if info, err := e.Info(); err == nil {
				size, files = info.Size(), 1
			}
		}
		if opts.DryRun {
			r.Bytes += size
			r.Files += files
			continue
		}
		if err := os.RemoveAll(full); err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("удаление %s: %v", full, err))
			continue
		}
		r.Bytes += size
		r.Files += files
	}
	return r
}

// ─── DNS ───

// processDNSCache сбрасывает кеш DNS так, как это описано в документации
// Apple: очистка кеша directory service и перезапуск mDNSResponder. Обе
// команды требуют root.
func processDNSCache(t Target, opts Options) Report {
	r := Report{Target: t}
	if opts.DryRun {
		r.Skipped = true
		r.SkippedReason = "DNS-кеш — действие, не имеет размера"
		return r
	}
	if out, err := exec.Command("dscacheutil", "-flushcache").CombinedOutput(); err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("dscacheutil -flushcache: %v: %s", err, strings.TrimSpace(string(out))))
	}
	if out, err := exec.Command("killall", "-HUP", "mDNSResponder").CombinedOutput(); err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("killall -HUP mDNSResponder (нужен sudo): %v: %s", err, strings.TrimSpace(string(out))))
	}
	return r
}

// ─── QuickLook ───

func processQuickLookCache(t Target, opts Options) Report {
	r := Report{Target: t}
	if opts.DryRun {
		r.Skipped = true
		r.SkippedReason = "кеш QuickLook сбрасывается системной утилитой, размер заранее неизвестен"
		return r
	}
	if out, err := exec.Command("qlmanage", "-r", "cache").CombinedOutput(); err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("qlmanage -r cache: %v: %s", err, strings.TrimSpace(string(out))))
	}
	return r
}

// ─── Homebrew ───

func processBrewCleanup(t Target, opts Options) Report {
	r := Report{Target: t}
	if _, err := exec.LookPath("brew"); err != nil {
		r.Skipped = true
		r.SkippedReason = "Homebrew не установлен"
		return r
	}
	if opts.DryRun {
		// brew умеет считать сам — спрашиваем его в режиме предпросмотра.
		out, err := exec.Command("brew", "cleanup", "--prune=all", "-n").Output()
		if err != nil {
			r.Skipped = true
			r.SkippedReason = "не удалось оценить объём (brew cleanup -n)"
			return r
		}
		r.Bytes = parseBrewFreedBytes(string(out))
		return r
	}
	out, err := exec.Command("brew", "cleanup", "--prune=all").CombinedOutput()
	if err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("brew cleanup: %v: %s", err, strings.TrimSpace(string(out))))
		return r
	}
	r.Bytes = parseBrewFreedBytes(string(out))
	return r
}

// parseBrewFreedBytes достаёт объём из строки вида
// "==> This operation has freed approximately 1.2GB of disk space".
func parseBrewFreedBytes(out string) int64 {
	// Обычный запуск говорит "has freed approximately", предпросмотр (-n) —
	// "would free approximately".
	for _, marker := range []string{"freed approximately ", "would free approximately "} {
		idx := strings.Index(out, marker)
		if idx < 0 {
			continue
		}
		rest := out[idx+len(marker):]
		end := strings.Index(rest, " of disk space")
		if end < 0 {
			continue
		}
		return parseHumanSize(strings.TrimSpace(rest[:end]))
	}
	return 0
}

// parseHumanSize разбирает "1.2GB", "512KB", "37MB" в байты.
func parseHumanSize(s string) int64 {
	units := []struct {
		suffix string
		mult   float64
	}{
		{"TB", 1 << 40}, {"GB", 1 << 30}, {"MB", 1 << 20}, {"KB", 1 << 10}, {"B", 1},
	}
	for _, u := range units {
		if !strings.HasSuffix(s, u.suffix) {
			continue
		}
		var value float64
		if _, err := fmt.Sscanf(strings.TrimSuffix(s, u.suffix), "%f", &value); err != nil {
			return 0
		}
		return int64(value * u.mult)
	}
	return 0
}

// ─── Локальные снимки Time Machine ───

// processLocalSnapshots удаляет локальные снимки APFS. Это не резервные
// копии на внешнем диске: снимки живут на системном томе и занимают место до
// тех пор, пока их не вытеснит сама macOS. Удаление через tmutil — штатный
// способ; сами файлы Time Machine мы не трогаем.
func processLocalSnapshots(t Target, opts Options) Report {
	r := Report{Target: t}
	if _, err := exec.LookPath("tmutil"); err != nil {
		r.Skipped = true
		r.SkippedReason = "tmutil недоступен"
		return r
	}
	snapshots, err := listLocalSnapshots()
	if err != nil {
		r.Errors = append(r.Errors, err.Error())
		return r
	}
	if opts.DryRun {
		// Размер снимка system не сообщает: он зависит от того, сколько
		// данных изменилось с момента его создания.
		r.Files = len(snapshots)
		r.Skipped = len(snapshots) == 0
		if r.Skipped {
			r.SkippedReason = "локальных снимков нет"
		} else {
			r.SkippedReason = fmt.Sprintf("найдено снимков: %d (объём известен только системе)", len(snapshots))
		}
		return r
	}
	for _, date := range snapshots {
		out, err := exec.Command("tmutil", "deletelocalsnapshots", date).CombinedOutput()
		if err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("tmutil deletelocalsnapshots %s (нужен sudo): %v: %s", date, err, strings.TrimSpace(string(out))))
			continue
		}
		r.Files++
		if opts.Verbose {
			opts.log("  [снимок] удалён локальный снимок %s", date)
		}
	}
	return r
}

// listLocalSnapshots возвращает даты снимков (com.apple.TimeMachine.<дата>).
func listLocalSnapshots() ([]string, error) {
	out, err := exec.Command("tmutil", "listlocalsnapshots", "/").Output()
	if err != nil {
		return nil, fmt.Errorf("tmutil listlocalsnapshots: %w", err)
	}
	const prefix = "com.apple.TimeMachine."
	var dates []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		idx := strings.Index(line, prefix)
		if idx < 0 {
			continue
		}
		date := strings.TrimSuffix(line[idx+len(prefix):], ".local")
		if date != "" {
			dates = append(dates, date)
		}
	}
	return dates, nil
}
