/**
 * Infrastructure Adapter: ElectronCleanerService
 * Implements ICleanerService using Electron IPC
 */
import type { ICleanerService, ScanOptions, CleanOptions } from '@application/ports/ICleanerService';
import { Category, ScanSummary, ScanResult, OperationLog } from '@domain/index';

export class ElectronCleanerService implements ICleanerService {
  async getCategories(): Promise<{ safe: Category[]; aggressive: Category[] }> {
    const result = await window.electronAPI.getCategories();
    
    return {
      safe: result.safe.map(c => Category.createSafe(c.id, c.name, c.description)),
      aggressive: result.aggressive.map(c => Category.createAggressive(c.id, c.name, c.description))
    };
  }

  async scan(options: ScanOptions): Promise<ScanSummary> {
    const result = await window.electronAPI.scan({
      aggressive: options.aggressive,
      categories: options.selection.getSelectedIds()
    });

    const scanResults = result.parsed.categories.map(c => 
      new ScanResult(c.id, c.name, c.sizeBytes, c.files)
    );

    return new ScanSummary(scanResults, 0);
  }

  async clean(options: CleanOptions): Promise<OperationLog> {
    const result = await window.electronAPI.clean({
      aggressive: options.aggressive,
      categories: options.selection.getSelectedIds(),
      yes: options.skipConfirmation
    });

    return OperationLog.empty().addRaw(result.output);
  }

  onScanProgress(callback: (log: string) => void): void {
    window.electronAPI.onScanProgress(callback);
  }

  onCleanProgress(callback: (log: string) => void): void {
    window.electronAPI.onCleanProgress(callback);
  }

  removeAllListeners(): void {
    window.electronAPI.removeAllListeners('scan-progress');
    window.electronAPI.removeAllListeners('clean-progress');
  }
}
