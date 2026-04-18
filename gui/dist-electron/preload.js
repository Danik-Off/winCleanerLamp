"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * Electron Preload Script
 * Secure bridge between renderer and main process
 */
const electron_1 = require("electron");
// Expose protected methods that allow the renderer process to use
// the ipcRenderer without exposing the entire object
const api = {
    getCategories: () => electron_1.ipcRenderer.invoke('get-categories'),
    scan: (options) => electron_1.ipcRenderer.invoke('scan', options),
    clean: (options) => electron_1.ipcRenderer.invoke('clean', options),
    getSysInfo: () => electron_1.ipcRenderer.invoke('get-sysinfo'),
    getLeftovers: () => electron_1.ipcRenderer.invoke('get-leftovers'),
    deleteLeftover: (folderPath) => electron_1.ipcRenderer.invoke('delete-leftover', folderPath),
    getDuplicates: (rootPaths) => electron_1.ipcRenderer.invoke('get-duplicates', rootPaths),
    getEmptyDirs: (rootPaths) => electron_1.ipcRenderer.invoke('get-empty-dirs', rootPaths),
    deleteEmptyDir: (dirPath) => electron_1.ipcRenderer.invoke('delete-empty-dir', dirPath),
    deleteFile: (filePath) => electron_1.ipcRenderer.invoke('delete-file', filePath),
    onScanProgress: (callback) => {
        electron_1.ipcRenderer.on('scan-progress', (_event, data) => callback(data));
    },
    onCleanProgress: (callback) => {
        electron_1.ipcRenderer.on('clean-progress', (_event, data) => callback(data));
    },
    removeAllListeners: (channel) => {
        electron_1.ipcRenderer.removeAllListeners(channel);
    },
    windowMinimize: () => electron_1.ipcRenderer.send('window-minimize'),
    windowMaximize: () => electron_1.ipcRenderer.send('window-maximize'),
    windowClose: () => electron_1.ipcRenderer.send('window-close'),
    windowIsMaximized: () => electron_1.ipcRenderer.invoke('window-is-maximized'),
};
electron_1.contextBridge.exposeInMainWorld('electronAPI', api);
console.log('Preload script loaded successfully');
//# sourceMappingURL=preload.js.map