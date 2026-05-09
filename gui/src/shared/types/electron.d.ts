/**
 * Shared Type Definitions for Electron API
 * This is a global ambient declaration file (non-module).
 * Types are available globally without imports.
 */

interface CategoryDto {
  id: string;
  name: string;
  description: string;
}

interface CategoriesResponseDto {
  safe: CategoryDto[];
  aggressive: CategoryDto[];
}

interface ScanOptionsDto {
  aggressive: boolean;
  categories?: string[];
}

interface ScanResultDto {
  output: string;
  parsed: {
    categories: Array<{
      id: string;
      name: string;
      size: string;
      sizeBytes: number;
      files: number;
    }>;
    totalBytes: number;
    totalFiles: number;
  };
  code: number;
}

interface CleanOptionsDto {
  aggressive: boolean;
  categories?: string[];
  yes: boolean;
}

interface CleanResultDto {
  output: string;
  error: string;
  code: number;
}

interface DeleteResultDto {
  success: boolean;
  error?: string;
  movedToRecycleBin?: boolean;
}

interface ElectronAPI {
  getCategories: () => Promise<CategoriesResponseDto>;
  scan: (options: ScanOptionsDto) => Promise<ScanResultDto>;
  clean: (options: CleanOptionsDto) => Promise<CleanResultDto>;
  getSysInfo: () => Promise<string>;
  getLeftovers: () => Promise<string>;
  deleteLeftover: (folderPath: string) => Promise<DeleteResultDto>;
  getDuplicates: (rootPaths: string) => Promise<string>;
  getEmptyDirs: (rootPaths: string) => Promise<string>;
  deleteEmptyDir: (dirPath: string) => Promise<DeleteResultDto>;
  deleteFile: (filePath: string) => Promise<DeleteResultDto>;
  openExternal: (path: string) => void;
  onScanProgress: (callback: (data: string) => void) => void;
  onCleanProgress: (callback: (data: string) => void) => void;
  removeAllListeners: (channel: string) => void;
  orphanScan: (configPath?: string) => Promise<{ output: string; error: string; code: number }>;
  orphanDiscover: (options?: { roots?: string }) => Promise<{ output: string; error: string; code: number }>;
  orphanClean: (options: { names: string; recycle?: boolean; cacheOnly?: boolean }) => Promise<{ output: string; error: string; code: number }>;
  orphanInfo: (displayName: string) => Promise<{ output: string; error: string; code: number }>;
  orphanList: (configPath?: string) => Promise<{ output: string; error: string; code: number }>;
  windowMinimize: () => void;
  windowMaximize: () => void;
  windowClose: () => void;
  windowIsMaximized: () => Promise<boolean>;
}

interface Window {
  electronAPI: ElectronAPI;
}
