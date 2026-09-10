# winCleanerLamp — консольная утилита (CLI)

CLI — это ядро проекта: один исполняемый файл на **Go 1.21+**, без сторонних Go-модулей. Для части действий вызываются системные утилиты Windows (`powershell`, `ipconfig`, `wevtutil`, `reg`).

> **Некоммерческий личный проект.** Автор не даёт гарантий и не несёт ответственности за последствия запуска. Перед удалением данных делайте резервные копии и используйте `--scan`.

---

## Назначение

- Оценить, сколько места можно освободить (`--scan`).
- Удалить файлы по **категориям** (временные папки, кеши браузеров и IDE, логи и т.д.) с учётом возраста файлов и безопасных путей.
- Отдельно — отчёт по «остаткам» программ (`--leftovers`), **без автоматического удаления**.

---

## Как определяется «мусор»

Три независимых сигнала:

1. **Расположение** — файл в известной «мусорной» директории (Temp, Prefetch, WER, кеши браузеров, IDE, мессенджеров и т.д.).
2. **Возраст** — у каждой категории свой минимальный возраст (`MinAgeHours`). Глобально можно ужесточить: `--min-age-hours N`.
3. **Имя/расширение** для отдельных типов (`thumbcache_*.db`, `*.dmp`, `*.evtx` и т.п.).

Заблокированные процессом файлы по возможности **пропускаются**, без остановки всего процесса очистки.

---

## Категории (безопасные)

| ID | Что затрагивается |
|----|-------------------|
| `user-temp` | `%TEMP%`, `%LOCALAPPDATA%\Temp` |
| `windows-temp` | `C:\Windows\Temp` (часто нужен админ) |
| `prefetch` | `C:\Windows\Prefetch` |
| `windows-update-cache` | `C:\Windows\SoftwareDistribution\Download` (>24ч) |
| `delivery-optimization` | P2P-кеш обновлений |
| `cbs-logs` | `C:\Windows\Logs\CBS` (>72ч) |
| `sd-datastore-logs` | `SoftwareDistribution\DataStore\Logs` |
| `windows-logs` | логи DISM / DPX / MoSetup / WindowsUpdate / setupapi |
| `panther` | `C:\Windows\Panther` |
| `wer` | Windows Error Reporting |
| `crash-dumps` | `%LOCALAPPDATA%\CrashDumps`, `Minidump`, `MEMORY.DMP` |
| `livekernel-reports` | `C:\Windows\LiveKernelReports` |
| `defender-scan-history` | история сканов Defender |
| `font-cache` | `FontCache*.dat`, `FNTCACHE.DAT` |
| `recent` | недавние файлы |
| `jump-lists` | Jump Lists |
| `inet-cache` | `INetCache` |
| `thumbnail-cache` | `thumbcache_*.db`, `iconcache_*.db` |
| `chrome-cache` / `edge-cache` / `brave-cache` | только кеши профиля по умолчанию |
| `firefox-cache` | `cache2` в профилях |
| `teams-cache` / `teams-new-cache` | Teams |
| `discord-cache` / `slack-cache` / `telegram-cache` | мессенджеры |
| `spotify-cache` | Spotify |
| `vscode-cache` | VS Code — только кеши/логи, не настройки пользователя целиком |
| `jetbrains-logs` | JetBrains |
| `office-cache` | Office File Cache |
| `adobe-media-cache` | Adobe Media Cache |
| `nuget-cache` / `pip-cache` / `npm-cache` / `yarn-cache` / `go-build-cache` / `gradle-cache` | кеши пакетных менеджеров |
| `steam-htmlcache` | Steam HTML cache |
| `nvidia-cache` / `amd-cache` / `dx-shader-cache` | GPU-кеши |
| `recycle-bin` | корзина (PowerShell) |
| `dns-cache` | `ipconfig /flushdns` |

Дополнительно в коде есть расширенный набор категорий (например `readyboot`, `store-cache`, кеши игровых лаунчеров, кеши CLI-инструментов разработчика — Composer/pnpm/Go modules/Cargo/Hugging Face, кеш RDP, автономных файлов (CSC) и др.) — полный актуальный список: **`win-cleaner-lamp.exe --list`**. Обоснование, откуда взят каждый новый пункт и почему он безопасен/агрессивен — в [`docs/research-junk-sources.md`](research-junk-sources.md).

---

## Категории (агрессивные)

Включаются только с **`--aggressive`** или явным перечислением в **`--categories`**:

| ID | Описание |
|----|----------|
| `windows-old` | `C:\Windows.old` |
| `windows-installer-upgrade` | `$WINDOWS.~BT`, `$WINDOWS.~WS`, `ESD\Windows` |
| `event-logs` | очистка `*.evtx` через `wevtutil` |
| `iis-logs` | `C:\inetpub\logs\LogFiles` |
| `downloads-old` | старые файлы в `Downloads` |
| `maven-cache` | локальный Maven-репозиторий |

---

## Что не трогается

В коде заданы ограничения (`IsPathSafeToDelete` в `internal/cleaner/safety.go` плюс список защищённых путей своей ОС в `safety_windows.go` / `safety_linux.go` / `safety_darwin.go`), в том числе:

- **Windows:** корни дисков, `System32`, `SysWOW64`, `WinSxS`, `Program Files`, `Program Files (x86)`, системные разделы профиля;
- **Linux:** `/`, `/usr`, `/etc`, `/boot`, `/proc`, `/sys`, `/dev`, `/var/lib`, `/snap`, домашний каталог целиком, `~/.ssh`, `~/.gnupg`;
- **macOS:** `/`, `/System`, `/usr`, `/Applications`, `/Library/Frameworks`, `/private/var/db`, `/private/var/vm`, `/Volumes`, домашний каталог целиком, `~/Library/Keychains`, `~/Library/Mail`, `~/Library/Mobile Documents`.

Проверка выполняется дважды: перед обработкой пути категории и на каждом файле при обходе дерева.

В браузерах намеренно затрагиваются в основном **кеши**, а не пароли, cookies и история целиком — см. исходники категорий.

---

## Сборка

Ядро собирается под свою ОС из отдельного каталога:

```powershell
go build -o win-cleaner-lamp.exe ./wincli
```

```bash
go build -o lin-cleaner-lamp ./lincli   # Linux
go build -o mac-cleaner-lamp ./maccli   # macOS
```

Оптимизация размера бинарника (как в релизных скриптах):

```powershell
go build -ldflags "-s -w" -o win-cleaner-lamp.exe ./wincli
```

Все три ядра сразу (из-под любой ОС, cgo не используется): `make build-all`.

---

## Примеры команд

```powershell
# Список категорий
.\win-cleaner-lamp.exe --list

# Оценка освобождаемого места (безопасные категории)
.\win-cleaner-lamp.exe --scan

# С агрессивными категориями
.\win-cleaner-lamp.exe --scan --aggressive

# Очистка с подтверждением
.\win-cleaner-lamp.exe --clean

# Без подтверждения
.\win-cleaner-lamp.exe --clean --yes

# Только выбранные категории
.\win-cleaner-lamp.exe --clean --categories user-temp,prefetch,recycle-bin

# Исключения
.\win-cleaner-lamp.exe --clean --exclude recycle-bin,dns-cache

# Агрессивная очистка
.\win-cleaner-lamp.exe --clean --aggressive --yes

# Глобальный минимальный возраст файлов (часы)
.\win-cleaner-lamp.exe --clean --min-age-hours 720

# Подробный лог
.\win-cleaner-lamp.exe --clean --verbose

# Остатки программ в AppData (только отчёт)
.\win-cleaner-lamp.exe --leftovers

# Системная информация (hiberfil, pagefile, WinSxS и т.д.)
.\win-cleaner-lamp.exe --sysinfo
```

Для доступа к некоторым системным путям запускайте консоль **от имени администратора**.

---

## Остатки программ (`--leftovers`)

Эвристика:

1. Читаются имена установленных программ из реестра (`Uninstall` и связанные ветки).
2. Имена токенизируются.
3. Папки первого уровня в `%APPDATA%`, `%LOCALAPPDATA%`, `ProgramData` сравниваются с этими токенами и белыми списками.
4. Подозрительные папки выводятся с размером.

Результат **может содержать ложные срабатывания** — перед удалением проверяйте вручную. Автоматического удаления нет.

---

## Архитектура исходников

| Файл | Роль |
|------|------|
| `wincli/`, `lincli/`, `maccli/` | Точки входа трёх ядер (по несколько строк каждая) |
| `internal/cli/cli.go` | Флаги, таблица, подтверждение, JSON-вывод — общие для всех ОС |
| `internal/cleaner/targets.go` | Модель категории и раскрытие путей; сами категории — в `targets_windows.go` / `targets_linux.go` / `targets_darwin.go` |
| `internal/cleaner/cleaner.go` | Scan/Clean и обход дерева; спец-действия — в `specials_<os>.go` |
| `internal/cleaner/safety.go` | Проверка безопасности пути; списки защищённых путей — в `safety_<os>.go` |
| `internal/cleaner/leftovers.go` | Логика `--leftovers`; корни и списки исключений — в `apps_<os>.go` |

Общая логика: `cleaner.Process(Target, Options) Report`; при `DryRun=true` выполняется только подсчёт.

Платформенные файлы разделены суффиксом имени (`_windows.go`, `_linux.go`, `_darwin.go`) — build-теги Go подключают только файлы своей ОС, поэтому в ядре каждой платформы нет чужого кода и нет ветвлений по `runtime.GOOS`.

---

## Дополнительные возможности (кратко)

- Параллельное сканирование: `--parallel N` (по умолчанию несколько воркеров), прогресс в консоли.
- Скрытие пустых категорий в отчёте сканирования; обратно: `--show-empty`.

---

## Отказ от ответственности

ПО поставляется «как есть». Вы несёте полную ответственность за запуск, в том числе с правами администратора и с агрессивными категориями.
