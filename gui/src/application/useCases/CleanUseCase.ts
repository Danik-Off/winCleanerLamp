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
  errorCount: number;
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
      const result = await this.cleanerService.clean({
        aggressive: request.aggressive,
        selection: request.selection,
        skipConfirmation: request.skipConfirmation
      });

      log = log.addRaw(result.log.toString());
      log = log.add(LogEntry.success('Очистка завершена'));

      return {
        log,
        bytesCleaned: result.bytesCleaned,
        filesCleaned: result.filesCleaned,
        errorCount: result.errorCount
      };
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : 'Unknown error';
      log = log.add(LogEntry.error(`Ошибка очистки: ${errorMsg}`, error));
      throw error;
    }
  }
}
