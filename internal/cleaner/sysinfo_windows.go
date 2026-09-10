//go:build windows

package cleaner

import "sync"

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
			Name:  "WinSxS",
			Path:  `C:\Windows\WinSxS`,
			Hint:  "    Хранилище компонентов Windows. НЕ удалять! Только через dism /Online /Cleanup-Image.",
			IsDir: true,
		},
		{
			Name:  "System Volume Information",
			Path:  `C:\System Volume Information`,
			Hint:  "    Точки восстановления и Volume Shadow Copy. Управление: vssadmin / «Защита системы».",
			IsDir: true,
		},
		{
			Name:  "Installer",
			Path:  `C:\Windows\Installer`,
			Hint:  "    Кеш MSI-установщиков. НЕ удалять вручную — поломает обновление/удаление программ.",
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

// SysInfoAdvice — подсказки по управлению системными файлами Windows.
// Печатаются в --sysinfo; сами эти файлы утилита не трогает.
func SysInfoAdvice() []string {
	return []string{
		"• hiberfil.sys  — отключить:  powercfg /h off",
		"• pagefile.sys  — размер:    Система → Дополнительные → Быстродействие → Виртуальная память",
		"• WinSxS        — анализ:    dism /Online /Cleanup-Image /AnalyzeComponentStore",
		"                  очистка:   dism /Online /Cleanup-Image /StartComponentCleanup /ResetBase",
	}
}
