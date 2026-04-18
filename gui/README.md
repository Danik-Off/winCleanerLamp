# WinCleanerLamp GUI

Графический интерфейс для `wincleanerlamp.exe` на базе **Electron + React + TypeScript + Material-UI (MUI)** с архитектурой **Clean Architecture / Onion Architecture**.

## Возможности

- Визуальный выбор категорий — чекбоксы для всех безопасных и агрессивных категорий
- Сканирование с прогрессом — живой лог и результаты в реальном времени
- Очистка одной кнопкой — подтверждение перед удалением
- Системная информация — размеры hiberfil.sys, pagefile.sys, WinSxS и т.д.
- Поиск остатков программ — сканирование AppData на "сиротские" папки
- Современный UI — Material Design с тёмной/светлой темой
- Чистая архитектура — Domain, Application, Infrastructure, Presentation слои

## Архитектура (Onion Architecture)

```
src/
├── domain/               # Ядро: бизнес-логика и сущности
│   ├── entities/         # Category, ScanResult, LeftoverItem, SystemInfo, OperationLog
│   └── index.ts
├── application/          # Слой применения: use cases и порты
│   ├── ports/            # Интерфейсы (ICleanerService, ISystemInfoService, ILeftoverService)
│   └── useCases/         # ScanUseCase, CleanUseCase, GetCategoriesUseCase, etc.
├── infrastructure/       # Внешний слой: адаптеры
│   └── adapters/         # ElectronCleanerService, ElectronSystemInfoService, ElectronLeftoverService
├── presentation/         # UI слой: React компоненты и хуки
│   ├── components/       # CleanupPanel, SystemInfoPanel, LeftoversPanel
│   └── hooks/            # useCategories, useScan, useClean, useSystemInfo, useLeftovers
├── container/            # DI контейнер (ручной, без фреймворков)
│   └── index.ts
└── shared/
    └── types/
        └── electron.d.ts # Глобальные типы Electron API (ambient declaration)

electron/                 # Electron main process (TypeScript → dist-electron/)
├── main.ts               # Главный процесс
└── preload.ts            # IPC bridge
```

## Структура проекта

```
gui/
├── package.json              # Зависимости и скрипты
├── tsconfig.json             # TypeScript конфигурация (renderer)
├── tsconfig.electron.json    # TypeScript конфигурация (main process)
├── tsconfig.node.json        # TypeScript конфигурация (Vite)
├── vite.config.ts            # Конфигурация Vite для React
├── index.html                # Точка входа HTML
├── electron/
│   ├── main.ts
│   └── preload.ts
└── src/
    ├── main.tsx              # Точка входа React
    ├── App.tsx               # Главный компонент с табами
    ├── domain/
    ├── application/
    ├── infrastructure/
    ├── presentation/
    ├── container/
    └── shared/
```

## Установка и запуск

### 1. Убедитесь что собран wincleanerlamp.exe

В корне проекта (родительская папка gui/):

```powershell
go build -o wincleanerlamp.exe .
```

### 2. Установите зависимости GUI

```powershell
cd gui
npm install
```

### 3. Проверка типов (TypeScript)

```powershell
npm run type-check
npm run lint
```

### 4. Режим разработки (с hot reload)

```powershell
# Сначала скомпилировать Electron main process
npm run build:electron

# Запустить Vite + Electron одновременно
npm run dev
```

### 5. Сборка production

```powershell
npm run build
```

Собирает:
- Фронтенд в `dist/`
- Electron в `dist-electron/`

## Архитектура IPC

```
┌─────────────┐      IPC       ┌─────────────┐
│  Renderer   │  ◄──────────►  │    Main     │
│  (React)    │   electronAPI  │   (Node.js) │
└─────────────┘                └─────────────┘
                                      │
                                      │ spawn
                                      ▼
                               ┌─────────────┐
                               │ wincleaner  │
                               │  lamp.exe   │
                               │   (Go CLI)  │
                               └─────────────┘
```

### ElectronAPI (глобальный тип из `electron.d.ts`)

```typescript
interface ElectronAPI {
  getCategories: () => Promise<CategoriesResponseDto>;
  scan: (options: ScanOptionsDto) => Promise<ScanResultDto>;
  clean: (options: CleanOptionsDto) => Promise<CleanResultDto>;
  getSysInfo: () => Promise<string>;
  getLeftovers: () => Promise<string>;
  onScanProgress: (callback: (data: string) => void) => void;
  onCleanProgress: (callback: (data: string) => void) => void;
  removeAllListeners: (channel: string) => void;
}
```

## Технологии

- **Electron** — обёртка для десктопного приложения
- **React 18** — UI библиотека
- **TypeScript 5** — строгая типизация
- **Material-UI (MUI) v5** — компоненты Material Design
- **Vite** — быстрый сборщик для разработки
- **Clean Architecture / Onion Architecture** — слоистая архитектура
- **Context Isolation + Preload** — безопасная архитектура Electron

## Вкладки интерфейса

1. **Очистка** — выбор категорий, сканирование, удаление
2. **Система** — информация о больших системных файлах
3. **Остатки программ** — поиск "сиротских" папок в AppData

## Безопасность

- **Context Isolation** включена — renderer не имеет доступа к Node.js API
- **Preload script** предоставляет только необходимые IPC методы
- Агрессивные категории требуют явного включения переключателем
- Диалог подтверждения перед удалением

## Разработка

### Добавление новых IPC методов

1. Добавьте тип в `src/shared/types/electron.d.ts`:
```typescript
interface ElectronAPI {
  myEvent: (args: MyEventArgs) => Promise<MyEventResult>;
}
```

2. Добавьте обработчик в `electron/main.ts`:
```typescript
ipcMain.handle('my-event', async (event, args) => {
  // ...
});
```

3. Добавьте вызов в `electron/preload.ts`:
```typescript
myEvent: (args) => ipcRenderer.invoke('my-event', args),
```

4. Создайте порт (интерфейс) в `src/application/ports/`
5. Создайте адаптер в `src/infrastructure/adapters/`
6. Создайте use case в `src/application/useCases/`
7. Создайте хук в `src/presentation/hooks/`
