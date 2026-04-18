package cleaner

import (
	"os"
	"sync"
	"time"
)

// SysInfoEntry — запись об одном системном файле или каталоге.
type SysInfoEntry struct {
	Name    string
	Path    string
	Size    int64 // -1 = не удалось вычислить (таймаут / нет доступа)
	Hint    string
	IsDir   bool
	Exists  bool
}

// GatherSysInfo возвращает информацию о «больших» системных файлах,
// которые нельзя трогать напрямую, но полезно знать их размер.
// Размеры каталогов вычисляются параллельно с таймаутом 3 сек на каждый.
func GatherSysInfo() []SysInfoEntry {
	entries := []SysInfoEntry{
		{
			Name: "hiberfil.sys",
			Path: `C:\hiberfil.sys`,
			Hint: "    Файл гибернации (~= размер ОЗУ). Отключите, если не пользуетесь режимом «сон»/быстрым запуском.",
		},
		{
			Name: "pagefile.sys",
			Path: `C:\pagefile.sys`,
			Hint: "    Файл подкачки. Удалять нельзя — только ограничить размер.",
		},
		{
			Name: "swapfile.sys",
			Path: `C:\swapfile.sys`,
			Hint: "    Файл свопа для UWP-приложений. Удалять нельзя.",
		},
		{
			Name: "MEMORY.DMP",
			Path: `C:\Windows\MEMORY.DMP`,
			Hint: "    Полный дамп ОЗУ после BSOD. Безопасно удалить (чистится категорией crash-dumps).",
		},
		{
			Name: "WinSxS",
			Path: `C:\Windows\WinSxS`,
			Hint: "    Хранилище компонентов Windows. НЕ удалять! Только через dism /Online /Cleanup-Image.",
			IsDir: true,
		},
		{
			Name: "System Volume Information",
			Path: `C:\System Volume Information`,
			Hint: "    Точки восстановления и Volume Shadow Copy. Управление: vssadmin / «Защита системы».",
			IsDir: true,
		},
		{
			Name: "Installer",
			Path: `C:\Windows\Installer`,
			Hint: "    Кеш MSI-установщиков. НЕ удалять вручную — поломает обновление/удаление программ.",
			IsDir: true,
		},
	}

	// Вычисляем размеры параллельно с таймаутом
	var wg sync.WaitGroup
	for i := range entries {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			entries[idx].Size, entries[idx].Exists = pathSizeFast(entries[idx].Path, entries[idx].IsDir)
		}(i)
	}
	wg.Wait()
	return entries
}

// pathSizeFast возвращает размер файла/каталога.
// Для файлов — мгновенный os.Lstat.
// Для каталогов — dirSize с таймаутом 3 секунды.
// Возвращает (size, exists). size=-1 если таймаут или ошибка доступа.
func pathSizeFast(p string, isDir bool) (int64, bool) {
	info, err := os.Lstat(p)
	if err != nil {
		return 0, false
	}
	if !info.IsDir() {
		return info.Size(), true
	}

	// Для каталогов используем горутину с таймаутом
	type result struct{ size int64 }
	ch := make(chan result, 1)
	go func() {
		sz, _ := dirSize(p)
		ch <- result{sz}
	}()

	select {
	case r := <-ch:
		return r.size, true
	case <-time.After(3 * time.Second):
		return -1, true // таймаут — каталог существует, но размер не удалось вычислить быстро
	}
}
