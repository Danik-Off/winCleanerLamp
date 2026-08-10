/**
 * Electron Preload Script
 * Secure bridge between renderer and main process
 */
import { contextBridge, ipcRenderer, IpcRendererEvent } from 'electron';

interface AutostartEntryDto {
  id: string;
  source: string;
  name: string;
  command?: string;
  location: string;
  enabled: boolean;
  canToggle: boolean;
}

interface DuplicateGroupDto {
  size: number;
  sizeFormatted: string;
  waste: number;
  wasteFormatted: string;
  paths: string[];
  riskFlag?: string;
}
interface DuplicatesResultDto {
  groups: DuplicateGroupDto[];
  scannedFiles: number;
  totalWaste: number;
  skippedRoots: string[];
  error?: string;
}
interface EmptyDirsResultDto {
  dirs: string[];
  error?: string;
}

interface UpdateInfoDto {
  version: string;
  releaseDate?: string;
  releaseNotes?: string;
}

interface BrokenShortcutDto {
  path: string;
  targetPath?: string;
  reason: string;
}
interface ShortcutsResultDto {
  broken: BrokenShortcutDto[];
  scanned: number;
  error?: string;
}

interface LargeFileDto {
  path: string;
  sizeBytes: number;
  sizeFormatted: string;
  modTime: string;
  inSystemDir: boolean;
}
interface LargeFilesResultDto {
  files: LargeFileDto[];
  scannedFiles: number;
  totalBytes: number;
  skippedRoots: string[];
  error?: string;
}

interface AuditExportResultDto {
  success: boolean;
  canceled?: boolean;
  path?: string;
  unknownCount?: number;
  installedCount?: number;
  error?: string;
}

interface ExportJsonResultDto {
  success: boolean;
  canceled?: boolean;
  path?: string;
  error?: string;
}

/**
 * ElectronAPI type (mirrored from src/shared/types/electron.d.ts)
 * Keep in sync with the shared type definition.
 */
interface ElectronAPI {
  getCategories: () => Promise<unknown>;
  scan: (options: { aggressive: boolean; categories?: string[] }) => Promise<unknown>;
  clean: (options: { aggressive: boolean; categories?: string[]; yes: boolean }) => Promise<unknown>;
  getSysInfo: () => Promise<string>;
  getLeftovers: () => Promise<string>;
  deleteLeftover: (folderPath: string) => Promise<{ success: boolean; error?: string }>;
  getDuplicates: (rootPaths: string) => Promise<DuplicatesResultDto>;
  getEmptyDirs: (rootPaths: string) => Promise<EmptyDirsResultDto>;
  deleteEmptyDir: (dirPath: string) => Promise<{ success: boolean; error?: string; movedToRecycleBin?: boolean }>;
  deleteFile: (filePath: string) => Promise<{ success: boolean; error?: string }>;
  /** Открывает локальный путь в системном обработчике (Проводник/приложение по умолчанию). Не для http(s)-ссылок. */
  openPath: (localPath: string) => void;
  onScanProgress: (callback: (data: string) => void) => void;
  onCleanProgress: (callback: (data: string) => void) => void;
  removeAllListeners: (channel: string) => void;
  orphanScan: (configPath?: string) => Promise<{ output: string; error: string; code: number }>;
  orphanDiscover: (options?: { roots?: string }) => Promise<{ output: string; error: string; code: number }>;
  orphanClean: (options: { names: string; recycle?: boolean; cacheOnly?: boolean; includeUserData?: boolean }) => Promise<{ output: string; error: string; code: number }>;
  orphanInfo: (displayName: string) => Promise<{ output: string; error: string; code: number }>;
  orphanList: (configPath?: string) => Promise<{ output: string; error: string; code: number }>;
  orphanTrack: (options: { path: string; name?: string; asCache?: boolean }) => Promise<{ success: boolean; error?: string }>;
  autostartList: () => Promise<{ entries: AutostartEntryDto[]; error: string }>;
  autostartToggle: (options: { id: string; enable: boolean }) => Promise<{ success: boolean; error?: string }>;
  getBrokenShortcuts: () => Promise<ShortcutsResultDto>;
  getLargeFiles: (options?: { roots?: string; minSizeMB?: number }) => Promise<LargeFilesResultDto>;
  launchUninstaller: (displayName: string) => Promise<{ success: boolean; error?: string }>;
  exportAudit: () => Promise<AuditExportResultDto>;
  exportJson: (options: { suggestedName?: string; data: unknown }) => Promise<ExportJsonResultDto>;
  checkForUpdates: () => Promise<{ success: boolean; error?: string }>;
  installUpdate: () => Promise<{ success: boolean; error?: string }>;
  onUpdateAvailable: (callback: (info: UpdateInfoDto) => void) => void;
  onUpdateNotAvailable: (callback: () => void) => void;
  onUpdateError: (callback: (message: string) => void) => void;
  onUpdateDownloadProgress: (callback: (percent: number) => void) => void;
  onUpdateDownloaded: (callback: () => void) => void;
  windowMinimize: () => void;
  windowMaximize: () => void;
  windowClose: () => void;
  windowIsMaximized: () => Promise<boolean>;
}

// Expose protected methods that allow the renderer process to use
// the ipcRenderer without exposing the entire object
const api: ElectronAPI = {
  getCategories: () => ipcRenderer.invoke('get-categories'),

  scan: (options) => ipcRenderer.invoke('scan', options),

  clean: (options) => ipcRenderer.invoke('clean', options),

  getSysInfo: () => ipcRenderer.invoke('get-sysinfo'),

  getLeftovers: () => ipcRenderer.invoke('get-leftovers'),

  deleteLeftover: (folderPath: string) => ipcRenderer.invoke('delete-leftover', folderPath),

  getDuplicates: (rootPaths: string) => ipcRenderer.invoke('get-duplicates', rootPaths),

  getEmptyDirs: (rootPaths: string) => ipcRenderer.invoke('get-empty-dirs', rootPaths),

  deleteEmptyDir: (dirPath: string) => ipcRenderer.invoke('delete-empty-dir', dirPath),

  deleteFile: (filePath: string) => ipcRenderer.invoke('delete-file', filePath),

  openPath: (localPath: string) => {
    const { shell } = require('electron');
    shell.openPath(localPath);
  },

  onScanProgress: (callback: (data: string) => void) => {
    ipcRenderer.on('scan-progress', (_event: IpcRendererEvent, data: string) => callback(data));
  },

  onCleanProgress: (callback: (data: string) => void) => {
    ipcRenderer.on('clean-progress', (_event: IpcRendererEvent, data: string) => callback(data));
  },

  removeAllListeners: (channel: string) => {
    ipcRenderer.removeAllListeners(channel);
  },

  orphanScan: (configPath?: string) => ipcRenderer.invoke('orphan-scan', configPath),
  orphanDiscover: (options?: { roots?: string }) => ipcRenderer.invoke('orphan-discover', options),
  orphanClean: (options: { names: string; recycle?: boolean; cacheOnly?: boolean; includeUserData?: boolean }) => ipcRenderer.invoke('orphan-clean', options),
  orphanInfo: (displayName: string) => ipcRenderer.invoke('orphan-info', displayName),
  orphanList: (configPath?: string) => ipcRenderer.invoke('orphan-list', configPath),
  orphanTrack: (options: { path: string; name?: string; asCache?: boolean }) => ipcRenderer.invoke('orphan-track', options),

  autostartList: () => ipcRenderer.invoke('autostart-list'),
  autostartToggle: (options: { id: string; enable: boolean }) => ipcRenderer.invoke('autostart-toggle', options),

  getBrokenShortcuts: () => ipcRenderer.invoke('get-broken-shortcuts'),
  getLargeFiles: (options?: { roots?: string; minSizeMB?: number }) => ipcRenderer.invoke('get-large-files', options),
  launchUninstaller: (displayName: string) => ipcRenderer.invoke('launch-uninstaller', displayName),
  exportAudit: () => ipcRenderer.invoke('export-audit'),
  exportJson: (options: { suggestedName?: string; data: unknown }) => ipcRenderer.invoke('export-json', options),

  checkForUpdates: () => ipcRenderer.invoke('check-for-updates'),
  installUpdate: () => ipcRenderer.invoke('install-update'),
  onUpdateAvailable: (callback: (info: UpdateInfoDto) => void) => {
    ipcRenderer.on('update-available', (_event: IpcRendererEvent, info: UpdateInfoDto) => callback(info));
  },
  onUpdateNotAvailable: (callback: () => void) => {
    ipcRenderer.on('update-not-available', () => callback());
  },
  onUpdateError: (callback: (message: string) => void) => {
    ipcRenderer.on('update-error', (_event: IpcRendererEvent, message: string) => callback(message));
  },
  onUpdateDownloadProgress: (callback: (percent: number) => void) => {
    ipcRenderer.on('update-download-progress', (_event: IpcRendererEvent, percent: number) => callback(percent));
  },
  onUpdateDownloaded: (callback: () => void) => {
    ipcRenderer.on('update-downloaded', () => callback());
  },

  windowMinimize: () => ipcRenderer.send('window-minimize'),
  windowMaximize: () => ipcRenderer.send('window-maximize'),
  windowClose: () => ipcRenderer.send('window-close'),
  windowIsMaximized: () => ipcRenderer.invoke('window-is-maximized'),
};

contextBridge.exposeInMainWorld('electronAPI', api);

console.log('Preload script loaded successfully');
