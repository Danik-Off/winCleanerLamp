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
  Folder as FolderIcon,
  Storage as StorageIcon,
  FolderOff as EmptyFolderIcon,
  AppRegistration as RegistryIcon,
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

  const totalItems = folders.length + empties.length + regKeys.length;

  return (
    <Box>
      {/* Header */}
      <Paper
        elevation={3}
        sx={{
          p: 3,
          mb: 2,
          background: (theme) =>
            theme.palette.mode === 'dark'
              ? 'linear-gradient(135deg, #1a237e 0%, #311b92 100%)'
              : 'linear-gradient(135deg, #e3f2fd 0%, #ede7f6 100%)',
          borderRadius: 2,
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
          <FolderDeleteIcon sx={{ mr: 1.5, fontSize: 28 }} />
          <Typography variant="h6" sx={{ fontWeight: 700 }}>
            Остатки удалённых программ
          </Typography>
        </Box>
        <Alert severity="warning" icon={<WarningIcon />} sx={{ mb: 2, borderRadius: 1.5 }}>
          Это эвристика. Сканируются AppData, ProgramData, Program Files, реестр HKCU\Software.
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

      {scanning && <LinearProgress sx={{ mb: 2, borderRadius: 1 }} />}

      {/* Stat Cards */}
      {result && totalItems > 0 && (
        <Grid container spacing={2} sx={{ mb: 2 }}>
          <Grid item xs={3}>
            <Card elevation={2} sx={{ borderRadius: 2, background: (t) => t.palette.mode === 'dark' ? 'linear-gradient(135deg,#1b5e20,#2e7d32)' : 'linear-gradient(135deg,#e8f5e9,#c8e6c9)' }}>
              <CardContent sx={{ textAlign: 'center', py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <StorageIcon sx={{ fontSize: 28, opacity: 0.8 }} />
                <Typography variant="h6" sx={{ fontWeight: 700 }}>{formatBytes(totalBytes)}</Typography>
                <Typography variant="caption" sx={{ opacity: 0.8 }}>Размер папок</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={3}>
            <Card elevation={2} sx={{ borderRadius: 2, background: (t) => t.palette.mode === 'dark' ? 'linear-gradient(135deg,#e65100,#f57c00)' : 'linear-gradient(135deg,#fff3e0,#ffe0b2)' }}>
              <CardContent sx={{ textAlign: 'center', py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <FolderIcon sx={{ fontSize: 28, opacity: 0.8 }} />
                <Typography variant="h6" sx={{ fontWeight: 700 }}>{folders.length}</Typography>
                <Typography variant="caption" sx={{ opacity: 0.8 }}>Папок-остатков</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={3}>
            <Card elevation={2} sx={{ borderRadius: 2, background: (t) => t.palette.mode === 'dark' ? 'linear-gradient(135deg,#4a148c,#6a1b9a)' : 'linear-gradient(135deg,#f3e5f5,#e1bee7)' }}>
              <CardContent sx={{ textAlign: 'center', py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <EmptyFolderIcon sx={{ fontSize: 28, opacity: 0.8 }} />
                <Typography variant="h6" sx={{ fontWeight: 700 }}>{empties.length}</Typography>
                <Typography variant="caption" sx={{ opacity: 0.8 }}>Пустых папок</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={3}>
            <Card elevation={2} sx={{ borderRadius: 2, background: (t) => t.palette.mode === 'dark' ? 'linear-gradient(135deg,#0d47a1,#1565c0)' : 'linear-gradient(135deg,#e3f2fd,#bbdefb)' }}>
              <CardContent sx={{ textAlign: 'center', py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <RegistryIcon sx={{ fontSize: 28, opacity: 0.8 }} />
                <Typography variant="h6" sx={{ fontWeight: 700 }}>{regKeys.length}</Typography>
                <Typography variant="caption" sx={{ opacity: 0.8 }}>Ключей реестра</Typography>
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
            sx={{ mb: 1.5, '& .MuiTab-root': { textTransform: 'none', fontWeight: 600, minHeight: 40 } }}
          >
            <Tab label={`Папки (${folders.length})`} icon={<FolderIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
            <Tab label={`Пустые (${empties.length})`} icon={<EmptyFolderIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
            <Tab label={`Реестр (${regKeys.length})`} icon={<RegistryIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
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

      {/* Tab 0: Folders */}
      {activeTab === 0 && folders.length > 0 && (
        <Paper variant="outlined" sx={{ maxHeight: 420, overflow: 'auto', borderRadius: 2 }}>
          <List dense disablePadding>
            {folders.map((item, idx) => (
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
                    <Typography variant="caption" sx={{ fontWeight: 600 }}>{idx + 1}</Typography>
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
                {idx < folders.length - 1 && <Divider />}
              </React.Fragment>
            ))}
          </List>
        </Paper>
      )}

      {/* Tab 1: Empty folders */}
      {activeTab === 1 && empties.length > 0 && (
        <Paper variant="outlined" sx={{ maxHeight: 420, overflow: 'auto', borderRadius: 2 }}>
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

      {/* Tab 2: Registry keys */}
      {activeTab === 2 && regKeys.length > 0 && (
        <Paper variant="outlined" sx={{ maxHeight: 420, overflow: 'auto', borderRadius: 2 }}>
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
        (activeTab === 0 && folders.length === 0) ||
        (activeTab === 1 && empties.length === 0) ||
        (activeTab === 2 && regKeys.length === 0)
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
