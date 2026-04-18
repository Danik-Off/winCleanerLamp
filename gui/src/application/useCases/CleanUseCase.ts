/**
 * Application Use Case: CleanUseCase
 * Orchestrates the cleaning operation
 */
import { CategorySelection, OperationLog, LogEntry } from '@domain/index';
import type { ICleanerService } from '../ports/ICleanerService';

export interface CleanRequest {
  aggressive: boolean;
  selection: CategorySelection;
  skipConfirmation: boolean;
}

export interface CleanResponse {
  log: OperationLog;
  bytesCleaned: number;
  filesCleaned: number;
}

export class CleanUseCase {
  constructor(
    private readonly cleanerService: ICleanerService
  ) {}

  async execute(request: CleanRequest): Promise<CleanResponse> {
    let log = OperationLog.empty();
    
    log = log.add(LogEntry.info(`Начало очистки: ${request.selection.count} категорий`));
    
    if (request.aggressive) {
      log = log.add(LogEntry.warn('Включен агрессивный режим!'));
    }

    try {
      const resultLog = await this.cleanerService.clean({
        aggressive: request.aggressive,
        selection: request.selection,
        skipConfirmation: request.skipConfirmation
      });

      log = log.addRaw(resultLog.toString());
      log = log.add(LogEntry.success('Очистка завершена'));

      // Parse cleaned amounts from log
      const bytesMatch = resultLog.toString().match(/освобождено:\s+([\d.]+\s*\w+)/i);
      const filesMatch = resultLog.toString().match(/(\d+)\s+файл/);

      return {
        log,
        bytesCleaned: bytesMatch ? this.parseSize(bytesMatch[1]) : 0,
        filesCleaned: filesMatch ? parseInt(filesMatch[1], 10) : 0
      };
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : 'Unknown error';
      log = log.add(LogEntry.error(`Ошибка очистки: ${errorMsg}`, error));
      throw error;
    }
  }

  private parseSize(sizeStr: string): number {
    const units: Record<string, number> = { 'B': 1, 'KB': 1024, 'MB': 1024**2, 'GB': 1024**3 };
    const match = sizeStr.match(/^([\d.]+)\s*(\w+)$/);
    if (!match) return 0;
    const [, num, unit] = match;
    return parseFloat(num) * (units[unit.toUpperCase()] || 1);
  }
}
