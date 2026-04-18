# winCleanerLamp

CLI-утилита на Go для очистки мусора в Windows. Безопасные категории, сухой прогон (`--scan`), подтверждение перед удалением.

## Где в Windows копится мусор

Ниже — места, откуда утилита подметает хлам. Все пути уже зашиты в программу (`internal/cleaner/targets.go`), и каждый можно включать/отключать флагами.

| Категория | Путь / действие | Зачем чистить |
|---|---|---|
| `user-temp` | `%TEMP%`, `%LOCALAPPDATA%\Temp` | Временные файлы приложений, часто не удаляются сами |
| `windows-temp` | `C:\Windows\Temp` | Системная `%TEMP%`. Требует прав администратора |
| `prefetch` | `C:\Windows\Prefetch` | Кеш предзагрузки приложений, пересоздаётся |
| `windows-update-cache` | `C:\Windows\SoftwareDistribution\Download` | Скачанные обновления старше суток |
| `delivery-optimization` | `...\DeliveryOptimization\Cache` | P2P-кеш обновлений |
| `cbs-logs` | `C:\Windows\Logs\CBS` | Логи обслуживания компонентов |
| `wer` | `...\WER\ReportArchive`, `ReportQueue` | Отчёты Windows Error Reporting |
| `crash-dumps` | `%LOCALAPPDATA%\CrashDumps`, `C:\Windows\Minidump`, `MEMORY.DMP` | Дампы падений |
| `recent` | `%APPDATA%\Microsoft\Windows\Recent` | Список недавних файлов |
| `inet-cache` | `%LOCALAPPDATA%\Microsoft\Windows\INetCache` | Кеш WinINet / IE / WebView |
| `thumbnail-cache` | `thumbcache_*.db`, `iconcache_*.db` | Кеш миниатюр и значков |
| `chrome-cache` / `edge-cache` / `brave-cache` | `User Data\Default\{Cache,Code Cache,GPUCache}` | Кеш браузеров (пароли/куки/история НЕ трогаются) |
| `firefox-cache` | `...\Firefox\Profiles\*\cache2` | Кеш Firefox во всех профилях |
| `nuget-cache` / `pip-cache` / `npm-cache` | кеши пакетных менеджеров | Восстанавливаются при следующей установке |
| `recycle-bin` | `Clear-RecycleBin` (PowerShell) | Корзина на всех дисках |
| `dns-cache` | `ipconfig /flushdns` | Сброс кеша DNS |

### Как определяется «остаточный» файл
- **По расположению** — файл лежит в одной из известных «мусорных» директорий выше.
- **По возрасту** — для кешей с `MinAgeHours` (например, `windows-update-cache` — >24ч, `cbs-logs` — >72ч, `nuget-cache` — >30 дней). Глобально можно задать `--min-age-hours N`.
- **По расширению** для спец-категорий (`thumbcache_*.db`, `iconcache_*.db`, `*.dmp`).
- **Заблокирован процессом** — тогда файл просто пропускается без ошибки остановки.

### Что НЕ трогается
- Корни дисков, `C:\Windows\System32`, `WinSxS`, `Program Files`, меню «Пуск», профили `Default`/`Public` и домашняя папка пользователя целиком (см. `safePath` в `cleaner.go`).
- Пароли, cookies, история, закладки браузеров.
- `Windows.old` и точки восстановления (слишком рискованно без явного согласия; можно добавить при необходимости).

## Сборка

```powershell
go build -o wincleanerlamp.exe
```

Нужен Go 1.21+. Только стандартная библиотека, внешних зависимостей нет.

## Использование

```powershell
# Показать все категории
.\wincleanerlamp.exe --list

# Посчитать, сколько можно освободить (ничего не удаляется)
.\wincleanerlamp.exe --scan

# Очистить всё с подтверждением
.\wincleanerlamp.exe --clean

# Очистить без подтверждения
.\wincleanerlamp.exe --clean --yes

# Только выбранные категории
.\wincleanerlamp.exe --clean --categories user-temp,prefetch,recycle-bin

# Исключить категории
.\wincleanerlamp.exe --clean --exclude recycle-bin,dns-cache

# Удалять только файлы старше 7 дней
.\wincleanerlamp.exe --clean --min-age-hours 168

# Подробный лог каждого действия
.\wincleanerlamp.exe --clean --verbose
```

Для очистки `C:\Windows\Temp`, `Prefetch`, `SoftwareDistribution` и корзины на системном диске запускайте консоль **от имени администратора**.

## Архитектура

```
main.go                          CLI: флаги, таблица, подтверждение
internal/cleaner/targets.go      Список категорий и раскрытие %ENV%
internal/cleaner/cleaner.go      Scan/Clean, спец-действия, safePath
```

`Process(Target, Options) Report` — единая точка для сканирования и удаления; при `DryRun=true` только считает размер.

## Дисклеймер

Используйте на свой страх и риск. Перед первым запуском обязательно сделайте `--scan` и просмотрите список. Утилита старается быть консервативной (не трогает системные каталоги, пропускает занятые файлы, требует подтверждения), но окончательную ответственность за удаление файлов несёте вы.
