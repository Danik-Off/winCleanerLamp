"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * Electron Main Process
 * TypeScript implementation of main process
 */
const electron_1 = require("electron");
const path_1 = __importDefault(require("path"));
const child_process_1 = require("child_process");
const fs_1 = __importDefault(require("fs"));
// Constants
const EXE_NAME = 'wincleanerlamp.exe';
const DEV_PORT = 3000;
/**
 * Get path to wincleanerlamp.exe
 */
function getExePath() {
    const devPath = path_1.default.join(__dirname, '..', '..', EXE_NAME);
    const prodPath = path_1.default.join(process.resourcesPath, EXE_NAME);
    return fs_1.default.existsSync(devPath) ? devPath : prodPath;
}
/**
 * Main Window reference
 */
let mainWindow = null;
/**
 * Create the main application window
 */
function createWindow() {
    mainWindow = new electron_1.BrowserWindow({
        width: 1200,
        height: 800,
        minWidth: 900,
        minHeight: 600,
        frame: false,
        titleBarStyle: 'hidden',
        webPreferences: {
            nodeIntegration: false,
            contextIsolation: true,
            preload: path_1.default.join(__dirname, 'preload.js'),
        },
        title: 'WinCleanerLamp GUI',
        show: false, // Show when ready
    });
    // Load the appropriate URL
    if (process.env.NODE_ENV === 'development') {
        mainWindow.loadURL(`http://localhost:${DEV_PORT}`);
        mainWindow.webContents.openDevTools();
    }
    else {
        mainWindow.loadFile(path_1.default.join(__dirname, '../dist/index.html'));
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
    electron_1.app.whenReady().then(() => {
        createWindow();
        electron_1.app.on('activate', () => {
            if (electron_1.BrowserWindow.getAllWindows().length === 0) {
                createWindow();
            }
        });
    });
    electron_1.app.on('window-all-closed', () => {
        if (process.platform !== 'darwin') {
            electron_1.app.quit();
        }
    });
}
catch (error) {
    console.error('Failed to start application:', error);
    electron_1.app.quit();
}
// IPC Handlers
/**
 * Execute CLI command and return output
 */
function executeCli(args) {
    return new Promise((resolve, reject) => {
        const exePath = getExePath();
        if (!fs_1.default.existsSync(exePath)) {
            reject(new Error(`Executable not found: ${exePath}`));
            return;
        }
        const options = {
            cwd: path_1.default.dirname(exePath),
        };
        const child = (0, child_process_1.spawn)(exePath, args, options);
        let stdout = '';
        let stderr = '';
        child.stdout?.on('data', (data) => {
            stdout += data.toString();
        });
        child.stderr?.on('data', (data) => {
            stderr += data.toString();
        });
        child.on('close', (code) => {
            resolve({
                stdout,
                stderr,
                code: code ?? -1,
            });
        });
        child.on('error', (error) => {
            reject(error);
        });
    });
}
// Get Categories IPC Handler
electron_1.ipcMain.handle('get-categories', async () => {
    const { stdout } = await executeCli(['--list']);
    return parseCategories(stdout);
});
// Scan IPC Handler
electron_1.ipcMain.handle('scan', async (event, options) => {
    const args = ['--scan'];
    if (options.aggressive)
        args.push('--aggressive');
    if (options.categories?.length)
        args.push('--categories', options.categories.join(','));
    const { stdout } = await executeCli(args);
    return {
        output: stdout,
        parsed: parseScanOutput(stdout),
        code: 0,
    };
});
// Clean IPC Handler
electron_1.ipcMain.handle('clean', async (event, options) => {
    const args = ['--clean'];
    if (options.aggressive)
        args.push('--aggressive');
    if (options.yes)
        args.push('--yes');
    if (options.categories?.length)
        args.push('--categories', options.categories.join(','));
    const { stdout, stderr, code } = await executeCli(args);
    return {
        output: stdout,
        error: stderr,
        code,
    };
});
// System Info IPC Handler
electron_1.ipcMain.handle('get-sysinfo', async () => {
    const { stdout } = await executeCli(['--sysinfo']);
    return stdout;
});
// Leftovers IPC Handler
electron_1.ipcMain.handle('get-leftovers', async () => {
    const { stdout } = await executeCli(['--leftovers']);
    return stdout;
});
// Duplicates IPC Handler
electron_1.ipcMain.handle('get-duplicates', async (_event, rootPaths) => {
    const { stdout } = await executeCli(['--duplicates', rootPaths]);
    return stdout;
});
// Empty Dirs IPC Handler
electron_1.ipcMain.handle('get-empty-dirs', async (_event, rootPaths) => {
    const { stdout } = await executeCli(['--empty-dirs', rootPaths]);
    return stdout;
});
// Delete Empty Dir IPC Handler
electron_1.ipcMain.handle('delete-empty-dir', async (_event, dirPath) => {
    try {
        if (!fs_1.default.existsSync(dirPath)) {
            return { success: false, error: 'Папка не найдена' };
        }
        const stat = fs_1.default.statSync(dirPath);
        if (!stat.isDirectory()) {
            return { success: false, error: 'Путь не является папкой' };
        }
        fs_1.default.rmSync(dirPath, { recursive: true, force: true });
        return { success: true };
    }
    catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        return { success: false, error: message };
    }
});
// Delete File IPC Handler (for duplicates)
electron_1.ipcMain.handle('delete-file', async (_event, filePath) => {
    try {
        if (!fs_1.default.existsSync(filePath)) {
            return { success: false, error: 'Файл не найден' };
        }
        fs_1.default.unlinkSync(filePath);
        return { success: true };
    }
    catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        return { success: false, error: message };
    }
});
// Delete Leftover Folder IPC Handler
electron_1.ipcMain.handle('delete-leftover', async (_event, folderPath) => {
    try {
        if (!fs_1.default.existsSync(folderPath)) {
            return { success: false, error: 'Папка не найдена' };
        }
        const stat = fs_1.default.statSync(folderPath);
        if (!stat.isDirectory()) {
            return { success: false, error: 'Путь не является папкой' };
        }
        fs_1.default.rmSync(folderPath, { recursive: true, force: true });
        return { success: true };
    }
    catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        return { success: false, error: message };
    }
});
// Window Control IPC Handlers
electron_1.ipcMain.on('window-minimize', () => {
    mainWindow?.minimize();
});
electron_1.ipcMain.on('window-maximize', () => {
    if (mainWindow?.isMaximized()) {
        mainWindow.unmaximize();
    }
    else {
        mainWindow?.maximize();
    }
});
electron_1.ipcMain.on('window-close', () => {
    mainWindow?.close();
});
electron_1.ipcMain.handle('window-is-maximized', () => {
    return mainWindow?.isMaximized() ?? false;
});
function parseCategories(output) {
    const lines = output.split('\n');
    const safe = [];
    const aggressive = [];
    let currentSection = null;
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
            const category = { id, name, description };
            if (currentSection === 'safe') {
                safe.push(category);
            }
            else {
                aggressive.push(category);
            }
        }
    }
    return { safe, aggressive };
}
function parseScanOutput(output) {
    const lines = output.split('\n');
    const categories = [];
    let totalBytes = 0;
    let totalFiles = 0;
    for (const line of lines) {
        // Parse table rows: id name size files
        // Size format: "1.12 GB", "392.72 MB", "0 B", or "-"
        const match = line.match(/^\s+(\S+)\s+(.+?)\s+([\d.]+\s*[KMGT]?B|-)\s+(\d+)\s*$/);
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
        // Parse total line: "ИТОГО: 1.85 GB в 57212 файлах"
        const totalMatch = line.match(/ИТОГО:\s+([\d.]+\s*[KMGT]?B)\s+в\s+(\d+)\s+файла/);
        if (totalMatch) {
            totalBytes = parseSize(totalMatch[1]);
            totalFiles = parseInt(totalMatch[2], 10);
        }
    }
    return { categories, totalBytes, totalFiles };
}
function parseSize(sizeStr) {
    const units = {
        B: 1,
        KB: 1024,
        MB: 1024 ** 2,
        GB: 1024 ** 3,
        TB: 1024 ** 4,
    };
    const match = sizeStr.match(/^([\d.]+)\s*(B|KB|MB|GB|TB)?$/);
    if (!match)
        return 0;
    const [, num, unit] = match;
    return parseFloat(num) * (unit ? units[unit] : 1);
}
//# sourceMappingURL=main.js.map