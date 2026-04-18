/**
 * Infrastructure Adapter: ElectronSystemInfoService
 * Implements ISystemInfoService using Electron IPC
 */
import type { ISystemInfoService } from '@application/ports/ISystemInfoService';
import { SystemInfoSummary, SystemFileInfo } from '@domain/index';

export class ElectronSystemInfoService implements ISystemInfoService {
  async getSystemInfo(): Promise<SystemInfoSummary> {
    const output = await window.electronAPI.getSysInfo();
    
    const files = this.parseSystemInfo(output);
    return new SystemInfoSummary(files);
  }

  private parseSystemInfo(output: string): SystemFileInfo[] {
    const lines = output.split('\n');
    const files: SystemFileInfo[] = [];
    
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      // Parse lines like: "  hiberfil.sys              12.42 GB  C:\hiberfil.sys"
      // or: "  WinSxS                    ~большой  C:\Windows\WinSxS"
      // or: "  swapfile.sys             не найден  C:\swapfile.sys"
      // Format: 2 spaces + name(25 chars) + size(12 chars right-aligned) + 2 spaces + path
      const match = line.match(/^\s{2}(\S.{0,24}?)\s+([\d.]+\s*[KMGT]?B|~большой|не найден|—)\s{2}(.+)$/);
      if (match) {
        const [, name, sizeStr, path] = match;
        let sizeBytes = 0;
        if (sizeStr.trim() === '~большой') {
          sizeBytes = -1; // marker for "too large to calculate"
        } else if (sizeStr.trim() === 'не найден' || sizeStr.trim() === '—') {
          sizeBytes = 0;
        } else {
          sizeBytes = this.parseSize(sizeStr.trim());
        }
        
        const hint = this.findHint(lines, i);
        
        files.push(SystemFileInfo.create(name.trim(), path.trim(), sizeBytes, hint));
      }
    }
    
    return files;
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

  private findHint(lines: string[], startIndex: number): string {
    // Look for hint in next 2 lines
    for (let i = startIndex + 1; i < Math.min(startIndex + 3, lines.length); i++) {
      const line = lines[i].trim();
      if (line.startsWith('•') || line.startsWith('-') || line.startsWith('→')) {
        return line;
      }
    }
    return '';
  }
}
