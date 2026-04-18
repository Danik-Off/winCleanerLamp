/**
 * Presentation Hook: useClean
 * Manages cleaning operations
 */
import { useState, useCallback } from 'react';
import { CategorySelection, OperationLog } from '@domain/index';
import { cleanUseCase } from '@container/index';

interface UseCleanReturn {
  cleaning: boolean;
  log: OperationLog | null;
  bytesCleaned: number;
  filesCleaned: number;
  error: string | null;
  clean: (aggressive: boolean, selection: CategorySelection) => Promise<void>;
  clear: () => void;
}

export function useClean(): UseCleanReturn {
  const [cleaning, setCleaning] = useState(false);
  const [log, setLog] = useState<OperationLog | null>(null);
  const [bytesCleaned, setBytesCleaned] = useState(0);
  const [filesCleaned, setFilesCleaned] = useState(0);
  const [error, setError] = useState<string | null>(null);

  const clean = useCallback(async (aggressive: boolean, selection: CategorySelection) => {
    setCleaning(true);
    setError(null);
    setLog(null);
    setBytesCleaned(0);
    setFilesCleaned(0);

    try {
      const response = await cleanUseCase.execute({
        aggressive,
        selection,
        skipConfirmation: true
      });
      setLog(response.log);
      setBytesCleaned(response.bytesCleaned);
      setFilesCleaned(response.filesCleaned);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Clean failed');
    } finally {
      setCleaning(false);
    }
  }, []);

  const clear = useCallback(() => {
    setLog(null);
    setError(null);
    setBytesCleaned(0);
    setFilesCleaned(0);
  }, []);

  return {
    cleaning,
    log,
    bytesCleaned,
    filesCleaned,
    error,
    clean,
    clear
  };
}
