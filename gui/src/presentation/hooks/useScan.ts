/**
 * Presentation Hook: useScan
 * Manages scanning operations
 */
import { useState, useCallback } from 'react';
import { ScanSummary, CategorySelection, OperationLog } from '@domain/index';
import { scanUseCase } from '@container/index';

interface UseScanReturn {
  scanning: boolean;
  result: ScanSummary | null;
  log: OperationLog | null;
  error: string | null;
  scan: (aggressive: boolean, selection: CategorySelection) => Promise<void>;
  clear: () => void;
}

export function useScan(): UseScanReturn {
  const [scanning, setScanning] = useState(false);
  const [result, setResult] = useState<ScanSummary | null>(null);
  const [log, setLog] = useState<OperationLog | null>(null);
  const [error, setError] = useState<string | null>(null);

  const scan = useCallback(async (aggressive: boolean, selection: CategorySelection) => {
    setScanning(true);
    setError(null);
    setResult(null);
    setLog(null);

    try {
      const response = await scanUseCase.execute({
        aggressive,
        selection
      });
      setResult(response.summary);
      setLog(response.log);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Scan failed');
    } finally {
      setScanning(false);
    }
  }, []);

  const clear = useCallback(() => {
    setResult(null);
    setLog(null);
    setError(null);
  }, []);

  return {
    scanning,
    result,
    log,
    error,
    scan,
    clear
  };
}
