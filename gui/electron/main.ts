/**
 * Electron Main Process
 * TypeScript implementation of main process
 */
import { app, BrowserWindow, dialog, ipcMain, IpcMainInvokeEvent } from 'electron';
import path from 'path';
import { spawn, SpawnOptionsWithoutStdio } from 'child_process';
import fs from 'fs';
import { autoUpdater } from 'electron-updater';

// Constants
const EXE_NAME = 'win-cleaner-lamp.exe';
const DEV_PORT = 3000;

/**
 * Путь к win-cleaner-lamp.exe:
 * - dev: корень репозитория (на уровень выше gui/)
 * - production: electron-builder кладёт бинарник в extraResources → каталог process.resourcesPath
 */
function getExePath(): string {
  const devPath = path.join(__dirname, '..', '..', EXE_NAME);
  if (!app.isPackaged) {
    return devPath;
  }
  const inResources = path.join(process.resourcesPath, EXE_NAME);
  if (fs.existsSync(inResources)) {
    return inResources;
  }
  const besideApp = path.join(path.dirname(process.execPath), EXE_NAME);
  if (fs.existsSync(besideApp)) {
    return besideApp;
  }
  return inResources;
}

/** Рендерер: только в упакованном приложении грузим dist; иначе легко словить localhost при NODE_ENV=development в системе */
function getIndexHtmlPath(): string {
  return path.join(app.getAppPath(), 'dist', 'index.html');
}

function shouldLoadDevServer(): boolean {
  return !app.isPackaged && process.env.NODE_ENV === 'development';
}

/**
 * Main Window reference
 */
let mainWindow: BrowserWindow | null = null;

/**
 * Create the main application window
 */
function createWindow(): void {
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    minWidth: 900,
    minHeight: 600,
    frame: false,
    titleBarStyle: 'hidden',
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      preload: path.join(__dirname, 'preload.js'),
    },
    title: 'WinCleanerLamp GUI',
    show: false, // Show when ready
  });

  // Упакованное приложение всегда с file:// из dist (не доверяем NODE_ENV — иначе пустое окно при dev в PATH)
  if (shouldLoadDevServer()) {
    mainWindow.loadURL(`http://localhost:${DEV_PORT}`);
    mainWindow.webContents.openDevTools();
  } else {
    const indexPath = getIndexHtmlPath();
    mainWindow.loadFile(indexPath);
  }

  // Show when ready to prevent visual flash
  mainWindow.once('ready-to-show', () => {
    mainWindow?.show();
  });

  mainWindow.on('closed', () => {
    mainWindow = null;
  });
}

// App lifecycle
try {
  app.whenReady().then(() => {
    createWindow();

    app.on('activate', () => {
      if (BrowserWindow.getAllWindows().length === 0) {
        createWindow();
      }
    });

    // Тихая фоновая проверка обновлений вскоре после старта — но даже она
    // не скачивает и не ставит ничего без явного согласия пользователя
    // (см. setupAutoUpdater ниже: autoDownload=false, диалог с Установить/Отмена).
    if (app.isPackaged) {
      setTimeout(() => {
        autoUpdater.checkForUpdates().catch((err) => {
          console.error('Background update check failed:', err);
        });
      }, 5000);
    }
  });

  app.on('window-all-closed', () => {
    if (process.platform !== 'darwin') {
      app.quit();
    }
  });
} catch (error) {
  console.error('Failed to start application:', error);
  app.quit();
}

// IPC Handlers

/**
 * Execute CLI command and return output.
 *
 * onProgress (опционально) получает живые статусные строки из stderr вида
 * "PROGRESS i/N Имя категории" — CLI пишет их даже в --json режиме именно
 * для того, чтобы GUI мог показывать реальный прогресс long-running
 * сканирований, не дожидаясь завершения процесса. Такие строки не попадают
 * в возвращаемый stderr (иначе замусорили бы диагностику ошибок).
 */
function executeCli(args: string[], onProgress?: (line: string) => void): Promise<{ stdout: string; stderr: string; code: number }> {
  return new Promise((resolve, reject) => {
    const exePath = getExePath();

    if (!fs.existsSync(exePath)) {
      reject(new Error(`Executable not found: ${exePath}`));
      return;
    }

    const options: SpawnOptionsWithoutStdio = {
      cwd: path.dirname(exePath),
    };

    const child = spawn(exePath, args, options);
    let stdout = '';
    const stderrLines: string[] = [];

    child.stdout?.on('data', (data: Buffer) => {
      stdout += data.toString();
    });

    child.stderr?.on('data', (data: Buffer) => {
      const chunk = data.toString();
      for (const line of chunk.split(/\r?\n/)) {
        if (!line) continue;
        const m = line.match(/^PROGRESS\s+(\d+)\/(\d+)\s+(.*)$/);
        if (m && onProgress) {
          onProgress(`[${m[1]}/${m[2]}] ${m[3]}`);
          continue;
        }
        stderrLines.push(line);
      }
    });

    child.on('close', (code: number | null) => {
      resolve({
        stdout,
        stderr: stderrLines.join('\n'),
        code: code ?? -1,
      });
    });

    child.on('error', (error: Error) => {
      reject(error);
    });
  });
}

// Get Categories IPC Handler — CLI отдаёт --list --json, парсинг текста регэкспами не нужен.
ipcMain.handle('get-categories', async () => {
  const { stdout } = await executeCli(['--list', '--json']);
  const parsed: JsonCategoriesResult = JSON.parse(stdout);
  return {
    safe: parsed.safe.map(toCategoryDto),
    aggressive: parsed.aggressive.map(toCategoryDto),
  };
});

// Scan IPC Handler
ipcMain.handle('scan', async (_event: IpcMainInvokeEvent, options: { aggressive: boolean; categories?: string[] }) => {
  const args = ['--scan', '--json'];
  if (options.aggressive) args.push('--aggressive');
  if (options.categories?.length) args.push('--categories', options.categories.join(','));

  const { stdout } = await executeCli(args, (line) => mainWindow?.webContents.send('scan-progress', line));
  const parsed: JsonScanResult = JSON.parse(stdout);

  return {
    output: buildScanLogText(parsed),
    parsed: {
      categories: parsed.categories.map((c) => ({
        id: c.id,
        name: c.name,
        size: formatBytes(c.bytes),
        sizeBytes: c.bytes,
        files: c.files,
      })),
      totalBytes: parsed.totalBytes,
      totalFiles: parsed.totalFiles,
    },
    code: 0,
  };
});

// Clean IPC Handler
ipcMain.handle('clean', async (_event: IpcMainInvokeEvent, options: { aggressive: boolean; categories?: string[]; yes: boolean }) => {
  const args = ['--clean', '--json'];
  if (options.aggressive) args.push('--aggressive');
  if (options.yes) args.push('--yes');
  if (options.categories?.length) args.push('--categories', options.categories.join(','));

  const { stdout, stderr, code } = await executeCli(args, (line) => mainWindow?.webContents.send('clean-progress', line));
  if (code !== 0 || !stdout.trim()) {
    // Отменено пользователем (нет --yes) или ошибка запуска — нет JSON на stdout.
    return { output: stdout || stderr, error: stderr, code, bytesCleaned: 0, filesCleaned: 0, errorCount: 0 };
  }
  const parsed: JsonCleanResult = JSON.parse(stdout);
  return {
    output: buildCleanLogText(parsed),
    error: stderr,
    code,
    bytesCleaned: parsed.cleanedBytes,
    filesCleaned: parsed.cleanedFiles,
    errorCount: parsed.errors?.length ?? 0,
  };
});

// System Info IPC Handler
ipcMain.handle('get-sysinfo', async () => {
  const { stdout } = await executeCli(['--sysinfo']);
  return stdout;
});

// Leftovers IPC Handler — структурированный JSON вместо парсинга текстовых таблиц.
ipcMain.handle('get-leftovers', async () => {
  const { stdout } = await executeCli(['--leftovers', '--json'], (line) => mainWindow?.webContents.send('scan-progress', line));
  return stdout;
});

// Duplicates IPC Handler — структурированный JSON вместо парсинга текста регэкспами.
interface JsonDuplicateGroup {
  hash: string;
  size: number;
  paths: string[];
  wasteSize: number;
  riskFlag?: string;
}
interface JsonDuplicatesResult {
  groups: JsonDuplicateGroup[] | null;
  totalWaste: number;
  totalFiles: number;
  scannedFiles: number;
  durationSeconds: number;
  skippedRoots?: string[];
  error?: string;
}

ipcMain.handle('get-duplicates', async (_event: IpcMainInvokeEvent, rootPaths: string) => {
  const { stdout } = await executeCli(['--duplicates', rootPaths, '--json'], (line) => mainWindow?.webContents.send('scan-progress', line));
  const parsed: JsonDuplicatesResult = JSON.parse(stdout);
  return {
    groups: (parsed.groups || []).map((g) => ({
      size: g.size,
      sizeFormatted: formatBytes(g.size),
      waste: g.wasteSize,
      wasteFormatted: formatBytes(g.wasteSize),
      paths: g.paths,
      riskFlag: g.riskFlag,
    })),
    scannedFiles: parsed.scannedFiles,
    totalWaste: parsed.totalWaste,
    skippedRoots: parsed.skippedRoots || [],
    error: parsed.error,
  };
});

// Empty Dirs IPC Handler
interface JsonEmptyDirResult {
  dirs: { path: string; depth: number }[] | null;
  total: number;
  error?: string;
}

ipcMain.handle('get-empty-dirs', async (_event: IpcMainInvokeEvent, rootPaths: string) => {
  const { stdout } = await executeCli(['--empty-dirs', rootPaths, '--json'], (line) => mainWindow?.webContents.send('scan-progress', line));
  const parsed: JsonEmptyDirResult = JSON.parse(stdout);
  return {
    dirs: (parsed.dirs || []).map((d) => d.path),
    error: parsed.error,
  };
});

// ─── Безопасное удаление ───
//
// Раньше эти три хендлера делали fs.unlinkSync/fs.rmSync/собственный
// PowerShell-скрипт прямо в Electron — без единой проверки безопасности пути
// (delete-file и delete-leftover вообще без какой-либо проверки) и с ломаным
// экранированием (`replace(/"/g, '\"')` не меняет строку). Теперь всё удаление
// идёт через CLI (--delete-path/--delete-dir), который использует единую
// cleaner.IsPathSafeToDelete и корректно экранированный PowerShell
// (-EncodedCommand). Electron здесь — только тонкая обёртка.

async function runDeleteCli(flag: '--delete-path' | '--delete-dir', targetPath: string): Promise<DeleteResultDto> {
  const { stdout, stderr } = await executeCli([flag, targetPath, '--json']);
  const raw = stdout.trim();
  if (!raw) {
    return { success: false, error: stderr || 'CLI не вернул результат' };
  }
  try {
    const parsed = JSON.parse(raw) as { success: boolean; movedToRecycleBin: boolean; error?: string };
    return { success: parsed.success, error: parsed.error, movedToRecycleBin: parsed.movedToRecycleBin };
  } catch {
    return { success: false, error: raw };
  }
}

interface DeleteResultDto {
  success: boolean;
  error?: string;
  movedToRecycleBin?: boolean;
}

// Delete Empty Dir IPC Handler — перемещение в Корзину (см. runDeleteCli).
ipcMain.handle('delete-empty-dir', async (_event: IpcMainInvokeEvent, dirPath: string) => {
  return runDeleteCli('--delete-dir', dirPath);
});

// Delete File IPC Handler (для дубликатов) — перемещение в Корзину.
ipcMain.handle('delete-file', async (_event: IpcMainInvokeEvent, filePath: string) => {
  return runDeleteCli('--delete-path', filePath);
});

// Delete Leftover Folder IPC Handler — перемещение в Корзину.
ipcMain.handle('delete-leftover', async (_event: IpcMainInvokeEvent, folderPath: string) => {
  return runDeleteCli('--delete-dir', folderPath);
});

// ─── Битые ярлыки ───
// Удаление найденных ярлыков идёт через уже существующий delete-file
// (--delete-path --json) — ярлык это обычный файл, отдельная команда не нужна.
interface JsonBrokenShortcut {
  path: string;
  targetPath?: string;
  reason: string;
}
interface JsonShortcutScanResult {
  broken: JsonBrokenShortcut[] | null;
  scanned: number;
}

ipcMain.handle('get-broken-shortcuts', async () => {
  const { stdout, stderr, code } = await executeCli(['--shortcuts-scan', '--json'], (line) => mainWindow?.webContents.send('scan-progress', line));
  if (code !== 0 || !stdout.trim()) {
    return { broken: [], scanned: 0, error: stderr || 'CLI не вернул результат' };
  }
  try {
    const parsed: JsonShortcutScanResult = JSON.parse(stdout);
    return { broken: parsed.broken || [], scanned: parsed.scanned, error: '' };
  } catch {
    return { broken: [], scanned: 0, error: stdout || stderr };
  }
});

// ─── Экспорт снимка для анализа ───
// Комбинирует orphan-discover (неизвестные папки) + список установленных
// программ в один JSON-файл — пользователь сам выбирает, куда сохранить,
// затем может проанализировать/дополнить orphaned_apps.json вручную.
// orphaned_apps.json при этом не меняется.
interface JsonAuditExport {
  generatedAt: string;
  unknownFolders: unknown[] | null;
  installedPrograms: unknown[] | null;
}

// Универсальный экспорт: рендерер уже держит в памяти результат сканирования
// (список остатков, ярлыков и т.п.) — просто просим пользователя выбрать
// файл и сохраняем то, что прислали, без повторного обращения к CLI.
// Используется кнопкой "Экспорт" в панелях, чтобы можно было потом вручную
// проанализировать находки и дополнить orphaned_apps.json/список категорий.
ipcMain.handle('export-json', async (_event: IpcMainInvokeEvent, options: { suggestedName?: string; data: unknown }) => {
  if (!mainWindow) {
    return { success: false, error: 'Окно приложения недоступно' };
  }
  const { canceled, filePath } = await dialog.showSaveDialog(mainWindow, {
    title: 'Сохранить для анализа',
    defaultPath: options.suggestedName || `wincleanerlamp-export-${new Date().toISOString().slice(0, 10)}.json`,
    filters: [{ name: 'JSON', extensions: ['json'] }],
  });
  if (canceled || !filePath) {
    return { success: false, canceled: true };
  }
  try {
    fs.writeFileSync(filePath, JSON.stringify(options.data, null, 2), 'utf-8');
    return { success: true, path: filePath };
  } catch (err) {
    return { success: false, error: err instanceof Error ? err.message : String(err) };
  }
});

ipcMain.handle('export-audit', async () => {
  if (!mainWindow) {
    return { success: false, error: 'Окно приложения недоступно' };
  }
  const { canceled, filePath } = await dialog.showSaveDialog(mainWindow, {
    title: 'Сохранить снимок для анализа',
    defaultPath: `wincleanerlamp-audit-${new Date().toISOString().slice(0, 10)}.json`,
    filters: [{ name: 'JSON', extensions: ['json'] }],
  });
  if (canceled || !filePath) {
    return { success: false, canceled: true };
  }

  const { stdout, stderr, code } = await executeCli(['--audit-export', filePath, '--json']);
  if (code !== 0 || !stdout.trim()) {
    return { success: false, error: stderr || 'CLI не вернул результат' };
  }
  try {
    const parsed = JSON.parse(stdout) as { success: boolean; path?: string; error?: string };
    if (!parsed.success) {
      return { success: false, error: parsed.error || 'неизвестная ошибка' };
    }
    // Читаем сохранённый файл, чтобы вернуть в рендерер краткую сводку (счётчики).
    let unknownCount = 0;
    let installedCount = 0;
    try {
      const raw = fs.readFileSync(filePath, 'utf-8');
      const audit = JSON.parse(raw) as JsonAuditExport;
      unknownCount = audit.unknownFolders?.length ?? 0;
      installedCount = audit.installedPrograms?.length ?? 0;
    } catch {
      // Файл создан, но сводку прочитать не удалось — не критично для успеха операции.
    }
    return { success: true, path: filePath, unknownCount, installedCount };
  } catch {
    return { success: false, error: stdout || stderr };
  }
});

// ─── Autostart Manager IPC Handlers ───
// Функция в первую очередь для GUI (см. docs/research-autostart.md) — CLI-сторона
// сознательно минимальна (--json-only, без текстового UX), обёртки здесь тонкие.

interface AutostartEntryDto {
  id: string;
  source: string;
  name: string;
  command?: string;
  location: string;
  enabled: boolean;
  canToggle: boolean;
}

ipcMain.handle('autostart-list', async () => {
  const { stdout, stderr, code } = await executeCli(['--autostart-list', '--json']);
  if (code !== 0 || !stdout.trim()) {
    return { entries: [] as AutostartEntryDto[], error: stderr };
  }
  try {
    const entries: AutostartEntryDto[] = JSON.parse(stdout);
    return { entries, error: '' };
  } catch {
    return { entries: [] as AutostartEntryDto[], error: stdout || stderr };
  }
});

ipcMain.handle('autostart-toggle', async (_event: IpcMainInvokeEvent, options: { id: string; enable: boolean }) => {
  const args = ['--autostart-set', options.id, options.enable ? '--autostart-enable' : '--autostart-disable', '--json'];
  const { stdout, stderr } = await executeCli(args);
  try {
    const parsed = JSON.parse(stdout.trim()) as { success: boolean; error?: string };
    return parsed;
  } catch {
    return { success: false, error: stderr || stdout };
  }
});

// ─── OrphanCleaner IPC Handlers ───

// Orphan Scan: check orphaned_apps.json entries — структурированный JSON.
ipcMain.handle('orphan-scan', async (_event: IpcMainInvokeEvent, configPath?: string) => {
  const args = ['--orphan-scan', '--json'];
  if (configPath) args.push('--orphan-config', configPath);
  const { stdout, stderr, code } = await executeCli(args, (line) => mainWindow?.webContents.send('scan-progress', line));
  return { output: stdout, error: stderr, code };
});

// Orphan Discover: find unknown folders
ipcMain.handle('orphan-discover', async (_event: IpcMainInvokeEvent, options?: { roots?: string; jsonOutput?: boolean }) => {
  const args = ['--orphan-discover', '--orphan-json'];
  if (options?.roots) args.push('--orphan-roots', options.roots);
  const { stdout, stderr, code } = await executeCli(args);
  return { output: stdout, error: stderr, code };
});

// Orphan Clean: delete leftovers for specified programs
ipcMain.handle('orphan-clean', async (_event: IpcMainInvokeEvent, options: { names: string; recycle?: boolean; cacheOnly?: boolean; includeUserData?: boolean }) => {
  const args = ['--orphan-clean', options.names];
  if (options.recycle) args.push('--orphan-recycle');
  if (options.cacheOnly) args.push('--orphan-cache-only');
  if (options.includeUserData) args.push('--orphan-include-user-data');
  args.push('--verbose');
  const { stdout, stderr, code } = await executeCli(args);
  return { output: stdout, error: stderr, code };
});

// Orphan Info: detailed info for a program
ipcMain.handle('orphan-info', async (_event: IpcMainInvokeEvent, displayName: string) => {
  const args = ['--orphan-info', displayName];
  const { stdout, stderr, code } = await executeCli(args);
  return { output: stdout, error: stderr, code };
});

// Orphan List: list all entries from orphaned_apps.json
ipcMain.handle('orphan-list', async (_event: IpcMainInvokeEvent, configPath?: string) => {
  const args = ['--orphan-list'];
  if (configPath) args.push('--orphan-config', configPath);
  const { stdout, stderr, code } = await executeCli(args);
  return { output: stdout, error: stderr, code };
});

// Orphan Track: add a discovered folder into orphaned_apps.json (explicit user action only).
ipcMain.handle('orphan-track', async (_event: IpcMainInvokeEvent, options: { path: string; name?: string; asCache?: boolean }) => {
  const args = ['--orphan-track', options.path, '--json'];
  if (options.name) args.push('--orphan-track-name', options.name);
  if (options.asCache) args.push('--orphan-track-cache');
  const { stdout, stderr } = await executeCli(args);
  try {
    return JSON.parse(stdout.trim()) as { success: boolean; error?: string };
  } catch {
    return { success: false, error: stderr || stdout };
  }
});

// ─── Автообновление (electron-updater + GitHub Releases) ───
//
// Поток: checkForUpdates() → событие 'update-available' с версией/описанием
// уходит в рендерер, который показывает диалог с кнопками Установить/Отмена
// (см. AboutPanel.tsx). Ничего не скачивается автоматически — autoDownload
// выключен намеренно, скачивание стартует только по нажатию «Установить»
// (install-update). После скачивания — автоматический перезапуск с установкой.
autoUpdater.autoDownload = false;
autoUpdater.autoInstallOnAppQuit = false;

autoUpdater.on('update-available', (info) => {
  mainWindow?.webContents.send('update-available', {
    version: info.version,
    releaseDate: info.releaseDate,
    releaseNotes: typeof info.releaseNotes === 'string' ? info.releaseNotes : '',
  });
});

autoUpdater.on('update-not-available', () => {
  mainWindow?.webContents.send('update-not-available');
});

autoUpdater.on('error', (err) => {
  const raw = err.message || String(err);
  console.error('autoUpdater error:', raw);
  mainWindow?.webContents.send('update-error', friendlyUpdateError(raw));
});

/**
 * electron-updater отдаёт сырые HTTP-ошибки builder-util-runtime (с полными
 * заголовками ответа в одну строку) — показывать это пользователю в диалоге
 * бессмысленно и пугающе. Здесь только текст для UI; полная ошибка всегда
 * уходит в консоль (см. выше) для диагностики.
 */
function friendlyUpdateError(raw: string): string {
  if (/latest\.yml/i.test(raw) || /404/.test(raw)) {
    return 'Сервер обновлений вернул неполный релиз (отсутствует latest.yml). Попробуйте позже — это будет исправлено в следующем релизе.';
  }
  if (/ENOTFOUND|ETIMEDOUT|ECONNREFUSED|net::/i.test(raw)) {
    return 'Не удалось подключиться к серверу обновлений. Проверьте интернет-соединение.';
  }
  const firstLine = raw.split('\n')[0].trim();
  return firstLine.length > 200 ? firstLine.slice(0, 200) + '…' : firstLine;
}

autoUpdater.on('download-progress', (progress) => {
  mainWindow?.webContents.send('update-download-progress', Math.round(progress.percent));
});

autoUpdater.on('update-downloaded', () => {
  mainWindow?.webContents.send('update-downloaded');
  // Небольшая пауза, чтобы рендерер успел показать "перезапуск..." перед тем,
  // как окно закроется.
  setTimeout(() => autoUpdater.quitAndInstall(), 1500);
});

ipcMain.handle('check-for-updates', async () => {
  if (!app.isPackaged) {
    return { success: false, error: 'Проверка обновлений доступна только в собранном приложении.' };
  }
  try {
    await autoUpdater.checkForUpdates();
    return { success: true };
  } catch (err) {
    return { success: false, error: err instanceof Error ? err.message : String(err) };
  }
});

ipcMain.handle('install-update', async () => {
  try {
    await autoUpdater.downloadUpdate();
    return { success: true };
  } catch (err) {
    return { success: false, error: err instanceof Error ? err.message : String(err) };
  }
});

// Window Control IPC Handlers
ipcMain.on('window-minimize', () => {
  mainWindow?.minimize();
});

ipcMain.on('window-maximize', () => {
  if (mainWindow?.isMaximized()) {
    mainWindow.unmaximize();
  } else {
    mainWindow?.maximize();
  }
});

ipcMain.on('window-close', () => {
  mainWindow?.close();
});

ipcMain.handle('window-is-maximized', () => {
  return mainWindow?.isMaximized() ?? false;
});

// ─── JSON DTO из CLI (--json) ───
//
// Раньше здесь разбирался текстовый вывод CLI регэкспами (parseCategories,
// parseScanOutput) — хрупко, ломалось при любой правке форматирования таблиц
// на Go-стороне. Теперь CLI умеет отдавать структурированный JSON напрямую
// (см. main.go: --json), поэтому парсинг сводится к JSON.parse.

interface JsonCategoryDto {
  id: string;
  name: string;
  description: string;
  aggressive: boolean;
}

interface JsonCategoriesResult {
  safe: JsonCategoryDto[];
  aggressive: JsonCategoryDto[];
}

function toCategoryDto(c: JsonCategoryDto): { id: string; name: string; description: string } {
  return { id: c.id, name: c.name, description: c.description };
}

interface JsonCategoryResult {
  id: string;
  name: string;
  bytes: number;
  files: number;
  skipped?: boolean;
  skippedReason?: string;
  errors?: string[];
}

interface JsonScanResult {
  categories: JsonCategoryResult[];
  totalBytes: number;
  totalFiles: number;
  durationSeconds: number;
}

interface JsonCleanResult {
  categories: JsonCategoryResult[];
  scannedBytes: number;
  scannedFiles: number;
  cleanedBytes: number;
  cleanedFiles: number;
  errors?: string[];
  durationSeconds: number;
}

function formatBytes(bytes: number): string {
  if (bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const k = 1024;
  const i = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(k)));
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${units[i]}`;
}

/** Человекочитаемый лог для панели "Лог операций" — строится из структурированных данных, а не из сырого текста CLI. */
function buildScanLogText(parsed: JsonScanResult): string {
  const lines = parsed.categories
    .filter((c) => c.bytes > 0 || c.files > 0)
    .map((c) => `  ${c.name}: ${formatBytes(c.bytes)} (${c.files} файлов)`);
  lines.push('');
  lines.push(`Найдено: ${formatBytes(parsed.totalBytes)} в ${parsed.totalFiles} файлах.`);
  return lines.join('\n');
}

function buildCleanLogText(parsed: JsonCleanResult): string {
  const lines = parsed.categories
    .filter((c) => c.bytes > 0 || c.files > 0)
    .map((c) => `  ${c.name}: ${formatBytes(c.bytes)} (${c.files} файлов)`);
  lines.push('');
  lines.push(`Готово. Освобождено: ${formatBytes(parsed.cleanedBytes)} в ${parsed.cleanedFiles} файлах.`);
  if (parsed.errors?.length) {
    lines.push(`Ошибок/пропусков: ${parsed.errors.length}`);
  }
  return lines.join('\n');
}
