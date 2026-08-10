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
  bytesCleaned: number;
  filesCleaned: number;
  errorCount: number;
}

interface DeleteResultDto {
  success: boolean;
  error?: string;
  movedToRecycleBin?: boolean;
}

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

interface ElectronAPI {
  getCategories: () => Promise<CategoriesResponseDto>;
  scan: (options: ScanOptionsDto) => Promise<ScanResultDto>;
  clean: (options: CleanOptionsDto) => Promise<CleanResultDto>;
  getSysInfo: () => Promise<string>;
  getLeftovers: () => Promise<string>;
  deleteLeftover: (folderPath: string) => Promise<DeleteResultDto>;
  getDuplicates: (rootPaths: string) => Promise<DuplicatesResultDto>;
  getEmptyDirs: (rootPaths: string) => Promise<EmptyDirsResultDto>;
  deleteEmptyDir: (dirPath: string) => Promise<DeleteResultDto>;
  deleteFile: (filePath: string) => Promise<DeleteResultDto>;
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

interface Window {
  electronAPI: ElectronAPI;
}
