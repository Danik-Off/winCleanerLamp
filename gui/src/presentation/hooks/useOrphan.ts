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
  orphanTrack: (path: string, name?: string, asCache?: boolean) => Promise<{ success: boolean; error?: string }>;
  clear: () => void;
}

interface JsonOrphanFoundPath {
  path: string;
  size: number;
  files: number;
  exists: boolean;
  isUserData: boolean;
  likelyUserData: boolean;
}
interface JsonOrphanScanResult {
  app: { displayName: string };
  foundPaths: JsonOrphanFoundPath[] | null;
  foundRegKeys: string[] | null;
  totalSize: number;
  totalFiles: number;
  programInstalled: boolean;
}

function formatBytes(bytes: number): string {
  if (bytes === -1) return '~большой';
  if (bytes <= 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function parseScanResults(output: string): OrphanScanItem[] {
  let raw: JsonOrphanScanResult[];
  try {
    raw = JSON.parse(output) || [];
  } catch {
    return [];
  }
  return raw.map((r) => ({
    displayName: r.app.displayName,
    totalSize: formatBytes(r.totalSize),
    totalFiles: r.totalFiles,
    paths: (r.foundPaths || []).map((p) => ({
      size: formatBytes(p.size),
      path: p.path,
      isUserData: p.isUserData || p.likelyUserData,
    })),
    regKeys: r.foundRegKeys || [],
    programInstalled: r.programInstalled,
  }));
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

  const orphanTrack = useCallback(async (path: string, name?: string, asCache?: boolean) => {
    try {
      return await window.electronAPI.orphanTrack({ path, name, asCache });
    } catch (err) {
      return { success: false, error: err instanceof Error ? err.message : 'Track failed' };
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
    orphanScan, orphanDiscover, orphanClean, orphanTrack, clear,
  };
}
