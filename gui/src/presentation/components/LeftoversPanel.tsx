/**
 * Leftovers Panel Component
 * Displays potentially orphaned folders, empty dirs, and registry keys
 */
import React, { useState, useMemo, useCallback } from 'react';
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
  TextField,
  InputAdornment,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  LinearProgress,
  Card,
  CardContent,
  Grid,
  Tabs,
  Tab,
  Stack,
} from '@mui/material';
import {
  Refresh as RefreshIcon,
  Warning as WarningIcon,
  Search as SearchIcon,
  Delete as DeleteIcon,
  Clear as ClearIcon,
  Storage as StorageIcon,
  FolderOff as EmptyFolderIcon,
  AppRegistration as RegistryIcon,
  Cached as CacheIcon,
  CheckCircle as KnownIcon,
  HelpOutline as UnknownIcon,
  LinkOff as ShortcutsIcon,
  BookmarkAdd as TrackIcon,
  BookmarkAdded as TrackedIcon,
} from '@mui/icons-material';
import { useLeftovers } from '../hooks';
import { ScanningIndicator } from './ScanningIndicator';
import type { LeftoverItem } from '@domain/index';

interface LeftoversPanelProps {
  onError: (error: string) => void;
}

interface BrokenShortcutItem {
  path: string;
  targetPath?: string;
  reason: string;
}

export function LeftoversPanel({ onError }: LeftoversPanelProps): JSX.Element {
  const { scanning, result, error, scan } = useLeftovers();
  const [filter, setFilter] = useState('');
  const [deleteTarget, setDeleteTarget] = useState<LeftoverItem | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deletedPaths, setDeletedPaths] = useState<Set<string>>(new Set());
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState(0);

  // Отслеживание неизвестных папок (добавление в orphaned_apps.json) — раньше
  // жило в отдельной вкладке-панели "Программы", которая по сути дублировала
  // этот же список неизвестных папок. Кнопка "добавить в отслеживание"
  // перенесена сюда, отдельная панель удалена.
  const [trackedPaths, setTrackedPaths] = useState<Set<string>>(new Set());
  const [trackingPath, setTrackingPath] = useState<string | null>(null);

  // Битые ярлыки — отдельный источник данных (своё сканирование), но
  // показывается как ещё одна вкладка в этой же секции, а не отдельным
  // пунктом навигации.
  const [shortcutsScanning, setShortcutsScanning] = useState(false);
  const [shortcutsScanned, setShortcutsScanned] = useState<number | null>(null);
  const [shortcutsBroken, setShortcutsBroken] = useState<BrokenShortcutItem[]>([]);
  const [shortcutDeleteTarget, setShortcutDeleteTarget] = useState<BrokenShortcutItem | null>(null);
  const [shortcutDeleting, setShortcutDeleting] = useState(false);
  const [shortcutDeleteError, setShortcutDeleteError] = useState<string | null>(null);

  if (error) {
    onError(error);
  }

  const applyFilter = useCallback(
    (items: LeftoverItem[]) => {
      const alive = items.filter((i) => !deletedPaths.has(i.path));
      if (!filter.trim()) return alive;
      const lf = filter.toLowerCase().trim();
      return alive.filter(
        (i) =>
          i.directoryName.toLowerCase().includes(lf) ||
          i.path.toLowerCase().includes(lf)
      );
    },
    [filter, deletedPaths]
  );

  const cacheItems = useMemo(
    () => (result ? applyFilter(result.cacheItems) : []),
    [result, applyFilter]
  );
  const removedProgramItems = useMemo(
    () => (result ? applyFilter(result.removedProgramItems) : []),
    [result, applyFilter]
  );
  const installedProgramItems = useMemo(
    () => (result ? applyFilter(result.installedProgramItems) : []),
    [result, applyFilter]
  );
  const orphanKnown = useMemo(
    () => [...removedProgramItems, ...installedProgramItems],
    [removedProgramItems, installedProgramItems]
  );
  const unknownFolders = useMemo(
    () => (result ? applyFilter(result.unknownFolders) : []),
    [result, applyFilter]
  );
  const folders = useMemo(
    () => (result ? applyFilter(result.folders) : []),
    [result, applyFilter]
  );
  const empties = useMemo(
    () => (result ? applyFilter(result.emptyFolders) : []),
    [result, applyFilter]
  );
  const regKeys = useMemo(
    () => (result ? applyFilter(result.registryKeys) : []),
    [result, applyFilter]
  );

  const totalBytes = useMemo(
    () => folders.reduce((s, i) => s + Math.max(0, i.sizeBytes), 0),
    [folders]
  );
  const cacheTotalBytes = useMemo(
    () => cacheItems.reduce((s, i) => s + Math.max(0, i.sizeBytes), 0),
    [cacheItems]
  );

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    setDeleteError(null);
    try {
      const res = await window.electronAPI.deleteLeftover(deleteTarget.path);
      if (res.success) {
        setDeletedPaths((prev) => new Set(prev).add(deleteTarget.path));
        setDeleteTarget(null);
      } else {
        setDeleteError(res.error || 'Неизвестная ошибка');
      }
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Ошибка удаления');
    } finally {
      setDeleting(false);
    }
  }, [deleteTarget]);

  const handleScan = useCallback(() => {
    setDeletedPaths(new Set());
    setFilter('');
    setActiveTab(0);
    scan();
  }, [scan]);

  const handleTrack = useCallback(async (path: string) => {
    setTrackingPath(path);
    try {
      const res = await window.electronAPI.orphanTrack({ path });
      if (res.success) {
        setTrackedPaths((prev) => new Set(prev).add(path));
      } else {
        onError(res.error || 'Не удалось добавить в orphaned_apps.json');
      }
    } finally {
      setTrackingPath(null);
    }
  }, [onError]);

  const handleShortcutsScan = useCallback(async () => {
    setShortcutsScanning(true);
    try {
      const res = await window.electronAPI.getBrokenShortcuts();
      if (res.error) {
        onError(res.error);
      }
      setShortcutsScanned(res.scanned);
      setShortcutsBroken(res.broken || []);
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Ошибка поиска ярлыков');
    } finally {
      setShortcutsScanning(false);
    }
  }, [onError]);

  const handleDeleteShortcut = useCallback(async () => {
    if (!shortcutDeleteTarget) return;
    setShortcutDeleting(true);
    setShortcutDeleteError(null);
    try {
      const res = await window.electronAPI.deleteFile(shortcutDeleteTarget.path);
      if (res.success) {
        setShortcutsBroken((prev) => prev.filter((b) => b.path !== shortcutDeleteTarget.path));
        setShortcutDeleteTarget(null);
      } else {
        setShortcutDeleteError(res.error || 'Неизвестная ошибка');
      }
    } catch (err) {
      setShortcutDeleteError(err instanceof Error ? err.message : 'Ошибка удаления');
    } finally {
      setShortcutDeleting(false);
    }
  }, [shortcutDeleteTarget]);

  const formatBytes = (bytes: number): string => {
    if (bytes === -1) return '~большой';
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const totalItems = folders.length + empties.length + regKeys.length + cacheItems.length;

  return (
    <Box>
      {/* Панель управления — контекстная кнопка сканирования: остатки для вкладок 0-4,
          отдельный поиск для вкладки "Ярлыки" (другой источник данных). */}
      <Box sx={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 1.5, mb: 2 }}>
        {activeTab === 5 ? (
          <>
            <Button
              variant="contained"
              onClick={handleShortcutsScan}
              disabled={shortcutsScanning}
              startIcon={shortcutsScanning ? <CircularProgress size={18} color="inherit" /> : <RefreshIcon />}
              sx={{ borderRadius: 2, fontWeight: 700, px: 3 }}
            >
              {shortcutsScanning ? 'Поиск...' : 'Найти битые ярлыки'}
            </Button>
            {shortcutsScanned !== null && (
              <Chip label={`проверено: ${shortcutsScanned}`} size="small" variant="outlined" sx={{ fontWeight: 600 }} />
            )}
            {shortcutsBroken.length > 0 && (
              <Chip label={`битых: ${shortcutsBroken.length}`} size="small" color="error" sx={{ fontWeight: 600 }} />
            )}
          </>
        ) : (
          <>
            <Button
              variant="contained"
              onClick={handleScan}
              disabled={scanning}
              startIcon={scanning ? <CircularProgress size={18} color="inherit" /> : <RefreshIcon />}
              sx={{ borderRadius: 2, fontWeight: 700, px: 3 }}
            >
              {scanning ? 'Сканирование...' : 'Сканировать остатки'}
            </Button>
            {result && totalItems > 0 && (
              <Chip label={`${totalItems} найдено`} size="small" color="warning" variant="outlined" sx={{ fontWeight: 600 }} />
            )}
            <Tooltip title="Поиск по AppData, ProgramData, Program Files и реестру — сопоставляет находки со списком установленных программ и базой orphaned_apps.json.">
              <WarningIcon sx={{ fontSize: 18, opacity: 0.4, cursor: 'help' }} />
            </Tooltip>
          </>
        )}
      </Box>

      <ScanningIndicator active={activeTab === 5 ? shortcutsScanning : scanning} />

      {/* Stat Cards */}
      {result && totalItems > 0 && (
        <Grid container spacing={2} sx={{ mb: 2 }}>
          <Grid item xs={6} sm={4} md={2}>
            <Card elevation={1} sx={{ borderRadius: 3, background: (t) => t.palette.mode === 'dark' ? 'linear-gradient(135deg,#1b5e20,#2e7d32)' : 'linear-gradient(135deg,#e8f5e9,#c8e6c9)' }}>
              <CardContent sx={{ textAlign: 'center', py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <StorageIcon sx={{ fontSize: 24, opacity: 0.8 }} />
                <Typography variant="body1" sx={{ fontWeight: 700 }}>{formatBytes(totalBytes)}</Typography>
                <Typography variant="caption" sx={{ opacity: 0.8, fontSize: '0.65rem' }}>Всего</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={6} sm={4} md={2}>
            <Card elevation={1} sx={{ borderRadius: 3, background: (t) => t.palette.mode === 'dark' ? 'linear-gradient(135deg,#006064,#00838f)' : 'linear-gradient(135deg,#e0f7fa,#b2ebf2)' }}>
              <CardContent sx={{ textAlign: 'center', py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <CacheIcon sx={{ fontSize: 24, opacity: 0.8 }} />
                <Typography variant="body1" sx={{ fontWeight: 700 }}>{cacheItems.length}</Typography>
                <Typography variant="caption" sx={{ opacity: 0.8, fontSize: '0.65rem' }}>Кеш ({formatBytes(cacheTotalBytes)})</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={6} sm={4} md={2}>
            <Card elevation={1} sx={{ borderRadius: 3, background: (t) => t.palette.mode === 'dark' ? 'linear-gradient(135deg,#e65100,#f57c00)' : 'linear-gradient(135deg,#fff3e0,#ffe0b2)' }}>
              <CardContent sx={{ textAlign: 'center', py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <KnownIcon sx={{ fontSize: 24, opacity: 0.8 }} />
                <Typography variant="body1" sx={{ fontWeight: 700 }}>{orphanKnown.length}</Typography>
                <Typography variant="caption" sx={{ opacity: 0.8, fontSize: '0.65rem' }}>Известные</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={6} sm={4} md={2}>
            <Card elevation={1} sx={{ borderRadius: 3, background: (t) => t.palette.mode === 'dark' ? 'linear-gradient(135deg,#bf360c,#d84315)' : 'linear-gradient(135deg,#fbe9e7,#ffccbc)' }}>
              <CardContent sx={{ textAlign: 'center', py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <UnknownIcon sx={{ fontSize: 24, opacity: 0.8 }} />
                <Typography variant="body1" sx={{ fontWeight: 700 }}>{unknownFolders.length}</Typography>
                <Typography variant="caption" sx={{ opacity: 0.8, fontSize: '0.65rem' }}>Неизвестные</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={6} sm={4} md={2}>
            <Card elevation={1} sx={{ borderRadius: 3, background: (t) => t.palette.mode === 'dark' ? 'linear-gradient(135deg,#4a148c,#6a1b9a)' : 'linear-gradient(135deg,#f3e5f5,#e1bee7)' }}>
              <CardContent sx={{ textAlign: 'center', py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <EmptyFolderIcon sx={{ fontSize: 24, opacity: 0.8 }} />
                <Typography variant="body1" sx={{ fontWeight: 700 }}>{empties.length}</Typography>
                <Typography variant="caption" sx={{ opacity: 0.8, fontSize: '0.65rem' }}>Пустые</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={6} sm={4} md={2}>
            <Card elevation={1} sx={{ borderRadius: 3, background: (t) => t.palette.mode === 'dark' ? 'linear-gradient(135deg,#0d47a1,#1565c0)' : 'linear-gradient(135deg,#e3f2fd,#bbdefb)' }}>
              <CardContent sx={{ textAlign: 'center', py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <RegistryIcon sx={{ fontSize: 24, opacity: 0.8 }} />
                <Typography variant="body1" sx={{ fontWeight: 700 }}>{regKeys.length}</Typography>
                <Typography variant="caption" sx={{ opacity: 0.8, fontSize: '0.65rem' }}>Реестр</Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}

      {/* Tabs + Filter — вкладки всегда видны (можно сразу перейти к "Ярлыки",
          не дожидаясь скана остатков — это независимый источник данных). */}
      <Tabs
        value={activeTab}
        onChange={(_, v) => setActiveTab(v)}
        variant="scrollable"
        scrollButtons="auto"
        sx={{
          mb: 1.5,
          bgcolor: (t) => (t.palette.mode === 'dark' ? '#1e293b' : '#ffffff'),
          border: '1px solid',
          borderColor: 'divider',
          borderRadius: 3,
          p: 0.5,
          '& .MuiTab-root': { textTransform: 'none', fontWeight: 600, minHeight: 36, borderRadius: 2, fontSize: '0.8rem', py: 0 },
        }}
      >
        <Tab label={`Кеш (${cacheItems.length})`} icon={<CacheIcon sx={{ fontSize: 16 }} />} iconPosition="start" />
        <Tab label={`Удалённые программы (${removedProgramItems.length})`} icon={<KnownIcon sx={{ fontSize: 16 }} />} iconPosition="start" />
        <Tab label={`Неизвестные (${unknownFolders.length})`} icon={<UnknownIcon sx={{ fontSize: 16 }} />} iconPosition="start" />
        <Tab label={`Пустые (${empties.length})`} icon={<EmptyFolderIcon sx={{ fontSize: 16 }} />} iconPosition="start" />
        <Tab label={`Реестр (${regKeys.length})`} icon={<RegistryIcon sx={{ fontSize: 16 }} />} iconPosition="start" />
        <Tab label={`Ярлыки${shortcutsBroken.length ? ` (${shortcutsBroken.length})` : ''}`} icon={<ShortcutsIcon sx={{ fontSize: 16 }} />} iconPosition="start" />
      </Tabs>
      {activeTab !== 5 && (
        <TextField
          fullWidth
          size="small"
          placeholder="Фильтр по имени или пути..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          sx={{ mb: 2 }}
          InputProps={{
            startAdornment: (<InputAdornment position="start"><SearchIcon sx={{ opacity: 0.5 }} /></InputAdornment>),
            endAdornment: filter ? (<InputAdornment position="end"><IconButton size="small" onClick={() => setFilter('')}><ClearIcon fontSize="small" /></IconButton></InputAdornment>) : null,
            sx: { borderRadius: 2 },
          }}
        />
      )}

      {!result && !scanning && activeTab !== 5 && (
        <Paper elevation={0} sx={{ p: 4, textAlign: 'center', borderRadius: 2, border: '1px solid', borderColor: (t) => (t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0') }}>
          <SearchIcon sx={{ fontSize: 48, opacity: 0.2, mb: 1 }} />
          <Typography variant="body2" sx={{ opacity: 0.5 }}>
            Нажмите «Сканировать остатки», чтобы найти кеш, остатки удалённых программ, неизвестные и пустые папки, ключи реестра.
          </Typography>
        </Paper>
      )}

      {/* Tab 0: Cache items (safe to delete) */}
      {activeTab === 0 && cacheItems.length > 0 && (
        <Paper variant="outlined" sx={{ maxHeight: 420, overflow: 'auto', borderRadius: 3, borderColor: 'divider' }}>
          <Alert severity="info" sx={{ m: 1, borderRadius: 1.5 }}>
            Кеш-файлы безопасно удалять — это временные данные, не затрагивающие настройки.
          </Alert>
          <List dense disablePadding>
            {cacheItems.map((item, idx) => (
              <React.Fragment key={item.path}>
                <ListItem
                  sx={{ py: 1.5, px: 2, '&:hover': { bgcolor: 'action.hover' } }}
                  secondaryAction={
                    <Tooltip title="Удалить кеш">
                      <IconButton edge="end" color="info" onClick={() => { setDeleteError(null); setDeleteTarget(item); }} size="small"
                        sx={{ border: '1px solid', borderColor: 'info.main', borderRadius: 1.5, '&:hover': { bgcolor: 'info.main', color: 'white' } }}>
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  }
                >
                  <Box sx={{ mr: 1.5, color: 'text.secondary', minWidth: 28, textAlign: 'center' }}>
                    <CacheIcon sx={{ fontSize: 18, opacity: 0.6 }} />
                  </Box>
                  <ListItemText
                    primary={item.directoryName}
                    secondary={item.path}
                    primaryTypographyProps={{ variant: 'body2', sx: { fontWeight: 600 } }}
                    secondaryTypographyProps={{ variant: 'caption', sx: { wordBreak: 'break-all', opacity: 0.7 } }}
                  />
                  <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center', mr: 2, flexShrink: 0 }}>
                    {item.orphanMatch && <Chip label={item.orphanMatch === 'cache' ? 'кеш' : item.orphanMatch} color="info" size="small" variant="outlined" sx={{ fontWeight: 600, fontSize: '0.7rem' }} />}
                    <Chip label={item.sizeFormatted} color="info" size="small" sx={{ fontWeight: 600 }} />
                  </Box>
                </ListItem>
                {idx < cacheItems.length - 1 && <Divider />}
              </React.Fragment>
            ))}
          </List>
        </Paper>
      )}

      {/* Tab 1: Known orphan items — сначала удалённые программы (приоритет), затем ещё установленные */}
      {activeTab === 1 && orphanKnown.length > 0 && (
        <Paper variant="outlined" sx={{ maxHeight: 420, overflow: 'auto', borderRadius: 3, borderColor: 'divider' }}>
          {removedProgramItems.length > 0 && (
            <>
              <Alert severity="success" icon={<KnownIcon />} sx={{ m: 1, borderRadius: 1.5 }}>
                ★ Программа удалена из системы — приоритетный кандидат на очистку ({removedProgramItems.length}).
              </Alert>
              <List dense disablePadding>
                {removedProgramItems.map((item, idx) => (
                  <React.Fragment key={item.path}>
                    <ListItem
                      sx={{ py: 1.5, px: 2, '&:hover': { bgcolor: 'action.hover' } }}
                      secondaryAction={
                        <Tooltip title="Удалить">
                          <IconButton edge="end" color="error" onClick={() => { setDeleteError(null); setDeleteTarget(item); }} size="small"
                            sx={{ border: '1px solid', borderColor: 'error.main', borderRadius: 1.5, '&:hover': { bgcolor: 'error.main', color: 'white' } }}>
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      }
                    >
                      <Box sx={{ mr: 1.5, color: 'text.secondary', minWidth: 28, textAlign: 'center' }}>
                        <KnownIcon sx={{ fontSize: 18, color: 'success.main' }} />
                      </Box>
                      <ListItemText
                        primary={item.directoryName}
                        secondary={item.path}
                        primaryTypographyProps={{ variant: 'body2', sx: { fontWeight: 600 } }}
                        secondaryTypographyProps={{ variant: 'caption', sx: { wordBreak: 'break-all', opacity: 0.7 } }}
                      />
                      <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center', mr: 2, flexShrink: 0 }}>
                        <Chip label={item.orphanMatch || 'остаток'} color="success" size="small" variant="outlined" sx={{ fontWeight: 600, fontSize: '0.7rem', maxWidth: 160 }} />
                        <Chip label={item.sizeFormatted} color="success" size="small" sx={{ fontWeight: 600 }} />
                      </Box>
                    </ListItem>
                    {idx < removedProgramItems.length - 1 && <Divider />}
                  </React.Fragment>
                ))}
              </List>
            </>
          )}

          {installedProgramItems.length > 0 && (
            <>
              <Alert severity="warning" sx={{ m: 1, borderRadius: 1.5 }}>
                Программа всё ещё установлена — это действующие данные, а не мусор. Удаляйте осторожно ({installedProgramItems.length}).
              </Alert>
              <List dense disablePadding>
                {installedProgramItems.map((item, idx) => (
                  <React.Fragment key={item.path}>
                    <ListItem
                      sx={{ py: 1.5, px: 2, '&:hover': { bgcolor: 'action.hover' } }}
                      secondaryAction={
                        <Tooltip title="Удалить">
                          <IconButton edge="end" color="error" onClick={() => { setDeleteError(null); setDeleteTarget(item); }} size="small"
                            sx={{ border: '1px solid', borderColor: 'error.main', borderRadius: 1.5, '&:hover': { bgcolor: 'error.main', color: 'white' } }}>
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      }
                    >
                      <Box sx={{ mr: 1.5, color: 'text.secondary', minWidth: 28, textAlign: 'center' }}>
                        <KnownIcon sx={{ fontSize: 18, color: 'warning.main' }} />
                      </Box>
                      <ListItemText
                        primary={item.directoryName}
                        secondary={item.path}
                        primaryTypographyProps={{ variant: 'body2', sx: { fontWeight: 600 } }}
                        secondaryTypographyProps={{ variant: 'caption', sx: { wordBreak: 'break-all', opacity: 0.7 } }}
                      />
                      <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center', mr: 2, flexShrink: 0 }}>
                        <Chip label={item.orphanMatch || 'остаток'} color="warning" size="small" variant="outlined" sx={{ fontWeight: 600, fontSize: '0.7rem', maxWidth: 160 }} />
                        <Chip label={item.sizeFormatted} color="warning" size="small" sx={{ fontWeight: 600 }} />
                      </Box>
                    </ListItem>
                    {idx < installedProgramItems.length - 1 && <Divider />}
                  </React.Fragment>
                ))}
              </List>
            </>
          )}
        </Paper>
      )}

      {/* Tab 2: Unknown folders */}
      {activeTab === 2 && unknownFolders.length > 0 && (
        <Paper variant="outlined" sx={{ maxHeight: 420, overflow: 'auto', borderRadius: 3, borderColor: 'divider' }}>
          <Alert severity="error" sx={{ m: 1, borderRadius: 1.5 }}>
            Эти папки не найдены в orphan DB. Проверьте вручную перед удалением.
          </Alert>
          <List dense disablePadding>
            {unknownFolders.map((item, idx) => (
              <React.Fragment key={item.path}>
                <ListItem
                  sx={{ py: 1.5, px: 2, '&:hover': { bgcolor: 'action.hover' } }}
                  secondaryAction={
                    <Stack direction="row" spacing={0.5} alignItems="center">
                      <Tooltip title={trackedPaths.has(item.path) ? 'Уже добавлено в orphaned_apps.json' : 'Добавить в отслеживание (orphaned_apps.json), чтобы система запоминала эту программу'}>
                        <span>
                          <IconButton
                            size="small"
                            color={trackedPaths.has(item.path) ? 'success' : 'default'}
                            disabled={trackedPaths.has(item.path) || trackingPath === item.path}
                            onClick={() => handleTrack(item.path)}
                          >
                            {trackingPath === item.path ? (
                              <CircularProgress size={16} />
                            ) : trackedPaths.has(item.path) ? (
                              <TrackedIcon sx={{ fontSize: 18 }} />
                            ) : (
                              <TrackIcon sx={{ fontSize: 18 }} />
                            )}
                          </IconButton>
                        </span>
                      </Tooltip>
                      <Tooltip title="Удалить папку">
                        <IconButton edge="end" color="error" onClick={() => { setDeleteError(null); setDeleteTarget(item); }} size="small"
                          sx={{ border: '1px solid', borderColor: 'error.main', borderRadius: 1.5, '&:hover': { bgcolor: 'error.main', color: 'white' } }}>
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    </Stack>
                  }
                >
                  <Box sx={{ mr: 1.5, color: 'text.secondary', minWidth: 28, textAlign: 'center' }}>
                    <UnknownIcon sx={{ fontSize: 18, opacity: 0.6 }} />
                  </Box>
                  <ListItemText
                    primary={
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                        {item.directoryName}
                        {item.likelyUserData && (
                          <Tooltip title="Похоже на пользовательские данные (сохранения, проекты, фото и т.п.) — проверьте особенно внимательно">
                            <WarningIcon sx={{ fontSize: 14, color: 'warning.main' }} />
                          </Tooltip>
                        )}
                      </Box>
                    }
                    secondary={item.path}
                    primaryTypographyProps={{ variant: 'body2', sx: { fontWeight: 600 } }}
                    secondaryTypographyProps={{ variant: 'caption', sx: { wordBreak: 'break-all', opacity: 0.7 } }}
                  />
                  <Box sx={{ textAlign: 'right', mr: 2, flexShrink: 0 }}>
                    <Chip
                      label={item.sizeFormatted}
                      color={item.sizeUnknown ? 'info' : item.sizeBytes > 1024 ** 3 ? 'error' : item.sizeBytes > 100 * 1024 ** 2 ? 'warning' : 'default'}
                      size="small"
                      sx={{ fontWeight: 600 }}
                    />
                    <Typography variant="caption" display="block" sx={{ mt: 0.5, opacity: 0.7 }}>
                      {item.fileCount} файлов
                    </Typography>
                  </Box>
                </ListItem>
                {idx < unknownFolders.length - 1 && <Divider />}
              </React.Fragment>
            ))}
          </List>
        </Paper>
      )}

      {/* Tab 3: Empty folders */}
      {activeTab === 3 && empties.length > 0 && (
        <Paper variant="outlined" sx={{ maxHeight: 420, overflow: 'auto', borderRadius: 3, borderColor: 'divider' }}>
          <List dense disablePadding>
            {empties.map((item, idx) => (
              <React.Fragment key={item.path}>
                <ListItem
                  sx={{ py: 1.2, px: 2, '&:hover': { bgcolor: 'action.hover' } }}
                  secondaryAction={
                    <Tooltip title="Удалить пустую папку">
                      <IconButton edge="end" color="error" onClick={() => { setDeleteError(null); setDeleteTarget(item); }} size="small"
                        sx={{ border: '1px solid', borderColor: 'error.main', borderRadius: 1.5, '&:hover': { bgcolor: 'error.main', color: 'white' } }}>
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  }
                >
                  <EmptyFolderIcon sx={{ mr: 1.5, fontSize: 20, opacity: 0.5 }} />
                  <ListItemText
                    primary={item.directoryName}
                    secondary={item.path}
                    primaryTypographyProps={{ variant: 'body2', sx: { fontWeight: 600 } }}
                    secondaryTypographyProps={{ variant: 'caption', sx: { wordBreak: 'break-all', opacity: 0.7 } }}
                  />
                  <Chip label="пусто" size="small" variant="outlined" sx={{ ml: 1 }} />
                </ListItem>
                {idx < empties.length - 1 && <Divider />}
              </React.Fragment>
            ))}
          </List>
        </Paper>
      )}

      {/* Tab 4: Registry keys */}
      {activeTab === 4 && regKeys.length > 0 && (
        <Paper variant="outlined" sx={{ maxHeight: 420, overflow: 'auto', borderRadius: 3, borderColor: 'divider' }}>
          <Alert severity="info" sx={{ m: 1, borderRadius: 1.5 }}>
            Ключи реестра можно удалить через regedit или reg delete. Автоудаление не поддерживается.
          </Alert>
          <List dense disablePadding>
            {regKeys.map((item, idx) => (
              <React.Fragment key={item.path}>
                <ListItem sx={{ py: 1.2, px: 2, '&:hover': { bgcolor: 'action.hover' } }}>
                  <RegistryIcon sx={{ mr: 1.5, fontSize: 20, opacity: 0.5 }} />
                  <ListItemText
                    primary={item.directoryName}
                    secondary={item.path}
                    primaryTypographyProps={{ variant: 'body2', sx: { fontWeight: 600 } }}
                    secondaryTypographyProps={{ variant: 'caption', sx: { wordBreak: 'break-all', opacity: 0.7 } }}
                  />
                  <Chip label="реестр" size="small" color="info" variant="outlined" sx={{ ml: 1 }} />
                </ListItem>
                {idx < regKeys.length - 1 && <Divider />}
              </React.Fragment>
            ))}
          </List>
        </Paper>
      )}

      {/* Tab 5: Broken shortcuts */}
      {activeTab === 5 && shortcutsBroken.length > 0 && (
        <Paper variant="outlined" sx={{ maxHeight: 420, overflow: 'auto', borderRadius: 3, borderColor: 'divider' }}>
          <List dense disablePadding>
            {shortcutsBroken.map((item, idx) => (
              <React.Fragment key={item.path}>
                <ListItem
                  sx={{ py: 1.5, px: 2, '&:hover': { bgcolor: 'action.hover' } }}
                  secondaryAction={
                    <Tooltip title="Удалить ярлык">
                      <IconButton edge="end" color="error" size="small"
                        onClick={() => { setShortcutDeleteError(null); setShortcutDeleteTarget(item); }}
                        sx={{ border: '1px solid', borderColor: 'error.main', borderRadius: 1.5, '&:hover': { bgcolor: 'error.main', color: 'white' } }}>
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  }
                >
                  <Box sx={{ mr: 1.5, color: 'text.secondary', minWidth: 28, textAlign: 'center' }}>
                    <ShortcutsIcon sx={{ fontSize: 18, color: '#ef4444' }} />
                  </Box>
                  <ListItemText
                    primary={item.path.split('\\').pop()}
                    secondary={item.targetPath ? `${item.path} → ${item.targetPath}` : item.path}
                    primaryTypographyProps={{ variant: 'body2', sx: { fontWeight: 600 } }}
                    secondaryTypographyProps={{ variant: 'caption', sx: { wordBreak: 'break-all', opacity: 0.7 } }}
                  />
                </ListItem>
                {idx < shortcutsBroken.length - 1 && <Divider />}
              </React.Fragment>
            ))}
          </List>
        </Paper>
      )}
      {activeTab === 5 && shortcutsScanned === null && !shortcutsScanning && (
        <Paper elevation={0} sx={{ p: 4, textAlign: 'center', borderRadius: 2, border: '1px solid', borderColor: (t) => (t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0') }}>
          <ShortcutsIcon sx={{ fontSize: 48, opacity: 0.2, mb: 1 }} />
          <Typography variant="body2" sx={{ opacity: 0.5 }}>
            Проверяет ярлыки (.lnk) на рабочем столе и в меню Пуск — те, что указывают на несуществующие файлы.
          </Typography>
          <Typography variant="caption" sx={{ opacity: 0.3, display: 'block', mt: 1 }}>
            Ярлыки на временно недоступные сетевые пути тоже могут попасть в список — проверьте перед удалением.
          </Typography>
        </Paper>
      )}
      {activeTab === 5 && !shortcutsScanning && shortcutsScanned !== null && shortcutsBroken.length === 0 && (
        <Alert severity="success" sx={{ borderRadius: 2 }}>Битых ярлыков не найдено.</Alert>
      )}

      {/* Empty state per tab (0-4) */}
      {!scanning && result && totalItems > 0 && activeTab !== 5 && (
        (activeTab === 0 && cacheItems.length === 0) ||
        (activeTab === 1 && orphanKnown.length === 0) ||
        (activeTab === 2 && unknownFolders.length === 0) ||
        (activeTab === 3 && empties.length === 0) ||
        (activeTab === 4 && regKeys.length === 0)
      ) && (
        <Alert severity="success" sx={{ borderRadius: 2 }}>
          {filter ? 'Ничего не найдено по фильтру.' : 'Нет элементов в этой категории.'}
        </Alert>
      )}

      {/* No results at all */}
      {!scanning && result && totalItems === 0 && activeTab !== 5 && (
        <Alert severity="success" sx={{ borderRadius: 2 }}>
          {deletedPaths.size > 0
            ? 'Все остатки удалены!'
            : 'Остатков не найдено! Система чиста.'}
        </Alert>
      )}

      {/* Delete Dialog */}
      <Dialog
        open={!!deleteTarget}
        onClose={() => !deleting && setDeleteTarget(null)}
        maxWidth="sm"
        fullWidth
        PaperProps={{ sx: { borderRadius: 2 } }}
      >
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <WarningIcon color="warning" />
          Подтвердите удаление
        </DialogTitle>
        <DialogContent>
          {deleteTarget && (
            <Box>
              <Typography gutterBottom>
                Вы собираетесь удалить {deleteTarget.isEmpty ? 'пустую папку' : 'папку'}:
              </Typography>
              <Paper variant="outlined" sx={{ p: 1.5, mb: 2, bgcolor: 'action.hover', borderRadius: 1.5 }}>
                <Typography variant="body2" sx={{ fontWeight: 600, wordBreak: 'break-all' }}>
                  {deleteTarget.path}
                </Typography>
                {deleteTarget.isFolder && (
                  <Typography variant="caption" color="text.secondary">
                    {deleteTarget.sizeFormatted} / {deleteTarget.fileCount} файлов
                  </Typography>
                )}
              </Paper>
              {deleteTarget.likelyUserData && (
                <Alert severity="warning" sx={{ borderRadius: 1.5, mb: 1 }}>
                  ⚠ Похоже на пользовательские данные (сохранения, проекты, фото и т.п.) — проверьте содержимое перед удалением.
                </Alert>
              )}
              <Alert severity="success" sx={{ borderRadius: 1.5 }}>
                Папка будет перемещена в Корзину (можно восстановить).
              </Alert>
              {deleteError && (
                <Alert severity="warning" sx={{ mt: 1, borderRadius: 1.5 }}>{deleteError}</Alert>
              )}
            </Box>
          )}
          {deleting && <LinearProgress sx={{ mt: 2 }} />}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDeleteTarget(null)} disabled={deleting} sx={{ borderRadius: 1.5 }}>Отмена</Button>
          <Button onClick={handleDelete} variant="contained" color="warning" disabled={deleting}
            startIcon={deleting ? <CircularProgress size={16} /> : <DeleteIcon />} sx={{ borderRadius: 1.5 }}>
            {deleting ? 'Удаление...' : 'В Корзину'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete Shortcut Dialog */}
      <Dialog
        open={!!shortcutDeleteTarget}
        onClose={() => !shortcutDeleting && setShortcutDeleteTarget(null)}
        maxWidth="sm"
        fullWidth
        PaperProps={{ sx: { borderRadius: 2 } }}
      >
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <WarningIcon color="warning" />
          Удалить ярлык?
        </DialogTitle>
        <DialogContent>
          {shortcutDeleteTarget && (
            <Box>
              <Paper variant="outlined" sx={{ p: 1.5, mb: 2, bgcolor: 'action.hover', borderRadius: 1.5 }}>
                <Typography variant="body2" sx={{ fontWeight: 600, wordBreak: 'break-all' }}>
                  {shortcutDeleteTarget.path}
                </Typography>
                {shortcutDeleteTarget.targetPath && (
                  <Typography variant="caption" color="text.secondary" sx={{ wordBreak: 'break-all', display: 'block' }}>
                    цель: {shortcutDeleteTarget.targetPath}
                  </Typography>
                )}
              </Paper>
              <Alert severity="success" sx={{ borderRadius: 1.5 }}>
                Ярлык будет перемещён в Корзину (можно восстановить).
              </Alert>
              {shortcutDeleteError && (
                <Alert severity="warning" sx={{ mt: 1, borderRadius: 1.5 }}>{shortcutDeleteError}</Alert>
              )}
            </Box>
          )}
          {shortcutDeleting && <LinearProgress sx={{ mt: 2 }} />}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setShortcutDeleteTarget(null)} disabled={shortcutDeleting} sx={{ borderRadius: 1.5 }}>Отмена</Button>
          <Button onClick={handleDeleteShortcut} variant="contained" color="warning" disabled={shortcutDeleting}
            startIcon={shortcutDeleting ? <CircularProgress size={16} /> : <DeleteIcon />} sx={{ borderRadius: 1.5 }}>
            {shortcutDeleting ? 'Удаление...' : 'В Корзину'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
