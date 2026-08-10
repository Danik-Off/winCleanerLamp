<div align="center">

# winCleanerLamp

**Очистка мусора на Windows: быстрый CLI на Go + удобный GUI на Electron**

[![CI](https://github.com/Danik-Off/winCleanerLamp/actions/workflows/ci.yml/badge.svg)](https://github.com/Danik-Off/winCleanerLamp/actions/workflows/ci.yml)
[![Release](https://github.com/Danik-Off/winCleanerLamp/actions/workflows/release.yml/badge.svg)](https://github.com/Danik-Off/winCleanerLamp/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![Platform](https://img.shields.io/badge/platform-Windows-0078D6?logo=windows&logoColor=white)](#)

[Возможности](#возможности) ·
[Установка](#установка) ·
[CLI](#cli) ·
[GUI](#gui) ·
[Документация](#документация) ·
[Релизы](#версия-и-релизы)

</div>

---

Набор инструментов для **Windows**: консольная утилита (**CLI**, чистый Go, без внешних зависимостей) и опциональный **графический интерфейс** (**GUI**, Electron + React + MUI), которые помогают оценить и освободить место на диске за счёт удаления кешей, временных файлов и типичного «мусора» по заранее заданным и предсказуемым правилам.

> Это **некоммерческий личный проект**: автор не предлагает платной поддержки и гарантий, не несёт ответственности за последствия использования. Изначально всё делалось **для себя**, потому что не нашлось удобных **бесплатных** альтернатив с **поддержкой и развитием**, которые бы устраивали по сценарию использования.

---

## Возможности

| | |
|---|---|
| **Сканирование без риска** | `--scan` только считает размер мусора, ничего не удаляет |
| **50+ категорий** | системные кеши, браузеры, IDE, мессенджеры, пакетные менеджеры, GPU-кеши и др. — полный список: `--list` |
| **Агрессивный режим** | `Windows.old`, `$WINDOWS.~BT`, логи событий и т.п. — только по явному запросу (`--aggressive`) |
| **Поиск остатков программ** | `--leftovers` / `--orphan-*` — папки и ключи реестра от удалённых программ, без автоудаления |
| **Дубликаты и крупные файлы** | `--duplicates`, `--large-files` — поиск того, что реально занимает место |
| **Битые ярлыки** | `--shortcuts-scan` — `.lnk`-файлы с несуществующей целью |
| **Автозагрузка** | `--autostart-list` / `--autostart-enable` / `--autostart-disable` |
| **Безопасное удаление** | перемещение в Корзину по умолчанию, отложенное удаление занятых файлов при перезагрузке |
| **JSON-вывод** | `--json` для скриптов и для GUI |
| **GUI поверх того же CLI** | никакой дублирующей логики — Electron просто вызывает `win-cleaner-lamp.exe` |

---

## Установка

**Вариант 1 — готовая сборка.** На странице [Releases](https://github.com/Danik-Off/winCleanerLamp/releases) лежат: `win-cleaner-lamp.exe` (CLI, можно использовать отдельно) и установщик/portable GUI (`WinCleanerLamp-Setup-*.exe`). Скачать и запустить — сборка обновляется автоматически при каждом релизе (см. [«Версия и релизы»](#версия-и-релизы)).

**Вариант 2 — из исходников.** Нужен [Go 1.21+](https://go.dev/dl/) для CLI и дополнительно [Node.js 20+](https://nodejs.org/) для GUI:

```powershell
# CLI
go build -o win-cleaner-lamp.exe .
.\win-cleaner-lamp.exe --scan

# GUI (использует уже собранный win-cleaner-lamp.exe из корня)
cd gui
npm install
npm run build:electron
npm run dev
```

Сборка установщика GUI (`npm run dist`) и полный dev-цикл — в [docs/gui.md](docs/gui.md).

---

## CLI

Один исполняемый файл, stdlib + системные утилиты Windows (`powershell`, `reg`, `wevtutil`, `ipconfig`), без внешних Go-модулей. Мусор определяется по трём независимым сигналам: **расположение**, **возраст файла** и **имя/расширение**. Заблокированные процессом файлы по возможности пропускаются, не останавливая всю очистку; по умолчанию удаление идёт через Корзину.

**Сканирование и очистка**

```powershell
# Список всех категорий (с описаниями)
.\win-cleaner-lamp.exe --list

# Оценка освобождаемого места (только безопасные категории)
.\win-cleaner-lamp.exe --scan

# То же самое, но включая агрессивные категории
.\win-cleaner-lamp.exe --scan --aggressive

# Очистка с интерактивным подтверждением
.\win-cleaner-lamp.exe --clean

# Точечная очистка конкретных категорий без диалога
.\win-cleaner-lamp.exe --clean --categories user-temp,inet-cache --yes

# Не трогать файлы младше 48 часов
.\win-cleaner-lamp.exe --clean --min-age-hours 48
```

**Диагностика диска**

```powershell
# Крупные системные файлы (hiberfil.sys, pagefile.sys, WinSxS…) и советы
.\win-cleaner-lamp.exe --sysinfo

# Самые крупные файлы (от 100 МБ) в профиле пользователя
.\win-cleaner-lamp.exe --large-files

# Дубликаты файлов в указанных папках
.\win-cleaner-lamp.exe --duplicates "C:\Users\Me\Downloads,C:\Users\Me\Pictures"

# Пустые папки
.\win-cleaner-lamp.exe --empty-dirs "C:\Users\Me\Downloads"

# Битые ярлыки (.lnk с несуществующей целью) на рабочем столе и в меню Пуск
.\win-cleaner-lamp.exe --shortcuts-scan
```

**Остатки удалённых программ и автозагрузка**

```powershell
# Быстрый отчёт по возможным остаткам в AppData/ProgramData (без удаления)
.\win-cleaner-lamp.exe --leftovers

# Проверка известной базы orphaned_apps.json (подтверждённый мусор)
.\win-cleaner-lamp.exe --orphan-scan

# Удаление мусора конкретной программы (только её кеш — безопасно)
.\win-cleaner-lamp.exe --orphan-clean "Название программы" --orphan-cache-only

# Записи автозагрузки: показать / выключить одну
.\win-cleaner-lamp.exe --autostart-list
.\win-cleaner-lamp.exe --autostart-set <id> --autostart-disable
```

**Интеграция со скриптами**

```powershell
# Структурированный JSON вместо текстовой таблицы (то же использует GUI)
.\win-cleaner-lamp.exe --scan --json
```

Полный список флагов, всех категорий (безопасных и агрессивных) и правил безопасного удаления — в **[docs/cli.md](docs/cli.md)**.

---

## GUI

Десктопное приложение на **Electron 29 + React 18 + TypeScript + MUI 5** для тех, кто не хочет запоминать флаги. Не дублирует логику очистки — под капотом тот же `win-cleaner-lamp.exe`, вызываемый через безопасный IPC (`contextIsolation` + `preload`, без прямого доступа рендерера к Node.js).

**Как пользоваться**

1. Открыть приложение — на вкладке **«Очистка»** отмечены галочками безопасные категории.
2. При необходимости включить агрессивные категории отдельным переключателем (по умолчанию выключены).
3. Нажать **«Сканировать»** — приложение покажет объём мусора по категориям, ничего не удаляя.
4. Нажать **«Очистить»** и подтвердить — лог выполнения виден в реальном времени.
5. Вкладка **«Система»** — размеры `hiberfil.sys`, `pagefile.sys`, `WinSxS` и советы по ним.
6. Вкладка **«Остатки»** — отчёт по папкам-кандидатам от удалённых программ (только отчёт, без автоудаления).

| Вкладка | Содержание |
|---|---|
| **Очистка** | категории (безопасные/агрессивные), сканирование, очистка с подтверждением, лог в реальном времени |
| **Система** | крупные системные файлы (`hiberfil.sys`, `pagefile.sys`, `WinSxS`) и советы |
| **Остатки** | отчёт по папкам-кандидатам от удалённых программ |

Доступна светлая и тёмная тема. Архитектура — Onion/Clean (`domain → application → infrastructure → presentation`). Подробнее, включая диаграмму потока данных, инструкции по разработке и сборке установщика — в **[docs/gui.md](docs/gui.md)**.

---

## Документация

| Документ | О чём |
|---|---|
| [docs/cli.md](docs/cli.md) | Все флаги, категории мусора, правила безопасного удаления, сборка |
| [docs/gui.md](docs/gui.md) | Установка, разработка, IPC, архитектура, сборка установщика |
| [docs/research-junk-sources.md](docs/research-junk-sources.md) | Обоснование источников мусора для новых категорий |
| [docs/research-autostart.md](docs/research-autostart.md) | Заметки по реализации модуля автозагрузки |

---

## Линтинг и код-стиль

Статический анализ — [golangci-lint](https://golangci-lint.run/), конфигурация в [.golangci.yml](.golangci.yml).

```powershell
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
golangci-lint run
```

Или через `Makefile`:

```powershell
make lint-install
make lint
```

---

## Версия и релизы

**Единый номер версии** задаётся в корневом [`package.json`](package.json). Версия в [`gui/package.json`](gui/package.json) при релизе подставляется автоматически (`scripts/sync-gui-version.js` в npm lifecycle `version`).

В корне включён [`.npmrc`](.npmrc) с `git-tag-version=true`: при поднятии версии через npm создаётся git-коммит и аннотированный тег вида `v1.2.3`.

**Релиз** (из корня репозитория, с чистым `git status`):

```powershell
npm run release:patch   # или release:minor / release:major
npm run release:push
```

GitHub Actions автоматически собирает CLI + GUI и публикует релиз при пуше тега `v*` или при изменении корневого `package.json`.

---

## Важно (дисклеймер)

- Программа **удаляет файлы** с диска. Перед очисткой используйте режим сканирования (`--scan`) и проверяйте список категорий.
- Автор **не гарантирует** отсутствие ошибок и **не несёт ответственности** за потерю данных, сбои системы или любой другой ущерб.
- Проект **не является коммерческим продуктом** и не заменяет консультации по безопасности данных.
- Использование — **на ваш страх и риск**.

## Лицензия

Код распространяется по лицензии **[MIT](LICENSE)**.
