/**
 * Application Use Case: GetLeftoversUseCase
 * Scans for leftover files from uninstalled programs
 */
import { LeftoverSummary } from '@domain/index';
import type { ILeftoverService } from '../ports/ILeftoverService';

export class GetLeftoversUseCase {
  constructor(
    private readonly leftoverService: ILeftoverService
  ) {}

  async execute(): Promise<LeftoverSummary> {
    return await this.leftoverService.scanLeftovers();
  }
}
