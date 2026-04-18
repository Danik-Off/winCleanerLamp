/**
 * Electron Preload Script
 * Secure bridge between renderer and main process
 */
import { contextBridge, ipcRenderer, IpcRendererEvent } from 'electron';

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
  getDuplicates: (rootPaths: string) => Promise<string>;
  getEmptyDirs: (rootPaths: string) => Promise<string>;
  deleteEmptyDir: (dirPath: string) => Promise<{ success: boolean; error?: string }>;
  deleteFile: (filePath: string) => Promise<{ success: boolean; error?: string }>;
  onScanProgress: (callback: (data: string) => void) => void;
  onCleanProgress: (callback: (data: string) => void) => void;
  removeAllListeners: (channel: string) => void;
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

  onScanProgress: (callback: (data: string) => void) => {
    ipcRenderer.on('scan-progress', (_event: IpcRendererEvent, data: string) => callback(data));
  },

  onCleanProgress: (callback: (data: string) => void) => {
    ipcRenderer.on('clean-progress', (_event: IpcRendererEvent, data: string) => callback(data));
  },

  removeAllListeners: (channel: string) => {
    ipcRenderer.removeAllListeners(channel);
  },

  windowMinimize: () => ipcRenderer.send('window-minimize'),
  windowMaximize: () => ipcRenderer.send('window-maximize'),
  windowClose: () => ipcRenderer.send('window-close'),
  windowIsMaximized: () => ipcRenderer.invoke('window-is-maximized'),
};

contextBridge.exposeInMainWorld('electronAPI', api);

console.log('Preload script loaded successfully');
