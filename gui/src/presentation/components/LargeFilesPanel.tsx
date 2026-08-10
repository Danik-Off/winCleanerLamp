/**
 * LargeFilesPanel — анализатор диска: самые крупные файлы в системе
 * (аналог WizTree/TreeSize). Ничего не удаляется автоматически — только
 * находит то, что реально занимает место, решение остаётся за пользователем.
 */
import { useState, useCallback } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  List,
  ListItem,
  ListItemText,
  Chip,
  CircularProgress,
  Alert,
  Divider,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  LinearProgress,
  Checkbox,
} from '@mui/material';
import {
  Refresh as RefreshIcon,
  Delete as DeleteIcon,
  Warning as WarningIcon,
  Storage as StorageIcon,
} from '@mui/icons-material';
import { ScanningIndicator } from './ScanningIndicator';

interface LargeFilesPanelProps {
  onError: (error: string) => void;
}

interface LargeFileItem {
  path: string;
  sizeBytes: number;
  sizeFormatted: string;
  modTime: string;
  inSystemDir: boolean;
}

function formatBytes(bytes: number): string {
  if (bytes <= 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

export function LargeFilesPanel({ onError }: LargeFilesPanelProps): JSX.Element {
  const [scanning, setScanning] = useState(false);
  const [files, setFiles] = useState<LargeFileItem[]>([]);
  const [scanned, setScanned] = useState<number | null>(null);
  const [totalBytes, setTotalBytes] = useState(0);
  const [deleteTarget, setDeleteTarget] = useState<LargeFileItem | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [bulkConfirmOpen, setBulkConfirmOpen] = useState(false);
  const [bulkDeleting, setBulkDeleting] = useState(false);

  const handleScan = useCallback(async () => {
    setScanning(true);
    setSelected(new Set());
    try {
      const res = await window.electronAPI.getLargeFiles();
      if (res.error) onError(res.error);
      setFiles(res.files || []);
      setScanned(res.scannedFiles);
      setTotalBytes(res.totalBytes);
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Ошибка поиска крупных файлов');
    } finally {
      setScanning(false);
    }
  }, [onError]);

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    setDeleteError(null);
    try {
      const res = await window.electronAPI.deleteFile(deleteTarget.path);
      if (res.success) {
        setFiles((prev) => prev.filter((f) => f.path !== deleteTarget.path));
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

  const toggleSelect = (path: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  };

  const allSelected = files.length > 0 && files.every((f) => selected.has(f.path));

  const handleBulkDelete = useCallback(async () => {
    setBulkDeleting(true);
    let failed = 0;
    for (const path of Array.from(selected)) {
      try {
        const res = await window.electronAPI.deleteFile(path);
        if (res.success) setFiles((prev) => prev.filter((f) => f.path !== path));
        else failed++;
      } catch {
        failed++;
      }
    }
    setSelected(new Set());
    setBulkDeleting(false);
    setBulkConfirmOpen(false);
    if (failed > 0) onError(`Не удалось удалить ${failed} файлов`);
  }, [selected, onError]);

  const selectedBytes = files.filter((f) => selected.has(f.path)).reduce((s, f) => s + f.sizeBytes, 0);

  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 1.5, mb: 1.5 }}>
        <Button
          variant="contained"
          onClick={handleScan}
          disabled={scanning}
          startIcon={scanning ? <CircularProgress size={18} color="inherit" /> : <RefreshIcon />}
          sx={{ borderRadius: 2, fontWeight: 700, px: 3 }}
        >
          {scanning ? 'Сканирование...' : 'Найти крупные файлы'}
        </Button>
        {scanned !== null && (
          <Chip label={`проверено файлов: ${scanned}`} size="small" variant="outlined" sx={{ fontWeight: 600 }} />
        )}
        {files.length > 0 && (
          <Chip label={formatBytes(totalBytes)} size="small" color="warning" sx={{ fontWeight: 600 }} />
        )}
      </Box>

      <Typography variant="caption" sx={{ display: 'block', opacity: 0.55, mb: 1.5 }}>
        Самые крупные файлы во всём профиле пользователя (от 100 МБ) — видео, установщики, образы дисков, старые бэкапы. Ничего не удаляется автоматически.
      </Typography>

      <ScanningIndicator active={scanning} />

      {files.length === 0 && scanned === null && !scanning && (
        <Paper elevation={0} sx={{ p: 4, textAlign: 'center', borderRadius: 2, border: '1px solid', borderColor: (t) => (t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0') }}>
          <StorageIcon sx={{ fontSize: 48, opacity: 0.2, mb: 1 }} />
          <Typography variant="body2" sx={{ opacity: 0.5 }}>
            Показывает файлы, которые реально занимают место на диске — не привязано к заранее заданным категориям мусора.
          </Typography>
        </Paper>
      )}

      {!scanning && scanned !== null && files.length === 0 && (
        <Alert severity="success" sx={{ borderRadius: 2 }}>Крупных файлов (от 100 МБ) не найдено.</Alert>
      )}

      {files.length > 0 && (
        <>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
            <Checkbox
              size="small"
              checked={allSelected}
              indeterminate={selected.size > 0 && !allSelected}
              onChange={() => setSelected(allSelected ? new Set() : new Set(files.map((f) => f.path)))}
              sx={{ p: 0.5 }}
            />
            <Typography variant="caption" sx={{ opacity: 0.7 }}>
              {selected.size > 0 ? `Выбрано: ${selected.size} (${formatBytes(selectedBytes)})` : 'Выбрать все'}
            </Typography>
            {selected.size > 0 && (
              <Button
                size="small"
                color="error"
                variant="outlined"
                onClick={() => setBulkConfirmOpen(true)}
                startIcon={<DeleteIcon sx={{ fontSize: 16 }} />}
                sx={{ ml: 'auto', borderRadius: 2, fontWeight: 600 }}
              >
                Удалить выбранное ({selected.size})
              </Button>
            )}
          </Box>

          <Paper variant="outlined" sx={{ maxHeight: 520, overflow: 'auto', borderRadius: 3, borderColor: 'divider' }}>
            <List dense disablePadding>
              {files.map((f, idx) => (
                <Box key={f.path}>
                  <ListItem
                    sx={{ py: 1.5, px: 2, '&:hover': { bgcolor: 'action.hover' } }}
                    secondaryAction={
                      <Tooltip title="Удалить файл">
                        <IconButton edge="end" color="error" size="small" onClick={() => { setDeleteError(null); setDeleteTarget(f); }}
                          sx={{ border: '1px solid', borderColor: 'error.main', borderRadius: 1.5, '&:hover': { bgcolor: 'error.main', color: 'white' } }}>
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    }
                  >
                    <Checkbox size="small" checked={selected.has(f.path)} onChange={() => toggleSelect(f.path)} sx={{ p: 0.5, mr: 0.5 }} />
                    <ListItemText
                      primary={
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                          {f.path.split('\\').pop()}
                          {f.inSystemDir && (
                            <Tooltip title="Внутри Program Files/Windows/ProgramData — может принадлежать установленной программе, проверьте перед удалением">
                              <WarningIcon sx={{ fontSize: 14, color: 'warning.main' }} />
                            </Tooltip>
                          )}
                        </Box>
                      }
                      secondary={f.path}
                      primaryTypographyProps={{ variant: 'body2', sx: { fontWeight: 600 } }}
                      secondaryTypographyProps={{ variant: 'caption', sx: { wordBreak: 'break-all', opacity: 0.7 } }}
                    />
                    <Chip label={f.sizeFormatted} size="small" color={f.sizeBytes > 1024 ** 3 ? 'error' : 'warning'} sx={{ fontWeight: 600, mr: 2, flexShrink: 0 }} />
                  </ListItem>
                  {idx < files.length - 1 && <Divider />}
                </Box>
              ))}
            </List>
          </Paper>
        </>
      )}

      {/* Delete Dialog */}
      <Dialog open={!!deleteTarget} onClose={() => !deleting && setDeleteTarget(null)} maxWidth="sm" fullWidth PaperProps={{ sx: { borderRadius: 2 } }}>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <WarningIcon color="warning" />
          Удалить файл?
        </DialogTitle>
        <DialogContent>
          {deleteTarget && (
            <Box>
              <Paper variant="outlined" sx={{ p: 1.5, mb: 2, bgcolor: 'action.hover', borderRadius: 1.5 }}>
                <Typography variant="body2" sx={{ fontWeight: 600, wordBreak: 'break-all' }}>{deleteTarget.path}</Typography>
                <Typography variant="caption" color="text.secondary">{deleteTarget.sizeFormatted}</Typography>
              </Paper>
              {deleteTarget.inSystemDir && (
                <Alert severity="warning" sx={{ borderRadius: 1.5, mb: 1 }}>
                  ⚠ Файл внутри системной/установочной папки — убедитесь, что он не нужен установленной программе.
                </Alert>
              )}
              <Alert severity="success" sx={{ borderRadius: 1.5 }}>Файл будет перемещён в Корзину (можно восстановить).</Alert>
              {deleteError && <Alert severity="warning" sx={{ mt: 1, borderRadius: 1.5 }}>{deleteError}</Alert>}
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

      {/* Bulk Delete Dialog */}
      <Dialog open={bulkConfirmOpen} onClose={() => !bulkDeleting && setBulkConfirmOpen(false)} maxWidth="sm" fullWidth PaperProps={{ sx: { borderRadius: 2 } }}>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <WarningIcon color="warning" />
          Удалить выбранное ({selected.size})?
        </DialogTitle>
        <DialogContent>
          <Typography gutterBottom>Будет удалено {selected.size} файлов ({formatBytes(selectedBytes)}).</Typography>
          <Alert severity="success" sx={{ borderRadius: 1.5 }}>Всё будет перемещено в Корзину (можно восстановить).</Alert>
          {bulkDeleting && <LinearProgress sx={{ mt: 2 }} />}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setBulkConfirmOpen(false)} disabled={bulkDeleting} sx={{ borderRadius: 1.5 }}>Отмена</Button>
          <Button onClick={handleBulkDelete} variant="contained" color="warning" disabled={bulkDeleting}
            startIcon={bulkDeleting ? <CircularProgress size={16} /> : <DeleteIcon />} sx={{ borderRadius: 1.5 }}>
            {bulkDeleting ? 'Удаление...' : 'В Корзину'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
