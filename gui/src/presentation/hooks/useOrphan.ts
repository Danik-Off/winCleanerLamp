/**
 * Presentation Hook: useOrphan
 * Manages orphan scanning, discovery, and cleanup operations
 */
import { useState, useCallback } from 'react';

export interface OrphanScanItem {
  displayName: string;
  totalSize: string;
  totalFiles: number;
  paths: { size: string; path: string; isUserData: boolean }[];
  regKeys: string[];
  /** false = программа сейчас НЕ установлена — приоритетный кандидат на очистку. */
  programInstalled: boolean;
}

const USER_DATA_MARKER = '⚠ [ПОЛЬЗОВАТЕЛЬСКИЕ ДАННЫЕ] ';

export interface DiscoverItem {
  path: string;
  size_mb: number;
  size_bytes: number;
  has_executable: boolean;
}

interface UseOrphanReturn {
  scanning: boolean;
  discovering: boolean;
  cleaning: boolean;
  scanResults: OrphanScanItem[];
  discoverResults: DiscoverItem[];
  error: string | null;
  scanOutput: string;
  discoverOutput: string;
  orphanScan: () => Promise<void>;
  orphanDiscover: (roots?: string) => Promise<void>;
  orphanClean: (names: string, recycle?: boolean, cacheOnly?: boolean, includeUserData?: boolean) => Promise<string>;
  clear: () => void;
}

function parseScanResults(output: string): OrphanScanItem[] {
  const items: OrphanScanItem[] = [];
  const lines = output.split('\n');
  let current: OrphanScanItem | null = null;

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) {
      if (current) {
        items.push(current);
        current = null;
      }
      continue;
    }

    // Program header line: "  DisplayName  (SIZE, N файлов)  [★ программа удалена|программа установлена]"
    const headerMatch = trimmed.match(/^(.+?)\s{2,}\((.+?),\s*(\d+)\s*файлов?\)/);
    if (headerMatch && !trimmed.startsWith('[')) {
      if (current) items.push(current);
      current = {
        displayName: headerMatch[1].trim(),
        totalSize: headerMatch[2].trim(),
        totalFiles: parseInt(headerMatch[3], 10),
        paths: [],
        regKeys: [],
        programInstalled: trimmed.includes('[программа установлена]'),
      };
      continue;
    }

    // Path line: "    [  SIZE] PATH" (путь может начинаться с маркера пользовательских данных)
    const pathMatch = trimmed.match(/^\[\s*([\d.]+\s*[KMGT]?B|~большой)\s*\]\s+(.+)$/);
    if (pathMatch && current) {
      let rawPath = pathMatch[2].trim();
      const isUserData = rawPath.startsWith(USER_DATA_MARKER);
      if (isUserData) rawPath = rawPath.slice(USER_DATA_MARKER.length);
      current.paths.push({ size: pathMatch[1].trim(), path: rawPath, isUserData });
      continue;
    }

    // Registry line: "    [реестр]   KEY"
    const regMatch = trimmed.match(/^\[реестр\]\s+(.+)$/);
    if (regMatch && current) {
      current.regKeys.push(regMatch[1].trim());
      continue;
    }

    // Standalone name line (no size info — no results yet)
    if (!trimmed.startsWith('[') && !trimmed.startsWith('-') && !trimmed.startsWith('ИТОГО') && !trimmed.startsWith('Для удаления') && !trimmed.startsWith('Сканирование') && !trimmed.startsWith('Найдены') && !trimmed.startsWith('Подтверждённых')) {
      if (current) items.push(current);
      current = {
        displayName: trimmed,
        totalSize: '0 B',
        totalFiles: 0,
        paths: [],
        regKeys: [],
        programInstalled: false,
      };
    }
  }
  if (current) items.push(current);
  return items;
}

function parseDiscoverResults(output: string): DiscoverItem[] {
  try {
    // CLI outputs JSON with --orphan-json
    const jsonStart = output.indexOf('[');
    const jsonEnd = output.lastIndexOf(']');
    if (jsonStart >= 0 && jsonEnd > jsonStart) {
      return JSON.parse(output.substring(jsonStart, jsonEnd + 1));
    }
  } catch {
    // fallback
  }
  return [];
}

export function useOrphan(): UseOrphanReturn {
  const [scanning, setScanning] = useState(false);
  const [discovering, setDiscovering] = useState(false);
  const [cleaning, setCleaning] = useState(false);
  const [scanResults, setScanResults] = useState<OrphanScanItem[]>([]);
  const [discoverResults, setDiscoverResults] = useState<DiscoverItem[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [scanOutput, setScanOutput] = useState('');
  const [discoverOutput, setDiscoverOutput] = useState('');

  const orphanScan = useCallback(async () => {
    setScanning(true);
    setError(null);
    try {
      const result = await window.electronAPI.orphanScan();
      setScanOutput(result.output);
      setScanResults(parseScanResults(result.output));
      if (result.error) setError(result.error);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Orphan scan failed');
    } finally {
      setScanning(false);
    }
  }, []);

  const orphanDiscover = useCallback(async (roots?: string) => {
    setDiscovering(true);
    setError(null);
    try {
      const result = await window.electronAPI.orphanDiscover(roots ? { roots } : undefined);
      setDiscoverOutput(result.output);
      setDiscoverResults(parseDiscoverResults(result.output));
      if (result.error) setError(result.error);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Discover failed');
    } finally {
      setDiscovering(false);
    }
  }, []);

  const orphanClean = useCallback(async (names: string, recycle?: boolean, cacheOnly?: boolean, includeUserData?: boolean): Promise<string> => {
    setCleaning(true);
    setError(null);
    try {
      const result = await window.electronAPI.orphanClean({ names, recycle, cacheOnly, includeUserData });
      if (result.error) setError(result.error);
      return result.output;
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Clean failed';
      setError(msg);
      return msg;
    } finally {
      setCleaning(false);
    }
  }, []);

  const clear = useCallback(() => {
    setScanResults([]);
    setDiscoverResults([]);
    setError(null);
    setScanOutput('');
    setDiscoverOutput('');
  }, []);

  return {
    scanning, discovering, cleaning,
    scanResults, discoverResults,
    error, scanOutput, discoverOutput,
    orphanScan, orphanDiscover, orphanClean, clear,
  };
}
