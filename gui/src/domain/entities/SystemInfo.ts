/**
 * Domain Entity: SystemFileInfo
 * Information about a large system file that cannot be auto-cleaned
 */
export class SystemFileInfo {
  constructor(
    public readonly name: string,
    public readonly path: string,
    public readonly sizeBytes: number,
    public readonly hint: string
  ) {}

  get sizeFormatted(): string {
    if (this.sizeBytes === -1) return '~большой';
    return formatBytes(this.sizeBytes);
  }

  get exists(): boolean {
    return this.sizeBytes > 0 || this.sizeBytes === -1;
  }

  get sizeUnknown(): boolean {
    return this.sizeBytes === -1;
  }

  static create(
    name: string,
    path: string,
    sizeBytes: number,
    hint: string
  ): SystemFileInfo {
    return new SystemFileInfo(name, path, sizeBytes, hint);
  }
}

/**
 * Domain Entity: SystemInfoSummary
 */
export class SystemInfoSummary {
  constructor(
    public readonly files: ReadonlyArray<SystemFileInfo>
  ) {}

  get totalBytes(): number {
    return this.files.reduce((sum, f) => sum + Math.max(0, f.sizeBytes), 0);
  }

  get existingFiles(): SystemFileInfo[] {
    return this.files.filter(f => f.exists);
  }

  get totalFormatted(): string {
    return formatBytes(this.totalBytes);
  }

  getFileByName(name: string): SystemFileInfo | undefined {
    return this.files.find(f => f.name === name);
  }

  getLargestFile(): SystemFileInfo | undefined {
    return this.existingFiles.sort((a, b) => b.sizeBytes - a.sizeBytes)[0];
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}
