//go:build windows

package cleaner

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// processSpecial выполняет нефайловые действия Windows-ядра: Корзина, DNS,
// журналы событий, хранилище компонентов, кеш миниатюр. Второе значение —
// false, если действие этому ядру неизвестно (тогда категория помечается
// пропущенной, см. Process).
func processSpecial(t Target, opts Options) (Report, bool) {
	switch t.Special {
	case SpecialRecycleBin:
		return processRecycleBin(t, opts), true
	case SpecialDNSCache:
		return processDNSCache(t, opts), true
	case SpecialThumbnailCache:
		return processThumbnailCache(t, opts), true
	case SpecialEventLogs:
		return processEventLogs(t, opts), true
	case SpecialComponentCleanup:
		return processComponentCleanup(t, opts), true
	}
	return Report{Target: t}, false
}

// platformProcessDir — категории Windows, где внутри корня нужно чистить не
// всё подряд, а конкретные подпапки профилей/пакетов. Возвращает true, если
// категория обработана здесь.
func platformProcessDir(root string, t Target, opts Options, r *Report) bool {
	switch t.ID {
	case "firefox-cache":
		// только cache2 во всех профилях
		processProfileSubdirs(root, []string{"cache2"}, t, opts, r)
		return true
	case "jetbrains-logs":
		// %LOCALAPPDATA%\JetBrains\<IDE>\{log,caches}
		processProfileSubdirs(root, []string{"log", "caches"}, t, opts, r)
		return true
	case "office-cache":
		// %LOCALAPPDATA%\Microsoft\Office\<ver>\OfficeFileCache
		processProfileSubdirs(root, []string{"OfficeFileCache"}, t, opts, r)
		return true
	case "teams-new-cache":
		processUWPPackageCache(root, "MSTeams_",
			[]string{`LocalCache\Microsoft\MSTeams\Cache`,
				`LocalCache\Microsoft\MSTeams\GPUCache`,
				`LocalCache\Microsoft\MSTeams\Code Cache`,
				`LocalCache\Microsoft\MSTeams\tmp`},
			t, opts, r)
		return true
	case "store-cache":
		processUWPPackageCache(root, "Microsoft.WindowsStore_",
			[]string{`LocalCache`, `LocalState\Cache`, `AC\INetCache`},
			t, opts, r)
		return true
	case "skype-cache":
		// %APPDATA%\Skype\<profile>\{media_messaging\media_cache_v3,skylib,cache}
		processProfileSubdirs(root, []string{
			"media_messaging/media_cache_v3",
			"media_messaging/media_cache",
			"skylib",
			"cache",
		}, t, opts, r)
		return true
	}
	return false
}

// processUWPPackageCache обрабатывает папки %LOCALAPPDATA%\Packages\<prefix>*
// и удаляет указанные относительные подпути внутри каждой найденной.
func processUWPPackageCache(root, prefix string, relPaths []string, t Target, opts Options, r *Report) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			r.Errors = append(r.Errors, err.Error())
		}
		return
	}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		base := filepath.Join(root, e.Name())
		for _, rel := range relPaths {
			p := filepath.Join(base, rel)
			if _, err := os.Stat(p); err == nil {
				walkAndDelete(p, t, opts, r, true)
			}
		}
	}
}

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

// processComponentCleanup запускает официальную очистку хранилища компонентов
// (WinSxS) через DISM. Без /ResetBase — убираются только замещённые версии
// компонентов, откат последнего обновления остаётся возможным. Размер
// заранее не оценивается: DISM сам решает, что можно безопасно убрать.
func processComponentCleanup(t Target, opts Options) Report {
	r := Report{Target: t}
	if opts.DryRun {
		r.Skipped = true
		r.SkippedReason = "очистка WinSxS через DISM — размер заранее неизвестен, оценивает сам DISM"
		return r
	}
	cmd := exec.Command("Dism.exe", "/Online", "/Cleanup-Image", "/StartComponentCleanup", "/Quiet", "/NoRestart")
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("Dism /StartComponentCleanup: %v: %s", err, strings.TrimSpace(string(out))))
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

// listDrives — буквы дисков, на которых есть файловая система.
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
