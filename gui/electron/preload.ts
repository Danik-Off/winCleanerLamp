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
  onScanProgress: (callback: (data: string) => void) => void;
  onCleanProgress: (callback: (data: string) => void) => void;
  removeAllListeners: (channel: string) => void;
}

// Expose protected methods that allow the renderer process to use
// the ipcRenderer without exposing the entire object
const api: ElectronAPI = {
  getCategories: () => ipcRenderer.invoke('get-categories'),

  scan: (options) => ipcRenderer.invoke('scan', options),

  clean: (options) => ipcRenderer.invoke('clean', options),

  getSysInfo: () => ipcRenderer.invoke('get-sysinfo'),

  getLeftovers: () => ipcRenderer.invoke('get-leftovers'),

  onScanProgress: (callback: (data: string) => void) => {
    ipcRenderer.on('scan-progress', (_event: IpcRendererEvent, data: string) => callback(data));
  },

  onCleanProgress: (callback: (data: string) => void) => {
    ipcRenderer.on('clean-progress', (_event: IpcRendererEvent, data: string) => callback(data));
  },

  removeAllListeners: (channel: string) => {
    ipcRenderer.removeAllListeners(channel);
  },
};

contextBridge.exposeInMainWorld('electronAPI', api);

console.log('Preload script loaded successfully');
