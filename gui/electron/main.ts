/**
 * Electron Main Process
 * TypeScript implementation of main process
 */
import { app, BrowserWindow, ipcMain, IpcMainInvokeEvent } from 'electron';
import path from 'path';
import { spawn, SpawnOptionsWithoutStdio } from 'child_process';
import fs from 'fs';

// Constants
const EXE_NAME = 'wincleanerlamp.exe';
const DEV_PORT = 3000;

/**
 * Get path to wincleanerlamp.exe
 */
function getExePath(): string {
  const devPath = path.join(__dirname, '..', '..', EXE_NAME);
  const prodPath = path.join(process.resourcesPath, EXE_NAME);
  return fs.existsSync(devPath) ? devPath : prodPath;
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
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      preload: path.join(__dirname, 'preload.js'),
    },
    title: 'WinCleanerLamp GUI',
    show: false, // Show when ready
  });

  // Load the appropriate URL
  if (process.env.NODE_ENV === 'development') {
    mainWindow.loadURL(`http://localhost:${DEV_PORT}`);
    mainWindow.webContents.openDevTools();
  } else {
    mainWindow.loadFile(path.join(__dirname, '../dist/index.html'));
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
  return {
    output: stdout,
    parsed: parseScanOutput(stdout),
    code: 0,
  };
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

// Leftovers IPC Handler
ipcMain.handle('get-leftovers', async () => {
  const { stdout } = await executeCli(['--leftovers']);
  return stdout;
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
    // Parse table rows: id name size files
    const match = line.match(/^\s+(\S+)\s+(.+?)\s+(\S+)\s+(\d+)$/);
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

    // Parse total line
    const totalMatch = line.match(/ИТОГО:\s+(\S+)\s+в\s+(\d+)\s+файла/);
    if (totalMatch) {
      totalBytes = parseSize(totalMatch[1]);
      totalFiles = parseInt(totalMatch[2], 10);
    }
  }

  return { categories, totalBytes, totalFiles };
}

function parseSize(sizeStr: string): number {
  const units: Record<string, number> = {
    B: 1,
    KB: 1024,
    MB: 1024 ** 2,
    GB: 1024 ** 3,
    TB: 1024 ** 4,
  };
  const match = sizeStr.match(/^([\d.]+)\s*(B|KB|MB|GB|TB)?$/);
  if (!match) return 0;
  const [, num, unit] = match;
  return parseFloat(num) * (unit ? units[unit] : 1);
}
