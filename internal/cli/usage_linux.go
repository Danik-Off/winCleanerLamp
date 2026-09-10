//go:build linux

package cli

// Тексты справки, специфичные для Linux-ядра.
const (
	osTitle                = "Linux"
	usageExampleCategories = "user-cache,thumbnails,trash"
	usageExampleExclude    = "trash,journal-logs"
	usageShortcutsHint     = ".desktop с несуществующим Exec"
	usageExampleFilePath   = "/home/user/путь/файл"
	usageExampleDirPath    = "/home/user/путь/папка"

	// Подписи, попадающие в описания флагов и в вывод команд.
	usageAggressiveExamples = "/var/tmp, старые логи, старые Downloads, кеши сборки"
	usageSystemDupDirs      = "/usr, /opt, /var, /etc"
	usageAppsSource         = "dpkg/rpm/pacman/flatpak/snap и ярлыки .desktop"
	usageExecutableLabel    = "исполняемых файлов"
)

func usageHints() []string {
	return []string{
		"Системные категории (кеши apt/dnf/pacman, journald, /var/tmp) требуют sudo — без него они будут пропущены.",
		"Кеши пакетных менеджеров чистятся их собственными командами (apt-get clean, dnf clean, paccache) — ядро удаляет только файлы кеша, не трогая базу пакетов.",
		"Категории snap/flatpak удаляют лишь отключённые ревизии и неиспользуемые runtime, установленные приложения не затрагиваются.",
	}
}
