/**
 * Application Port: ISystemInfoService
 * Defines the interface for system information retrieval
 */
import { SystemInfoSummary } from '@domain/index';

export interface ISystemInfoService {
  /**
   * Get system file information
   */
  getSystemInfo(): Promise<SystemInfoSummary>;
}
