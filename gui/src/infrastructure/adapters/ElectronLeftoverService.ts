/**
 * Infrastructure Adapter: ElectronLeftoverService
 * Implements ILeftoverService using Electron IPC
 */
import type { ILeftoverService } from '@application/ports/ILeftoverService';
import type { LeftoverType, InstalledProgramInfo } from '@domain/index';
import { LeftoverSummary, LeftoverItem } from '@domain/index';

interface JsonLeftoverCandidate {
  path: string;
  size: number;
  files: number;
  reason: string;
  type: 'folder' | 'registry' | 'empty';
  orphanMatch?: string;
  cacheHit: boolean;
  installedMatch: boolean;
  likelyUserData: boolean;
}

interface JsonInstalledProgram {
  displayName: string;
  publisher?: string;
  installLocation?: string;
  inOrphanDB: boolean;
}

interface JsonLeftoversResult {
  candidates: JsonLeftoverCandidate[] | null;
  installed: JsonInstalledProgram[] | null;
  error?: string;
}

export class ElectronLeftoverService implements ILeftoverService {
  async scanLeftovers(): Promise<LeftoverSummary> {
    const raw = await window.electronAPI.getLeftovers();
    const parsed: JsonLeftoversResult = JSON.parse(raw);
    if (parsed.error) {
      throw new Error(parsed.error);
    }

    const items = (parsed.candidates || []).map((c) => new LeftoverItem(
      c.path,
      c.size,
      c.files,
      c.reason,
      c.type as LeftoverType,
      c.orphanMatch || '',
      c.cacheHit,
      c.installedMatch,
      c.likelyUserData
    ));

    const installedPrograms: InstalledProgramInfo[] = (parsed.installed || []).map((p) => ({
      displayName: p.displayName,
      publisher: p.publisher,
      installLocation: p.installLocation,
      inOrphanDB: p.inOrphanDB,
    }));

    return new LeftoverSummary(items, installedPrograms);
  }
}
