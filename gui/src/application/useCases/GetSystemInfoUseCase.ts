/**
 * Application Use Case: GetSystemInfoUseCase
 * Retrieves system file information
 */
import { SystemInfoSummary } from '@domain/index';
import type { ISystemInfoService } from '../ports/ISystemInfoService';

export class GetSystemInfoUseCase {
  constructor(
    private readonly systemInfoService: ISystemInfoService
  ) {}

  async execute(): Promise<SystemInfoSummary> {
    return await this.systemInfoService.getSystemInfo();
  }
}
