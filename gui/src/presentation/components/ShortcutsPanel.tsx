/**
 * ShortcutsPanel — поиск битых ярлыков (.lnk с несуществующей целью)
 * на рабочем столе и в меню Пуск.
 */
import { useState, useCallback } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Chip,
  CircularProgress,
  Alert,
  Tooltip,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  LinearProgress,
} from '@mui/material';
import {
  Refresh as RefreshIcon,
  LinkOff as BrokenLinkIcon,
  Delete as DeleteIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import { ScanningIndicator } from './ScanningIndicator';

interface ShortcutsPanelProps {
  onError: (error: string) => void;
}

interface BrokenShortcutItem {
  path: string;
  targetPath?: string;
  reason: string;
}

export function ShortcutsPanel({ onError }: ShortcutsPanelProps): JSX.Element {
  const [scanning, setScanning] = useState(false);
  const [scanned, setScanned] = useState<number | null>(null);
  const [broken, setBroken] = useState<BrokenShortcutItem[]>([]);
  const [deleteTarget, setDeleteTarget] = useState<BrokenShortcutItem | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const handleScan = useCallback(async () => {
    setScanning(true);
    try {
      const res = await window.electronAPI.getBrokenShortcuts();
      if (res.error) {
        onError(res.error);
      }
      setScanned(res.scanned);
      setBroken(res.broken || []);
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Ошибка поиска ярлыков');
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
        setBroken((prev) => prev.filter((b) => b.path !== deleteTarget.path));
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

  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 1.5, mb: 2 }}>
        <Button
          variant="contained"
          onClick={handleScan}
          disabled={scanning}
          startIcon={scanning ? <CircularProgress size={18} color="inherit" /> : <RefreshIcon />}
          sx={{ borderRadius: 2, fontWeight: 700, px: 3 }}
        >
          {scanning ? 'Поиск...' : 'Найти битые ярлыки'}
        </Button>
        {scanned !== null && (
          <Chip label={`проверено: ${scanned}`} size="small" variant="outlined" sx={{ fontWeight: 600 }} />
        )}
        {broken.length > 0 && (
          <Chip label={`битых: ${broken.length}`} size="small" color="error" sx={{ fontWeight: 600 }} />
        )}
      </Box>

      <ScanningIndicator active={scanning} />

      {scanned === null && !scanning && (
        <Paper
          elevation={0}
          sx={{
            p: 4,
            textAlign: 'center',
            borderRadius: 2,
            border: '1px solid',
            borderColor: (t) => (t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0'),
          }}
        >
          <BrokenLinkIcon sx={{ fontSize: 48, opacity: 0.2, mb: 1 }} />
          <Typography variant="body2" sx={{ opacity: 0.5 }}>
            Проверяет ярлыки (.lnk) на рабочем столе и в меню Пуск — те, что указывают на несуществующие файлы.
          </Typography>
          <Typography variant="caption" sx={{ opacity: 0.3, display: 'block', mt: 1 }}>
            Ярлыки на временно недоступные сетевые пути тоже могут попасть в список — проверьте перед удалением.
          </Typography>
        </Paper>
      )}

      {!scanning && scanned !== null && broken.length === 0 && (
        <Alert severity="success" sx={{ borderRadius: 2 }}>
          Битых ярлыков не найдено.
        </Alert>
      )}

      {broken.length > 0 && (
        <List disablePadding>
          {broken.map((item) => (
            <Paper
              key={item.path}
              elevation={0}
              sx={{
                mb: 0.5,
                borderRadius: 2,
                border: '1px solid',
                borderColor: (t) => (t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0'),
              }}
            >
              <ListItem
                sx={{ py: 0.75 }}
                secondaryAction={
                  <Tooltip title="Удалить ярлык">
                    <IconButton
                      edge="end"
                      color="error"
                      size="small"
                      onClick={() => {
                        setDeleteError(null);
                        setDeleteTarget(item);
                      }}
                    >
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                }
              >
                <ListItemIcon sx={{ minWidth: 36 }}>
                  <BrokenLinkIcon sx={{ fontSize: 18, color: '#ef4444' }} />
                </ListItemIcon>
                <ListItemText
                  primary={
                    <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.75rem', wordBreak: 'break-all', pr: 4 }}>
                      {item.path}
                    </Typography>
                  }
                  secondary={
                    item.targetPath ? (
                      <Typography variant="caption" sx={{ fontFamily: 'monospace', fontSize: '0.68rem', opacity: 0.6, wordBreak: 'break-all' }}>
                        → {item.targetPath}
                      </Typography>
                    ) : (
                      <Typography variant="caption" sx={{ opacity: 0.6 }}>{item.reason}</Typography>
                    )
                  }
                />
              </ListItem>
            </Paper>
          ))}
        </List>
      )}

      {/* Delete Dialog */}
      <Dialog open={!!deleteTarget} onClose={() => !deleting && setDeleteTarget(null)} maxWidth="sm" fullWidth PaperProps={{ sx: { borderRadius: 2 } }}>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <WarningIcon color="warning" />
          Удалить ярлык?
        </DialogTitle>
        <DialogContent>
          {deleteTarget && (
            <Box>
              <Paper variant="outlined" sx={{ p: 1.5, mb: 2, bgcolor: 'action.hover', borderRadius: 1.5 }}>
                <Typography variant="body2" sx={{ fontWeight: 600, wordBreak: 'break-all' }}>
                  {deleteTarget.path}
                </Typography>
                {deleteTarget.targetPath && (
                  <Typography variant="caption" color="text.secondary" sx={{ wordBreak: 'break-all' }}>
                    цель: {deleteTarget.targetPath}
                  </Typography>
                )}
              </Paper>
              <Alert severity="success" sx={{ borderRadius: 1.5 }}>
                Ярлык будет перемещён в Корзину (можно восстановить).
              </Alert>
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
    </Box>
  );
}
