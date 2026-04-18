/**
 * Application Use Case: ScanUseCase
 * Orchestrates the scanning operation
 */
import { ScanSummary, CategorySelection, OperationLog, LogEntry } from '@domain/index';
import type { ICleanerService } from '../ports/ICleanerService';

export interface ScanRequest {
  aggressive: boolean;
  selection: CategorySelection;
}

export interface ScanResponse {
  summary: ScanSummary;
  log: OperationLog;
}

export class ScanUseCase {
  constructor(
    private readonly cleanerService: ICleanerService
  ) {}

  async execute(request: ScanRequest): Promise<ScanResponse> {
    const startTime = Date.now();
    let log = OperationLog.empty();

    log = log.add(LogEntry.info(`Начало сканирования: ${request.selection.count} категорий`));

    try {
      const summary = await this.cleanerService.scan({
        aggressive: request.aggressive,
        selection: request.selection
      });

      const duration = Date.now() - startTime;
      log = log.add(LogEntry.success(`Сканирование завершено за ${duration}ms`));
      log = log.add(LogEntry.info(`Найдено: ${summary.totalFormatted} в ${summary.totalFiles} файлах`));

      return { summary, log };
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : 'Unknown error';
      log = log.add(LogEntry.error(`Ошибка сканирования: ${errorMsg}`, error));
      throw error;
    }
  }
}
