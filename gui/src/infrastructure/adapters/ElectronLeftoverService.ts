/**
 * Infrastructure Adapter: ElectronLeftoverService
 * Implements ILeftoverService using Electron IPC
 */
import type { ILeftoverService } from '@application/ports/ILeftoverService';
import type { LeftoverType } from '@domain/index';
import { LeftoverSummary, LeftoverItem } from '@domain/index';

export class ElectronLeftoverService implements ILeftoverService {
  async scanLeftovers(): Promise<LeftoverSummary> {
    const output = await window.electronAPI.getLeftovers();
    
    const items = this.parseLeftovers(output);
    return new LeftoverSummary(items);
  }

  private parseLeftovers(output: string): LeftoverItem[] {
    const lines = output.split('\n');
    const items: LeftoverItem[] = [];
    let currentSection: LeftoverType = 'folder';
    let currentCacheHit = false;
    
    for (const line of lines) {
      // Detect section headers (new categorized format)
      if (line.includes('=== Кеш программ из orphan DB')) {
        currentSection = 'folder';

        currentCacheHit = true;
        continue;
      }
      if (line.includes('=== Известные остатки из orphan DB')) {
        currentSection = 'folder';

        currentCacheHit = false;
        continue;
      }
      if (line.includes('=== Неизвестные папки')) {
        currentSection = 'folder';

        currentCacheHit = false;
        continue;
      }
      // Legacy format support
      if (line.includes('=== Папки-остатки')) {
        currentSection = 'folder';

        currentCacheHit = false;
        continue;
      }
      if (line.includes('=== Пустые папки')) {
        currentSection = 'empty';
        continue;
      }
      if (line.includes('=== Ключи реестра')) {
        currentSection = 'registry';
        continue;
      }
      
      // Skip separator/header lines
      if (line.includes('---') || line.includes('РАЗМЕР') || line.includes('ФАЙЛОВ')) continue;
      // Stop at total line
      if (line.includes('ИТОГО:')) break;
      
      const trimmed = line.trim();
      if (!trimmed) continue;

      // Parse folder data lines: "  SIZE  FILES  TAG  PATH"
      if (currentSection === 'folder') {
        const parsed = this.parseFolderLine(trimmed, currentCacheHit);
        if (parsed) items.push(parsed);
        continue;
      }

      // Parse empty folder lines: "  [пусто]  PATH"
      if (currentSection === 'empty') {
        const emptyMatch = trimmed.match(/^\[пусто\]\s+(.+)$/);
        if (emptyMatch) {
          items.push(LeftoverItem.create(emptyMatch[1].trim(), 0, 0, 'Пустая папка', 'empty'));
        }
        continue;
      }

      // Parse registry lines: "  [реестр]  PATH"
      if (currentSection === 'registry') {
        const regMatch = trimmed.match(/^\[реестр\]\s+(.+)$/);
        if (regMatch) {
          items.push(LeftoverItem.create(regMatch[1].trim(), 0, 0, 'Ключ реестра без программы', 'registry'));
        }
        continue;
      }
    }
    
    return items;
  }

  private parseFolderLine(line: string, sectionCacheHit: boolean): LeftoverItem | null {
    // 4-column format: "SIZE  FILES  [TAG]  PATH"
    // TAG is bracketed: [кеш], [?], [Google Chrome], etc.
    const match4 = line.match(/^([\d.]+\s*[KMGT]?B|~большой)\s+(\d+)\s+\[([^\]]*)\]\s+(.+)$/);
    if (match4) {
      const [, sizeStr, files, tag, path] = match4;
      const sizeBytes = sizeStr.trim() === '~большой' ? -1 : this.parseSize(sizeStr.trim());
      const isCacheTag = tag === 'кеш' || sectionCacheHit;
      const isUnknownTag = tag === '?';
      const orphanName = (!isCacheTag && !isUnknownTag) ? tag : '';
      const reason = isCacheTag
        ? `кеш (orphan DB)`
        : isUnknownTag
          ? 'нет в orphan DB'
          : `orphan DB: ${orphanName}`;
      return LeftoverItem.create(
        path.trim(),
        sizeBytes,
        parseInt(files, 10),
        reason,
        'folder',
        isCacheTag ? 'cache' : orphanName,
        isCacheTag
      );
    }
    // Legacy 3-column: "SIZE  FILES  PATH"
    const match3 = line.match(/^([\d.]+\s*[KMGT]?B|~большой)\s+(\d+)\s+(.+)$/);
    if (!match3) return null;
    const [, sizeStr, files, path] = match3;
    const sizeBytes = sizeStr.trim() === '~большой' ? -1 : this.parseSize(sizeStr.trim());
    return LeftoverItem.create(path.trim(), sizeBytes, parseInt(files, 10), 'Нет в списке установленных программ', 'folder');
  }

  private parseSize(sizeStr: string): number {
    const units: Record<string, number> = {
      'B': 1, 'KB': 1024, 'MB': 1024**2, 'GB': 1024**3, 'TB': 1024**4
    };
    const match = sizeStr.match(/^([\d.]+)\s*(\w+)$/);
    if (!match) return 0;
    const [, num, unit] = match;
    return parseFloat(num) * (units[unit.toUpperCase()] || 1);
  }
}
