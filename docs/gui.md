# winCleanerLamp — графический интерфейс (GUI)

GUI — это **десктопное приложение** на **Electron 29**, **React 18**, **TypeScript** и **Material UI (MUI) 5**. Оно не дублирует логику очистки: внутри запускается тот же **`win-cleaner-lamp.exe`**, что и в CLI, через безопасный **IPC** из главного процесса Electron.

> **Некоммерческий личный проект.** Сделан в первую очередь для удобного сценария «открыл окно — отсканировал — почистил», без обязательств по поддержке. **Автор не несёт ответственности** за сбои, потерю данных или любой вред. Используйте на свой риск после **`--scan` в CLI** или сканирования в GUI.

---

## Зачем GUI, если есть CLI

У консоли полный контроль и скрипты; у GUI — наглядный выбор категорий, переключатель агрессивного режима, лог в реальном времени и вкладки «Система» / «Остатки» без запоминания флагов. Оба варианта используют **один и тот же бинарник** CLI рядом с приложением (в dev — в корне репозитория относительно `gui`).

---

## Возможности интерфейса

| Вкладка / блок | Содержание |
|----------------|------------|
| **Очистка** | Чекбоксы категорий (безопасные и агрессивные), сканирование, очистка с подтверждением, лог вывода |
| **Программы** | Список установленных программ (`--apps-list`) с деинсталляцией (`--uninstall-launch`); если для программы известна база остатков — после удаления сразу предлагается очистить её мусор (`--orphan-scan` / `--orphan-clean`) |
| **Система** | Вывод `--sysinfo`: крупные системные файлы и подсказки (hiberfil, pagefile, WinSxS и т.д.) |
| **Остатки** | Результат `--leftovers`: отчёт по папкам-кандидатам без автоматического удаления |

Тема оформления: светлая / тёмная (в рамках настроек приложения).

---

## Архитектура (Onion / Clean)

```
gui/src/
├── domain/              # Сущности: категории, результаты сканирования и т.д.
├── application/         # Сценарии (use cases) и порты (интерфейсы сервисов)
├── infrastructure/      # Адаптеры: вызовы Electron IPC
├── presentation/        # React-компоненты и хуки
├── container/           # Сборка зависимостей (ручной DI)
└── shared/types/        # Типы, в т.ч. electron.d.ts

gui/electron/
├── main.ts              # Главный процесс: окно, IPC, spawn win-cleaner-lamp.exe
└── preload.ts           # Ограниченный мост contextBridge → renderer
```

**Безопасность:** включены **context isolation** и **preload** — страница React не получает прямой доступ к Node.js; только объявленные методы `electronAPI`.

---

## Поток данных (упрощённо)

```mermaid
flowchart LR
  subgraph renderer [Renderer React]
    UI[Компоненты]
  end
  subgraph main [Main Electron]
    IPC[IPC handlers]
    CLI[win-cleaner-lamp.exe]
  end
  UI -->|invoke| IPC
  IPC -->|spawn stdout/stderr| CLI
  CLI -->|строки лога| IPC
  IPC -->|события| UI
```

---

## Требования

- **Windows**, **Linux** или **macOS** — упаковка идёт под ту ОС, на которой запущена (Linux: для цели `rpm` нужен `rpmbuild`, пакет `rpm`).
- **Node.js** 20+ и **npm**.
- **Go** 1.21+ — для сборки ядра текущей ОС в корень репозитория (скрипты `pack` / `dist` делают это автоматически, см. `gui/build-cli.cjs`).

---

## Установка зависимостей и разработка

```powershell
cd gui
npm install
```

Скомпилировать main-процесс Electron (нужно перед первым `npm run dev`):

```powershell
npm run build:electron
```

Режим разработки (Vite на порту 3000 + Electron):

```powershell
npm run dev
```

В development `win-cleaner-lamp.exe` ожидается в **родительской** папке относительно `gui` (корень репозитория). Соберите CLI:

```powershell
cd ..
go build -o win-cleaner-lamp.exe ./wincli
cd gui
```

---

## Production-сборка

Скрипт **`npm run build`** в каталоге `gui`:

- проверяет и собирает TypeScript renderer;
- собирает фронтенд Vite в `gui/dist/`;
- компилирует Electron в `gui/dist-electron/`.

Полный цикл **CLI + GUI + упаковка** (каталог `gui/dist-release/`):

```powershell
cd gui
npm run pack
```

Пакеты для текущей ОС (electron-builder сам выбирает платформу, на которой запущен):

```powershell
npm run dist
```

| ОС | Цели (`build` в `gui/package.json`) | Ядро в `resources/` |
|---|---|---|
| Windows | NSIS-установщик + portable (x64) | `win-cleaner-lamp.exe` + `orphaned_apps.windows.json` |
| Linux | AppImage, deb, rpm | `lin-cleaner-lamp` + `orphaned_apps.linux.json` |
| macOS | dmg + zip (x64 и arm64, без подписи) | `mac-cleaner-lamp` + `orphaned_apps.darwin.json` |

`pack` и `dist` сначала выполняют **`build:cli`** (`node build-cli.cjs` — `go build` ядра текущей ОС в корень репозитория: `wincli` / `lincli` / `maccli`, карта в `gui/cli-target.cjs`), затем **`verify:cli`** (проверка, что файл есть — иначе сборка падает с понятной ошибкой), затем `build`, затем **electron-builder**.

Бинарник ядра попадает в **`resources/`** рядом с `app.asar` (**`extraResources`** своей платформы в `package.json`), а не рядом с исполняемым файлом GUI. Главный процесс ищет его в `process.resourcesPath` под именем, зависящим от `process.platform` (`EXE_NAME` в `electron/main.ts`).

Автообновление (electron-updater) работает в Windows и в Linux только из AppImage: deb/rpm обновляются пакетным менеджером, а неподписанная macOS-сборка не может установить обновление — в этих случаях GUI сообщает, что новую версию нужно скачать со страницы Releases.

Если ядро уже собрано в корне репозитория, достаточно **`npm run dist:electron`** (`verify:cli` + `build` + `electron-builder` без Go). Такой шаг используется в GitHub Actions (`release.yml`: три параллельных джоба Windows / Linux / macOS) после отдельного шага `go build`.

---

## Версия приложения

Версия в установщике берётся из **`gui/package.json`** → поле `version`. При релизе через **`npm run release:patch`** (или `minor` / `major`) из корня репозитория npm поднимает версию в корневом `package.json`, затем скрипт **`version`** копирует её в `gui/package.json` и создаётся git-тег `v…` — см. раздел про версии в корневом [`README.md`](../README.md).

---

## Расширение IPC

Если нужен новый метод:

1. Тип в `src/shared/types/electron.d.ts`.
2. Обработчик `ipcMain.handle` в `electron/main.ts`.
3. Экспорт в `electron/preload.ts` через `contextBridge`.
4. Порт в `application/ports/`, адаптер в `infrastructure/`, use case и при необходимости хук в `presentation/hooks/`.

---

## Ограничения и дисклеймер

- GUI **не отменяет** риски удаления данных: это оболочка над CLI.
- Агрессивные категории требуют явного включения в UI; всё равно читайте описание категорий в [docs/cli.md](cli.md).
- Проект **не коммерческий**; претензии к «службе поддержки» не предусмотрены.

Если что-то пошло не так — в первую очередь проверьте лог в окне и запуск `win-cleaner-lamp.exe` из той же папки вручную в консоли.
