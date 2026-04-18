# winCleanerLamp

CLI-утилита на Go для очистки мусора в Windows. Без внешних зависимостей (только stdlib + системные утилиты `powershell`, `ipconfig`, `wevtutil`, `reg`).

## Как определить, что файл — «мусор»

Программа использует три независимых сигнала:

1. **По расположению** — файл лежит в известной мусорной директории (Temp, Prefetch, WER, кеши браузеров/IDE/мессенджеров и т.п.). Полный список — ниже.
2. **По возрасту** — у каждой категории свой `MinAgeHours` (обновления Windows — 24ч, CBS-логи — 72ч, `go-build` — 14д, `maven` — 60д и т.д.). Можно ужесточить глобально флагом `--min-age-hours N`.
3. **По имени/расширению** для специальных категорий (`thumbcache_*.db`, `iconcache_*.db`, `*.dmp`, `*.evtx`).

Заблокированные процессом файлы тихо пропускаются — программа не останавливается на ошибке.

Для подозрительных остатков после удаления ПО отдельно есть команда `--leftovers`: сравнивает папки в `AppData\Roaming`, `AppData\Local`, `ProgramData` со списком установленных программ из реестра (`HKLM/HKCU\...\Uninstall`) и выводит «сиротские» папки. **Удаление** их автоматически не делается — только отчёт, потому что эвристика может ошибаться.

## Категории (безопасные)

| ID | Что чистит |
|---|---|
| `user-temp` | `%TEMP%`, `%LOCALAPPDATA%\Temp` |
| `windows-temp` | `C:\Windows\Temp` (нужен админ) |
| `prefetch` | `C:\Windows\Prefetch` |
| `windows-update-cache` | `C:\Windows\SoftwareDistribution\Download` (>24ч) |
| `delivery-optimization` | P2P-кеш обновлений |
| `cbs-logs` | `C:\Windows\Logs\CBS` (>72ч) |
| `sd-datastore-logs` | `SoftwareDistribution\DataStore\Logs` |
| `windows-logs` | DISM / DPX / MoSetup / WindowsUpdate / setupapi logs |
| `panther` | `C:\Windows\Panther` — логи установки ОС |
| `wer` | Windows Error Reporting (архив + очередь) |
| `crash-dumps` | `%LOCALAPPDATA%\CrashDumps`, `C:\Windows\Minidump`, `MEMORY.DMP` |
| `livekernel-reports` | `C:\Windows\LiveKernelReports` |
| `defender-scan-history` | История сканов Defender |
| `font-cache` | `FontCache*.dat`, `FNTCACHE.DAT` |
| `recent` | `%APPDATA%\Microsoft\Windows\Recent` |
| `jump-lists` | AutomaticDestinations / CustomDestinations |
| `inet-cache` | `%LOCALAPPDATA%\Microsoft\Windows\INetCache` |
| `thumbnail-cache` | `thumbcache_*.db`, `iconcache_*.db` |
| `chrome-cache` / `edge-cache` / `brave-cache` | `User Data\Default\{Cache, Code Cache, GPUCache}` |
| `firefox-cache` | `...\Firefox\Profiles\*\cache2` |
| `teams-cache` / `teams-new-cache` | Classic + новый MSTeams_* |
| `discord-cache` / `slack-cache` / `telegram-cache` | мессенджеры |
| `spotify-cache` | Data / Storage / Browser |
| `vscode-cache` | `%APPDATA%\Code\{Cache,CachedData,CachedExtensions,Code Cache,GPUCache,logs,User\workspaceStorage}` |
| `jetbrains-logs` | `%LOCALAPPDATA%\JetBrains\*\{log,caches}` |
| `office-cache` | `%LOCALAPPDATA%\Microsoft\Office\*\OfficeFileCache` |
| `adobe-media-cache` | `Adobe\Common\Media Cache*` |
| `nuget-cache` / `pip-cache` / `npm-cache` / `yarn-cache` / `go-build-cache` / `gradle-cache` | кеши пакетных менеджеров и билд-систем |
| `steam-htmlcache` | HTML-кеш + логи Steam |
| `nvidia-cache` / `amd-cache` / `dx-shader-cache` | шейдерные кеши GPU |
| `recycle-bin` | Корзина на всех дисках (PowerShell `Clear-RecycleBin`) |
| `dns-cache` | `ipconfig /flushdns` |

## Категории (агрессивные — только с `--aggressive` или явно через `--categories`)

| ID | Что чистит |
|---|---|
| `windows-old` | `C:\Windows.old` — старая ОС после апгрейда, десятки ГБ, откат невозможен |
| `windows-installer-upgrade` | `C:\$WINDOWS.~BT`, `$WINDOWS.~WS`, `C:\ESD\Windows` |
| `event-logs` | Все журналы `*.evtx` через `wevtutil cl` |
| `iis-logs` | `C:\inetpub\logs\LogFiles` |
| `downloads-old` | Файлы из `%USERPROFILE%\Downloads` старше 90 дней |
| `maven-cache` | `%USERPROFILE%\.m2\repository` (долго пересобирать) |

## Что НЕ трогается никогда

Защита в `safePath` (`internal/cleaner/cleaner.go`):

- Корни дисков, `C:\Windows\System32`, `SysWOW64`, `WinSxS`
- `Program Files`, `Program Files (x86)`
- Меню «Пуск», `Users\Default`, `Users\Public`
- Корень домашней папки пользователя

Плюс в кешах браузеров намеренно выбраны только `Cache/Code Cache/GPUCache` — пароли, cookies, история и закладки **не трогаются**.

## Сборка и запуск

```powershell
go build -o wincleanerlamp.exe
```

Нужен Go 1.21+. Внешних модулей нет.

```powershell
# Список всех категорий (безопасные + агрессивные)
.\wincleanerlamp.exe --list

# Посчитать, сколько можно освободить
.\wincleanerlamp.exe --scan

# Посчитать вместе с агрессивными
.\wincleanerlamp.exe --scan --aggressive

# Очистить всё безопасное (с подтверждением)
.\wincleanerlamp.exe --clean

# Без подтверждения
.\wincleanerlamp.exe --clean --yes

# Только выбранные категории
.\wincleanerlamp.exe --clean --categories user-temp,prefetch,recycle-bin

# Исключить категории
.\wincleanerlamp.exe --clean --exclude recycle-bin,dns-cache

# Включая агрессивные
.\wincleanerlamp.exe --clean --aggressive --yes

# Только файлы старше 30 дней (глобально)
.\wincleanerlamp.exe --clean --min-age-hours 720

# Подробный лог каждого файла
.\wincleanerlamp.exe --clean --verbose

# Найти возможные остатки удалённых программ (отчёт, не удаляет!)
.\wincleanerlamp.exe --leftovers
```

Для полного эффекта (`C:\Windows\Temp`, `Prefetch`, `SoftwareDistribution`, корзина системного диска, `Windows.old`, event logs) запускайте консоль **от имени администратора**.

## Как ищутся остатки программ (`--leftovers`)

1. Читаем `DisplayName` из:
   - `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`
   - `HKLM\SOFTWARE\WOW6432Node\...\Uninstall`
   - `HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`
   - плюс вендоры из `HKLM\SOFTWARE\*`
2. Токенизируем имена (буквы/цифры, длиной ≥3).
3. Для каждой папки первого уровня в `%APPDATA%`, `%LOCALAPPDATA%`, `C:\ProgramData`:
   - если имя в whitelist системных/вендорских папок — пропускаем;
   - если совпадает с каким-либо токеном установленной программы — пропускаем;
   - иначе считаем размер и выводим как кандидата.
4. Сортируем по убыванию размера.

Эвристика ложно-положительная по природе — всегда **проверяйте папки вручную**, прежде чем удалять.

## Архитектура

```
main.go                              CLI, флаги, таблица, подтверждение, --leftovers
internal/cleaner/targets.go          Список категорий + раскрытие %ENV%
internal/cleaner/cleaner.go          Scan/Clean, спец-действия, safePath
internal/cleaner/leftovers.go        Поиск остатков удалённых программ через реестр
```

Единая точка — `cleaner.Process(Target, Options) Report`; `DryRun=true` считает размер без удаления.

## Дисклеймер

Используйте на свой страх и риск. Перед первым запуском сделайте `--scan` (и `--scan --aggressive`, если собираетесь использовать агрессивный режим). Программа консервативна по умолчанию, но окончательную ответственность за удалённые файлы несёте вы.
