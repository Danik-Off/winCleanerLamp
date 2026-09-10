//go:build darwin

package cleaner

import (
	"os/exec"
	"strings"
	"sync"
)

// GatherSysInfo возвращает информацию о «больших» местах macOS, которые
// нельзя чистить обычным удалением файлов, но полезно знать их размер.
func GatherSysInfo() []SysInfoEntry {
	entries := []SysInfoEntry{
		{
			Name:  "sleepimage",
			Path:  "/private/var/vm/sleepimage",
			Hint:  "    Образ памяти для режима сна (~= объём ОЗУ). Удалять нельзя — управляется pmset hibernatemode.",
			IsDir: false,
		},
		{
			Name:  "swapfiles",
			Path:  "/private/var/vm",
			Hint:  "    Файлы подкачки. Удалять нельзя — размером управляет сама macOS.",
			IsDir: true,
		},
		{
			Name:  "~/Library/Caches",
			Path:  userCacheDir(),
			Hint:  "    Кеш приложений пользователя. Чистится категорией user-cache.",
			IsDir: true,
		},
		{
			Name:  "/Library/Caches",
			Path:  "/Library/Caches",
			Hint:  "    Общесистемный кеш. Чистится категорией system-cache (нужен sudo).",
			IsDir: true,
		},
		{
			Name:  "Xcode DerivedData",
			Path:  sub(userLibraryDir(), "Developer/Xcode/DerivedData"),
			Hint:  "    Промежуточные сборки Xcode. Чистится категорией xcode-derived-data.",
			IsDir: true,
		},
		{
			Name:  "CoreSimulator",
			Path:  sub(userLibraryDir(), "Developer/CoreSimulator"),
			Hint:  "    Образы симуляторов iOS. Удалять только через xcrun simctl delete unavailable.",
			IsDir: true,
		},
		{
			Name:  "Резервные копии iOS",
			Path:  sub(userAppSupportDir(), "MobileSync/Backup"),
			Hint:  "    Копии устройств. Это НЕ кеш — удаление необратимо (категория ios-backups).",
			IsDir: true,
		},
		{
			Name:  "/private/var/log",
			Path:  "/private/var/log",
			Hint:  "    Системные журналы. Старые файлы чистит категория system-logs (нужен sudo).",
			IsDir: true,
		},
	}

	var wg sync.WaitGroup
	for i := range entries {
		if entries[i].Path == "" {
			continue
		}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			entries[idx].Size, entries[idx].Exists = pathSizeFast(entries[idx].Path, entries[idx].IsDir)
		}(i)
	}
	wg.Wait()

	// Локальные снимки Time Machine занимают место на системном томе, но
	// файлами не представлены — показываем хотя бы их количество.
	if n := localSnapshotCount(); n > 0 {
		entries = append(entries, SysInfoEntry{
			Name:   "Снимки Time Machine",
			Path:   "локальные снимки APFS",
			Size:   -1,
			Exists: true,
			Hint:   "    Занимают место на системном томе. Удаление: категория local-snapshots (tmutil).",
		})
	}
	return entries
}

// SysInfoAdvice — подсказки по управлению большими системными объектами.
func SysInfoAdvice() []string {
	return []string{
		"• sleepimage   — режим сна:  sudo pmset -a hibernatemode 0 (образ перестанет создаваться)",
		"• swapfiles    — размером управляет macOS, вручную не удаляют",
		"• снимки TM    — просмотр:   tmutil listlocalsnapshots /",
		"• симуляторы   — очистка:    xcrun simctl delete unavailable",
		"• Homebrew     — очистка:    brew cleanup --prune=all (категория brew-cleanup)",
	}
}

// localSnapshotCount — сколько локальных снимков APFS сейчас на системном
// томе.
func localSnapshotCount() int {
	if _, err := exec.LookPath("tmutil"); err != nil {
		return 0
	}
	out, err := exec.Command("tmutil", "listlocalsnapshots", "/").Output()
	if err != nil {
		return 0
	}
	n := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "com.apple.TimeMachine.") {
			n++
		}
	}
	return n
}
