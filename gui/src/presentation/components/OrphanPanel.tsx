/**
 * OrphanPanel Component
 * GUI for orphaned program leftovers: scan, discover, clean
 */
import { useState } from 'react';
import {
  Paper,
  Typography,
  Button,
  Box,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Chip,
  CircularProgress,
  Alert,
  Tabs,
  Tab,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Stack,
  Collapse,
  IconButton,
} from '@mui/material';
import {
  Search as SearchIcon,
  FolderDelete as FolderDeleteIcon,
  Explore as DiscoverIcon,
  Delete as DeleteIcon,
  Refresh as RefreshIcon,
  Warning as WarningIcon,
  Folder as FolderIcon,
  AppRegistration as RegistryIcon,
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  DeleteSweep as CleanIcon,
} from '@mui/icons-material';
import { useOrphan } from '../hooks';

interface OrphanPanelProps {
  onError: (error: string) => void;
}

export function OrphanPanel({ onError }: OrphanPanelProps): JSX.Element {
  const {
    scanning, discovering, cleaning,
    scanResults, discoverResults,
    error, orphanScan, orphanDiscover, orphanClean,
  } = useOrphan();

  const [activeTab, setActiveTab] = useState(0);
  const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set());
  const [cleanDialog, setCleanDialog] = useState<{ name: string; recycle: boolean; cacheOnly: boolean } | null>(null);
  const [cleanOutput, setCleanOutput] = useState<string | null>(null);

  const toggleExpand = (name: string) => {
    setExpandedItems((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  };

  const handleClean = async () => {
    if (!cleanDialog) return;
    const output = await orphanClean(cleanDialog.name, cleanDialog.recycle, cleanDialog.cacheOnly);
    setCleanOutput(output);
    setCleanDialog(null);
    // Re-scan after cleaning
    orphanScan();
  };


  return (
    <Box>
      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
        <Box>
          <Typography variant="h6" sx={{ fontWeight: 700, fontSize: '1.1rem' }}>
            Остатки удалённых программ
          </Typography>
          <Typography variant="caption" sx={{ opacity: 0.5 }}>
            Поиск и очистка файлов от программ, описанных в orphaned_apps.json
          </Typography>
        </Box>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2, borderRadius: 2 }} onClose={() => onError('')}>
          {error}
        </Alert>
      )}

      {/* Tabs */}
      <Paper
        elevation={0}
        sx={{
          mb: 2,
          borderRadius: 2,
          border: '1px solid',
          borderColor: (t) => t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0',
          bgcolor: (t) => t.palette.mode === 'dark' ? '#0f1629' : '#ffffff',
        }}
      >
        <Tabs
          value={activeTab}
          onChange={(_, v) => setActiveTab(v)}
          sx={{
            minHeight: 40,
            '& .MuiTab-root': { minHeight: 40, fontSize: '0.8rem', fontWeight: 600, textTransform: 'none' },
          }}
        >
          <Tab icon={<SearchIcon sx={{ fontSize: 16 }} />} iconPosition="start" label="Scan (по JSON)" />
          <Tab icon={<DiscoverIcon sx={{ fontSize: 16 }} />} iconPosition="start" label="Discover" />
        </Tabs>
      </Paper>

      {/* Tab: Scan */}
      {activeTab === 0 && (
        <Box>
          <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
            <Button
              variant="contained"
              startIcon={scanning ? <CircularProgress size={16} color="inherit" /> : <RefreshIcon />}
              onClick={orphanScan}
              disabled={scanning}
              sx={{ borderRadius: 2, fontWeight: 700, px: 3 }}
            >
              {scanning ? 'Сканирование...' : 'Сканировать'}
            </Button>
            {scanResults.length > 0 && (
              <Chip
                label={`${scanResults.length} программ`}
                size="small"
                color="warning"
                variant="outlined"
                sx={{ fontWeight: 600 }}
              />
            )}
          </Box>

          {scanResults.length === 0 && !scanning && (
            <Paper
              elevation={0}
              sx={{
                p: 4,
                textAlign: 'center',
                borderRadius: 2,
                border: '1px solid',
                borderColor: (t) => t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0',
              }}
            >
              <FolderDeleteIcon sx={{ fontSize: 48, opacity: 0.2, mb: 1 }} />
              <Typography variant="body2" sx={{ opacity: 0.5 }}>
                Нажмите "Сканировать" чтобы проверить orphaned_apps.json
              </Typography>
            </Paper>
          )}

          {/* Scan Results */}
          <List disablePadding>
            {scanResults.map((item) => (
              <Paper
                key={item.displayName}
                elevation={0}
                sx={{
                  mb: 1,
                  borderRadius: 2,
                  border: '1px solid',
                  borderColor: (t) => t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0',
                  overflow: 'hidden',
                }}
              >
                <ListItem
                  sx={{ cursor: 'pointer', py: 1 }}
                  onClick={() => toggleExpand(item.displayName)}
                  secondaryAction={
                    <Stack direction="row" spacing={0.5} alignItems="center">
                      <Chip label={item.totalSize} size="small" variant="outlined" sx={{ fontWeight: 600, fontSize: '0.7rem' }} />
                      <Tooltip title="Удалить только кеш">
                        <IconButton
                          size="small"
                          color="info"
                          onClick={(e) => {
                            e.stopPropagation();
                            setCleanDialog({ name: item.displayName, recycle: false, cacheOnly: true });
                          }}
                        >
                          <CleanIcon sx={{ fontSize: 16 }} />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Удалить всё">
                        <IconButton
                          size="small"
                          color="error"
                          onClick={(e) => {
                            e.stopPropagation();
                            setCleanDialog({ name: item.displayName, recycle: false, cacheOnly: false });
                          }}
                        >
                          <DeleteIcon sx={{ fontSize: 16 }} />
                        </IconButton>
                      </Tooltip>
                      {expandedItems.has(item.displayName) ? <ExpandLessIcon sx={{ fontSize: 18 }} /> : <ExpandMoreIcon sx={{ fontSize: 18 }} />}
                    </Stack>
                  }
                >
                  <ListItemIcon sx={{ minWidth: 36 }}>
                    <FolderDeleteIcon color="warning" sx={{ fontSize: 20 }} />
                  </ListItemIcon>
                  <ListItemText
                    primary={item.displayName}
                    secondary={`${item.totalFiles} файлов`}
                    primaryTypographyProps={{ fontWeight: 600, fontSize: '0.85rem' }}
                    secondaryTypographyProps={{ fontSize: '0.7rem' }}
                  />
                </ListItem>

                <Collapse in={expandedItems.has(item.displayName)}>
                  <Box sx={{ px: 2, pb: 1.5, pt: 0.5 }}>
                    {item.paths.map((p) => (
                      <Box key={p.path} sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 0.3 }}>
                        <FolderIcon sx={{ fontSize: 14, opacity: 0.4 }} />
                        <Typography variant="caption" sx={{ fontFamily: 'monospace', fontSize: '0.7rem', flex: 1, wordBreak: 'break-all' }}>
                          {p.path}
                        </Typography>
                        <Chip label={p.size} size="small" sx={{ fontSize: '0.6rem', height: 18 }} />
                      </Box>
                    ))}
                    {item.regKeys.map((k) => (
                      <Box key={k} sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 0.3 }}>
                        <RegistryIcon sx={{ fontSize: 14, opacity: 0.4, color: '#f59e0b' }} />
                        <Typography variant="caption" sx={{ fontFamily: 'monospace', fontSize: '0.7rem', wordBreak: 'break-all' }}>
                          {k}
                        </Typography>
                      </Box>
                    ))}
                  </Box>
                </Collapse>
              </Paper>
            ))}
          </List>
        </Box>
      )}

      {/* Tab: Discover */}
      {activeTab === 1 && (
        <Box>
          <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
            <Button
              variant="contained"
              startIcon={discovering ? <CircularProgress size={16} color="inherit" /> : <DiscoverIcon />}
              onClick={() => orphanDiscover()}
              disabled={discovering}
              sx={{ borderRadius: 2, fontWeight: 700, px: 3 }}
            >
              {discovering ? 'Поиск...' : 'Найти неизвестные папки'}
            </Button>
            {discoverResults.length > 0 && (
              <Chip
                label={`${discoverResults.length} папок`}
                size="small"
                color="info"
                variant="outlined"
                sx={{ fontWeight: 600 }}
              />
            )}
          </Box>

          {discoverResults.length === 0 && !discovering && (
            <Paper
              elevation={0}
              sx={{
                p: 4,
                textAlign: 'center',
                borderRadius: 2,
                border: '1px solid',
                borderColor: (t) => t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0',
              }}
            >
              <DiscoverIcon sx={{ fontSize: 48, opacity: 0.2, mb: 1 }} />
              <Typography variant="body2" sx={{ opacity: 0.5 }}>
                Сканирует Program Files, ProgramData, AppData и ищет папки,
                не связанные с установленными программами
              </Typography>
              <Typography variant="caption" sx={{ opacity: 0.3, display: 'block', mt: 1 }}>
                Результаты только информативные — ничего не удаляется автоматически
              </Typography>
            </Paper>
          )}

          {/* Discover Results */}
          <List disablePadding>
            {discoverResults.map((item) => (
              <Paper
                key={item.path}
                elevation={0}
                sx={{
                  mb: 0.5,
                  borderRadius: 2,
                  border: '1px solid',
                  borderColor: (t) => t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0',
                }}
              >
                <ListItem sx={{ py: 0.75 }}>
                  <ListItemIcon sx={{ minWidth: 36 }}>
                    <FolderIcon sx={{ fontSize: 18, color: item.has_executable ? '#f59e0b' : '#64748b' }} />
                  </ListItemIcon>
                  <ListItemText
                    primary={
                      <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.75rem', wordBreak: 'break-all' }}>
                        {item.path}
                      </Typography>
                    }
                    secondary={
                      <Stack direction="row" spacing={0.5} sx={{ mt: 0.25 }}>
                        <Chip label={`${item.size_mb} MB`} size="small" sx={{ fontSize: '0.6rem', height: 18 }} />
                        {item.has_executable && (
                          <Chip
                            icon={<WarningIcon sx={{ fontSize: 12 }} />}
                            label="есть .exe"
                            size="small"
                            color="warning"
                            variant="outlined"
                            sx={{ fontSize: '0.6rem', height: 18 }}
                          />
                        )}
                      </Stack>
                    }
                  />
                </ListItem>
              </Paper>
            ))}
          </List>
        </Box>
      )}

      {/* Clean Result Dialog */}
      {cleanOutput && (
        <Dialog open={!!cleanOutput} onClose={() => setCleanOutput(null)} maxWidth="sm" fullWidth>
          <DialogTitle sx={{ fontWeight: 700 }}>Результат очистки</DialogTitle>
          <DialogContent>
            <Box
              sx={{
                fontFamily: 'monospace',
                fontSize: '0.75rem',
                whiteSpace: 'pre-wrap',
                p: 2,
                borderRadius: 1,
                bgcolor: (t) => t.palette.mode === 'dark' ? '#0b1120' : '#f1f5f9',
                maxHeight: 400,
                overflow: 'auto',
              }}
            >
              {cleanOutput}
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setCleanOutput(null)}>Закрыть</Button>
          </DialogActions>
        </Dialog>
      )}

      {/* Confirm Clean Dialog */}
      <Dialog
        open={!!cleanDialog}
        onClose={() => setCleanDialog(null)}
        PaperProps={{ sx: { borderRadius: 3, minWidth: 440 } }}
      >
        <DialogTitle sx={{ fontWeight: 700 }}>
          {cleanDialog?.cacheOnly ? 'Очистить кеш?' : 'Удалить все остатки?'}
        </DialogTitle>
        <DialogContent>
          <Chip
            icon={<CleanIcon />}
            label={cleanDialog?.name}
            color={cleanDialog?.cacheOnly ? 'info' : 'error'}
            variant="outlined"
            sx={{ fontWeight: 600, mb: 2 }}
          />

          {cleanDialog?.cacheOnly ? (
            <Alert severity="info" sx={{ borderRadius: 2, fontSize: '0.8rem' }}>
              Будут удалены только <strong>кеш-файлы</strong> (временные данные).
              Настройки, профили и личные данные не будут затронуты.
            </Alert>
          ) : (
            <Alert severity="warning" sx={{ borderRadius: 2, fontSize: '0.8rem' }}>
              <strong>⚠ ВНИМАНИЕ!</strong> Будут удалены <strong>все</strong> остатки программы:
              настройки, профили, сохранения, кеш и ключи реестра.
              Это может привести к потере личных данных!
            </Alert>
          )}

          <Stack direction="row" spacing={1} sx={{ mt: 2 }}>
            <Button
              size="small"
              variant={cleanDialog?.recycle ? 'contained' : 'outlined'}
              onClick={() => cleanDialog && setCleanDialog({ ...cleanDialog, recycle: !cleanDialog.recycle })}
              sx={{ borderRadius: 2, fontSize: '0.75rem' }}
            >
              {cleanDialog?.recycle ? '✓ В корзину' : 'В корзину'}
            </Button>
            {!cleanDialog?.cacheOnly && (
              <Button
                size="small"
                variant="outlined"
                color="info"
                onClick={() => cleanDialog && setCleanDialog({ ...cleanDialog, cacheOnly: true })}
                sx={{ borderRadius: 2, fontSize: '0.75rem' }}
              >
                Только кеш
              </Button>
            )}
            {cleanDialog?.cacheOnly && (
              <Button
                size="small"
                variant="outlined"
                color="error"
                onClick={() => cleanDialog && setCleanDialog({ ...cleanDialog, cacheOnly: false })}
                sx={{ borderRadius: 2, fontSize: '0.75rem' }}
              >
                Полная очистка
              </Button>
            )}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCleanDialog(null)}>Отмена</Button>
          <Button
            variant="contained"
            color={cleanDialog?.cacheOnly ? 'info' : 'error'}
            onClick={handleClean}
            disabled={cleaning}
            startIcon={cleaning ? <CircularProgress size={16} color="inherit" /> : <DeleteIcon />}
          >
            {cleaning ? 'Удаление...' : cleanDialog?.cacheOnly ? 'Очистить кеш' : 'Удалить всё'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
