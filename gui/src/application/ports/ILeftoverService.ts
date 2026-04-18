/**
 * Application Port: ILeftoverService
 * Defines the interface for leftover file scanning
 */
import { LeftoverSummary } from '@domain/index';

export interface ILeftoverService {
  /**
   * Scan for leftover files from uninstalled programs
   */
  scanLeftovers(): Promise<LeftoverSummary>;
}
