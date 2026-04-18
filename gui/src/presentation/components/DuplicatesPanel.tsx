/**
 * Duplicates Panel Component
 * Finds and displays duplicate files with option to delete extras
 */
import React, { useState, useCallback } from 'react';
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
  Radio,
} from '@mui/material';
import {
  ContentCopy as DupIcon,
  Refresh as RefreshIcon,
  Warning as WarningIcon,
  Search as SearchIcon,
  Delete as DeleteIcon,
  Clear as ClearIcon,
  Storage as StorageIcon,
  FileCopy as FileIcon,
  Folder as FolderIcon,
} from '@mui/icons-material';

interface DupGroup {
  size: number;
  sizeFormatted: string;
  waste: number;
  wasteFormatted: string;
  paths: string[];
}

interface DuplicatesPanelProps {
  onError: (error: string) => void;
}

function formatBytes(bytes: number): string {
  if (bytes <= 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

export function DuplicatesPanel({ onError }: DuplicatesPanelProps): JSX.Element {
  const [scanning, setScanning] = useState(false);
  const [rootPath, setRootPath] = useState('');
  const [groups, setGroups] = useState<DupGroup[]>([]);
  const [scannedFiles, setScannedFiles] = useState(0);
  const [totalWaste, setTotalWaste] = useState(0);
  const [hasScanResult, setHasScanResult] = useState(false);
  const [filter, setFilter] = useState('');

  const [deleteTarget, setDeleteTarget] = useState<{ group: DupGroup; path: string } | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deletedPaths, setDeletedPaths] = useState<Set<string>>(new Set());
  const [deleteError, setDeleteError] = useState<string | null>(null);

  // Selected "keep" path per group (index)
  const [keepSelection, setKeepSelection] = useState<Record<number, string>>({});

  const handleScan = useCallback(async () => {
    const paths = rootPath.trim();
    if (!paths) {
      onError('Укажите папки для сканирования');
      return;
    }
    setScanning(true);
    setGroups([]);
    setHasScanResult(false);
    setDeletedPaths(new Set());
    setKeepSelection({});
    try {
      const output = await window.electronAPI.getDuplicates(paths);
      const parsed = parseDuplicatesOutput(output);
      setGroups(parsed.groups);
      setScannedFiles(parsed.scannedFiles);
      setTotalWaste(parsed.totalWaste);
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
      const res = await window.electronAPI.deleteFile(deleteTarget.path);
      if (res.success) {
        setDeletedPaths(prev => new Set(prev).add(deleteTarget.path));
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

  const filteredGroups = groups.filter(g => {
    if (!filter.trim()) return true;
    const lf = filter.toLowerCase();
    return g.paths.some(p => p.toLowerCase().includes(lf));
  });

  return (
    <Box>
      {/* Header */}
      <Paper
        elevation={3}
        sx={{
          p: 3, mb: 2, borderRadius: 2,
          background: (t) => t.palette.mode === 'dark'
            ? 'linear-gradient(135deg, #4a148c 0%, #6a1b9a 100%)'
            : 'linear-gradient(135deg, #f3e5f5 0%, #e1bee7 100%)',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
          <DupIcon sx={{ mr: 1.5, fontSize: 28 }} />
          <Typography variant="h6" sx={{ fontWeight: 700 }}>Поиск дубликатов файлов</Typography>
        </Box>
        <Alert severity="info" sx={{ mb: 2, borderRadius: 1.5 }}>
          Сканирование по размеру + SHA-256 хэшу. Укажите папки через запятую.
        </Alert>
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'flex-start' }}>
          <TextField
            fullWidth
            size="small"
            placeholder="C:\Users\ИмяПользователя\Documents, D:\Фото"
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
            {scanning ? 'Поиск...' : 'Найти дубликаты'}
          </Button>
        </Box>
      </Paper>

      {scanning && <LinearProgress sx={{ mb: 2, borderRadius: 1 }} />}

      {/* Stats */}
      {hasScanResult && (
        <Grid container spacing={2} sx={{ mb: 2 }}>
          <Grid item xs={4}>
            <Card elevation={2} sx={{ borderRadius: 2, background: (t) => t.palette.mode === 'dark' ? 'linear-gradient(135deg,#4a148c,#6a1b9a)' : 'linear-gradient(135deg,#f3e5f5,#e1bee7)' }}>
              <CardContent sx={{ textAlign: 'center', py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <StorageIcon sx={{ fontSize: 28, opacity: 0.8 }} />
                <Typography variant="h6" sx={{ fontWeight: 700 }}>{formatBytes(totalWaste)}</Typography>
                <Typography variant="caption" sx={{ opacity: 0.8 }}>Можно освободить</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={4}>
            <Card elevation={2} sx={{ borderRadius: 2, background: (t) => t.palette.mode === 'dark' ? 'linear-gradient(135deg,#e65100,#f57c00)' : 'linear-gradient(135deg,#fff3e0,#ffe0b2)' }}>
              <CardContent sx={{ textAlign: 'center', py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <FileIcon sx={{ fontSize: 28, opacity: 0.8 }} />
                <Typography variant="h6" sx={{ fontWeight: 700 }}>{filteredGroups.length}</Typography>
                <Typography variant="caption" sx={{ opacity: 0.8 }}>Групп дубликатов</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={4}>
            <Card elevation={2} sx={{ borderRadius: 2, background: (t) => t.palette.mode === 'dark' ? 'linear-gradient(135deg,#0d47a1,#1565c0)' : 'linear-gradient(135deg,#e3f2fd,#bbdefb)' }}>
              <CardContent sx={{ textAlign: 'center', py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <FolderIcon sx={{ fontSize: 28, opacity: 0.8 }} />
                <Typography variant="h6" sx={{ fontWeight: 700 }}>{scannedFiles}</Typography>
                <Typography variant="caption" sx={{ opacity: 0.8 }}>Файлов просканировано</Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}

      {/* Filter */}
      {hasScanResult && groups.length > 0 && (
        <TextField
          fullWidth size="small"
          placeholder="Фильтр по имени файла или пути..."
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

      {/* Groups */}
      {filteredGroups.length > 0 && (
        <Paper variant="outlined" sx={{ maxHeight: 480, overflow: 'auto', borderRadius: 2 }}>
          <List dense disablePadding>
            {filteredGroups.map((group, gi) => {
              const livePaths = group.paths.filter(p => !deletedPaths.has(p));
              if (livePaths.length < 2) return null;
              const kept = keepSelection[gi] || livePaths[0];
              return (
                <React.Fragment key={gi}>
                  <Box sx={{ bgcolor: 'action.hover', px: 2, py: 1, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <Typography variant="body2" sx={{ fontWeight: 700 }}>
                      Группа {gi + 1}: {group.sizeFormatted} × {livePaths.length} файлов
                    </Typography>
                    <Chip label={`Экономия: ${group.wasteFormatted}`} size="small" color="warning" sx={{ fontWeight: 600 }} />
                  </Box>
                  {livePaths.map((p) => {
                    const fileName = p.split('\\').pop() || p;
                    const isKept = p === kept;
                    return (
                      <ListItem key={p} sx={{ py: 0.8, px: 2, opacity: isKept ? 1 : 0.85 }}>
                        <Tooltip title={isKept ? 'Оставить этот файл' : 'Выбрать как основной'}>
                          <Radio
                            checked={isKept}
                            onChange={() => setKeepSelection(prev => ({ ...prev, [gi]: p }))}
                            size="small"
                            sx={{ mr: 1 }}
                          />
                        </Tooltip>
                        <ListItemText
                          primary={fileName}
                          secondary={p}
                          primaryTypographyProps={{ variant: 'body2', fontWeight: isKept ? 700 : 400 }}
                          secondaryTypographyProps={{ variant: 'caption', sx: { wordBreak: 'break-all', opacity: 0.7 } }}
                        />
                        {!isKept && (
                          <Tooltip title="Удалить дубликат">
                            <IconButton edge="end" color="error" size="small"
                              onClick={() => { setDeleteError(null); setDeleteTarget({ group, path: p }); }}
                              sx={{ border: '1px solid', borderColor: 'error.main', borderRadius: 1.5, '&:hover': { bgcolor: 'error.main', color: 'white' } }}>
                              <DeleteIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        )}
                        {isKept && (
                          <Chip label="оставить" size="small" color="success" variant="outlined" sx={{ ml: 1 }} />
                        )}
                      </ListItem>
                    );
                  })}
                  {gi < filteredGroups.length - 1 && <Divider />}
                </React.Fragment>
              );
            })}
          </List>
        </Paper>
      )}

      {/* Empty state */}
      {!scanning && hasScanResult && groups.length === 0 && (
        <Alert severity="success" sx={{ borderRadius: 2 }}>
          Дубликатов не найдено! Все файлы уникальны.
        </Alert>
      )}

      {/* Delete Dialog */}
      <Dialog open={!!deleteTarget} onClose={() => !deleting && setDeleteTarget(null)} maxWidth="sm" fullWidth PaperProps={{ sx: { borderRadius: 2 } }}>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <WarningIcon color="error" />
          Удалить дубликат
        </DialogTitle>
        <DialogContent>
          {deleteTarget && (
            <Box>
              <Typography gutterBottom>Вы собираетесь удалить файл:</Typography>
              <Paper variant="outlined" sx={{ p: 1.5, mb: 2, bgcolor: 'action.hover', borderRadius: 1.5 }}>
                <Typography variant="body2" sx={{ fontWeight: 600, wordBreak: 'break-all' }}>{deleteTarget.path}</Typography>
                <Typography variant="caption" color="text.secondary">{deleteTarget.group.sizeFormatted}</Typography>
              </Paper>
              <Alert severity="error" sx={{ borderRadius: 1.5 }}>Это действие необратимо!</Alert>
              {deleteError && <Alert severity="warning" sx={{ mt: 1, borderRadius: 1.5 }}>{deleteError}</Alert>}
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

/** Parse CLI output from --duplicates */
function parseDuplicatesOutput(output: string): { groups: DupGroup[]; scannedFiles: number; totalWaste: number } {
  const lines = output.split('\n');
  const groups: DupGroup[] = [];
  let current: DupGroup | null = null;
  let scannedFiles = 0;
  let totalWaste = 0;

  for (const line of lines) {
    // "Просканировано 12345 файлов за 3.2 сек."
    const scannedMatch = line.match(/Просканировано\s+(\d+)\s+файлов/);
    if (scannedMatch) {
      scannedFiles = parseInt(scannedMatch[1], 10);
      continue;
    }

    // "  === Группа 1: 1.23 MB (x3 файлов) ==="
    const groupMatch = line.match(/===\s+Группа\s+\d+:\s+(.+?)\s+\(x(\d+)\s+файлов\)\s+===/);
    if (groupMatch) {
      if (current) groups.push(current);
      current = { size: 0, sizeFormatted: groupMatch[1], waste: 0, wasteFormatted: '', paths: [] };
      continue;
    }

    // "    C:\path\to\file"
    if (current && line.match(/^\s{4}\S/) && !line.includes('Экономия')) {
      current.paths.push(line.trim());
      continue;
    }

    // "    Экономия при удалении лишних: 2.46 MB"
    const wasteMatch = line.match(/Экономия при удалении лишних:\s+(.+)/);
    if (wasteMatch && current) {
      current.wasteFormatted = wasteMatch[1].trim();
      current.waste = parseSizeStr(current.wasteFormatted);
      continue;
    }

    // "  ИТОГО: 5 групп дубликатов, 15 файлов, ~12.3 MB можно освободить"
    const totalMatch = line.match(/~([\d.]+\s*\w+)\s+можно освободить/);
    if (totalMatch) {
      totalWaste = parseSizeStr(totalMatch[1]);
    }
  }
  if (current && current.paths.length >= 2) groups.push(current);

  // Parse size for each group
  for (const g of groups) {
    g.size = parseSizeStr(g.sizeFormatted);
  }

  return { groups, scannedFiles, totalWaste };
}

function parseSizeStr(s: string): number {
  const units: Record<string, number> = { 'B': 1, 'KB': 1024, 'MB': 1024 ** 2, 'GB': 1024 ** 3, 'TB': 1024 ** 4 };
  const m = s.match(/^([\d.]+)\s*(\w+)$/);
  if (!m) return 0;
  return parseFloat(m[1]) * (units[m[2].toUpperCase()] || 1);
}
