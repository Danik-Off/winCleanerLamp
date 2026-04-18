package cleaner

import (
	"os"
)

// SysInfoEntry — запись об одном системном файле или каталоге.
type SysInfoEntry struct {
	Name string
	Path string
	Size int64
	Hint string
}

// GatherSysInfo возвращает информацию о «больших» системных файлах,
// которые нельзя трогать напрямую, но полезно знать их размер.
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
		},
		{
			Name: "System Volume Information",
			Path: `C:\System Volume Information`,
			Hint: "    Точки восстановления и Volume Shadow Copy. Управление: vssadmin / «Защита системы».",
		},
		{
			Name: "Installer",
			Path: `C:\Windows\Installer`,
			Hint: "    Кеш MSI-установщиков. НЕ удалять вручную — поломает обновление/удаление программ.",
		},
	}

	for i := range entries {
		entries[i].Size = pathSize(entries[i].Path)
	}
	return entries
}

func pathSize(p string) int64 {
	info, err := os.Lstat(p)
	if err != nil {
		return 0
	}
	if !info.IsDir() {
		return info.Size()
	}
	sz, _ := dirSize(p)
	return sz
}
