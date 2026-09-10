//go:build linux

package cleaner

// AllTargets — перечень безопасных к очистке категорий для Linux.
//
// Пути собираются по XDG Base Directory Specification (XDG_CACHE_HOME,
// XDG_CONFIG_HOME, XDG_DATA_HOME с обычными fallback-ами ~/.cache, ~/.config,
// ~/.local/share), а не жёстко от ~ — иначе на системах с переопределёнными
// переменными очистка просто ничего не находила бы.
//
// Источники (общеизвестные места накопления мусора в Linux):
//   - ~/.cache                       — пользовательский кеш (XDG)
//   - ~/.local/share/Trash           — Корзина (FreeDesktop Trash spec)
//   - /tmp, /var/tmp                 — временные файлы (чистятся с MinAge)
//   - /var/log, journald             — системные журналы
//   - /var/cache/<пакетный менеджер> — скачанные пакеты apt/dnf/pacman/zypper
//   - flatpak/snap                   — неиспользуемые runtime и отключённые ревизии
//   - кеши браузеров, IDE и пакетных менеджеров языков программирования
func AllTargets() []Target {
	cache := userCacheDir()
	cfg := userConfigDir()
	data := userDataDir()
	home := userHomeDir()

	targets := []Target{
		{
			ID:          "user-cache",
			Name:        "Пользовательский кеш (~/.cache)",
			Description: "Содержимое XDG-каталога кеша. Приложения создают его заново по мере работы.",
			Paths:       paths(cache),
			KeepRoot:    true,
			MinAgeHours: 24,
			// Внутри ~/.cache некоторые приложения хранят не кеш, а рабочее
			// состояние: ключи сессии keyring, база dconf, «недоделанные»
			// загрузки менеджеров пакетов. Их не трогаем.
			ForbidSubstrings: []string{"/keyring", "/gnome-keyring", "/dconf", "/.cache/rclone"},
		},
		{
			ID:          "thumbnails",
			Name:        "Кеш миниатюр",
			Description: "~/.cache/thumbnails и старый ~/.thumbnails — превью файлов, создаются заново.",
			Paths:       paths(sub(cache, "thumbnails"), sub(home, ".thumbnails")),
			KeepRoot:    true,
		},
		{
			ID:          "trash",
			Name:        "Корзина",
			Description: "~/.local/share/Trash (FreeDesktop Trash spec): files, info и expunged.",
			Special:     SpecialTrash,
		},
		{
			ID:          "fontconfig-cache",
			Name:        "Кеш шрифтов (fontconfig)",
			Description: "~/.cache/fontconfig — пересоздаётся при первом обращении к шрифтам.",
			Paths:       paths(sub(cache, "fontconfig")),
			KeepRoot:    true,
		},
		{
			ID:          "mesa-shader-cache",
			Name:        "Кеш шейдеров Mesa",
			Description: "~/.cache/mesa_shader_cache — кеш скомпилированных шейдеров драйвера.",
			Paths:       paths(sub(cache, "mesa_shader_cache"), sub(cache, "mesa_shader_cache_db")),
			KeepRoot:    true,
		},
		{
			ID:          "nvidia-shader-cache",
			Name:        "Кеш шейдеров NVIDIA",
			Description: "~/.nv/GLCache и ~/.cache/nvidia — кеш компиляции шейдеров.",
			Paths:       paths(sub(home, ".nv/GLCache"), sub(cache, "nvidia")),
			KeepRoot:    true,
		},
		{
			ID:          "recently-used",
			Name:        "Список недавних файлов",
			Description: "~/.local/share/recently-used.xbel — история открытых файлов в файловых менеджерах.",
			Paths:       paths(sub(data, "recently-used.xbel"), sub(home, ".recently-used.xbel")),
		},
		{
			ID:          "xsession-errors",
			Name:        "Журнал ошибок сессии",
			Description: "~/.xsession-errors и .old — растут неограниченно у некоторых сессий.",
			Paths:       paths(sub(home, ".xsession-errors.old")),
		},

		// ---- Браузеры ----
		{
			ID:          "chrome-cache",
			Name:        "Кеш Google Chrome",
			Description: "~/.cache/google-chrome — кеш всех профилей.",
			Paths:       paths(sub(cache, "google-chrome")),
			KeepRoot:    true,
		},
		{
			ID:          "chromium-cache",
			Name:        "Кеш Chromium",
			Description: "~/.cache/chromium — кеш всех профилей.",
			Paths:       paths(sub(cache, "chromium")),
			KeepRoot:    true,
		},
		{
			ID:          "edge-cache",
			Name:        "Кеш Microsoft Edge",
			Description: "~/.cache/microsoft-edge.",
			Paths:       paths(sub(cache, "microsoft-edge"), sub(cache, "microsoft-edge-beta")),
			KeepRoot:    true,
		},
		{
			ID:          "brave-cache",
			Name:        "Кеш Brave",
			Description: "~/.cache/BraveSoftware.",
			Paths:       paths(sub(cache, "BraveSoftware")),
			KeepRoot:    true,
		},
		{
			ID:          "opera-cache",
			Name:        "Кеш Opera / Vivaldi / Yandex",
			Description: "~/.cache/opera, ~/.cache/vivaldi, ~/.cache/yandex-browser.",
			Paths:       paths(sub(cache, "opera"), sub(cache, "vivaldi"), sub(cache, "yandex-browser")),
			KeepRoot:    true,
		},
		{
			ID:          "firefox-cache",
			Name:        "Кеш Mozilla Firefox",
			Description: "cache2 во всех профилях ~/.cache/mozilla/firefox (обрабатывается по профилям).",
			Paths:       paths(sub(cache, "mozilla/firefox")),
			KeepRoot:    true,
		},

		// ---- Пакетные менеджеры языков и сборка ----
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
			Description: "~/.cache/yarn.",
			Paths:       paths(sub(cache, "yarn")),
			KeepRoot:    true,
		},
		{
			ID:          "pnpm-store",
			Name:        "Хранилище pnpm",
			Description: "~/.local/share/pnpm/store — общий кеш пакетов pnpm.",
			Paths:       paths(sub(data, "pnpm/store")),
			KeepRoot:    true,
			Aggressive:  true,
		},
		{
			ID:          "pip-cache",
			Name:        "Кеш pip",
			Description: "~/.cache/pip — скачанные wheel-пакеты Python.",
			Paths:       paths(sub(cache, "pip")),
			KeepRoot:    true,
		},
		{
			ID:          "go-build-cache",
			Name:        "Кеш сборки Go",
			Description: "~/.cache/go-build — пересоздаётся при следующей сборке.",
			Paths:       paths(sub(cache, "go-build")),
			KeepRoot:    true,
		},
		{
			ID:          "go-mod-cache",
			Name:        "Кеш загрузок модулей Go",
			Description: "~/go/pkg/mod/cache/download — модули будут скачаны заново при сборке.",
			Paths:       paths(sub(home, "go/pkg/mod/cache/download")),
			KeepRoot:    true,
			Aggressive:  true,
		},
		{
			ID:          "cargo-cache",
			Name:        "Кеш Cargo (Rust)",
			Description: "~/.cargo/registry/cache и ~/.cargo/registry/src — скачиваются заново.",
			Paths:       paths(sub(home, ".cargo/registry/cache"), sub(home, ".cargo/registry/src")),
			KeepRoot:    true,
			Aggressive:  true,
		},
		{
			ID:          "gradle-cache",
			Name:        "Кеш Gradle",
			Description: "~/.gradle/caches — сборки скачают зависимости заново.",
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
			ID:          "composer-cache",
			Name:        "Кеш Composer (PHP)",
			Description: "~/.cache/composer.",
			Paths:       paths(sub(cache, "composer")),
			KeepRoot:    true,
		},
		{
			ID:          "nuget-cache",
			Name:        "Кеш NuGet (.NET)",
			Description: "~/.nuget/packages — пакеты будут восстановлены при сборке.",
			Paths:       paths(sub(home, ".nuget/packages")),
			KeepRoot:    true,
			Aggressive:  true,
		},
		{
			ID:          "huggingface-cache",
			Name:        "Кеш моделей Hugging Face",
			Description: "~/.cache/huggingface — модели скачиваются заново, объём может быть очень большим.",
			Paths:       paths(sub(cache, "huggingface")),
			KeepRoot:    true,
			Aggressive:  true,
		},

		// ---- Редакторы и IDE ----
		{
			ID:          "vscode-cache",
			Name:        "Кеш и логи VS Code",
			Description: "Cache, CachedData, Code Cache, GPUCache и logs профиля Code.",
			Paths: paths(
				sub(cfg, "Code/Cache"), sub(cfg, "Code/CachedData"),
				sub(cfg, "Code/Code Cache"), sub(cfg, "Code/GPUCache"),
				sub(cfg, "Code/logs"), sub(cfg, "Code/User/workspaceStorage"),
			),
			KeepRoot: true,
		},
		{
			ID:          "jetbrains-logs",
			Name:        "Логи и кеши JetBrains",
			Description: "~/.cache/JetBrains/<IDE>/{log,caches} — обрабатывается по каждой IDE.",
			Paths:       paths(sub(cache, "JetBrains")),
			KeepRoot:    true,
		},

		// ---- Мессенджеры и медиа ----
		{
			ID:          "discord-cache",
			Name:        "Кеш Discord",
			Description: "Cache, Code Cache и GPUCache профиля Discord.",
			Paths: paths(
				sub(cfg, "discord/Cache"), sub(cfg, "discord/Code Cache"), sub(cfg, "discord/GPUCache"),
			),
			KeepRoot: true,
		},
		{
			ID:          "slack-cache",
			Name:        "Кеш Slack",
			Description: "Cache, Code Cache, GPUCache и логи Slack.",
			Paths: paths(
				sub(cfg, "Slack/Cache"), sub(cfg, "Slack/Code Cache"),
				sub(cfg, "Slack/GPUCache"), sub(cfg, "Slack/logs"),
			),
			KeepRoot: true,
		},
		{
			ID:          "signal-cache",
			Name:        "Кеш Signal",
			Description: "Cache, Code Cache, GPUCache и логи Signal (сообщения не затрагиваются).",
			Paths: paths(
				sub(cfg, "Signal/Cache"), sub(cfg, "Signal/Code Cache"),
				sub(cfg, "Signal/GPUCache"), sub(cfg, "Signal/logs"),
			),
			KeepRoot: true,
		},
		{
			ID:          "telegram-cache",
			Name:        "Кеш Telegram Desktop",
			Description: "cache и media_cache внутри tdata/user_data — сама переписка не затрагивается.",
			Paths: paths(
				sub(data, "TelegramDesktop/tdata/user_data/cache"),
				sub(data, "TelegramDesktop/tdata/user_data/media_cache"),
			),
			KeepRoot: true,
		},
		{
			ID:          "spotify-cache",
			Name:        "Кеш Spotify",
			Description: "~/.cache/spotify — скачанные для офлайна треки будут загружены заново.",
			Paths:       paths(sub(cache, "spotify")),
			KeepRoot:    true,
		},
		{
			ID:          "zoom-logs",
			Name:        "Логи и кеш Zoom",
			Description: "~/.zoom/logs и ~/.cache/zoom.",
			Paths:       paths(sub(home, ".zoom/logs"), sub(cache, "zoom")),
			KeepRoot:    true,
		},
		{
			ID:          "obs-studio-logs",
			Name:        "Логи OBS Studio",
			Description: "~/.config/obs-studio/logs и crashes.",
			Paths:       paths(sub(cfg, "obs-studio/logs"), sub(cfg, "obs-studio/crashes")),
			KeepRoot:    true,
		},
		{
			ID:          "steam-cache",
			Name:        "Кеш и логи Steam",
			Description: "appcache/httpcache и logs — Steam создаст их заново (игры не затрагиваются).",
			Paths: paths(
				sub(data, "Steam/appcache/httpcache"), sub(data, "Steam/logs"),
				sub(home, ".steam/steam/appcache/httpcache"), sub(home, ".steam/steam/logs"),
			),
			KeepRoot: true,
		},
		{
			ID:          "flatpak-app-cache",
			Name:        "Кеш flatpak-приложений",
			Description: "~/.var/app/<приложение>/cache — обрабатывается по каждому приложению.",
			Paths:       paths(sub(home, ".var/app")),
			KeepRoot:    true,
		},

		// ---- Системные категории (нужен root) ----
		{
			ID:          "package-cache",
			Name:        "Кеш пакетного менеджера",
			Description: "Скачанные пакеты apt/dnf/pacman/zypper. Очистка штатной командой (нужен root).",
			Special:     SpecialPackageCache,
		},
		{
			ID:          "journal-logs",
			Name:        "Журналы systemd (journald)",
			Description: "Усечение журнала до последних 7 дней через journalctl --vacuum-time (нужен root).",
			Special:     SpecialJournalVacuum,
		},
		{
			ID:          "flatpak-unused",
			Name:        "Неиспользуемые runtime flatpak",
			Description: "flatpak uninstall --unused — удаляет только runtime, на которые не ссылается ни одно приложение.",
			Special:     SpecialFlatpakUnused,
		},
		{
			ID:          "dns-cache",
			Name:        "DNS-кеш",
			Description: "resolvectl flush-caches (systemd-resolved).",
			Special:     SpecialDNSCache,
		},
		{
			ID:          "crash-reports",
			Name:        "Отчёты о падениях",
			Description: "/var/crash и ~/.cache/abrt — дампы упавших программ.",
			Paths:       paths("/var/crash", sub(cache, "abrt")),
			KeepRoot:    true,
			MinAgeHours: 24,
		},

		// ---- Агрессивные категории ----
		{
			ID:          "user-temp",
			Name:        "Временные файлы /tmp",
			Description: "Файлы старше 48 часов в /tmp. Сокеты и служебные каталоги сессии не трогаются.",
			Paths:       paths("/tmp"),
			KeepRoot:    true,
			MinAgeHours: 48,
			ForbidSubstrings: []string{
				"/.x11-unix", "/.ice-unix", "/.font-unix", "/.xim-unix", "/.test-unix",
				"/systemd-private-", "/snap-private-tmp", "/ssh-", "/.mount_",
			},
			Aggressive: true,
		},
		{
			ID:          "var-temp",
			Name:        "Временные файлы /var/tmp",
			Description: "Файлы старше 7 дней в /var/tmp (нужен root). Каталоги systemd-private не трогаются.",
			Paths:       paths("/var/tmp"),
			KeepRoot:    true,
			MinAgeHours: 24 * 7,
			ForbidSubstrings: []string{
				"/systemd-private-", "/snap-private-tmp",
			},
			Aggressive: true,
		},
		{
			ID:          "system-logs",
			Name:        "Ротированные системные логи",
			Description: "Старые файлы в /var/log (*.gz, *.1 и т.п.) старше 7 дней (нужен root).",
			Paths:       paths("/var/log"),
			KeepRoot:    true,
			MinAgeHours: 24 * 7,
			// Активные журналы journald трогаем не напрямую, а через
			// journalctl --vacuum-time (категория journal-logs).
			ForbidSubstrings: []string{"/journal/"},
			Aggressive:       true,
		},
		{
			ID:          "snap-disabled",
			Name:        "Отключённые ревизии snap",
			Description: "Старые (disabled) ревизии установленных snap-пакетов (нужен root).",
			Special:     SpecialSnapDisabled,
			Aggressive:  true,
		},
		{
			ID:          "downloads-old",
			Name:        "Старые загрузки (~/Downloads)",
			Description: "Файлы в ~/Downloads старше 90 дней. Это пользовательские файлы — проверьте список перед очисткой!",
			Paths:       paths(sub(home, "Downloads"), sub(home, "Загрузки")),
			KeepRoot:    true,
			MinAgeHours: 24 * 90,
			Aggressive:  true,
		},
	}

	return dropEmptyTargets(targets)
}
