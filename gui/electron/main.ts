/**
 * Electron Main Process
 * TypeScript implementation of main process
 */
import { app, BrowserWindow, ipcMain, IpcMainInvokeEvent } from 'electron';
import path from 'path';
import { spawn, SpawnOptionsWithoutStdio } from 'child_process';
import fs from 'fs';

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
 * Execute CLI command and return output
 */
function executeCli(args: string[]): Promise<{ stdout: string; stderr: string; code: number }> {
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
    let stderr = '';

    child.stdout?.on('data', (data: Buffer) => {
      stdout += data.toString();
    });

    child.stderr?.on('data', (data: Buffer) => {
      stderr += data.toString();
    });

    child.on('close', (code: number | null) => {
      resolve({
        stdout,
        stderr,
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

  const { stdout } = await executeCli(args);
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

  const { stdout, stderr, code } = await executeCli(args);
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

// Leftovers IPC Handler (enhanced with orphan DB)
ipcMain.handle('get-leftovers', async () => {
  const { stdout } = await executeCli(['--leftovers']);
  return stdout;
});

// Leftovers Extended: includes orphan DB cross-referencing + installed programs
ipcMain.handle('get-leftovers-ex', async (_event: IpcMainInvokeEvent, options?: { logFile?: string }) => {
  const args = ['--leftovers'];
  if (options?.logFile) args.push('--leftovers-log', options.logFile);
  const { stdout, stderr, code } = await executeCli(args);
  return { output: stdout, error: stderr, code };
});

// Duplicates IPC Handler
ipcMain.handle('get-duplicates', async (_event: IpcMainInvokeEvent, rootPaths: string) => {
  const { stdout } = await executeCli(['--duplicates', rootPaths]);
  return stdout;
});

// Empty Dirs IPC Handler
ipcMain.handle('get-empty-dirs', async (_event: IpcMainInvokeEvent, rootPaths: string) => {
  const { stdout } = await executeCli(['--empty-dirs', rootPaths]);
  return stdout;
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

// ─── OrphanCleaner IPC Handlers ───

// Orphan Scan: check orphaned_apps.json entries
ipcMain.handle('orphan-scan', async (_event: IpcMainInvokeEvent, configPath?: string) => {
  const args = ['--orphan-scan'];
  if (configPath) args.push('--orphan-config', configPath);
  const { stdout, stderr, code } = await executeCli(args);
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
