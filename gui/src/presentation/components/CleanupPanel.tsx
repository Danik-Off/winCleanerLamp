/**
 * Cleanup Panel Component
 * Main cleaning interface with category selection
 */
import { useState, useCallback } from 'react';
import {
  Box,
  Grid,
  Paper,
  Typography,
  Button,
  Checkbox,
  FormControlLabel,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  ListItemIcon,
  LinearProgress,
  Switch,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Card,
  CardContent,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Alert,
} from '@mui/material';
import {
  Refresh as RefreshIcon,
  Delete as DeleteIcon,
  ExpandMore as ExpandMoreIcon,
  Warning as WarningIcon,
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
} from '@mui/icons-material';
import { useCategories, useScan, useClean } from '../hooks';

interface CleanupPanelProps {
  onError: (error: string) => void;
}

export function CleanupPanel({ onError }: CleanupPanelProps): JSX.Element {
  const [aggressive, setAggressive] = useState(false);
  const [confirmDialog, setConfirmDialog] = useState(false);

  const {
    categories,
    selection,
    loading: categoriesLoading,
    error: categoriesError,
    toggleCategory,
    selectAllSafe,
    selectAllAggressive,
    deselectAllSafe,
    deselectAllAggressive,
  } = useCategories();
  
  // Categories loaded via useEffect in the hook

  const {
    scanning,
    result: scanResult,
    log: scanLog,
    error: scanError,
    scan,
    clear: clearScan,
  } = useScan();

  const {
    cleaning,
    log: cleanLog,
    bytesCleaned,
    filesCleaned,
    errorCount,
    error: cleanError,
    clean,
    clear: clearClean,
  } = useClean();

  const [cleanDone, setCleanDone] = useState(false);

  const handleScan = useCallback(async () => {
    clearScan();
    clearClean();
    try {
      await scan(aggressive, selection);
    } catch {
      onError(scanError || 'Scan failed');
    }
  }, [aggressive, selection, scan, clearScan, clearClean, scanError, onError]);

  const handleClean = useCallback(async () => {
    setConfirmDialog(false);
    setCleanDone(false);
    try {
      await clean(aggressive, selection);
      setCleanDone(true);
      // Auto-rescan after cleaning to update results
      try {
        await scan(aggressive, selection);
      } catch { /* ignore rescan errors */ }
    } catch {
      onError(cleanError || 'Clean failed');
    }
  }, [aggressive, selection, clean, scan, cleanError, onError]);

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  // Show error if any
  if (categoriesError) {
    return (
      <Alert severity="error" sx={{ mb: 2 }}>
        {categoriesError}
      </Alert>
    );
  }

  return (
    <Grid container spacing={3}>
      {/* Categories Panel */}
      <Grid item xs={12} md={5}>
        <Paper
          elevation={3}
          sx={{
            p: 2.5,
            height: '100%',
            borderRadius: 2,
            background: (theme) =>
              theme.palette.mode === 'dark'
                ? 'linear-gradient(180deg, #1e1e2f 0%, #1a1a2e 100%)'
                : undefined,
          }}
        >
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Typography variant="h6" sx={{ fontWeight: 700 }}>Категории</Typography>
            <FormControlLabel
              control={
                <Switch
                  checked={aggressive}
                  onChange={(e) => setAggressive(e.target.checked)}
                  color="warning"
                />
              }
              label={
                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                  <WarningIcon color="warning" sx={{ mr: 0.5, fontSize: 18 }} />
                  <Typography variant="caption" sx={{ fontWeight: 600 }}>Агрессивный</Typography>
                </Box>
              }
            />
          </Box>

          {/* Safe Categories */}
          <Accordion defaultExpanded sx={{ borderRadius: '8px !important', '&:before': { display: 'none' }, mb: 1 }}>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                Безопасные ({categories.safe.length})
              </Typography>
            </AccordionSummary>
            <AccordionDetails sx={{ p: 0 }}>
              <Box sx={{ mb: 1, px: 1 }}>
                <Button size="small" onClick={selectAllSafe} sx={{ textTransform: 'none', fontWeight: 600 }}>Все</Button>
                <Button size="small" onClick={deselectAllSafe} sx={{ textTransform: 'none', fontWeight: 600 }}>Ничего</Button>
              </Box>
              <List dense sx={{ maxHeight: 300, overflow: 'auto' }}>
                {categories.safe.map((cat) => (
                  <ListItem key={cat.id} disablePadding>
                    <ListItemButton onClick={() => toggleCategory(cat.id)} dense sx={{ borderRadius: 1 }}>
                      <ListItemIcon>
                        <Checkbox
                          edge="start"
                          checked={selection.isSelected(cat.id)}
                          tabIndex={-1}
                          disableRipple
                          color="primary"
                        />
                      </ListItemIcon>
                      <ListItemText
                        primary={cat.name}
                        secondary={cat.id}
                        primaryTypographyProps={{ variant: 'body2', fontWeight: 500 }}
                        secondaryTypographyProps={{ variant: 'caption', sx: { opacity: 0.6 } }}
                      />
                    </ListItemButton>
                  </ListItem>
                ))}
              </List>
            </AccordionDetails>
          </Accordion>

          {/* Aggressive Categories */}
          {aggressive && (
            <Accordion defaultExpanded sx={{ borderRadius: '8px !important', '&:before': { display: 'none' }, mt: 1 }}>
              <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                <Typography variant="subtitle2" color="warning.main" sx={{ fontWeight: 700 }}>
                  Агрессивные ({categories.aggressive.length})
                </Typography>
              </AccordionSummary>
              <AccordionDetails sx={{ p: 0 }}>
                <Box sx={{ mb: 1, px: 1 }}>
                  <Button size="small" onClick={selectAllAggressive} sx={{ textTransform: 'none', fontWeight: 600 }}>Все</Button>
                  <Button size="small" onClick={deselectAllAggressive} sx={{ textTransform: 'none', fontWeight: 600 }}>Ничего</Button>
                </Box>
                <List dense sx={{ maxHeight: 200, overflow: 'auto' }}>
                  {categories.aggressive.map((cat) => (
                    <ListItem key={cat.id} disablePadding>
                      <ListItemButton onClick={() => toggleCategory(cat.id)} dense sx={{ borderRadius: 1 }}>
                        <ListItemIcon>
                          <Checkbox
                            edge="start"
                            checked={selection.isSelected(cat.id)}
                            tabIndex={-1}
                            disableRipple
                            color="warning"
                          />
                        </ListItemIcon>
                        <ListItemText
                          primary={cat.name}
                          secondary={cat.id}
                          primaryTypographyProps={{ variant: 'body2', color: 'warning.main', fontWeight: 500 }}
                          secondaryTypographyProps={{ variant: 'caption', sx: { opacity: 0.6 } }}
                        />
                      </ListItemButton>
                    </ListItem>
                  ))}
                </List>
              </AccordionDetails>
            </Accordion>
          )}

          <Box sx={{ mt: 2.5, display: 'flex', gap: 1 }}>
            <Button
              variant="contained"
              startIcon={<RefreshIcon />}
              onClick={handleScan}
              disabled={scanning || cleaning || categoriesLoading}
              fullWidth
              sx={{ borderRadius: 2, textTransform: 'none', fontWeight: 600, py: 1.2 }}
            >
              {scanning ? 'Сканирование...' : 'Сканировать'}
            </Button>
            <Button
              variant="contained"
              color="error"
              startIcon={<DeleteIcon />}
              onClick={() => setConfirmDialog(true)}
              disabled={scanning || cleaning || !scanResult}
              fullWidth
              sx={{ borderRadius: 2, textTransform: 'none', fontWeight: 600, py: 1.2 }}
            >
              Очистить
            </Button>
          </Box>
        </Paper>
      </Grid>

      {/* Results Panel */}
      <Grid item xs={12} md={7}>
        <Paper
          elevation={3}
          sx={{
            p: 2.5,
            height: '100%',
            borderRadius: 2,
            background: (theme) =>
              theme.palette.mode === 'dark'
                ? 'linear-gradient(180deg, #1e1e2f 0%, #1a1a2e 100%)'
                : undefined,
          }}
        >
          <Typography variant="h6" gutterBottom sx={{ fontWeight: 700 }}>Результаты</Typography>

          {(scanning || cleaning) && <LinearProgress sx={{ mb: 2, borderRadius: 1 }} />}

          {/* Post-clean result banner */}
          {cleanDone && !cleaning && (
            <Alert
              severity={errorCount > 0 ? 'warning' : 'success'}
              icon={errorCount > 0 ? <ErrorIcon /> : <CheckCircleIcon />}
              sx={{ mb: 2, borderRadius: 2 }}
              onClose={() => setCleanDone(false)}
            >
              <Typography variant="body2" sx={{ fontWeight: 600 }}>
                {bytesCleaned > 0
                  ? `Освобождено: ${formatBytes(bytesCleaned)} в ${filesCleaned} файлах`
                  : 'Очистка завершена, нечего удалять'}
              </Typography>
              {errorCount > 0 && (
                <Typography variant="caption" sx={{ opacity: 0.85 }}>
                  Ошибок/пропусков: {errorCount} (файлы заняты или требуют прав администратора)
                </Typography>
              )}
            </Alert>
          )}

          {scanResult && (
            <Card
              sx={{
                mb: 2,
                borderRadius: 2,
                background: (theme) =>
                  theme.palette.mode === 'dark'
                    ? 'linear-gradient(135deg, #1b5e20 0%, #2e7d32 100%)'
                    : 'linear-gradient(135deg, #43a047 0%, #66bb6a 100%)',
              }}
            >
              <CardContent>
                <Typography variant="h5" color="white" sx={{ fontWeight: 700 }}>
                  Найдено: {formatBytes(scanResult.totalBytes)}
                </Typography>
                <Typography variant="body2" color="white" sx={{ opacity: 0.85 }}>
                  {scanResult.totalFiles} файлов в {scanResult.nonEmptyResults.length} категориях
                </Typography>
              </CardContent>
            </Card>
          )}

          {scanResult && scanResult.nonEmptyResults.length > 0 && (
            <Paper variant="outlined" sx={{ maxHeight: 200, overflow: 'auto', mb: 2, borderRadius: 2 }}>
              <List dense disablePadding>
                {scanResult.nonEmptyResults.map((cat) => (
                  <ListItem key={cat.categoryId} sx={{ py: 1 }}>
                    <ListItemText
                      primary={cat.categoryName}
                      secondary={`${cat.fileCount} файлов`}
                      primaryTypographyProps={{ fontWeight: 500, variant: 'body2' }}
                      secondaryTypographyProps={{ variant: 'caption', sx: { opacity: 0.7 } }}
                    />
                    <Chip
                      label={cat.sizeFormatted}
                      size="small"
                      color={cat.sizeBytes > 1024 * 1024 * 100 ? 'warning' : 'default'}
                      sx={{ fontWeight: 600 }}
                    />
                  </ListItem>
                ))}
              </List>
            </Paper>
          )}

          <Typography variant="subtitle2" gutterBottom sx={{ fontWeight: 600 }}>Лог:</Typography>
          <Paper
            variant="outlined"
            sx={{
              p: 1.5,
              bgcolor: (theme) =>
                theme.palette.mode === 'dark' ? '#0d0d0d' : '#fafafa',
              fontFamily: '"JetBrains Mono", "Fira Code", "Consolas", monospace',
              fontSize: '0.75rem',
              whiteSpace: 'pre-wrap',
              maxHeight: 300,
              overflow: 'auto',
              borderRadius: 2,
              '&::-webkit-scrollbar': { width: 5 },
              '&::-webkit-scrollbar-thumb': { bgcolor: 'action.disabled', borderRadius: 3 },
            }}
          >
            {(scanLog?.all.map(e => e.message).join('\n')) ||
             (cleanLog?.all.map(e => e.message).join('\n')) ||
             'Нажмите "Сканировать" для начала...'}
          </Paper>
        </Paper>
      </Grid>

      {/* Confirm Dialog */}
      <Dialog
        open={confirmDialog}
        onClose={() => setConfirmDialog(false)}
        PaperProps={{ sx: { borderRadius: 2 } }}
      >
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <WarningIcon color="warning" />
          Подтвердите удаление
        </DialogTitle>
        <DialogContent>
          {scanResult && (
            <Typography>
              Вы собираетесь удалить {formatBytes(scanResult.totalBytes)} в {scanResult.totalFiles} файлах.
            </Typography>
          )}
          {aggressive && (
            <Alert severity="warning" sx={{ mt: 2, borderRadius: 1.5 }}>
              Включён агрессивный режим! Это может удалить важные данные.
            </Alert>
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setConfirmDialog(false)} sx={{ borderRadius: 1.5 }}>Отмена</Button>
          <Button onClick={handleClean} variant="contained" color="error" sx={{ borderRadius: 1.5 }}>
            Удалить
          </Button>
        </DialogActions>
      </Dialog>
    </Grid>
  );
}
