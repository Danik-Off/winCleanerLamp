/**
 * Autostart Panel Component
 * Управление автозагрузкой: Run-ключи реестра, папки автозагрузки, задания
 * планировщика с триггером "при входе в систему". См. docs/research-autostart.md.
 *
 * Включение/выключение обратимо (StartupApproved/schtasks state) — исходная
 * команда автозапуска никогда не удаляется.
 */
import { useState, useCallback, useEffect, useMemo } from 'react';
import {
  Paper,
  Typography,
  Button,
  Box,
  List,
  ListItem,
  ListItemText,
  Chip,
  CircularProgress,
  Alert,
  Divider,
  Switch,
  TextField,
  InputAdornment,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  PowerSettingsNew as AutostartIcon,
  Refresh as RefreshIcon,
  Search as SearchIcon,
  Clear as ClearIcon,
} from '@mui/icons-material';
import { ScanningIndicator } from './ScanningIndicator';

interface AutostartEntryDto {
  id: string;
  source: string;
  name: string;
  command?: string;
  location: string;
  enabled: boolean;
  canToggle: boolean;
}

interface AutostartPanelProps {
  onError: (error: string) => void;
}

const SOURCE_LABELS: Record<string, string> = {
  'run-hkcu': 'Реестр — Run (текущий пользователь)',
  'run-hklm': 'Реестр — Run (все пользователи)',
  'run-hklm32': 'Реестр — Run (32-бит)',
  'startup-folder-user': 'Папка автозагрузки (пользователь)',
  'startup-folder-common': 'Папка автозагрузки (все пользователи)',
  'scheduled-task': 'Задания планировщика (при входе)',
};

const SOURCE_ORDER = [
  'run-hkcu', 'run-hklm', 'run-hklm32',
  'startup-folder-user', 'startup-folder-common', 'scheduled-task',
];

interface HistoryAction {
  id: string;
  name: string;
  /** Состояние ДО этого действия — на него откатывает Undo. */
  previousEnabled: boolean;
}

export function AutostartPanel({ onError }: AutostartPanelProps): JSX.Element {
  const [loading, setLoading] = useState(false);
  const [entries, setEntries] = useState<AutostartEntryDto[]>([]);
  const [pendingIds, setPendingIds] = useState<Set<string>>(new Set());
  const [filter, setFilter] = useState('');
  const [loaded, setLoaded] = useState(false);
  // Последнее действие для баннера "Undo" — обратимость и так гарантирована
  // на уровне реестра (StartupApproved никогда не удаляет команду), но
  // пользователю удобнее одной кнопкой отменить случайный клик, не вспоминая
  // какое было состояние до этого.
  const [lastAction, setLastAction] = useState<HistoryAction | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await window.electronAPI.autostartList();
      if (res.error && res.entries.length === 0) {
        onError(res.error);
      }
      setEntries(res.entries);
      setLoaded(true);
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Не удалось получить список автозагрузки');
    } finally {
      setLoading(false);
    }
  }, [onError]);

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleToggle = useCallback(async (entry: AutostartEntryDto, recordHistory = true) => {
    const nextEnabled = !entry.enabled;
    setPendingIds((prev) => new Set(prev).add(entry.id));
    // Оптимистичное обновление — переключатель отзывчив, откатываем при ошибке.
    setEntries((prev) => prev.map((e) => (e.id === entry.id ? { ...e, enabled: nextEnabled } : e)));
    try {
      const res = await window.electronAPI.autostartToggle({ id: entry.id, enable: nextEnabled });
      if (!res.success) {
        setEntries((prev) => prev.map((e) => (e.id === entry.id ? { ...e, enabled: entry.enabled } : e)));
        onError(res.error || 'Не удалось изменить состояние автозагрузки');
      } else if (recordHistory) {
        setLastAction({ id: entry.id, name: entry.name, previousEnabled: entry.enabled });
      }
    } catch (err) {
      setEntries((prev) => prev.map((e) => (e.id === entry.id ? { ...e, enabled: entry.enabled } : e)));
      onError(err instanceof Error ? err.message : 'Ошибка переключения автозагрузки');
    } finally {
      setPendingIds((prev) => {
        const next = new Set(prev);
        next.delete(entry.id);
        return next;
      });
    }
  }, [onError]);

  const handleUndo = useCallback(async () => {
    if (!lastAction) return;
    const current = entries.find((e) => e.id === lastAction.id);
    if (!current) {
      setLastAction(null);
      return;
    }
    setLastAction(null);
    // current.enabled сейчас равно !previousEnabled — handleToggle снова
    // инвертирует и вернёт к previousEnabled. recordHistory=false, чтобы
    // "отменить" не порождало собственную запись для повторной отмены.
    await handleToggle(current, false);
  }, [lastAction, entries, handleToggle]);

  const filtered = useMemo(() => {
    const lf = filter.trim().toLowerCase();
    if (!lf) return entries;
    return entries.filter((e) => e.name.toLowerCase().includes(lf) || (e.command || '').toLowerCase().includes(lf));
  }, [entries, filter]);

  const grouped = useMemo(() => {
    const map = new Map<string, AutostartEntryDto[]>();
    for (const e of filtered) {
      const list = map.get(e.source) || [];
      list.push(e);
      map.set(e.source, list);
    }
    return map;
  }, [filtered]);

  const enabledCount = entries.filter((e) => e.enabled).length;

  return (
    <Box>
      <Paper
        elevation={2}
        sx={{
          p: 3, mb: 2, borderRadius: 3,
          background: (t) => t.palette.mode === 'dark'
            ? 'linear-gradient(135deg, #4a148c 0%, #6a1b9a 100%)'
            : 'linear-gradient(135deg, #f3e5f5 0%, #e1bee7 100%)',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
          <AutostartIcon sx={{ mr: 1.5, fontSize: 28 }} />
          <Typography variant="h6" sx={{ fontWeight: 700 }}>Автозагрузка</Typography>
        </Box>
        <Alert severity="info" sx={{ mb: 2, borderRadius: 1.5 }}>
          Реестр (Run, все пользователи и 32-бит), обе папки автозагрузки и задания планировщика с триггером «при входе».
          Выключение обратимо — команда не удаляется, только помечается неактивной (как в Диспетчере задач).
        </Alert>
        <Button
          variant="contained"
          onClick={load}
          disabled={loading}
          startIcon={loading ? <CircularProgress size={20} /> : <RefreshIcon />}
          sx={{ borderRadius: 2, textTransform: 'none', fontWeight: 600 }}
        >
          {loading ? 'Загрузка...' : 'Обновить'}
        </Button>
        {loaded && (
          <Chip
            label={`${enabledCount} из ${entries.length} включено`}
            size="small"
            sx={{ ml: 1.5, fontWeight: 600 }}
          />
        )}
      </Paper>

      <ScanningIndicator active={loading} />

      {lastAction && (
        <Alert
          severity="info"
          sx={{ mb: 2, borderRadius: 2 }}
          onClose={() => setLastAction(null)}
          action={
            <Button color="inherit" size="small" onClick={handleUndo} sx={{ fontWeight: 700 }}>
              Отменить
            </Button>
          }
        >
          «{lastAction.name}»: {lastAction.previousEnabled ? 'выключено' : 'включено'}
        </Alert>
      )}

      {loaded && entries.length > 0 && (
        <TextField
          fullWidth
          size="small"
          placeholder="Фильтр по имени или команде..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          sx={{ mb: 2 }}
          InputProps={{
            startAdornment: <InputAdornment position="start"><SearchIcon sx={{ opacity: 0.5 }} /></InputAdornment>,
            endAdornment: filter ? <InputAdornment position="end"><IconButton size="small" onClick={() => setFilter('')}><ClearIcon fontSize="small" /></IconButton></InputAdornment> : null,
            sx: { borderRadius: 2 },
          }}
        />
      )}

      {loaded && entries.length === 0 && (
        <Alert severity="success" sx={{ borderRadius: 2 }}>Записей автозагрузки не найдено.</Alert>
      )}

      {SOURCE_ORDER.map((source) => {
        const list = grouped.get(source);
        if (!list || list.length === 0) return null;
        return (
          <Paper key={source} variant="outlined" sx={{ mb: 2, borderRadius: 3, borderColor: 'divider', overflow: 'hidden' }}>
            <Box sx={{ bgcolor: 'action.hover', px: 2, py: 1 }}>
              <Typography variant="body2" sx={{ fontWeight: 700 }}>
                {SOURCE_LABELS[source] || source} ({list.length})
              </Typography>
            </Box>
            <List dense disablePadding>
              {list.map((entry, idx) => (
                <Box key={entry.id}>
                  <ListItem sx={{ py: 1, px: 2 }}>
                    <Tooltip title={entry.enabled ? 'Выключить' : 'Включить'}>
                      <span>
                        <Switch
                          checked={entry.enabled}
                          disabled={!entry.canToggle || pendingIds.has(entry.id)}
                          onChange={() => handleToggle(entry)}
                          size="small"
                          color="success"
                        />
                      </span>
                    </Tooltip>
                    <ListItemText
                      sx={{ ml: 1 }}
                      primary={entry.name}
                      secondary={entry.command || entry.location}
                      primaryTypographyProps={{ variant: 'body2', sx: { fontWeight: 600, opacity: entry.enabled ? 1 : 0.55 } }}
                      secondaryTypographyProps={{ variant: 'caption', sx: { wordBreak: 'break-all', opacity: entry.enabled ? 0.7 : 0.4 } }}
                    />
                    <Chip
                      label={entry.enabled ? 'включено' : 'выключено'}
                      size="small"
                      color={entry.enabled ? 'success' : 'default'}
                      variant={entry.enabled ? 'filled' : 'outlined'}
                      sx={{ fontWeight: 600, fontSize: '0.65rem', flexShrink: 0 }}
                    />
                  </ListItem>
                  {idx < list.length - 1 && <Divider />}
                </Box>
              ))}
            </List>
          </Paper>
        );
      })}
    </Box>
  );
}
