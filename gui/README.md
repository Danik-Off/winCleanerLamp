# WinCleanerLamp GUI

Исходники графической оболочки (Electron + React + TypeScript + MUI).

Полное описание установки, архитектуры, IPC и сборки — в **[`docs/gui.md`](../docs/gui.md)** в корне репозитория.

Кратко:

```powershell
cd gui
npm install
npm run build:electron
# в корне репозитория: go build -o wincleanerlamp.exe .
npm run dev
```

Production: `npm run pack` или `npm run dist` (собирают CLI и приложение).
