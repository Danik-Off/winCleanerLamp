/**
 * Presentation Hook: useLeftovers
 * Manages leftover scanning operations
 */
import { useState, useCallback } from 'react';
import { LeftoverSummary } from '@domain/index';
import { getLeftoversUseCase } from '@container/index';

interface UseLeftoversReturn {
  scanning: boolean;
  result: LeftoverSummary | null;
  error: string | null;
  scan: () => Promise<void>;
  clear: () => void;
}

export function useLeftovers(): UseLeftoversReturn {
  const [scanning, setScanning] = useState(false);
  const [result, setResult] = useState<LeftoverSummary | null>(null);
  const [error, setError] = useState<string | null>(null);

  const scan = useCallback(async () => {
    setScanning(true);
    setError(null);
    try {
      const result = await getLeftoversUseCase.execute();
      setResult(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Leftover scan failed');
    } finally {
      setScanning(false);
    }
  }, []);

  const clear = useCallback(() => {
    setResult(null);
    setError(null);
  }, []);

  return {
    scanning,
    result,
    error,
    scan,
    clear
  };
}
