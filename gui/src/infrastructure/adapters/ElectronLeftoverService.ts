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
    
    for (const line of lines) {
      // Detect section headers
      if (line.includes('=== Папки-остатки')) {
        currentSection = 'folder';
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

      // Parse folder data lines: "  SIZE  FILES  PATH"
      if (currentSection === 'folder') {
        const parsed = this.parseFolderLine(trimmed);
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

  private parseFolderLine(line: string): LeftoverItem | null {
    // Parse lines like: "46.31 GB       68396  C:\Users\..."
    // or: "~большой         0  C:\Users\..."
    const match = line.match(/^([\d.]+\s*[KMGT]?B|~большой)\s+(\d+)\s+(.+)$/);
    if (!match) return null;
    
    const [, sizeStr, files, path] = match;
    let sizeBytes: number;
    if (sizeStr.trim() === '~большой') {
      sizeBytes = -1;
    } else {
      sizeBytes = this.parseSize(sizeStr.trim());
    }
    
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
