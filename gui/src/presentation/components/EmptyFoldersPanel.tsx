/**
 * Empty Folders Panel Component
 * Finds and displays empty directories with option to delete
 */
import React, { useState, useCallback, useMemo } from 'react';
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
  ListItemIcon,
} from '@mui/material';
import {
  FolderOff as EmptyFolderIcon,
  Refresh as RefreshIcon,
  Warning as WarningIcon,
  Search as SearchIcon,
  Delete as DeleteIcon,
  Clear as ClearIcon,
  FolderOpen as FolderOpenIcon,
  RestoreFromTrash as RestoreIcon,
} from '@mui/icons-material';

interface EmptyFoldersPanelProps {
  onError: (error: string) => void;
}

export function EmptyFoldersPanel({ onError }: EmptyFoldersPanelProps): JSX.Element {
  const [scanning, setScanning] = useState(false);
  const [rootPath, setRootPath] = useState('');
  const [dirs, setDirs] = useState<string[]>([]);
  const [hasScanResult, setHasScanResult] = useState(false);
  const [filter, setFilter] = useState('');

  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deletedPaths, setDeletedPaths] = useState<Set<string>>(new Set());
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deleteResults, setDeleteResults] = useState<Array<{path: string; success: boolean; movedToRecycleBin: boolean}>>([]);
  const [showReport, setShowReport] = useState(false);

  const handleOpenInExplorer = useCallback((dirPath: string) => {
    // Открыть папку в Проводнике
    window.electronAPI.openExternal(dirPath);
  }, []);

  const handleScan = useCallback(async () => {
    const paths = rootPath.trim();
    if (!paths) {
      onError('Укажите папки для сканирования');
      return;
    }
    setScanning(true);
    setDirs([]);
    setHasScanResult(false);
    setDeletedPaths(new Set());
    try {
      const output = await window.electronAPI.getEmptyDirs(paths);
      const parsed = parseEmptyDirsOutput(output);
      setDirs(parsed);
      setHasScanResult(true);
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Ошибка сканирования');
    } finally {
      setScanning(false);
    }
  }, [rootPath, onError]);

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    setDeleteError(null);
    try {
      const res = await window.electronAPI.deleteEmptyDir(deleteTarget);
      if (res.success) {
        setDeletedPaths(prev => new Set(prev).add(deleteTarget));
        setDeleteResults(prev => [...prev, {
          path: deleteTarget,
          success: true,
          movedToRecycleBin: res.movedToRecycleBin || false,
        }]);
        setDeleteTarget(null);
      } else {
        setDeleteError(res.error || 'Ошибка удаления');
      }
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Ошибка');
    } finally {
      setDeleting(false);
    }
  }, [deleteTarget]);

  const handleDeleteAll = useCallback(async () => {
    const alive = dirs.filter(d => !deletedPaths.has(d));
    const lf = filter.toLowerCase().trim();
    const toDelete = lf ? alive.filter(d => d.toLowerCase().includes(lf)) : alive;
    
    let successCount = 0;
    let errorCount = 0;
    const results: Array<{path: string; success: boolean; movedToRecycleBin: boolean}> = [];

    for (const d of toDelete) {
      try {
        const res = await window.electronAPI.deleteEmptyDir(d);
        if (res.success) {
          setDeletedPaths(prev => new Set(prev).add(d));
          successCount++;
          results.push({ path: d, success: true, movedToRecycleBin: res.movedToRecycleBin || false });
        } else {
          errorCount++;
          results.push({ path: d, success: false, movedToRecycleBin: false });
        }
      } catch {
        errorCount++;
        results.push({ path: d, success: false, movedToRecycleBin: false });
      }
    }

    setDeleteResults(results);
    setShowReport(true);
  }, [dirs, filter, deletedPaths]);

  const filteredDirs = useMemo(() => {
    const alive = dirs.filter(d => !deletedPaths.has(d));
    if (!filter.trim()) return alive;
    const lf = filter.toLowerCase();
    return alive.filter(d => d.toLowerCase().includes(lf));
  }, [dirs, filter, deletedPaths]);

  return (
    <Box>
      {/* Header */}
      <Paper
        elevation={2}
        sx={{
          p: 3, mb: 2, borderRadius: 3,
          background: (t) => t.palette.mode === 'dark'
            ? 'linear-gradient(135deg, #bf360c 0%, #e64a19 100%)'
            : 'linear-gradient(135deg, #fbe9e7 0%, #ffccbc 100%)',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
          <EmptyFolderIcon sx={{ mr: 1.5, fontSize: 28 }} />
          <Typography variant="h6" sx={{ fontWeight: 700 }}>Поиск пустых папок</Typography>
        </Box>
        <Alert severity="info" sx={{ mb: 2, borderRadius: 1.5 }}>
          Находит пустые папки (включая содержащие только Thumbs.db, desktop.ini). 
          <Box component="span" sx={{ display: 'block', mt: 0.5, fontSize: '0.85em' }}>
            ✅ Удаление в Корзину • ⚠️ Системные папки защищены
          </Box>
        </Alert>
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'flex-start' }}>
          <TextField
            fullWidth
            size="small"
            placeholder="C:\Users\ИмяПользователя, D:\Проекты"
            value={rootPath}
            onChange={(e) => setRootPath(e.target.value)}
            InputProps={{ sx: { borderRadius: 2 } }}
          />
          <Button
            variant="contained"
            onClick={handleScan}
            disabled={scanning}
            startIcon={scanning ? <CircularProgress size={20} /> : <RefreshIcon />}
            sx={{ borderRadius: 2, textTransform: 'none', fontWeight: 600, whiteSpace: 'nowrap', minWidth: 160 }}
          >
            {scanning ? 'Поиск...' : 'Найти пустые'}
          </Button>
        </Box>
      </Paper>

      {scanning && <LinearProgress sx={{ mb: 2, borderRadius: 2, height: 6 }} />}

      {/* Stats */}
      {hasScanResult && (
        <Grid container spacing={2} sx={{ mb: 2 }}>
          <Grid item xs={12} sm={6}>
            <Card elevation={1} sx={{ borderRadius: 3, background: (t) => t.palette.mode === 'dark' ? 'linear-gradient(135deg,#bf360c,#e64a19)' : 'linear-gradient(135deg,#fbe9e7,#ffccbc)' }}>
              <CardContent sx={{ textAlign: 'center', py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <EmptyFolderIcon sx={{ fontSize: 28, opacity: 0.8 }} />
                <Typography variant="h6" sx={{ fontWeight: 700 }}>{filteredDirs.length}</Typography>
                <Typography variant="caption" sx={{ opacity: 0.8 }}>Пустых папок</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6}>
            <Card elevation={1} sx={{ borderRadius: 3, background: (t) => t.palette.mode === 'dark' ? 'linear-gradient(135deg,#1b5e20,#2e7d32)' : 'linear-gradient(135deg,#e8f5e9,#c8e6c9)' }}>
              <CardContent sx={{ textAlign: 'center', py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <DeleteIcon sx={{ fontSize: 28, opacity: 0.8 }} />
                <Typography variant="h6" sx={{ fontWeight: 700 }}>{deletedPaths.size}</Typography>
                <Typography variant="caption" sx={{ opacity: 0.8 }}>Удалено</Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}

      {/* Filter + delete all */}
      {hasScanResult && dirs.length > 0 && (
        <Box sx={{ display: 'flex', gap: 1, mb: 2, alignItems: 'center' }}>
          <TextField
            fullWidth size="small"
            placeholder="Фильтр по пути..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            InputProps={{
              startAdornment: <InputAdornment position="start"><SearchIcon sx={{ opacity: 0.5 }} /></InputAdornment>,
              endAdornment: filter ? <InputAdornment position="end"><IconButton size="small" onClick={() => setFilter('')}><ClearIcon fontSize="small" /></IconButton></InputAdornment> : null,
              sx: { borderRadius: 2 },
            }}
          />
          {filteredDirs.length > 0 && (
            <Tooltip title="Удалить все отображаемые пустые папки">
              <Button
                variant="outlined"
                color="error"
                size="small"
                startIcon={<DeleteIcon />}
                onClick={handleDeleteAll}
                sx={{ borderRadius: 2, textTransform: 'none', fontWeight: 600, whiteSpace: 'nowrap' }}
              >
                Удалить все ({filteredDirs.length})
              </Button>
            </Tooltip>
          )}
        </Box>
      )}

      {/* List */}
      {filteredDirs.length > 0 && (
        <Paper variant="outlined" sx={{ maxHeight: 420, overflow: 'auto', borderRadius: 3, borderColor: 'divider' }}>
          <List dense disablePadding>
            {filteredDirs.map((dir, idx) => {
              const name = dir.split('\\').pop() || dir;
              return (
                <React.Fragment key={dir}>
                  <ListItem
                    sx={{ py: 1, px: 2, '&:hover': { bgcolor: 'action.hover' } }}
                    secondaryAction={
                      <Box sx={{ display: 'flex', gap: 0.5 }}>
                        <Tooltip title="Открыть в Проводнике">
                          <IconButton edge="end" size="small"
                            onClick={() => handleOpenInExplorer(dir)}
                            sx={{ color: 'primary.main' }}>
                            <FolderOpenIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="Удалить пустую папку">
                          <IconButton edge="end" color="error" size="small"
                            onClick={() => { setDeleteError(null); setDeleteTarget(dir); }}
                            sx={{ border: '1px solid', borderColor: 'error.main', borderRadius: 1.5, '&:hover': { bgcolor: 'error.main', color: 'white' } }}>
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </Box>
                    }
                  >
                    <ListItemIcon sx={{ minWidth: 36 }}>
                      <EmptyFolderIcon sx={{ fontSize: 20, opacity: 0.5 }} />
                    </ListItemIcon>
                    <ListItemText
                      primary={name}
                      secondary={dir}
                      primaryTypographyProps={{ variant: 'body2', fontWeight: 600 }}
                      secondaryTypographyProps={{ variant: 'caption', sx: { wordBreak: 'break-all', opacity: 0.7 } }}
                    />
                    <Chip label="пусто" size="small" variant="outlined" sx={{ ml: 1 }} />
                  </ListItem>
                  {idx < filteredDirs.length - 1 && <Divider />}
                </React.Fragment>
              );
            })}
          </List>
        </Paper>
      )}

      {/* Empty state */}
      {!scanning && hasScanResult && filteredDirs.length === 0 && (
        <Alert severity="success" sx={{ borderRadius: 2 }}>
          {deletedPaths.size > 0 ? 'Все пустые папки удалены!' : filter ? 'Ничего не найдено по фильтру.' : 'Пустых папок не найдено!'}
        </Alert>
      )}

      {/* Delete Dialog */}
      <Dialog open={!!deleteTarget} onClose={() => !deleting && setDeleteTarget(null)} maxWidth="sm" fullWidth PaperProps={{ sx: { borderRadius: 2 } }}>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <WarningIcon color="warning" />
          Удалить пустую папку
        </DialogTitle>
        <DialogContent>
          {deleteTarget && (
            <Box>
              <Typography variant="body2" sx={{ mb: 1 }}>Вы собираетесь переместить в Корзину:</Typography>
              <Paper variant="outlined" sx={{ p: 1.5, mb: 2, bgcolor: 'action.hover', borderRadius: 1.5 }}>
                <Typography variant="body2" sx={{ fontWeight: 600, wordBreak: 'break-all' }}>{deleteTarget}</Typography>
              </Paper>
              <Alert severity="success" sx={{ borderRadius: 1.5, mb: 1 }}>
                <RestoreIcon sx={{ fontSize: 16, verticalAlign: 'middle', mr: 0.5 }} />
                Папка будет перемещена в Корзину (можно восстановить)
              </Alert>
              <Alert severity="info" sx={{ borderRadius: 1.5, mt: 1 }}>
                Системные папки (Windows, Program Files) защищены и не могут быть удалены.
              </Alert>
              {deleteError && <Alert severity="error" sx={{ mt: 1, borderRadius: 1.5 }}>{deleteError}</Alert>}
            </Box>
          )}
          {deleting && <LinearProgress sx={{ mt: 2 }} />}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDeleteTarget(null)} disabled={deleting} sx={{ borderRadius: 1.5 }}>Отмена</Button>
          <Button onClick={handleDelete} variant="contained" color="warning" disabled={deleting}
            startIcon={deleting ? <CircularProgress size={16} /> : <RestoreIcon />} sx={{ borderRadius: 1.5 }}>
            {deleting ? 'Перемещение...' : 'Переместить в Корзину'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Report Dialog */}
      <Dialog open={showReport} onClose={() => setShowReport(false)} maxWidth="md" fullWidth PaperProps={{ sx: { borderRadius: 2 } }}>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <RestoreIcon sx={{ color: 'success.main' }} />
          Отчет об удалении
        </DialogTitle>
        <DialogContent dividers>
          <Box sx={{ mb: 2 }}>
            <Typography variant="body1" sx={{ fontWeight: 600, mb: 1 }}>
              Результат массового удаления:
            </Typography>
            <Grid container spacing={2}>
              <Grid item xs={6}>
                <Paper variant="outlined" sx={{ p: 2, textAlign: 'center', bgcolor: 'success.light' }}>
                  <Typography variant="h4" color="success.main">
                    {deleteResults.filter(r => r.success).length}
                  </Typography>
                  <Typography variant="caption">Успешно перемещено</Typography>
                </Paper>
              </Grid>
              <Grid item xs={6}>
                <Paper variant="outlined" sx={{ p: 2, textAlign: 'center', bgcolor: 'error.light' }}>
                  <Typography variant="h4" color="error.main">
                    {deleteResults.filter(r => !r.success).length}
                  </Typography>
                  <Typography variant="caption">Ошибок</Typography>
                </Paper>
              </Grid>
            </Grid>
          </Box>
          
          {deleteResults.some(r => !r.success) && (
            <Alert severity="warning" sx={{ mb: 2 }}>
              Некоторые папки не удалось удалить. Проверьте права доступа и закройте программы, использующие эти папки.
            </Alert>
          )}
          
          <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>Детали:</Typography>
          <Paper variant="outlined" sx={{ maxHeight: 200, overflow: 'auto' }}>
            <List dense>
              {deleteResults.map((result, idx) => (
                <ListItem key={idx} sx={{ py: 0.5 }}>
                  <ListItemIcon sx={{ minWidth: 36 }}>
                    {result.success ? (
                      <RestoreIcon fontSize="small" color="success" />
                    ) : (
                      <WarningIcon fontSize="small" color="error" />
                    )}
                  </ListItemIcon>
                  <ListItemText
                    primary={result.path}
                    secondary={result.success ? 'Перемещено в Корзину' : 'Ошибка удаления'}
                    primaryTypographyProps={{ 
                      variant: 'body2', 
                      sx: { wordBreak: 'break-all', fontSize: '0.8rem' } 
                    }}
                  />
                </ListItem>
              ))}
            </List>
          </Paper>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setShowReport(false)} variant="contained" sx={{ borderRadius: 1.5 }}>
            Закрыть
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

/** Parse CLI output from --empty-dirs */
function parseEmptyDirsOutput(output: string): string[] {
  const dirs: string[] = [];
  for (const line of output.split('\n')) {
    const match = line.match(/\[пусто\]\s+(.+)/);
    if (match) {
      dirs.push(match[1].trim());
    }
  }
  return dirs;
}
