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
    
    for (const line of lines) {
      // Parse lines like: "hiberfil.sys              12.42 GB  C:\hiberfil.sys"
      const match = line.match(/^(\S+)\s+([\d.]+\s*\w+)\s+(.+)$/);
      if (match) {
        const [, name, sizeStr, path] = match;
        const sizeBytes = this.parseSize(sizeStr.trim());
        
        // Find the hint in following lines
        const hint = this.findHint(lines, lines.indexOf(line));
        
        files.push(SystemFileInfo.create(name, path.trim(), sizeBytes, hint));
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
