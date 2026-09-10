//go:build linux

package cleaner

import (
	"os"
	"strconv"
	"strings"
	"sync"
)

// GatherSysInfo возвращает информацию о «больших» местах Linux-системы,
// которые нельзя чистить обычным удалением файлов, но полезно знать их
// размер: своп, журналы, кеши пакетных менеджеров, образы контейнеров.
func GatherSysInfo() []SysInfoEntry {
	entries := []SysInfoEntry{
		{
			Name: "swap",
			Path: swapPath(),
			Hint: "    Файл или раздел подкачки. Удалять нельзя — размер меняется настройками swap (swapon/zram).",
		},
		{
			Name:  "journald",
			Path:  "/var/log/journal",
			Hint:  "    Журналы systemd. Очистка: journalctl --vacuum-time=7d (категория journal-logs).",
			IsDir: true,
		},
		{
			Name:  "/var/log",
			Path:  "/var/log",
			Hint:  "    Системные логи. Ротацию настраивает logrotate; старые файлы чистит категория system-logs.",
			IsDir: true,
		},
		{
			Name:  "кеш пакетов",
			Path:  packageCacheDir(),
			Hint:  "    Скачанные пакеты. Очистка: категория package-cache (apt-get clean / dnf clean packages / pacman -Sc).",
			IsDir: true,
		},
		{
			Name:  "snap",
			Path:  "/var/lib/snapd/snaps",
			Hint:  "    Образы snap, включая старые ревизии. Уменьшить: snap set system refresh.retain=2 (категория snap-disabled).",
			IsDir: true,
		},
		{
			Name:  "flatpak",
			Path:  "/var/lib/flatpak",
			Hint:  "    Приложения и runtime flatpak. Очистка неиспользуемых: категория flatpak-unused.",
			IsDir: true,
		},
		{
			Name:  "docker",
			Path:  "/var/lib/docker",
			Hint:  "    Образы и тома Docker. НЕ удалять вручную — только docker system prune.",
			IsDir: true,
		},
		{
			Name:  "~/.cache",
			Path:  userCacheDir(),
			Hint:  "    Пользовательский кеш. Чистится категорией user-cache.",
			IsDir: true,
		},
	}

	// Вычисляем размеры параллельно с таймаутом
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

	// Своп-раздел — блочное устройство: Lstat покажет нулевой размер, а
	// реальный объём знает только ядро (третья колонка /proc/swaps).
	for i := range entries {
		if entries[i].Name == "swap" && entries[i].Size <= 0 && entries[i].Path != "" {
			if kb := swapSizeKB(); kb > 0 {
				entries[i].Size = kb * 1024
				entries[i].Exists = true
			}
		}
	}
	return entries
}

// SysInfoAdvice — подсказки по управлению большими системными объектами.
func SysInfoAdvice() []string {
	return []string{
		"• swap        — размер:   swapon --show, zramctl (файл подкачки не удаляют вручную)",
		"• journald    — усечение: journalctl --vacuum-time=7d или --vacuum-size=200M",
		"• пакеты      — очистка:  apt-get clean / dnf clean packages / pacman -Sc",
		"• snap        — хранить меньше ревизий: snap set system refresh.retain=2",
		"• docker      — очистка:  docker system prune (учтите: удаляет неиспользуемые образы и тома)",
	}
}

// swapPath — первый своп из /proc/swaps (файл или раздел). Пустая строка,
// если своп не используется.
func swapPath() string {
	data, err := os.ReadFile("/proc/swaps")
	if err != nil {
		return ""
	}
	for i, line := range strings.Split(string(data), "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // заголовок
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			return fields[0]
		}
	}
	return ""
}

// packageCacheDir — каталог кеша того пакетного менеджера, который есть в
// системе.
func packageCacheDir() string {
	for _, c := range packageCacheCleaners() {
		if _, err := os.Stat(c.dir); err == nil {
			return c.dir
		}
	}
	return ""
}

// swapSizeKB — размер свопа по данным /proc/swaps; используется только для
// диагностики размера раздела, который dirSize посчитать не может.
func swapSizeKB() int64 {
	data, err := os.ReadFile("/proc/swaps")
	if err != nil {
		return 0
	}
	for i, line := range strings.Split(string(data), "\n") {
		if i == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		if n, err := strconv.ParseInt(fields[2], 10, 64); err == nil {
			return n
		}
	}
	return 0
}
