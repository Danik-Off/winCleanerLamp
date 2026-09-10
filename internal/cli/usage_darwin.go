//go:build darwin

package cli

// Тексты справки, специфичные для macOS-ядра.
const (
	osTitle                = "macOS"
	usageExampleCategories = "user-cache,trash,xcode-derived-data"
	usageExampleExclude    = "trash,ios-backups"
	usageShortcutsHint     = "битые псевдонимы и .app-ссылки"
	usageExampleFilePath   = "/Users/user/путь/файл"
	usageExampleDirPath    = "/Users/user/путь/папка"

	// Подписи, попадающие в описания флагов и в вывод команд.
	usageAggressiveExamples = "$TMPDIR, локальные снимки Time Machine, резервные копии iOS, старые Downloads"
	usageSystemDupDirs      = "/System, /Applications, /Library, /usr"
	usageAppsSource         = "/Applications и Homebrew"
	usageExecutableLabel    = "исполняемых файлов"
)

func usageHints() []string {
	return []string{
		"Системные категории (/Library/Caches, /private/var/log) требуют sudo — без него они будут пропущены.",
		"Часть папок в ~/Library защищена SIP/TCC: без выданного терминалу «Полного доступа к диску» они читаются, но не удаляются.",
		"Резервные копии iOS (--categories ios-backups) — это НЕ кеш: удаляйте их только осознанно, восстановить их нельзя.",
	}
}
