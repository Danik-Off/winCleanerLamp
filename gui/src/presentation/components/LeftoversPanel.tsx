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
} from '@mui/material';
import {
  FolderDelete as FolderDeleteIcon,
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
} from '@mui/icons-material';
import { useLeftovers } from '../hooks';
import type { LeftoverItem } from '@domain/index';

interface LeftoversPanelProps {
  onError: (error: string) => void;
}

export function LeftoversPanel({ onError }: LeftoversPanelProps): JSX.Element {
  const { scanning, result, error, scan } = useLeftovers();
  const [filter, setFilter] = useState('');
  const [deleteTarget, setDeleteTarget] = useState<LeftoverItem | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deletedPaths, setDeletedPaths] = useState<Set<string>>(new Set());
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState(0);

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
  const orphanKnown = useMemo(
    () => (result ? applyFilter(result.orphanKnownItems) : []),
    [result, applyFilter]
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
      {/* Header */}
      <Paper
        elevation={2}
        sx={{
          p: 3,
          mb: 2,
          background: (theme) =>
            theme.palette.mode === 'dark'
              ? 'linear-gradient(135deg, #1a237e 0%, #311b92 100%)'
              : 'linear-gradient(135deg, #e3f2fd 0%, #ede7f6 100%)',
          borderRadius: 3,
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
          <FolderDeleteIcon sx={{ mr: 1.5, fontSize: 28 }} />
          <Typography variant="h6" sx={{ fontWeight: 700, color: 'text.primary' }}>
            Остатки удалённых программ
          </Typography>
        </Box>
        <Alert severity="info" icon={<WarningIcon />} sx={{ mb: 2, borderRadius: 1.5 }}>
          Поиск остатков + кеш из orphan DB. Сканируются AppData, ProgramData, Program Files, реестр.
        </Alert>
        <Button
          variant="contained"
          onClick={handleScan}
          disabled={scanning}
          startIcon={scanning ? <CircularProgress size={20} /> : <RefreshIcon />}
          size="large"
          sx={{ borderRadius: 2, textTransform: 'none', fontWeight: 600 }}
        >
          {scanning ? 'Сканирование...' : 'Сканировать остатки'}
        </Button>
      </Paper>

      {scanning && <LinearProgress sx={{ mb: 2, borderRadius: 2, height: 6 }} />}

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

      {/* Tabs + Filter */}
      {result && totalItems > 0 && (
        <>
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
            <Tab label={`Известные (${orphanKnown.length})`} icon={<KnownIcon sx={{ fontSize: 16 }} />} iconPosition="start" />
            <Tab label={`Неизвестные (${unknownFolders.length})`} icon={<UnknownIcon sx={{ fontSize: 16 }} />} iconPosition="start" />
            <Tab label={`Пустые (${empties.length})`} icon={<EmptyFolderIcon sx={{ fontSize: 16 }} />} iconPosition="start" />
            <Tab label={`Реестр (${regKeys.length})`} icon={<RegistryIcon sx={{ fontSize: 16 }} />} iconPosition="start" />
          </Tabs>
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
        </>
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

      {/* Tab 1: Known orphan items */}
      {activeTab === 1 && orphanKnown.length > 0 && (
        <Paper variant="outlined" sx={{ maxHeight: 420, overflow: 'auto', borderRadius: 3, borderColor: 'divider' }}>
          <Alert severity="warning" sx={{ m: 1, borderRadius: 1.5 }}>
            Эти папки известны из orphan DB. Удаление может затронуть настройки и профили.
          </Alert>
          <List dense disablePadding>
            {orphanKnown.map((item, idx) => (
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
                {idx < orphanKnown.length - 1 && <Divider />}
              </React.Fragment>
            ))}
          </List>
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
                    <Tooltip title="Удалить папку">
                      <IconButton edge="end" color="error" onClick={() => { setDeleteError(null); setDeleteTarget(item); }} size="small"
                        sx={{ border: '1px solid', borderColor: 'error.main', borderRadius: 1.5, '&:hover': { bgcolor: 'error.main', color: 'white' } }}>
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  }
                >
                  <Box sx={{ mr: 1.5, color: 'text.secondary', minWidth: 28, textAlign: 'center' }}>
                    <UnknownIcon sx={{ fontSize: 18, opacity: 0.6 }} />
                  </Box>
                  <ListItemText
                    primary={item.directoryName}
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

      {/* Empty state per tab */}
      {!scanning && result && totalItems > 0 && (
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
      {!scanning && result && totalItems === 0 && (
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
          <WarningIcon color="error" />
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
              <Alert severity="error" sx={{ borderRadius: 1.5 }}>
                Это действие необратимо! Папка будет удалена безвозвратно.
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
          <Button onClick={handleDelete} variant="contained" color="error" disabled={deleting}
            startIcon={deleting ? <CircularProgress size={16} /> : <DeleteIcon />} sx={{ borderRadius: 1.5 }}>
            {deleting ? 'Удаление...' : 'Удалить'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
