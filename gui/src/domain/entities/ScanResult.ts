/**
 * Domain Entity: ScanResult
 * Represents the result of a junk scan operation
 */
export class ScanResult {
  constructor(
    public readonly categoryId: string,
    public readonly categoryName: string,
    public readonly sizeBytes: number,
    public readonly fileCount: number,
    public readonly isSkipped: boolean = false
  ) {}

  get sizeFormatted(): string {
    return formatBytes(this.sizeBytes);
  }

  get isEmpty(): boolean {
    return this.sizeBytes === 0 && this.fileCount === 0;
  }

  static createEmpty(categoryId: string, categoryName: string): ScanResult {
    return new ScanResult(categoryId, categoryName, 0, 0, false);
  }
}

/**
 * Domain Entity: ScanSummary
 * Aggregate of all scan results
 */
export class ScanSummary {
  constructor(
    public readonly results: ReadonlyArray<ScanResult>,
    public readonly durationMs: number
  ) {}

  get totalBytes(): number {
    return this.results.reduce((sum, r) => sum + r.sizeBytes, 0);
  }

  get totalFiles(): number {
    return this.results.reduce((sum, r) => sum + r.fileCount, 0);
  }

  get totalCategories(): number {
    return this.results.length;
  }

  get nonEmptyResults(): ScanResult[] {
    return this.results.filter(r => !r.isEmpty);
  }

  get totalFormatted(): string {
    return formatBytes(this.totalBytes);
  }

  getResultById(categoryId: string): ScanResult | undefined {
    return this.results.find(r => r.categoryId === categoryId);
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}
