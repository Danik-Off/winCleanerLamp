/**
 * Infrastructure Adapter: ElectronLeftoverService
 * Implements ILeftoverService using Electron IPC
 */
import type { ILeftoverService } from '@application/ports/ILeftoverService';
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
    let inTable = false;
    
    for (const line of lines) {
      // Detect table start
      if (line.includes('РАЗМЕР') && line.includes('ФАЙЛОВ')) {
        inTable = true;
        continue;
      }
      
      // Skip separator lines
      if (line.includes('---') || line.includes('===')) continue;
      
      // Parse data lines
      if (inTable && line.trim()) {
        const parsed = this.parseDataLine(line);
        if (parsed) {
          items.push(parsed);
        }
      }
      
      // Stop at total line
      if (line.includes('ИТОГО')) break;
    }
    
    return items;
  }

  private parseDataLine(line: string): LeftoverItem | null {
    // Parse lines like: "46.31 GB       68396  C:\Users\...\AppData\Roaming\.minecraft"
    const match = line.match(/([\d.]+\s*\w+)\s+(\d+)\s+(.+)$/);
    if (!match) return null;
    
    const [, sizeStr, files, path] = match;
    const sizeBytes = this.parseSize(sizeStr.trim());
    
    return LeftoverItem.create(path.trim(), sizeBytes, parseInt(files, 10));
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
