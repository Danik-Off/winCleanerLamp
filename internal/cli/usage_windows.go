//go:build windows

package cli

// Тексты справки, специфичные для Windows-ядра: примеры путей, названия
// категорий и подсказки про права администратора.
const (
	osTitle                = "Windows"
	usageExampleCategories = "user-temp,prefetch,recycle-bin"
	usageExampleExclude    = "recycle-bin,dns-cache"
	usageShortcutsHint     = ".lnk с несуществующей целью"
	usageExampleFilePath   = `C:\путь\файл`
	usageExampleDirPath    = `C:\путь\папка`

	// Подписи, попадающие в описания флагов и в вывод команд.
	usageAggressiveExamples = `Windows.old, $WINDOWS.~BT, event-logs, старые Downloads`
	usageSystemDupDirs      = "Program Files, Windows, ProgramData"
	usageAppsSource         = "реестр Uninstall"
	usageExecutableLabel    = ".exe"
)

func usageHints() []string {
	return []string{
		`Для очистки C:\Windows\Temp, Prefetch, SoftwareDistribution и т.п. запускайте консоль от имени администратора.`,
	}
}
