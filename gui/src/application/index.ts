/**
 * Application Layer Exports
 * Use cases and orchestration
 */

export { ScanUseCase, type ScanRequest, type ScanResponse } from './useCases/ScanUseCase';
export { CleanUseCase, type CleanRequest, type CleanResponse } from './useCases/CleanUseCase';
export { GetCategoriesUseCase, type CategoriesResponse } from './useCases/GetCategoriesUseCase';
export { GetLeftoversUseCase } from './useCases/GetLeftoversUseCase';
export { GetSystemInfoUseCase } from './useCases/GetSystemInfoUseCase';

export type {
  ICleanerService,
  ScanOptions,
  CleanOptions,
  ISystemInfoService,
  ILeftoverService
} from './ports';
