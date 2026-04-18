/**
 * Presentation Hook: useSystemInfo
 * Manages system information state
 */
import { useState, useCallback, useEffect } from 'react';
import { SystemInfoSummary } from '@domain/index';
import { getSystemInfoUseCase } from '@container/index';

interface UseSystemInfoReturn {
  info: SystemInfoSummary | null;
  loading: boolean;
  error: string | null;
  loadInfo: () => Promise<void>;
}

export function useSystemInfo(): UseSystemInfoReturn {
  const [info, setInfo] = useState<SystemInfoSummary | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadInfo = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getSystemInfoUseCase.execute();
      setInfo(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load system info');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadInfo();
  }, [loadInfo]);

  return {
    info,
    loading,
    error,
    loadInfo
  };
}
