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

interface ElectronAPI {
  getCategories: () => Promise<CategoriesResponseDto>;
  scan: (options: ScanOptionsDto) => Promise<ScanResultDto>;
  clean: (options: CleanOptionsDto) => Promise<CleanResultDto>;
  getSysInfo: () => Promise<string>;
  getLeftovers: () => Promise<string>;
  onScanProgress: (callback: (data: string) => void) => void;
  onCleanProgress: (callback: (data: string) => void) => void;
  removeAllListeners: (channel: string) => void;
}

interface Window {
  electronAPI: ElectronAPI;
}
