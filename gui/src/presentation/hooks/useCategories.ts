/**
 * Presentation Hook: useCategories
 * Manages category state and selection
 */
import { useState, useCallback, useEffect } from 'react';
import { Category, CategorySelection } from '@domain/index';
import { getCategoriesUseCase } from '@container/index';

interface UseCategoriesReturn {
  categories: {
    safe: Category[];
    aggressive: Category[];
  };
  selection: CategorySelection;
  loading: boolean;
  error: string | null;
  loadCategories: () => Promise<void>;
  toggleCategory: (id: string) => void;
  selectAllSafe: () => void;
  selectAllAggressive: () => void;
  deselectAllSafe: () => void;
  deselectAllAggressive: () => void;
}

export function useCategories(): UseCategoriesReturn {
  const [categories, setCategories] = useState<{ safe: Category[]; aggressive: Category[] }>({
    safe: [],
    aggressive: []
  });
  const [selection, setSelection] = useState<CategorySelection>(CategorySelection.empty());
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadCategories = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getCategoriesUseCase.execute();
      setCategories({
        safe: result.safe,
        aggressive: result.aggressive
      });
      setSelection(result.defaultSelection);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load categories');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadCategories();
  }, [loadCategories]);

  const toggleCategory = useCallback((id: string) => {
    setSelection(prev => prev.toggle(id));
  }, []);

  const selectAllSafe = useCallback(() => {
    setSelection(prev => {
      let newSelection = prev;
      categories.safe.forEach(c => {
        newSelection = newSelection.select(c.id);
      });
      return newSelection;
    });
  }, [categories.safe]);

  const selectAllAggressive = useCallback(() => {
    setSelection(prev => {
      let newSelection = prev;
      categories.aggressive.forEach(c => {
        newSelection = newSelection.select(c.id);
      });
      return newSelection;
    });
  }, [categories.aggressive]);

  const deselectAllSafe = useCallback(() => {
    setSelection(prev => {
      let newSelection = prev;
      categories.safe.forEach(c => {
        newSelection = newSelection.deselect(c.id);
      });
      return newSelection;
    });
  }, [categories.safe]);

  const deselectAllAggressive = useCallback(() => {
    setSelection(prev => {
      let newSelection = prev;
      categories.aggressive.forEach(c => {
        newSelection = newSelection.deselect(c.id);
      });
      return newSelection;
    });
  }, [categories.aggressive]);

  return {
    categories,
    selection,
    loading,
    error,
    loadCategories,
    toggleCategory,
    selectAllSafe,
    selectAllAggressive,
    deselectAllSafe,
    deselectAllAggressive
  };
}
