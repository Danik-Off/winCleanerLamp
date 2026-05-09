/**
 * Electron Main Process
 * TypeScript implementation of main process
 */
import { app, BrowserWindow, ipcMain, IpcMainInvokeEvent, shell } from 'electron';
import path from 'path';
import { spawn, SpawnOptionsWithoutStdio } from 'child_process';
import fs from 'fs';

// Constants
const EXE_NAME = 'wincleanerlamp.exe';
const DEV_PORT = 3000;

/**
 * Путь к wincleanerlamp.exe:
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

// Get Categories IPC Handler
ipcMain.handle('get-categories', async () => {
  const { stdout } = await executeCli(['--list']);
  return parseCategories(stdout);
});

// Scan IPC Handler
ipcMain.handle('scan', async (event: IpcMainInvokeEvent, options: { aggressive: boolean; categories?: string[] }) => {
  const args = ['--scan'];
  if (options.aggressive) args.push('--aggressive');
  if (options.categories?.length) args.push('--categories', options.categories.join(','));

  const { stdout } = await executeCli(args);
  const parsed = parseScanOutput(stdout);
  
  // Debug logging
  console.log('=== SCAN OUTPUT ===');
  console.log('Categories found:', parsed.categories.length);
  console.log('Total bytes:', parsed.totalBytes);
  console.log('Total files:', parsed.totalFiles);
  if (parsed.categories.length > 0) {
    console.log('First category:', JSON.stringify(parsed.categories[0]));
  }
  
  const result = {
    output: stdout,
    parsed,
    code: 0,
  };
  
  console.log('Returning result:', JSON.stringify(result.parsed));
  return result;
});

// Clean IPC Handler
ipcMain.handle('clean', async (event: IpcMainInvokeEvent, options: { aggressive: boolean; categories?: string[]; yes: boolean }) => {
  const args = ['--clean'];
  if (options.aggressive) args.push('--aggressive');
  if (options.yes) args.push('--yes');
  if (options.categories?.length) args.push('--categories', options.categories.join(','));

  const { stdout, stderr, code } = await executeCli(args);
  return {
    output: stdout,
    error: stderr,
    code,
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

// Delete Empty Dir IPC Handler
ipcMain.handle('delete-empty-dir', async (_event: IpcMainInvokeEvent, dirPath: string) => {
  try {
    if (!fs.existsSync(dirPath)) {
      return { success: false, error: 'Папка не найдена' };
    }
    const stat = fs.statSync(dirPath);
    if (!stat.isDirectory()) {
      return { success: false, error: 'Путь не является папкой' };
    }

    // Проверка безопасности: запрещаем удаление системных директорий
    const absPath = path.resolve(dirPath);
    const lowerPath = absPath.toLowerCase();

    const forbiddenPaths = [
      'c:\\windows',
      'c:\\program files',
      'c:\\program files (x86)',
      'c:\\programdata',
      'c:\\users\\all users',
      'c:\\users\\default',
      'c:\\users\\public',
      'c:\\perflogs',
    ];

    for (const forbidden of forbiddenPaths) {
      if (lowerPath.startsWith(forbidden)) {
        return { 
          success: false, 
          error: 'Удаление этой папки небезопасно (системная директория)' 
        };
      }
    }

    // Запрещаем удаление корня диска
    if (absPath.length <= 3) {
      return { success: false, error: 'Нельзя удалить корень диска' };
    }

    // Перемещаем в Корзину через PowerShell
    const psScript = `
      $shell = New-Object -ComObject Shell.Application
      $folder = $shell.Namespace(0)
      $folder.ParseName("${absPath.replace(/"/g, '\"')}").MoveToHere()
    `;

    const { execSync } = await import('child_process');
    execSync(`powershell -NoProfile -WindowStyle Hidden -Command "${psScript.replace(/\r?\n/g, ' ')}"`, {
      stdio: 'pipe',
      timeout: 10000,
    });

    return { success: true, movedToRecycleBin: true };
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return { success: false, error: `Не удалось переместить в корзину: ${message}` };
  }
});

// Delete File IPC Handler (for duplicates)
ipcMain.handle('delete-file', async (_event: IpcMainInvokeEvent, filePath: string) => {
  try {
    if (!fs.existsSync(filePath)) {
      return { success: false, error: 'Файл не найден' };
    }
    fs.unlinkSync(filePath);
    return { success: true };
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return { success: false, error: message };
  }
});

// Delete Leftover Folder IPC Handler
ipcMain.handle('delete-leftover', async (_event: IpcMainInvokeEvent, folderPath: string) => {
  try {
    if (!fs.existsSync(folderPath)) {
      return { success: false, error: 'Папка не найдена' };
    }
    const stat = fs.statSync(folderPath);
    if (!stat.isDirectory()) {
      return { success: false, error: 'Путь не является папкой' };
    }
    fs.rmSync(folderPath, { recursive: true, force: true });
    return { success: true };
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return { success: false, error: message };
  }
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
ipcMain.handle('orphan-clean', async (_event: IpcMainInvokeEvent, options: { names: string; recycle?: boolean; cacheOnly?: boolean }) => {
  const args = ['--orphan-clean', options.names];
  if (options.recycle) args.push('--orphan-recycle');
  if (options.cacheOnly) args.push('--orphan-cache-only');
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

// Category Parser
interface CategoryDto {
  id: string;
  name: string;
  description: string;
}

interface CategoriesResult {
  safe: CategoryDto[];
  aggressive: CategoryDto[];
}

function parseCategories(output: string): CategoriesResult {
  const lines = output.split('\n');
  const safe: CategoryDto[] = [];
  const aggressive: CategoryDto[] = [];
  let currentSection: 'safe' | 'aggressive' | null = null;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    if (line.includes('безопасные по умолчанию')) {
      currentSection = 'safe';
      continue;
    }
    if (line.includes('Агрессивные категории')) {
      currentSection = 'aggressive';
      continue;
    }

    const match = line.match(/^\s{2}(\S+)\s*$/);
    if (match && currentSection) {
      const id = match[1];
      const name = lines[i + 1]?.trim() ?? '';
      const description = lines[i + 2]?.trim() ?? '';

      const category: CategoryDto = { id, name, description };

      if (currentSection === 'safe') {
        safe.push(category);
      } else {
        aggressive.push(category);
      }
    }
  }

  return { safe, aggressive };
}

// Scan Output Parser
interface ScanResultDto {
  id: string;
  name: string;
  size: string;
  sizeBytes: number;
  files: number;
}

interface ScanParsedResult {
  categories: ScanResultDto[];
  totalBytes: number;
  totalFiles: number;
}

function parseScanOutput(output: string): ScanParsedResult {
  const lines = output.split('\n');
  const categories: ScanResultDto[] = [];
  let totalBytes = 0;
  let totalFiles = 0;

  for (const line of lines) {
    // Parse table rows from CLI:
    // Format: "  {id}  {name}  {size}  {files}"
    // Example: "  event-logs  Журналы событий Windows (*.evtx)   346.12 MB     400"
    // Use \s{2,} for 2 or more spaces (variable column spacing)
    const match = line.match(/^\s{2}(\S+)\s{2,}(.+?)\s{2,}([\d.,]+\s*[KMGT]?B|-)\s{2,}(\d+)\s*$/);
    if (match) {
      const [, id, name, sizeStr, files] = match;
      const sizeBytes = parseSize(sizeStr.trim());
      categories.push({
        id: id.trim(),
        name: name.trim(),
        size: sizeStr.trim(),
        sizeBytes,
        files: parseInt(files, 10),
      });
      totalBytes += sizeBytes;
      totalFiles += parseInt(files, 10);
    }

    // Parse total line: "  ИТОГО: 425.83 MB в 1494 файлах  (скрыто...)"
    const totalMatch = line.match(/ИТОГО:\s+([\d.,]+\s*[KMGT]?B|-)\s+в\s+(\d+)\s+файла/);
    if (totalMatch) {
      totalBytes = parseSize(totalMatch[1]);
      totalFiles = parseInt(totalMatch[2], 10);
    }
  }

  return { categories, totalBytes, totalFiles };
}

function parseSize(sizeStr: string): number {
  if (sizeStr === '-' || sizeStr === '') return 0;
  
  const units: Record<string, number> = {
    B: 1,
    KB: 1024,
    MB: 1024 ** 2,
    GB: 1024 ** 3,
    TB: 1024 ** 4,
  };
  
  // Handle formats like "346.12 MB", "491.13 KB", "735 B", "4 B"
  const match = sizeStr.match(/^([\d.,]+)\s*(B|KB|MB|GB|TB)?$/i);
  if (!match) return 0;
  
  const [, numStr, unit] = match;
  // Replace comma with dot for European format if needed
  const num = parseFloat(numStr.replace(',', '.'));
  return num * (unit ? units[unit.toUpperCase()] || 1 : 1);
}
