/**
 * Domain Layer Exports
 * Core business logic and entities
 */

export { Category, CategorySelection } from './entities/Category';
export { ScanResult, ScanSummary } from './entities/ScanResult';
export { LeftoverItem, LeftoverSummary } from './entities/LeftoverItem';
export type { LeftoverType } from './entities/LeftoverItem';
export { SystemFileInfo, SystemInfoSummary } from './entities/SystemInfo';
export { LogEntry, LogLevel, OperationLog } from './entities/OperationLog';

export type { Category as ICategory } from './entities/Category';
export type { ScanResult as IScanResult } from './entities/ScanResult';
export type { LeftoverItem as ILeftoverItem } from './entities/LeftoverItem';
export type { SystemFileInfo as ISystemFileInfo } from './entities/SystemInfo';
export type { LogEntry as ILogEntry } from './entities/OperationLog';
