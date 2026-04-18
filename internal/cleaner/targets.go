package cleaner

import (
	"os"
	"path/filepath"
	"strings"
)

// Target описывает одну категорию мусора.
type Target struct {
	ID          string   // короткий идентификатор для --categories
	Name        string   // человекочитаемое имя
	Description string   // что именно чистится и почему это безопасно
	Paths       []string // пути (могут содержать %ENV% и ~)
	// Если true — содержимое директории удаляется, но сама директория сохраняется.
	KeepRoot bool
	// Минимальный возраст файла в часах. 0 = без ограничения.
	MinAgeHours int
	// Если путь содержит один из этих подстрок после раскрытия — пропускаем (защита).
	ForbidSubstrings []string
	// Специальный обработчик (для нефайловых действий, напр. корзина / DNS).
	Special SpecialAction
	// Aggressive — категория потенциально опасна / требует явного согласия.
	// Включается только с флагом --aggressive.
	Aggressive bool
}

type SpecialAction string

const (
	SpecialNone           SpecialAction = ""
	SpecialRecycleBin     SpecialAction = "recycle_bin"
	SpecialDNSCache       SpecialAction = "dns_cache"
	SpecialThumbnailCache SpecialAction = "thumbnail_cache"
	SpecialEventLogs      SpecialAction = "event_logs"
)

// AllTargets — перечень безопасных к очистке категорий для Windows.
// Источники (общеизвестные места накопления мусора в Windows):
//   - %TEMP%, %TMP%, C:\Windows\Temp                 — временные файлы
//   - C:\Windows\Prefetch                            — кеш запуска приложений
//   - C:\Windows\SoftwareDistribution\Download       — кеш Windows Update
//   - C:\Windows\Logs\CBS                            — логи обслуживания компонентов
//   - %LOCALAPPDATA%\Microsoft\Windows\WER           — отчёты об ошибках
//   - %LOCALAPPDATA%\CrashDumps, C:\Windows\Minidump — дампы падений
//   - %LOCALAPPDATA%\Microsoft\Windows\Explorer\*.db — кеш миниатюр / icon cache
//   - %LOCALAPPDATA%\Microsoft\Windows\INetCache     — кеш IE/WebView
//   - %APPDATA%\Microsoft\Windows\Recent             — список недавних файлов
//   - Кеши браузеров (Chrome / Edge / Firefox / Brave)
//   - Корзина (все диски)                             — через Shell API / PowerShell
//   - DNS-кеш                                         — ipconfig /flushdns
func AllTargets() []Target {
	return []Target{
		{
			ID:          "user-temp",
			Name:        "Временные файлы пользователя (%TEMP%)",
			Description: "Файлы в %LOCALAPPDATA%\\Temp. Создаются приложениями и часто не удаляются.",
			Paths:       []string{`%TEMP%`, `%LOCALAPPDATA%\Temp`},
			KeepRoot:    true,
		},
		{
			ID:          "windows-temp",
			Name:        "Временные файлы Windows (C:\\Windows\\Temp)",
			Description: "Системная временная папка. Требуются права администратора для полной очистки.",
			Paths:       []string{`C:\Windows\Temp`},
			KeepRoot:    true,
		},
		{
			ID:          "prefetch",
			Name:        "Windows Prefetch",
			Description: "Кеш предзагрузки приложений. Безопасно удаляется, Windows пересоздаст по мере использования.",
			Paths:       []string{`C:\Windows\Prefetch`},
			KeepRoot:    true,
		},
		{
			ID:          "windows-update-cache",
			Name:        "Кеш Windows Update",
			Description: "Старые скачанные обновления в SoftwareDistribution\\Download.",
			Paths:       []string{`C:\Windows\SoftwareDistribution\Download`},
			KeepRoot:    true,
			MinAgeHours: 24,
		},
		{
			ID:          "delivery-optimization",
			Name:        "Delivery Optimization (P2P кеш обновлений)",
			Description: "Файлы P2P-доставки обновлений Windows.",
			Paths:       []string{`C:\Windows\ServiceProfiles\NetworkService\AppData\Local\Microsoft\Windows\DeliveryOptimization\Cache`},
			KeepRoot:    true,
		},
		{
			ID:          "cbs-logs",
			Name:        "Логи обслуживания компонентов (CBS)",
			Description: "Старые логи C:\\Windows\\Logs\\CBS.",
			Paths:       []string{`C:\Windows\Logs\CBS`},
			KeepRoot:    true,
			MinAgeHours: 72,
		},
		{
			ID:          "wer",
			Name:        "Отчёты об ошибках Windows (WER)",
			Description: "Архивированные и очередные отчёты Windows Error Reporting.",
			Paths: []string{
				`%LOCALAPPDATA%\Microsoft\Windows\WER\ReportArchive`,
				`%LOCALAPPDATA%\Microsoft\Windows\WER\ReportQueue`,
				`%PROGRAMDATA%\Microsoft\Windows\WER\ReportArchive`,
				`%PROGRAMDATA%\Microsoft\Windows\WER\ReportQueue`,
			},
			KeepRoot: true,
		},
		{
			ID:          "crash-dumps",
			Name:        "Дампы падений приложений и ядра",
			Description: "*.dmp файлы. Обычно не нужны обычному пользователю.",
			Paths: []string{
				`%LOCALAPPDATA%\CrashDumps`,
				`C:\Windows\Minidump`,
				`C:\Windows\MEMORY.DMP`,
			},
			KeepRoot: true,
		},
		{
			ID:          "recent",
			Name:        "Список недавних файлов",
			Description: "Ярлыки в %APPDATA%\\Microsoft\\Windows\\Recent.",
			Paths:       []string{`%APPDATA%\Microsoft\Windows\Recent`},
			KeepRoot:    true,
		},
		{
			ID:          "inet-cache",
			Name:        "Кеш IE / WebView / WinINet",
			Description: "Кеш системного HTTP-стека (используется многими приложениями).",
			Paths:       []string{`%LOCALAPPDATA%\Microsoft\Windows\INetCache`},
			KeepRoot:    true,
		},
		{
			ID:          "thumbnail-cache",
			Name:        "Кеш миниатюр / значков",
			Description: "Файлы thumbcache_*.db и iconcache_*.db. Windows пересоздаст.",
			Paths:       []string{`%LOCALAPPDATA%\Microsoft\Windows\Explorer`},
			Special:     SpecialThumbnailCache,
		},
		{
			ID:          "chrome-cache",
			Name:        "Кеш Google Chrome",
			Description: "Кеш страниц и кода Chrome (не трогает пароли/cookies/историю).",
			Paths: []string{
				`%LOCALAPPDATA%\Google\Chrome\User Data\Default\Cache`,
				`%LOCALAPPDATA%\Google\Chrome\User Data\Default\Code Cache`,
				`%LOCALAPPDATA%\Google\Chrome\User Data\Default\GPUCache`,
			},
			KeepRoot: true,
		},
		{
			ID:          "edge-cache",
			Name:        "Кеш Microsoft Edge",
			Description: "Кеш страниц и кода Edge.",
			Paths: []string{
				`%LOCALAPPDATA%\Microsoft\Edge\User Data\Default\Cache`,
				`%LOCALAPPDATA%\Microsoft\Edge\User Data\Default\Code Cache`,
				`%LOCALAPPDATA%\Microsoft\Edge\User Data\Default\GPUCache`,
			},
			KeepRoot: true,
		},
		{
			ID:          "brave-cache",
			Name:        "Кеш Brave",
			Description: "Кеш страниц и кода Brave.",
			Paths: []string{
				`%LOCALAPPDATA%\BraveSoftware\Brave-Browser\User Data\Default\Cache`,
				`%LOCALAPPDATA%\BraveSoftware\Brave-Browser\User Data\Default\Code Cache`,
			},
			KeepRoot: true,
		},
		{
			ID:          "firefox-cache",
			Name:        "Кеш Firefox",
			Description: "Все cache2/ папки во всех профилях Firefox.",
			Paths:       []string{`%LOCALAPPDATA%\Mozilla\Firefox\Profiles`},
			// очищаем через glob в cleaner.go
		},
		{
			ID:          "nuget-cache",
			Name:        "Кеш NuGet",
			Description: "Кеш пакетов NuGet (%USERPROFILE%\\.nuget\\packages). Восстанавливается при следующей сборке.",
			Paths:       []string{`%USERPROFILE%\.nuget\packages`},
			KeepRoot:    true,
			MinAgeHours: 24 * 30,
		},
		{
			ID:          "pip-cache",
			Name:        "Кеш pip",
			Description: "%LOCALAPPDATA%\\pip\\Cache — восстанавливается при следующей установке.",
			Paths:       []string{`%LOCALAPPDATA%\pip\Cache`},
			KeepRoot:    true,
		},
		{
			ID:          "npm-cache",
			Name:        "Кеш npm",
			Description: "%APPDATA%\\npm-cache — безопасно удаляется.",
			Paths:       []string{`%APPDATA%\npm-cache`, `%LOCALAPPDATA%\npm-cache`},
			KeepRoot:    true,
		},
		{
			ID:          "recycle-bin",
			Name:        "Корзина (все диски)",
			Description: "Очистка корзины через Shell API (PowerShell Clear-RecycleBin).",
			Special:     SpecialRecycleBin,
		},
		{
			ID:          "dns-cache",
			Name:        "DNS-кеш",
			Description: "ipconfig /flushdns",
			Special:     SpecialDNSCache,
		},

		// ---- Дополнительные безопасные категории ----
		{
			ID:          "jump-lists",
			Name:        "Jump Lists (авто/пользовательские)",
			Description: "%APPDATA%\\Microsoft\\Windows\\Recent\\{AutomaticDestinations,CustomDestinations}.",
			Paths: []string{
				`%APPDATA%\Microsoft\Windows\Recent\AutomaticDestinations`,
				`%APPDATA%\Microsoft\Windows\Recent\CustomDestinations`,
			},
			KeepRoot: true,
		},
		{
			ID:          "windows-logs",
			Name:        "Разные логи Windows",
			Description: "C:\\Windows\\Logs (кроме CBS, который отдельно), setupapi*.log.",
			Paths: []string{
				`C:\Windows\Logs\DISM`,
				`C:\Windows\Logs\DPX`,
				`C:\Windows\Logs\MoSetup`,
				`C:\Windows\Logs\WindowsUpdate`,
				`C:\Windows\Logs\waasmedic`,
				`C:\Windows\inf\setupapi.dev.log`,
				`C:\Windows\inf\setupapi.app.log`,
				`C:\Windows\inf\setupapi.setup.log`,
				`C:\Windows\inf\setupapi.offline.log`,
			},
			KeepRoot:    true,
			MinAgeHours: 72,
		},
		{
			ID:          "panther",
			Name:        "Логи установки/обновления Windows (Panther)",
			Description: "C:\\Windows\\Panther и C:\\Windows\\Panther\\UnattendGC.",
			Paths:       []string{`C:\Windows\Panther`},
			KeepRoot:    true,
			MinAgeHours: 24 * 7,
		},
		{
			ID:          "livekernel-reports",
			Name:        "LiveKernelReports (отчёты о зависаниях ядра)",
			Description: "C:\\Windows\\LiveKernelReports — большие .dmp отчёты ядра.",
			Paths:       []string{`C:\Windows\LiveKernelReports`},
			KeepRoot:    true,
		},
		{
			ID:          "defender-scan-history",
			Name:        "История сканирований Windows Defender",
			Description: "C:\\ProgramData\\Microsoft\\Windows Defender\\Scans\\History — логи сканов.",
			Paths:       []string{`C:\ProgramData\Microsoft\Windows Defender\Scans\History`},
			KeepRoot:    true,
			MinAgeHours: 24 * 7,
		},
		{
			ID:          "font-cache",
			Name:        "Кеш шрифтов",
			Description: "FontCache*.dat. Пересоздаётся. Может потребоваться перезапуск службы FontCache.",
			Paths: []string{
				`C:\Windows\ServiceProfiles\LocalService\AppData\Local\FontCache`,
				`C:\Windows\System32\FNTCACHE.DAT`,
			},
			KeepRoot: true,
		},
		{
			ID:          "sd-datastore-logs",
			Name:        "Логи DataStore Windows Update",
			Description: "C:\\Windows\\SoftwareDistribution\\DataStore\\Logs.",
			Paths:       []string{`C:\Windows\SoftwareDistribution\DataStore\Logs`},
			KeepRoot:    true,
			MinAgeHours: 72,
		},
		{
			ID:          "teams-cache",
			Name:        "Кеш Microsoft Teams (classic)",
			Description: "%APPDATA%\\Microsoft\\Teams\\{Cache,Code Cache,GPUCache,tmp,blob_storage,Service Worker\\CacheStorage}.",
			Paths: []string{
				`%APPDATA%\Microsoft\Teams\Cache`,
				`%APPDATA%\Microsoft\Teams\Code Cache`,
				`%APPDATA%\Microsoft\Teams\GPUCache`,
				`%APPDATA%\Microsoft\Teams\tmp`,
				`%APPDATA%\Microsoft\Teams\blob_storage`,
				`%APPDATA%\Microsoft\Teams\Service Worker\CacheStorage`,
			},
			KeepRoot: true,
		},
		{
			ID:          "teams-new-cache",
			Name:        "Кеш Microsoft Teams (new)",
			Description: "%LOCALAPPDATA%\\Packages\\MSTeams_*\\LocalCache\\...",
			Paths:       []string{`%LOCALAPPDATA%\Packages`},
			// чистится специализированной логикой по glob — ниже в cleaner.go
		},
		{
			ID:          "discord-cache",
			Name:        "Кеш Discord",
			Description: "%APPDATA%\\discord\\{Cache,Code Cache,GPUCache}.",
			Paths: []string{
				`%APPDATA%\discord\Cache`,
				`%APPDATA%\discord\Code Cache`,
				`%APPDATA%\discord\GPUCache`,
			},
			KeepRoot: true,
		},
		{
			ID:          "slack-cache",
			Name:        "Кеш Slack",
			Description: "%APPDATA%\\Slack\\{Cache,Code Cache,GPUCache,Service Worker\\CacheStorage}.",
			Paths: []string{
				`%APPDATA%\Slack\Cache`,
				`%APPDATA%\Slack\Code Cache`,
				`%APPDATA%\Slack\GPUCache`,
				`%APPDATA%\Slack\Service Worker\CacheStorage`,
			},
			KeepRoot: true,
		},
		{
			ID:          "telegram-cache",
			Name:        "Кеш Telegram Desktop",
			Description: "%APPDATA%\\Telegram Desktop\\tdata\\user_data\\cache и emoji/webview caches.",
			Paths: []string{
				`%APPDATA%\Telegram Desktop\tdata\user_data\cache`,
				`%APPDATA%\Telegram Desktop\tdata\emoji`,
				`%APPDATA%\Telegram Desktop\tdata\webview`,
			},
			KeepRoot: true,
		},
		{
			ID:          "spotify-cache",
			Name:        "Кеш Spotify",
			Description: "%LOCALAPPDATA%\\Spotify\\Data и Storage (загруженные треки/обложки).",
			Paths: []string{
				`%LOCALAPPDATA%\Spotify\Data`,
				`%LOCALAPPDATA%\Spotify\Storage`,
				`%LOCALAPPDATA%\Spotify\Browser`,
			},
			KeepRoot: true,
		},
		{
			ID:          "vscode-cache",
			Name:        "Кеш VS Code",
			Description: "%APPDATA%\\Code\\{Cache,CachedData,CachedExtensions,Code Cache,GPUCache,logs}.",
			Paths: []string{
				`%APPDATA%\Code\Cache`,
				`%APPDATA%\Code\CachedData`,
				`%APPDATA%\Code\CachedExtensions`,
				`%APPDATA%\Code\CachedExtensionVSIXs`,
				`%APPDATA%\Code\Code Cache`,
				`%APPDATA%\Code\GPUCache`,
				`%APPDATA%\Code\logs`,
				`%APPDATA%\Code\User\workspaceStorage`,
			},
			KeepRoot:    true,
			MinAgeHours: 24 * 3,
		},
		{
			ID:          "jetbrains-logs",
			Name:        "Логи и кеши JetBrains IDE",
			Description: "%LOCALAPPDATA%\\JetBrains\\*\\{log,caches} во всех IDE.",
			Paths:       []string{`%LOCALAPPDATA%\JetBrains`},
			// чистится glob-логикой ниже
		},
		{
			ID:          "office-cache",
			Name:        "Кеш Office Document Cache",
			Description: "%LOCALAPPDATA%\\Microsoft\\Office\\*\\OfficeFileCache (не трогает несохранённые).",
			Paths:       []string{`%LOCALAPPDATA%\Microsoft\Office`},
			// чистится glob-логикой (только OfficeFileCache)
		},
		{
			ID:          "adobe-media-cache",
			Name:        "Кеш медиа Adobe (Premiere/AE и т.д.)",
			Description: "%APPDATA%\\Adobe\\Common\\{Media Cache,Media Cache Files}.",
			Paths: []string{
				`%APPDATA%\Adobe\Common\Media Cache`,
				`%APPDATA%\Adobe\Common\Media Cache Files`,
			},
			KeepRoot: true,
		},
		{
			ID:          "go-build-cache",
			Name:        "Кеш сборки Go",
			Description: "%LOCALAPPDATA%\\go-build — восстанавливается при следующей сборке.",
			Paths:       []string{`%LOCALAPPDATA%\go-build`},
			KeepRoot:    true,
			MinAgeHours: 24 * 14,
		},
		{
			ID:          "gradle-cache",
			Name:        "Кеш Gradle",
			Description: "%USERPROFILE%\\.gradle\\caches — восстанавливается при следующей сборке.",
			Paths:       []string{`%USERPROFILE%\.gradle\caches`},
			KeepRoot:    true,
			MinAgeHours: 24 * 30,
		},
		{
			ID:          "yarn-cache",
			Name:        "Кеш Yarn",
			Description: "%LOCALAPPDATA%\\Yarn\\Cache.",
			Paths:       []string{`%LOCALAPPDATA%\Yarn\Cache`},
			KeepRoot:    true,
		},
		{
			ID:          "steam-htmlcache",
			Name:        "HTML-кеш Steam",
			Description: "%LOCALAPPDATA%\\Steam\\htmlcache и logs клиента.",
			Paths: []string{
				`%LOCALAPPDATA%\Steam\htmlcache`,
				`C:\Program Files (x86)\Steam\logs`,
				`C:\Program Files (x86)\Steam\appcache\httpcache`,
			},
			KeepRoot: true,
		},
		{
			ID:          "nvidia-cache",
			Name:        "Кеш драйверов NVIDIA (GLCache/DXCache)",
			Description: "%LOCALAPPDATA%\\NVIDIA\\{GLCache,DXCache,D3DSCache}.",
			Paths: []string{
				`%LOCALAPPDATA%\NVIDIA\GLCache`,
				`%LOCALAPPDATA%\NVIDIA\DXCache`,
				`%LOCALAPPDATA%\NVIDIA\D3DSCache`,
				`%LOCALAPPDATA%\NVIDIA Corporation\NV_Cache`,
			},
			KeepRoot: true,
		},
		{
			ID:          "amd-cache",
			Name:        "Кеш шейдеров AMD",
			Description: "%LOCALAPPDATA%\\AMD\\{DxCache,GLCache,VkCache}.",
			Paths: []string{
				`%LOCALAPPDATA%\AMD\DxCache`,
				`%LOCALAPPDATA%\AMD\GLCache`,
				`%LOCALAPPDATA%\AMD\VkCache`,
			},
			KeepRoot: true,
		},
		{
			ID:          "dx-shader-cache",
			Name:        "DirectX Shader Cache",
			Description: "%LOCALAPPDATA%\\D3DSCache.",
			Paths:       []string{`%LOCALAPPDATA%\D3DSCache`},
			KeepRoot:    true,
		},

		// ---- Агрессивные (только с --aggressive) ----
		{
			ID:          "windows-old",
			Name:        "C:\\Windows.old (старая ОС после обновления)",
			Description: "Папка старой установки Windows. Может занимать десятки ГБ. После удаления откат невозможен.",
			Paths:       []string{`C:\Windows.old`},
			Aggressive:  true,
		},
		{
			ID:          "windows-installer-upgrade",
			Name:        "$WINDOWS.~BT / $WINDOWS.~WS (кеш обновления ОС)",
			Description: "Временные файлы установки крупных обновлений Windows.",
			Paths: []string{
				`C:\$WINDOWS.~BT`,
				`C:\$WINDOWS.~WS`,
				`C:\ESD\Windows`,
			},
			Aggressive: true,
		},
		{
			ID:          "event-logs",
			Name:        "Журналы событий Windows (*.evtx)",
			Description: "C:\\Windows\\System32\\winevt\\Logs. Очищаются через wevtutil cl. Потеряете историю системных событий.",
			Special:     SpecialEventLogs,
			Aggressive:  true,
		},
		{
			ID:          "iis-logs",
			Name:        "Логи IIS",
			Description: "C:\\inetpub\\logs\\LogFiles.",
			Paths:       []string{`C:\inetpub\logs\LogFiles`},
			KeepRoot:    true,
			MinAgeHours: 24 * 30,
			Aggressive:  true,
		},
		{
			ID:          "downloads-old",
			Name:        "Старые файлы из Downloads (>90 дней)",
			Description: "%USERPROFILE%\\Downloads: удаляются файлы, не менявшиеся >90 дней. Проверьте перед запуском!",
			Paths:       []string{`%USERPROFILE%\Downloads`},
			KeepRoot:    true,
			MinAgeHours: 24 * 90,
			Aggressive:  true,
		},
		{
			ID:          "maven-cache",
			Name:        "Локальный репозиторий Maven",
			Description: "%USERPROFILE%\\.m2\\repository — восстанавливается при следующей сборке (может быть долго).",
			Paths:       []string{`%USERPROFILE%\.m2\repository`},
			KeepRoot:    true,
			MinAgeHours: 24 * 60,
			Aggressive:  true,
		},
	}
}

// ExpandPath раскрывает %ENV% переменные и ~ в путь Windows.
func ExpandPath(p string) string {
	out := os.Expand(rawWinToUnixVars(p), os.Getenv)
	if strings.HasPrefix(out, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			out = filepath.Join(home, strings.TrimPrefix(out, "~"))
		}
	}
	return out
}

// rawWinToUnixVars превращает %VAR% в ${VAR} для os.Expand.
func rawWinToUnixVars(p string) string {
	var b strings.Builder
	i := 0
	for i < len(p) {
		if p[i] == '%' {
			j := strings.IndexByte(p[i+1:], '%')
			if j > 0 {
				name := p[i+1 : i+1+j]
				b.WriteString("${")
				b.WriteString(name)
				b.WriteString("}")
				i += j + 2
				continue
			}
		}
		b.WriteByte(p[i])
		i++
	}
	return b.String()
}
