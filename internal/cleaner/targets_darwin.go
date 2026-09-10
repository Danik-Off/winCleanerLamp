//go:build darwin

package cleaner

// AllTargets — перечень безопасных к очистке категорий для macOS.
//
// Источники (общеизвестные места накопления мусора в macOS):
//   - ~/Library/Caches, ~/Library/Logs        — кеши и логи приложений
//   - /Library/Caches, /private/var/log       — то же на уровне системы (нужен sudo)
//   - ~/.Trash                                — Корзина
//   - $TMPDIR (/private/var/folders/.../T)    — временные файлы пользователя
//   - ~/Library/Developer/Xcode/DerivedData   — промежуточные результаты сборок
//   - ~/Library/Developer/CoreSimulator       — образы и кеши симуляторов iOS
//   - ~/Library/Caches/Homebrew               — скачанные бутылки Homebrew
//   - QuickLook, DNS, локальные снимки Time Machine — через системные утилиты
//
// Отдельно стоит помнить про SIP и TCC: часть каталогов в ~/Library
// (например, Mail, Messages, Safari) защищена системой, и без выданного
// терминалу «Полного доступа к диску» удаление в них не сработает. Такие
// каталоги сюда не включены.
func AllTargets() []Target {
	cache := userCacheDir()
	logs := userLogsDir()
	support := userAppSupportDir()
	home := userHomeDir()
	tmp := userTempDir()

	targets := []Target{
		{
			ID:          "user-cache",
			Name:        "Кеш приложений (~/Library/Caches)",
			Description: "Кеши всех программ пользователя. Создаются заново по мере работы.",
			Paths:       paths(cache),
			KeepRoot:    true,
			MinAgeHours: 24,
			// В Caches некоторые программы держат не кеш, а рабочее
			// состояние и учётные данные — их не трогаем.
			ForbidSubstrings: []string{
				"/com.apple.keychainaccess", "/CloudKit", "/com.apple.Safari/SafeBrowsing",
			},
		},
		{
			ID:          "user-logs",
			Name:        "Логи приложений (~/Library/Logs)",
			Description: "Журналы программ пользователя старше 7 дней.",
			Paths:       paths(logs),
			KeepRoot:    true,
			MinAgeHours: 24 * 7,
		},
		{
			ID:          "crash-reports",
			Name:        "Отчёты о падениях",
			Description: "DiagnosticReports пользователя и системы (.crash, .ips, .spin).",
			Paths: paths(
				sub(logs, "DiagnosticReports"),
				"/Library/Logs/DiagnosticReports",
			),
			KeepRoot:    true,
			MinAgeHours: 24,
		},
		{
			ID:          "trash",
			Name:        "Корзина",
			Description: "~/.Trash — содержимое Корзины текущего пользователя.",
			Special:     SpecialTrash,
		},
		{
			ID:          "saved-app-state",
			Name:        "Сохранённые состояния окон",
			Description: "~/Library/Saved Application State — программы откроются с чистого листа.",
			Paths:       paths(sub(userLibraryDir(), "Saved Application State")),
			KeepRoot:    true,
			Aggressive:  true,
		},
		{
			ID:          "quicklook-cache",
			Name:        "Кеш QuickLook",
			Description: "Сброс кеша превью через qlmanage -r cache.",
			Special:     SpecialQuickLookCache,
		},
		{
			ID:          "dns-cache",
			Name:        "DNS-кеш",
			Description: "dscacheutil -flushcache и перезапуск mDNSResponder.",
			Special:     SpecialDNSCache,
		},

		// ---- Браузеры ----
		{
			ID:          "safari-cache",
			Name:        "Кеш Safari",
			Description: "~/Library/Caches/com.apple.Safari (история и закладки не затрагиваются).",
			Paths:       paths(sub(cache, "com.apple.Safari")),
			KeepRoot:    true,
		},
		{
			ID:          "chrome-cache",
			Name:        "Кеш Google Chrome",
			Description: "~/Library/Caches/Google/Chrome — кеш всех профилей.",
			Paths:       paths(sub(cache, "Google/Chrome")),
			KeepRoot:    true,
		},
		{
			ID:          "edge-cache",
			Name:        "Кеш Microsoft Edge",
			Description: "~/Library/Caches/Microsoft Edge.",
			Paths:       paths(sub(cache, "Microsoft Edge")),
			KeepRoot:    true,
		},
		{
			ID:          "brave-cache",
			Name:        "Кеш Brave",
			Description: "~/Library/Caches/BraveSoftware.",
			Paths:       paths(sub(cache, "BraveSoftware")),
			KeepRoot:    true,
		},
		{
			ID:          "firefox-cache",
			Name:        "Кеш Mozilla Firefox",
			Description: "cache2 во всех профилях ~/Library/Caches/Firefox/Profiles.",
			Paths:       paths(sub(cache, "Firefox/Profiles")),
			KeepRoot:    true,
		},

		// ---- Разработка ----
		{
			ID:          "xcode-derived-data",
			Name:        "Xcode DerivedData",
			Description: "Промежуточные результаты сборок Xcode — пересобираются автоматически.",
			Paths:       paths(sub(userLibraryDir(), "Developer/Xcode/DerivedData")),
			KeepRoot:    true,
		},
		{
			ID:          "xcode-device-support",
			Name:        "Xcode iOS DeviceSupport",
			Description: "Символы отладки для подключённых устройств. Скачиваются заново при подключении.",
			Paths: paths(
				sub(userLibraryDir(), "Developer/Xcode/iOS DeviceSupport"),
				sub(userLibraryDir(), "Developer/Xcode/watchOS DeviceSupport"),
			),
			KeepRoot:   true,
			Aggressive: true,
		},
		{
			ID:          "simulator-caches",
			Name:        "Кеши симуляторов iOS",
			Description: "~/Library/Developer/CoreSimulator/Caches — сами симуляторы не удаляются.",
			Paths:       paths(sub(userLibraryDir(), "Developer/CoreSimulator/Caches")),
			KeepRoot:    true,
		},
		{
			ID:          "cocoapods-cache",
			Name:        "Кеш CocoaPods",
			Description: "~/Library/Caches/CocoaPods — поды скачиваются заново.",
			Paths:       paths(sub(cache, "CocoaPods")),
			KeepRoot:    true,
		},
		{
			ID:          "homebrew-cache",
			Name:        "Кеш Homebrew (файлы)",
			Description: "~/Library/Caches/Homebrew — скачанные бутылки и исходники.",
			Paths:       paths(sub(cache, "Homebrew")),
			KeepRoot:    true,
		},
		{
			ID:          "brew-cleanup",
			Name:        "Homebrew: старые версии",
			Description: "brew cleanup --prune=all — удаляет устаревшие версии установленных формул.",
			Special:     SpecialBrewCleanup,
		},
		{
			ID:          "npm-cache",
			Name:        "Кеш npm",
			Description: "~/.npm/_cacache — скачанные пакеты, восстанавливается сам.",
			Paths:       paths(sub(home, ".npm/_cacache")),
			KeepRoot:    true,
		},
		{
			ID:          "yarn-cache",
			Name:        "Кеш Yarn",
			Description: "~/Library/Caches/Yarn.",
			Paths:       paths(sub(cache, "Yarn")),
			KeepRoot:    true,
		},
		{
			ID:          "pip-cache",
			Name:        "Кеш pip",
			Description: "~/Library/Caches/pip — скачанные wheel-пакеты Python.",
			Paths:       paths(sub(cache, "pip")),
			KeepRoot:    true,
		},
		{
			ID:          "go-build-cache",
			Name:        "Кеш сборки Go",
			Description: "~/Library/Caches/go-build — пересоздаётся при следующей сборке.",
			Paths:       paths(sub(cache, "go-build")),
			KeepRoot:    true,
		},
		{
			ID:          "go-mod-cache",
			Name:        "Кеш загрузок модулей Go",
			Description: "~/go/pkg/mod/cache/download — модули будут скачаны заново.",
			Paths:       paths(sub(home, "go/pkg/mod/cache/download")),
			KeepRoot:    true,
			Aggressive:  true,
		},
		{
			ID:          "cargo-cache",
			Name:        "Кеш Cargo (Rust)",
			Description: "~/.cargo/registry/cache и ~/.cargo/registry/src.",
			Paths:       paths(sub(home, ".cargo/registry/cache"), sub(home, ".cargo/registry/src")),
			KeepRoot:    true,
			Aggressive:  true,
		},
		{
			ID:          "gradle-cache",
			Name:        "Кеш Gradle",
			Description: "~/.gradle/caches — зависимости будут скачаны заново.",
			Paths:       paths(sub(home, ".gradle/caches")),
			KeepRoot:    true,
			Aggressive:  true,
		},
		{
			ID:          "maven-cache",
			Name:        "Локальный репозиторий Maven",
			Description: "~/.m2/repository — зависимости будут скачаны заново.",
			Paths:       paths(sub(home, ".m2/repository")),
			KeepRoot:    true,
			Aggressive:  true,
		},
		{
			ID:          "huggingface-cache",
			Name:        "Кеш моделей Hugging Face",
			Description: "~/.cache/huggingface — модели скачиваются заново, объём может быть очень большим.",
			Paths:       paths(sub(home, ".cache/huggingface")),
			KeepRoot:    true,
			Aggressive:  true,
		},
		{
			ID:          "vscode-cache",
			Name:        "Кеш и логи VS Code",
			Description: "Cache, CachedData, Code Cache, GPUCache и logs профиля Code.",
			Paths: paths(
				sub(support, "Code/Cache"), sub(support, "Code/CachedData"),
				sub(support, "Code/Code Cache"), sub(support, "Code/GPUCache"),
				sub(support, "Code/logs"),
			),
			KeepRoot: true,
		},
		{
			ID:          "jetbrains-logs",
			Name:        "Логи и кеши JetBrains",
			Description: "~/Library/Logs/JetBrains и ~/Library/Caches/JetBrains (по каждой IDE).",
			Paths:       paths(sub(logs, "JetBrains"), sub(cache, "JetBrains")),
			KeepRoot:    true,
		},

		// ---- Мессенджеры и медиа ----
		{
			ID:          "slack-cache",
			Name:        "Кеш Slack",
			Description: "Cache, Code Cache, GPUCache и логи Slack.",
			Paths: paths(
				sub(support, "Slack/Cache"), sub(support, "Slack/Code Cache"),
				sub(support, "Slack/GPUCache"), sub(support, "Slack/logs"),
			),
			KeepRoot: true,
		},
		{
			ID:          "discord-cache",
			Name:        "Кеш Discord",
			Description: "Cache, Code Cache и GPUCache профиля Discord.",
			Paths: paths(
				sub(support, "discord/Cache"), sub(support, "discord/Code Cache"),
				sub(support, "discord/GPUCache"),
			),
			KeepRoot: true,
		},
		{
			ID:          "signal-cache",
			Name:        "Кеш Signal",
			Description: "Cache, Code Cache, GPUCache и логи Signal (сообщения не затрагиваются).",
			Paths: paths(
				sub(support, "Signal/Cache"), sub(support, "Signal/Code Cache"),
				sub(support, "Signal/GPUCache"), sub(support, "Signal/logs"),
			),
			KeepRoot: true,
		},
		{
			ID:          "telegram-cache",
			Name:        "Кеш Telegram Desktop",
			Description: "~/Library/Caches/ru.keepcoder.Telegram — переписка (Group Containers) не затрагивается.",
			Paths:       paths(sub(cache, "ru.keepcoder.Telegram")),
			KeepRoot:    true,
		},
		{
			ID:          "spotify-cache",
			Name:        "Кеш Spotify",
			Description: "~/Library/Caches/com.spotify.client — офлайн-треки будут загружены заново.",
			Paths:       paths(sub(cache, "com.spotify.client")),
			KeepRoot:    true,
		},
		{
			ID:          "zoom-logs",
			Name:        "Логи и кеш Zoom",
			Description: "~/Library/Logs/zoom.us и ~/Library/Caches/us.zoom.xos.",
			Paths:       paths(sub(logs, "zoom.us"), sub(cache, "us.zoom.xos")),
			KeepRoot:    true,
		},
		{
			ID:          "steam-cache",
			Name:        "Кеш и логи Steam",
			Description: "appcache/httpcache и logs — Steam создаст их заново (игры не затрагиваются).",
			Paths: paths(
				sub(support, "Steam/appcache/httpcache"),
				sub(support, "Steam/logs"),
			),
			KeepRoot: true,
		},

		// ---- Системные категории (нужен sudo) ----
		{
			ID:          "system-cache",
			Name:        "Системный кеш (/Library/Caches)",
			Description: "Общесистемные кеши приложений (нужен sudo).",
			Paths:       paths("/Library/Caches"),
			KeepRoot:    true,
			MinAgeHours: 24,
		},
		{
			ID:          "system-logs",
			Name:        "Системные логи (/private/var/log)",
			Description: "Ротированные системные журналы старше 7 дней (нужен sudo).",
			Paths:       paths("/private/var/log"),
			KeepRoot:    true,
			MinAgeHours: 24 * 7,
			// Unified logging (.logarchive/Persist) читается только штатными
			// средствами — не трогаем.
			ForbidSubstrings: []string{"/Persist/", ".logarchive"},
			Aggressive:       true,
		},

		// ---- Агрессивные категории ----
		{
			ID:          "user-temp",
			Name:        "Временные файлы ($TMPDIR)",
			Description: "Файлы старше 48 часов в личном каталоге /private/var/folders/.../T.",
			Paths:       paths(tmp),
			KeepRoot:    true,
			MinAgeHours: 48,
			// Внутри контейнера TMPDIR соседствует с каталогом C (кеш
			// системных служб) — трогаем только T, а внутри него пропускаем
			// служебные сокеты.
			ForbidSubstrings: []string{"/com.apple.LaunchServices", "/mds", "/.CFUserTextEncoding"},
			Aggressive:       true,
		},
		{
			ID:          "local-snapshots",
			Name:        "Локальные снимки Time Machine",
			Description: "tmutil deletelocalsnapshots — освобождает место, занятое локальными снимками APFS.",
			Special:     SpecialLocalSnapshots,
			Aggressive:  true,
		},
		{
			ID:          "ios-backups",
			Name:        "Резервные копии iOS",
			Description: "~/Library/Application Support/MobileSync/Backup — это НЕ кеш: восстановить копии нельзя.",
			Paths:       paths(sub(support, "MobileSync/Backup")),
			KeepRoot:    true,
			Aggressive:  true,
		},
		{
			ID:          "xcode-archives",
			Name:        "Архивы сборок Xcode",
			Description: "~/Library/Developer/Xcode/Archives — собранные архивы приложений, пересобираются только вручную.",
			Paths:       paths(sub(userLibraryDir(), "Developer/Xcode/Archives")),
			KeepRoot:    true,
			Aggressive:  true,
		},
		{
			ID:          "downloads-old",
			Name:        "Старые загрузки (~/Downloads)",
			Description: "Файлы в ~/Downloads старше 90 дней. Это пользовательские файлы — проверьте список перед очисткой!",
			Paths:       paths(sub(home, "Downloads")),
			KeepRoot:    true,
			MinAgeHours: 24 * 90,
			Aggressive:  true,
		},
	}

	return dropEmptyTargets(targets)
}
