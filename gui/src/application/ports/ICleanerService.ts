/**
 * Application Port: ICleanerService
 * Defines the interface for cleaning operations
 */
import { ScanSummary, Category, CategorySelection, OperationLog } from '@domain/index';

export interface ScanOptions {
  readonly aggressive: boolean;
  readonly selection: CategorySelection;
}

export interface CleanOptions {
  readonly aggressive: boolean;
  readonly selection: CategorySelection;
  readonly skipConfirmation: boolean;
}

export interface ICleanerService {
  /**
   * Scan for junk files
   */
  scan(options: ScanOptions): Promise<ScanSummary>;

  /**
   * Clean junk files
   */
  clean(options: CleanOptions): Promise<OperationLog>;

  /**
   * Get available categories
   */
  getCategories(): Promise<{
    safe: Category[];
    aggressive: Category[];
  }>;

  /**
   * Subscribe to scan progress
   */
  onScanProgress(callback: (log: string) => void): void;

  /**
   * Subscribe to clean progress
   */
  onCleanProgress(callback: (log: string) => void): void;

  /**
   * Unsubscribe from progress events
   */
  removeAllListeners(): void;
}
