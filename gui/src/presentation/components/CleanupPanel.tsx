/**
 * Cleanup Panel Component
 * Main cleaning interface with category selection
 */
import React, { useState, useCallback } from 'react';
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
  Divider,
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
} from '@mui/icons-material';
import { useCategories, useScan, useClean } from '../hooks';
import { LogLevel } from '@domain/index';

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
    loadCategories,
    toggleCategory,
    selectAllSafe,
    selectAllAggressive,
    deselectAllSafe,
    deselectAllAggressive,
  } = useCategories();

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
    error: cleanError,
    clean,
    clear: clearClean,
  } = useClean();

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
    try {
      await clean(aggressive, selection);
    } catch {
      onError(cleanError || 'Clean failed');
    }
  }, [aggressive, selection, clean, cleanError, onError]);

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
        <Paper elevation={2} sx={{ p: 2, height: '100%' }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Typography variant="h6">Категории</Typography>
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
                  <Typography variant="caption">Агрессивный</Typography>
                </Box>
              }
            />
          </Box>

          {/* Safe Categories */}
          <Accordion defaultExpanded>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography variant="subtitle2" sx={{ fontWeight: 'bold' }}>
                Безопасные ({categories.safe.length})
              </Typography>
            </AccordionSummary>
            <AccordionDetails sx={{ p: 0 }}>
              <Box sx={{ mb: 1 }}>
                <Button size="small" onClick={selectAllSafe}>Все</Button>
                <Button size="small" onClick={deselectAllSafe}>Ничего</Button>
              </Box>
              <List dense sx={{ maxHeight: 300, overflow: 'auto' }}>
                {categories.safe.map((cat) => (
                  <ListItem key={cat.id} disablePadding>
                    <ListItemButton onClick={() => toggleCategory(cat.id)} dense>
                      <ListItemIcon>
                        <Checkbox
                          edge="start"
                          checked={selection.isSelected(cat.id)}
                          tabIndex={-1}
                          disableRipple
                        />
                      </ListItemIcon>
                      <ListItemText
                        primary={cat.name}
                        secondary={cat.id}
                        primaryTypographyProps={{ variant: 'body2' }}
                        secondaryTypographyProps={{ variant: 'caption' }}
                      />
                    </ListItemButton>
                  </ListItem>
                ))}
              </List>
            </AccordionDetails>
          </Accordion>

          {/* Aggressive Categories */}
          {aggressive && (
            <Accordion defaultExpanded sx={{ mt: 1 }}>
              <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                <Typography variant="subtitle2" color="warning.main" sx={{ fontWeight: 'bold' }}>
                  Агрессивные ({categories.aggressive.length})
                </Typography>
              </AccordionSummary>
              <AccordionDetails sx={{ p: 0 }}>
                <Box sx={{ mb: 1 }}>
                  <Button size="small" onClick={selectAllAggressive}>Все</Button>
                  <Button size="small" onClick={deselectAllAggressive}>Ничего</Button>
                </Box>
                <List dense sx={{ maxHeight: 200, overflow: 'auto' }}>
                  {categories.aggressive.map((cat) => (
                    <ListItem key={cat.id} disablePadding>
                      <ListItemButton onClick={() => toggleCategory(cat.id)} dense>
                        <ListItemIcon>
                          <Checkbox
                            edge="start"
                            checked={selection.isSelected(cat.id)}
                            tabIndex={-1}
                            disableRipple
                          />
                        </ListItemIcon>
                        <ListItemText
                          primary={cat.name}
                          secondary={cat.id}
                          primaryTypographyProps={{ variant: 'body2', color: 'warning.main' }}
                          secondaryTypographyProps={{ variant: 'caption' }}
                        />
                      </ListItemButton>
                    </ListItem>
                  ))}
                </List>
              </AccordionDetails>
            </Accordion>
          )}

          <Box sx={{ mt: 2, display: 'flex', gap: 1 }}>
            <Button
              variant="contained"
              startIcon={<RefreshIcon />}
              onClick={handleScan}
              disabled={scanning || cleaning || categoriesLoading}
              fullWidth
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
            >
              Очистить
            </Button>
          </Box>
        </Paper>
      </Grid>

      {/* Results Panel */}
      <Grid item xs={12} md={7}>
        <Paper elevation={2} sx={{ p: 2, height: '100%' }}>
          <Typography variant="h6" gutterBottom>Результаты</Typography>

          {(scanning || cleaning) && <LinearProgress sx={{ mb: 2 }} />}

          {scanResult && (
            <Card sx={{ mb: 2, bgcolor: 'success.dark' }}>
              <CardContent>
                <Typography variant="h5" color="white">
                  Найдено: {formatBytes(scanResult.totalBytes)}
                </Typography>
                <Typography variant="body2" color="white" sx={{ opacity: 0.8 }}>
                  {scanResult.totalFiles} файлов в {scanResult.nonEmptyResults.length} категориях
                </Typography>
              </CardContent>
            </Card>
          )}

          {scanResult && scanResult.nonEmptyResults.length > 0 && (
            <Paper variant="outlined" sx={{ maxHeight: 200, overflow: 'auto', mb: 2 }}>
              <List dense>
                {scanResult.nonEmptyResults.map((cat) => (
                  <ListItem key={cat.categoryId}>
                    <ListItemText
                      primary={`${cat.categoryName} (${cat.sizeFormatted})`}
                      secondary={`${cat.fileCount} файлов`}
                    />
                    <Chip
                      label={cat.sizeFormatted}
                      size="small"
                      color={cat.sizeBytes > 1024 * 1024 * 100 ? 'warning' : 'default'}
                    />
                  </ListItem>
                ))}
              </List>
            </Paper>
          )}

          <Typography variant="subtitle2" gutterBottom>Лог:</Typography>
          <Paper
            variant="outlined"
            sx={{
              p: 1,
              bgcolor: 'background.paper',
              fontFamily: 'monospace',
              fontSize: '0.75rem',
              whiteSpace: 'pre-wrap',
              maxHeight: 300,
              overflow: 'auto',
            }}
          >
            {(scanLog?.all.map(e => e.message).join('\n')) ||
             (cleanLog?.all.map(e => e.message).join('\n')) ||
             'Нажмите "Сканировать" для начала...'}
          </Paper>
        </Paper>
      </Grid>

      {/* Confirm Dialog */}
      <Dialog open={confirmDialog} onClose={() => setConfirmDialog(false)}>
        <DialogTitle>
          <WarningIcon color="warning" sx={{ mr: 1 }} />
          Подтвердите удаление
        </DialogTitle>
        <DialogContent>
          {scanResult && (
            <Typography>
              Вы собираетесь удалить {formatBytes(scanResult.totalBytes)} в {scanResult.totalFiles} файлах.
            </Typography>
          )}
          {aggressive && (
            <Alert severity="warning" sx={{ mt: 2 }}>
              Включён агрессивный режим! Это может удалить важные данные.
            </Alert>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmDialog(false)}>Отмена</Button>
          <Button onClick={handleClean} variant="contained" color="error">
            Удалить
          </Button>
        </DialogActions>
      </Dialog>
    </Grid>
  );
}
