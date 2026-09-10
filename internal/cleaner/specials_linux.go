//go:build linux

package cleaner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// processSpecial выполняет нефайловые действия Linux-ядра: Корзина, DNS,
// journald, кеш пакетного менеджера, flatpak и snap. Второе значение —
// false, если действие этому ядру неизвестно (тогда категория помечается
// пропущенной, см. Process).
func processSpecial(t Target, opts Options) (Report, bool) {
	switch t.Special {
	case SpecialTrash:
		return processTrash(t, opts), true
	case SpecialDNSCache:
		return processDNSCache(t, opts), true
	case SpecialJournalVacuum:
		return processJournalVacuum(t, opts), true
	case SpecialPackageCache:
		return processPackageCache(t, opts), true
	case SpecialFlatpakUnused:
		return processFlatpakUnused(t, opts), true
	case SpecialSnapDisabled:
		return processSnapDisabled(t, opts), true
	}
	return Report{Target: t}, false
}

// platformProcessDir — категории Linux, где внутри корня чистятся конкретные
// подпапки профилей, а не всё содержимое.
func platformProcessDir(root string, t Target, opts Options, r *Report) bool {
	switch t.ID {
	case "firefox-cache":
		// ~/.cache/mozilla/firefox/<профиль>/cache2
		processProfileSubdirs(root, []string{"cache2"}, t, opts, r)
		return true
	case "jetbrains-logs":
		// ~/.cache/JetBrains/<IDE>/{log,caches}
		processProfileSubdirs(root, []string{"log", "caches"}, t, opts, r)
		return true
	case "flatpak-app-cache":
		// ~/.var/app/<приложение>/cache — настройки (config) и данные (data)
		// приложения при этом не трогаются.
		processProfileSubdirs(root, []string{"cache"}, t, opts, r)
		return true
	}
	return false
}

// ─── Корзина (FreeDesktop Trash spec) ───

func processTrash(t Target, opts Options) Report {
	r := Report{Target: t}
	dir := trashDir()
	if dir == "" {
		r.Skipped = true
		r.SkippedReason = "не удалось определить каталог Корзины"
		return r
	}
	for _, sub := range []string{"files", "info", "expunged"} {
		p := filepath.Join(dir, sub)
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if opts.DryRun {
			size, files := dirSize(p)
			r.Bytes += size
			r.Files += files
			continue
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("чтение %s: %v", p, err))
			continue
		}
		for _, e := range entries {
			full := filepath.Join(p, e.Name())
			size, files := dirSize(full)
			if err := os.RemoveAll(full); err != nil {
				r.Errors = append(r.Errors, fmt.Sprintf("удаление %s: %v", full, err))
				continue
			}
			r.Bytes += size
			r.Files += files
		}
	}
	return r
}

// ─── DNS ───

// processDNSCache сбрасывает кеш systemd-resolved. Других общесистемных
// DNS-кешей в типичной установке нет: сам ядро/glibc кеш не держит, а
// dnsmasq/nscd есть далеко не везде — если resolvectl отсутствует,
// сообщаем об этом, а не делаем вид, что кеш сброшен.
func processDNSCache(t Target, opts Options) Report {
	r := Report{Target: t}
	if opts.DryRun {
		r.Skipped = true
		r.SkippedReason = "DNS-кеш — действие, не имеет размера"
		return r
	}
	for _, args := range [][]string{
		{"resolvectl", "flush-caches"},
		{"systemd-resolve", "--flush-caches"},
	} {
		if _, err := exec.LookPath(args[0]); err != nil {
			continue
		}
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("%s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out))))
			continue
		}
		return r
	}
	r.Errors = append(r.Errors, "не найден resolvectl/systemd-resolve — системный DNS-кеш не обнаружен")
	return r
}

// ─── journald ───

// journalVacuumTime — до какого возраста усекается журнал. 7 дней — то же
// значение, что стоит по умолчанию в большинстве дистрибутивов для
// MaxRetentionSec, то есть свежая диагностика остаётся на месте.
const journalVacuumTime = "7d"

func processJournalVacuum(t Target, opts Options) Report {
	r := Report{Target: t}
	if opts.DryRun {
		// Размер журнала = размер /var/log/journal (persistent) или
		// /run/log/journal (volatile).
		for _, p := range []string{"/var/log/journal", "/run/log/journal"} {
			size, files := dirSize(p)
			r.Bytes += size
			r.Files += files
		}
		return r
	}
	if _, err := exec.LookPath("journalctl"); err != nil {
		r.Errors = append(r.Errors, "journalctl не найден — systemd-журналы не используются")
		return r
	}
	out, err := exec.Command("journalctl", "--vacuum-time="+journalVacuumTime).CombinedOutput()
	if err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("journalctl --vacuum-time: %v: %s", err, strings.TrimSpace(string(out))))
	}
	return r
}

// ─── Кеш пакетного менеджера ───

// packageCacheCleaner — как чистится кеш конкретного пакетного менеджера:
// каталог для оценки размера и штатная команда очистки. Своими руками файлы
// пакетов не удаляем — менеджер ведёт по ним собственный учёт.
type packageCacheCleaner struct {
	tool  string
	dir   string
	clean []string
}

func packageCacheCleaners() []packageCacheCleaner {
	return []packageCacheCleaner{
		{"apt-get", "/var/cache/apt/archives", []string{"apt-get", "clean"}},
		{"dnf", "/var/cache/dnf", []string{"dnf", "clean", "packages"}},
		{"yum", "/var/cache/yum", []string{"yum", "clean", "packages"}},
		{"zypper", "/var/cache/zypp/packages", []string{"zypper", "clean", "--all"}},
		{"pacman", "/var/cache/pacman/pkg", []string{"pacman", "-Sc", "--noconfirm"}},
		{"apk", "/var/cache/apk", []string{"apk", "cache", "clean"}},
	}
}

func processPackageCache(t Target, opts Options) Report {
	r := Report{Target: t}
	found := false
	for _, c := range packageCacheCleaners() {
		if _, err := exec.LookPath(c.tool); err != nil {
			continue
		}
		found = true
		if opts.DryRun {
			size, files := dirSize(c.dir)
			r.Bytes += size
			r.Files += files
			continue
		}
		out, err := exec.Command(c.clean[0], c.clean[1:]...).CombinedOutput()
		if err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("%s: %v: %s", strings.Join(c.clean, " "), err, strings.TrimSpace(string(out))))
			continue
		}
		if opts.Verbose {
			opts.log("  [пакеты] %s", strings.Join(c.clean, " "))
		}
	}
	if !found {
		r.Skipped = true
		r.SkippedReason = "не найден поддерживаемый пакетный менеджер (apt/dnf/yum/zypper/pacman/apk)"
	}
	return r
}

// ─── flatpak ───

func processFlatpakUnused(t Target, opts Options) Report {
	r := Report{Target: t}
	if _, err := exec.LookPath("flatpak"); err != nil {
		r.Skipped = true
		r.SkippedReason = "flatpak не установлен"
		return r
	}
	if opts.DryRun {
		// Точный размер известен только самому flatpak, а его вывод
		// (`flatpak uninstall --unused` без -y) интерактивен — не гадаем.
		r.Skipped = true
		r.SkippedReason = "размер неиспользуемых runtime заранее неизвестен — оценивает сам flatpak"
		return r
	}
	out, err := exec.Command("flatpak", "uninstall", "--unused", "-y").CombinedOutput()
	if err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("flatpak uninstall --unused: %v: %s", err, strings.TrimSpace(string(out))))
	}
	return r
}

// ─── snap ───

// snapListEntry — строка вывода `snap list --all --unicode=never`.
type snapListEntry struct {
	Name     string
	Revision string
	Disabled bool
}

func processSnapDisabled(t Target, opts Options) Report {
	r := Report{Target: t}
	if _, err := exec.LookPath("snap"); err != nil {
		r.Skipped = true
		r.SkippedReason = "snapd не установлен"
		return r
	}
	entries, err := listDisabledSnaps()
	if err != nil {
		r.Errors = append(r.Errors, err.Error())
		return r
	}
	for _, e := range entries {
		// Размер отключённой ревизии — размер её squashfs-файла в
		// /var/lib/snapd/snaps (сам каталог мы не трогаем: удалением
		// занимается snapd).
		size := snapRevisionSize(e)
		if opts.DryRun {
			r.Bytes += size
			r.Files++
			continue
		}
		out, err := exec.Command("snap", "remove", "--purge", e.Name, "--revision="+e.Revision).CombinedOutput()
		if err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("snap remove %s (ревизия %s): %v: %s", e.Name, e.Revision, err, strings.TrimSpace(string(out))))
			continue
		}
		r.Bytes += size
		r.Files++
		if opts.Verbose {
			opts.log("  [snap] удалена отключённая ревизия %s %s", e.Name, e.Revision)
		}
	}
	return r
}

// listDisabledSnaps возвращает отключённые (disabled) ревизии — это старые
// версии, оставленные snapd для отката; активная версия каждого пакета в
// список не попадает.
func listDisabledSnaps() ([]snapListEntry, error) {
	out, err := exec.Command("snap", "list", "--all", "--unicode=never", "--color=never").Output()
	if err != nil {
		return nil, fmt.Errorf("snap list --all: %w", err)
	}
	var result []snapListEntry
	for i, line := range strings.Split(string(out), "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // заголовок таблицы
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		// Name Version Rev Tracking Publisher Notes
		notes := fields[len(fields)-1]
		if !strings.Contains(notes, "disabled") {
			continue
		}
		result = append(result, snapListEntry{Name: fields[0], Revision: fields[2], Disabled: true})
	}
	return result, nil
}

func snapRevisionSize(e snapListEntry) int64 {
	p := filepath.Join("/var/lib/snapd/snaps", fmt.Sprintf("%s_%s.snap", e.Name, e.Revision))
	info, err := os.Stat(p)
	if err != nil {
		return 0
	}
	return info.Size()
}
