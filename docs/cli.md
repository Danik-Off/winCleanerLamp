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

Дополнительно в коде есть расширенный набор категорий (например `readyboot`, `store-cache`, кеши игровых лаунчеров и др.) — полный актуальный список: **`wincleanerlamp.exe --list`**.

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

В коде заданы ограничения (`safePath` в `internal/cleaner/cleaner.go`), в том числе:

- корни дисков, `System32`, `SysWOW64`, `WinSxS`;
- `Program Files`, `Program Files (x86)`;
- системные разделы профиля, которые не должны чиститься как «кеш».

В браузерах намеренно затрагиваются в основном **кеши**, а не пароли, cookies и история целиком — см. исходники категорий.

---

## Сборка

```powershell
go build -o wincleanerlamp.exe .
```

Оптимизация размера бинарника (как в релизных скриптах):

```powershell
go build -ldflags "-s -w" -o wincleanerlamp.exe .
```

---

## Примеры команд

```powershell
# Список категорий
.\wincleanerlamp.exe --list

# Оценка освобождаемого места (безопасные категории)
.\wincleanerlamp.exe --scan

# С агрессивными категориями
.\wincleanerlamp.exe --scan --aggressive

# Очистка с подтверждением
.\wincleanerlamp.exe --clean

# Без подтверждения
.\wincleanerlamp.exe --clean --yes

# Только выбранные категории
.\wincleanerlamp.exe --clean --categories user-temp,prefetch,recycle-bin

# Исключения
.\wincleanerlamp.exe --clean --exclude recycle-bin,dns-cache

# Агрессивная очистка
.\wincleanerlamp.exe --clean --aggressive --yes

# Глобальный минимальный возраст файлов (часы)
.\wincleanerlamp.exe --clean --min-age-hours 720

# Подробный лог
.\wincleanerlamp.exe --clean --verbose

# Остатки программ в AppData (только отчёт)
.\wincleanerlamp.exe --leftovers

# Системная информация (hiberfil, pagefile, WinSxS и т.д.)
.\wincleanerlamp.exe --sysinfo
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
| `main.go` | Точка входа, флаги, таблица, подтверждение, `--leftovers` |
| `internal/cleaner/targets.go` | Категории и подстановка `%ENV%` |
| `internal/cleaner/cleaner.go` | Scan/Clean, спец-действия, `safePath` |
| `internal/cleaner/leftovers.go` | Логика `--leftovers` |

Общая логика: `cleaner.Process(Target, Options) Report`; при `DryRun=true` выполняется только подсчёт.

---

## Дополнительные возможности (кратко)

- Параллельное сканирование: `--parallel N` (по умолчанию несколько воркеров), прогресс в консоли.
- Скрытие пустых категорий в отчёте сканирования; обратно: `--show-empty`.

---

## Отказ от ответственности

ПО поставляется «как есть». Вы несёте полную ответственность за запуск, в том числе с правами администратора и с агрессивными категориями.
