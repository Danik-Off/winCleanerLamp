/**
 * Dependency Injection Container
 * Manual DI without external libraries
 */
import { ElectronCleanerService } from '@infrastructure/adapters/ElectronCleanerService';
import { ElectronSystemInfoService } from '@infrastructure/adapters/ElectronSystemInfoService';
import { ElectronLeftoverService } from '@infrastructure/adapters/ElectronLeftoverService';

import { ScanUseCase } from '@application/useCases/ScanUseCase';
import { CleanUseCase } from '@application/useCases/CleanUseCase';
import { GetCategoriesUseCase } from '@application/useCases/GetCategoriesUseCase';
import { GetLeftoversUseCase } from '@application/useCases/GetLeftoversUseCase';
import { GetSystemInfoUseCase } from '@application/useCases/GetSystemInfoUseCase';

// Infrastructure Singletons
const cleanerService = new ElectronCleanerService();
const systemInfoService = new ElectronSystemInfoService();
const leftoverService = new ElectronLeftoverService();

// Use Case Instances
export const scanUseCase = new ScanUseCase(cleanerService);
export const cleanUseCase = new CleanUseCase(cleanerService);
export const getCategoriesUseCase = new GetCategoriesUseCase(cleanerService);
export const getLeftoversUseCase = new GetLeftoversUseCase(leftoverService);
export const getSystemInfoUseCase = new GetSystemInfoUseCase(systemInfoService);

// Services Export
export { cleanerService, systemInfoService, leftoverService };
