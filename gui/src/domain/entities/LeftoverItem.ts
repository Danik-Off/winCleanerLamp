/**
 * Domain Entity: LeftoverItem
 * Represents a potentially orphaned folder/registry key from uninstalled software
 */
export type LeftoverType = 'folder' | 'registry' | 'empty';

export class LeftoverItem {
  constructor(
    public readonly path: string,
    public readonly sizeBytes: number,
    public readonly fileCount: number,
    public readonly reason: string,
    public readonly type: LeftoverType = 'folder'
  ) {}

  get sizeFormatted(): string {
    if (this.sizeBytes === -1) return '~большой';
    return formatBytes(this.sizeBytes);
  }

  get sizeUnknown(): boolean {
    return this.sizeBytes === -1;
  }

  get directoryName(): string {
    const sep = this.path.includes('\\') ? '\\' : '\\';
    return this.path.split(sep).pop() || this.path;
  }

  get isRegistry(): boolean {
    return this.type === 'registry';
  }

  get isEmpty(): boolean {
    return this.type === 'empty';
  }

  get isFolder(): boolean {
    return this.type === 'folder';
  }

  static create(
    path: string,
    sizeBytes: number,
    fileCount: number,
    reason: string = 'Нет в списке установленных программ',
    itemType: LeftoverType = 'folder'
  ): LeftoverItem {
    return new LeftoverItem(path, sizeBytes, fileCount, reason, itemType);
  }
}

/**
 * Domain Entity: LeftoverSummary
 */
export class LeftoverSummary {
  constructor(
    public readonly items: ReadonlyArray<LeftoverItem>
  ) {}

  get totalBytes(): number {
    return this.items.reduce((sum, item) => sum + Math.max(0, item.sizeBytes), 0);
  }

  get totalFiles(): number {
    return this.items.reduce((sum, item) => sum + item.fileCount, 0);
  }

  get folderCount(): number {
    return this.folders.length;
  }

  get totalFormatted(): string {
    return formatBytes(this.totalBytes);
  }

  get folders(): LeftoverItem[] {
    return this.items.filter(i => i.isFolder);
  }

  get emptyFolders(): LeftoverItem[] {
    return this.items.filter(i => i.isEmpty);
  }

  get registryKeys(): LeftoverItem[] {
    return this.items.filter(i => i.isRegistry);
  }

  get sortedBySize(): LeftoverItem[] {
    return [...this.items].sort((a, b) => b.sizeBytes - a.sizeBytes);
  }

  get topItems(): LeftoverItem[] {
    return this.sortedBySize.slice(0, 20);
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}
