/**
 * Cleanup Panel Component
 * Main cleaning interface with category selection - Material Design
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
  Chip,
  TextField,
  InputAdornment,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Alert,
  Stack,
  ButtonBase,
  Collapse,
  IconButton,
} from '@mui/material';
import {
  Refresh as RefreshIcon,
  Delete as DeleteIcon,
  Warning as WarningIcon,
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  TrendingDown as TrendingDownIcon,
  Search as SearchIcon,
  Clear as ClearIcon,
  Bolt as BoltIcon,
  Shield as ShieldIcon,
  RocketLaunch as RocketIcon,
  KeyboardArrowDown as ArrowDownIcon,
  KeyboardArrowUp as ArrowUpIcon,
} from '@mui/icons-material';
import { useCategories, useScan, useClean } from '../hooks';

interface CleanupPanelProps {
  onError: (error: string) => void;
}

export function CleanupPanel({ onError }: CleanupPanelProps): JSX.Element {
  const [aggressive, setAggressive] = useState(false);
  const [confirmDialog, setConfirmDialog] = useState(false);
  const [search, setSearch] = useState('');
  const [presetLabel, setPresetLabel] = useState('Кастом');
  const [showCategories, setShowCategories] = useState(false);
  const [showLog, setShowLog] = useState(false);

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

  const searchValue = search.trim().toLowerCase();
  const safeCategories = categories.safe.filter((cat) =>
    cat.name.toLowerCase().includes(searchValue)
  );
  const aggressiveCategories = categories.aggressive.filter((cat) =>
    cat.name.toLowerCase().includes(searchValue)
  );
  const selectedSafeCount = categories.safe.filter((cat) => selection.isSelected(cat.id)).length;
  const selectedAggressiveCount = categories.aggressive.filter((cat) => selection.isSelected(cat.id)).length;
  const totalSelected = selectedSafeCount + selectedAggressiveCount;
  const totalAvailable = categories.safe.length + categories.aggressive.length;
  const riskLevel = selectedAggressiveCount > 0 ? 'Высокий риск' : 'Безопасный режим';
  const potentialCleanup = scanResult?.totalBytes ?? 0;

  const applyPreset = useCallback(
    (preset: 'quick' | 'balanced' | 'deep') => {
      deselectAllSafe();
      deselectAllAggressive();

      const safeKeywords: Record<typeof preset, string[]> = {
        quick: ['temp', 'cache', 'thumb', 'лог', 'log'],
        balanced: ['temp', 'cache', 'thumb', 'лог', 'log', 'recycle', 'корз'],
        deep: [],
      };

      const shouldSelectSafe = (name: string): boolean => {
        if (preset === 'deep') return true;
        const normalized = name.toLowerCase();
        return safeKeywords[preset].some((kw) => normalized.includes(kw));
      };

      categories.safe.forEach((cat) => {
        if (shouldSelectSafe(cat.name)) {
          toggleCategory(cat.id);
        }
      });

      if (preset === 'deep') {
        setAggressive(true);
        categories.aggressive.forEach((cat) => toggleCategory(cat.id));
        setPresetLabel('Максимальный');
      } else if (preset === 'balanced') {
        setAggressive(false);
        setPresetLabel('Рекомендуемый');
      } else {
        setAggressive(false);
        setPresetLabel('Быстрый');
      }
    },
    [categories.safe, categories.aggressive, deselectAllSafe, deselectAllAggressive, toggleCategory]
  );

  if (categoriesError) {
    return (
      <Alert severity="error" sx={{ mb: 2 }}>
        {categoriesError}
      </Alert>
    );
  }

  const presets = [
    {
      key: 'quick' as const,
      label: 'Быстрый',
      desc: 'Temp, кэш, логи',
      icon: <BoltIcon sx={{ fontSize: 22 }} />,
      color: '#3b82f6',
      bg: (dark: boolean) => dark ? '#172554' : '#eff6ff',
    },
    {
      key: 'balanced' as const,
      label: 'Рекомендуемый',
      desc: 'Безопасная очистка',
      icon: <ShieldIcon sx={{ fontSize: 22 }} />,
      color: '#10b981',
      bg: (dark: boolean) => dark ? '#052e16' : '#ecfdf5',
    },
    {
      key: 'deep' as const,
      label: 'Максимальный',
      desc: 'Всё, включая агрессивные',
      icon: <RocketIcon sx={{ fontSize: 22 }} />,
      color: '#f59e0b',
      bg: (dark: boolean) => dark ? '#451a03' : '#fffbeb',
    },
  ];

  return (
    <Box sx={{ pb: 0 }}>
      {/* ── Step 1: Presets ── */}
      <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 1.5, opacity: 0.5, textTransform: 'uppercase', letterSpacing: '0.05em', fontSize: '0.7rem' }}>
        Выберите сценарий
      </Typography>
      <Grid container spacing={1.5} sx={{ mb: 3 }}>
        {presets.map((p) => (
          <Grid item xs={12} sm={4} key={p.key}>
            <ButtonBase
              onClick={() => applyPreset(p.key)}
              sx={{
                width: '100%',
                borderRadius: 3,
                overflow: 'hidden',
                textAlign: 'left',
              }}
            >
              <Paper
                elevation={0}
                sx={{
                  p: 2,
                  width: '100%',
                  borderRadius: 3,
                  border: '1px solid',
                  borderColor: presetLabel === p.label
                    ? p.color
                    : (t) => t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0',
                  bgcolor: presetLabel === p.label
                    ? (t) => p.bg(t.palette.mode === 'dark')
                    : 'background.paper',
                  transition: 'all 0.2s',
                  '&:hover': {
                    borderColor: `${p.color}99`,
                    bgcolor: (t) => p.bg(t.palette.mode === 'dark'),
                  },
                }}
              >
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                  <Box sx={{ color: p.color, display: 'flex' }}>{p.icon}</Box>
                  <Box>
                    <Typography variant="body2" sx={{ fontWeight: 700 }}>{p.label}</Typography>
                    <Typography variant="caption" sx={{ opacity: 0.6 }}>{p.desc}</Typography>
                  </Box>
                </Box>
              </Paper>
            </ButtonBase>
          </Grid>
        ))}
      </Grid>

      {/* ── Step 2: Stats row ── */}
      <Grid container spacing={1.5} sx={{ mb: 2 }}>
        <Grid item xs={6} sm={3}>
          <Paper
            elevation={0}
            sx={{
              p: 1.5,
              borderRadius: 2.5,
              border: '1px solid',
              borderColor: (t) => t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0',
              bgcolor: 'background.paper',
              textAlign: 'center',
            }}
          >
            <Typography variant="h5" sx={{ fontWeight: 800, color: 'primary.main' }}>
              {totalSelected}
            </Typography>
            <Typography variant="caption" sx={{ opacity: 0.5 }}>из {totalAvailable}</Typography>
          </Paper>
        </Grid>
        <Grid item xs={6} sm={3}>
          <Paper
            elevation={0}
            sx={{
              p: 1.5,
              borderRadius: 2.5,
              border: '1px solid',
              borderColor: (t) => t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0',
              bgcolor: 'background.paper',
              textAlign: 'center',
            }}
          >
            <Typography variant="h5" sx={{ fontWeight: 800, color: 'success.main' }}>
              {formatBytes(potentialCleanup)}
            </Typography>
            <Typography variant="caption" sx={{ opacity: 0.5 }}>потенциал</Typography>
          </Paper>
        </Grid>
        <Grid item xs={6} sm={3}>
          <Paper
            elevation={0}
            sx={{
              p: 1.5,
              borderRadius: 2.5,
              border: '1px solid',
              borderColor: (t) => t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0',
              bgcolor: 'background.paper',
              textAlign: 'center',
            }}
          >
            <Typography variant="h5" sx={{ fontWeight: 800, color: selectedAggressiveCount > 0 ? 'warning.main' : 'text.secondary' }}>
              {selectedAggressiveCount > 0 ? 'High' : 'Low'}
            </Typography>
            <Typography variant="caption" sx={{ opacity: 0.5 }}>риск</Typography>
          </Paper>
        </Grid>
        <Grid item xs={6} sm={3}>
          <Paper
            elevation={0}
            sx={{
              p: 1.5,
              borderRadius: 2.5,
              border: '1px solid',
              borderColor: (t) => t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0',
              bgcolor: 'background.paper',
              textAlign: 'center',
            }}
          >
            <Typography variant="h5" sx={{ fontWeight: 800, color: 'text.primary' }}>
              {presetLabel}
            </Typography>
            <Typography variant="caption" sx={{ opacity: 0.5 }}>сценарий</Typography>
          </Paper>
        </Grid>
      </Grid>

      {/* ── Progress bar ── */}
      {(scanning || cleaning) && (
        <LinearProgress
          sx={{
            mb: 2,
            borderRadius: 2,
            height: 5,
            bgcolor: (t) => t.palette.mode === 'dark' ? '#1e293b' : '#e2e8f0',
            '& .MuiLinearProgress-bar': {
              borderRadius: 2,
              background: 'linear-gradient(90deg, #3b82f6 0%, #06b6d4 100%)',
            },
          }}
        />
      )}

      {/* ── Post-clean result banner ── */}
      {cleanDone && !cleaning && (
        <Alert
          severity={errorCount > 0 ? 'warning' : 'success'}
          icon={errorCount > 0 ? <ErrorIcon /> : <CheckCircleIcon />}
          sx={{ mb: 2, borderRadius: 2.5 }}
          onClose={() => setCleanDone(false)}
        >
          <Typography variant="body2" sx={{ fontWeight: 600 }}>
            {bytesCleaned > 0
              ? `Освобождено: ${formatBytes(bytesCleaned)} в ${filesCleaned} файлах`
              : 'Очистка завершена, нечего удалять'}
          </Typography>
          {errorCount > 0 && (
            <Typography variant="caption" sx={{ opacity: 0.9, display: 'block', mt: 0.5 }}>
              Ошибок/пропусков: {errorCount} (файлы заняты или требуют прав администратора)
            </Typography>
          )}
        </Alert>
      )}

      {/* ── Scan Result Summary ── */}
      {scanResult && (
        <Paper
          elevation={0}
          sx={{
            mb: 2,
            p: 2.5,
            borderRadius: 3,
            background: (t) =>
              t.palette.mode === 'dark'
                ? 'linear-gradient(135deg, #064e3b 0%, #065f46 100%)'
                : 'linear-gradient(135deg, #ecfdf5 0%, #d1fae5 100%)',
            border: '1px solid',
            borderColor: (t) => t.palette.mode === 'dark' ? '#047857' : '#a7f3d0',
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <Box sx={{
              p: 1.5,
              bgcolor: 'rgba(255,255,255,0.15)',
              borderRadius: 2.5,
              display: 'flex',
            }}>
              <TrendingDownIcon sx={{ color: (t) => t.palette.mode === 'dark' ? 'white' : '#065f46', fontSize: 28 }} />
            </Box>
            <Box sx={{ flex: 1 }}>
              <Typography variant="h6" sx={{ fontWeight: 800, color: (t) => t.palette.mode === 'dark' ? 'white' : '#065f46' }}>
                {formatBytes(scanResult.totalBytes)}
              </Typography>
              <Typography variant="caption" sx={{ opacity: 0.8, color: (t) => t.palette.mode === 'dark' ? '#a7f3d0' : '#065f46' }}>
                {scanResult.totalFiles} файлов в {scanResult.nonEmptyResults.length} категориях
              </Typography>
            </Box>
          </Box>
        </Paper>
      )}

      {/* ── Detailed Results ── */}
      {scanResult && scanResult.nonEmptyResults.length > 0 && (
        <Paper
          elevation={0}
          sx={{
            mb: 2,
            borderRadius: 2.5,
            border: '1px solid',
            borderColor: (t) => t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0',
            overflow: 'hidden',
          }}
        >
          <List dense disablePadding sx={{ maxHeight: 220, overflow: 'auto' }}>
            {scanResult.nonEmptyResults.map((cat, idx) => (
              <ListItem
                key={cat.categoryId}
                sx={{
                  py: 1,
                  px: 2,
                  borderBottom: idx < scanResult.nonEmptyResults.length - 1 ? '1px solid' : 'none',
                  borderColor: 'divider',
                }}
              >
                <ListItemText
                  primary={cat.categoryName}
                  secondary={`${cat.fileCount} файлов`}
                  primaryTypographyProps={{ fontWeight: 600, variant: 'body2' }}
                  secondaryTypographyProps={{ variant: 'caption', sx: { opacity: 0.6 } }}
                />
                <Chip
                  label={cat.sizeFormatted}
                  size="small"
                  color={cat.sizeBytes > 1024 * 1024 * 100 ? 'warning' : 'default'}
                  sx={{ fontWeight: 600, borderRadius: 1.5, height: 24 }}
                />
              </ListItem>
            ))}
          </List>
        </Paper>
      )}

      {scanResult && scanResult.nonEmptyResults.length === 0 && !scanning && (
        <Alert severity="success" sx={{ mb: 2, borderRadius: 2 }}>
          Ничего критичного не найдено.
        </Alert>
      )}

      {/* ── Expandable: Category selection ── */}
      <Paper
        elevation={0}
        sx={{
          mb: 2,
          borderRadius: 2.5,
          border: '1px solid',
          borderColor: (t) => t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0',
          overflow: 'hidden',
        }}
      >
        <ButtonBase
          onClick={() => setShowCategories(!showCategories)}
          sx={{
            width: '100%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            px: 2,
            py: 1.5,
            textAlign: 'left',
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Typography variant="body2" sx={{ fontWeight: 700 }}>
              Категории очистки
            </Typography>
            <Chip
              label={`${totalSelected} выбрано`}
              size="small"
              color={totalSelected > 0 ? 'primary' : 'default'}
              sx={{ fontWeight: 600, height: 22, fontSize: '0.7rem' }}
            />
          </Box>
          {showCategories ? <ArrowUpIcon sx={{ opacity: 0.5 }} /> : <ArrowDownIcon sx={{ opacity: 0.5 }} />}
        </ButtonBase>

        <Collapse in={showCategories}>
          <Box sx={{ px: 2, pb: 2 }}>
            {/* Search + aggressive toggle */}
            <Box sx={{ display: 'flex', gap: 1, mb: 1.5, alignItems: 'center' }}>
              <TextField
                size="small"
                fullWidth
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Поиск..."
                InputProps={{
                  startAdornment: <InputAdornment position="start"><SearchIcon sx={{ opacity: 0.4, fontSize: 18 }} /></InputAdornment>,
                  endAdornment: search ? <InputAdornment position="end"><IconButton size="small" onClick={() => setSearch('')}><ClearIcon sx={{ fontSize: 16 }} /></IconButton></InputAdornment> : null,
                  sx: { borderRadius: 2, height: 36 },
                }}
              />
              <FormControlLabel
                control={
                  <Switch
                    checked={aggressive}
                    onChange={(e) => setAggressive(e.target.checked)}
                    color="warning"
                    size="small"
                  />
                }
                label={
                  <Typography variant="caption" sx={{ fontWeight: 600, whiteSpace: 'nowrap' }}>
                    Агрессивный
                  </Typography>
                }
                sx={{ m: 0, ml: 1 }}
              />
            </Box>

            {/* Safe categories */}
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 0.5 }}>
              <Typography variant="caption" sx={{ fontWeight: 700, color: 'primary.main' }}>
                <CheckCircleIcon sx={{ fontSize: 14, mr: 0.5, verticalAlign: 'middle' }} />
                Безопасные ({safeCategories.length})
              </Typography>
              <Box sx={{ display: 'flex', gap: 0.5 }}>
                <Chip label="Все" size="small" onClick={selectAllSafe} clickable
                  sx={{ height: 22, fontSize: '0.65rem', fontWeight: 600, bgcolor: 'primary.main', color: 'white' }} />
                <Chip label="Сброс" size="small" onClick={deselectAllSafe} clickable variant="outlined"
                  sx={{ height: 22, fontSize: '0.65rem', fontWeight: 600 }} />
              </Box>
            </Box>
            <Paper
              variant="outlined"
              sx={{
                maxHeight: 200,
                overflow: 'auto',
                borderRadius: 2,
                mb: 1.5,
                bgcolor: (t) => t.palette.mode === 'dark' ? '#0b1120' : '#f8fafc',
              }}
            >
              <List dense disablePadding>
                {safeCategories.map((cat) => (
                  <ListItem key={cat.id} disablePadding>
                    <ListItemButton onClick={() => toggleCategory(cat.id)} dense sx={{ py: 0.25, px: 1 }}>
                      <ListItemIcon sx={{ minWidth: 30 }}>
                        <Checkbox checked={selection.isSelected(cat.id)} tabIndex={-1} disableRipple size="small" color="primary" />
                      </ListItemIcon>
                      <ListItemText primary={cat.name} primaryTypographyProps={{ variant: 'body2', fontSize: '0.8rem', fontWeight: 500, noWrap: true }} />
                    </ListItemButton>
                  </ListItem>
                ))}
              </List>
            </Paper>

            {/* Aggressive categories */}
            {aggressive && (
              <>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 0.5 }}>
                  <Typography variant="caption" sx={{ fontWeight: 700, color: 'warning.main' }}>
                    <WarningIcon sx={{ fontSize: 14, mr: 0.5, verticalAlign: 'middle' }} />
                    Агрессивные ({aggressiveCategories.length})
                  </Typography>
                  <Box sx={{ display: 'flex', gap: 0.5 }}>
                    <Chip label="Все" size="small" onClick={selectAllAggressive} clickable
                      sx={{ height: 22, fontSize: '0.65rem', fontWeight: 600, bgcolor: '#dc2626', color: 'white' }} />
                    <Chip label="Сброс" size="small" onClick={deselectAllAggressive} clickable variant="outlined"
                      sx={{ height: 22, fontSize: '0.65rem', fontWeight: 600, borderColor: '#dc2626', color: '#dc2626' }} />
                  </Box>
                </Box>
                <Paper
                  variant="outlined"
                  sx={{
                    maxHeight: 160,
                    overflow: 'auto',
                    borderRadius: 2,
                    bgcolor: (t) => t.palette.mode === 'dark' ? '#1a0a0a' : '#fef2f2',
                    borderColor: (t) => t.palette.mode === 'dark' ? '#7f1d1d' : '#fecaca',
                  }}
                >
                  <List dense disablePadding>
                    {aggressiveCategories.map((cat) => (
                      <ListItem key={cat.id} disablePadding>
                        <ListItemButton onClick={() => toggleCategory(cat.id)} dense sx={{ py: 0.25, px: 1 }}>
                          <ListItemIcon sx={{ minWidth: 30 }}>
                            <Checkbox checked={selection.isSelected(cat.id)} tabIndex={-1} disableRipple size="small" color="warning" />
                          </ListItemIcon>
                          <ListItemText primary={cat.name} primaryTypographyProps={{ variant: 'body2', fontSize: '0.8rem', fontWeight: 500, color: 'warning.main', noWrap: true }} />
                        </ListItemButton>
                      </ListItem>
                    ))}
                  </List>
                </Paper>
              </>
            )}
          </Box>
        </Collapse>
      </Paper>

      {/* ── Expandable: Log ── */}
      <Paper
        elevation={0}
        sx={{
          mb: 2,
          borderRadius: 2.5,
          border: '1px solid',
          borderColor: (t) => t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0',
          overflow: 'hidden',
        }}
      >
        <ButtonBase
          onClick={() => setShowLog(!showLog)}
          sx={{
            width: '100%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            px: 2,
            py: 1.5,
            textAlign: 'left',
          }}
        >
          <Typography variant="body2" sx={{ fontWeight: 700 }}>
            Лог операций
          </Typography>
          {showLog ? <ArrowUpIcon sx={{ opacity: 0.5 }} /> : <ArrowDownIcon sx={{ opacity: 0.5 }} />}
        </ButtonBase>
        <Collapse in={showLog}>
          <Box
            sx={{
              px: 2,
              pb: 2,
              fontFamily: '"JetBrains Mono", "Fira Code", "Consolas", monospace',
              fontSize: '0.72rem',
              whiteSpace: 'pre-wrap',
              maxHeight: 200,
              overflow: 'auto',
              color: 'text.secondary',
              lineHeight: 1.6,
            }}
          >
            {(scanLog?.all.map(e => e.message).join('\n')) ||
             (cleanLog?.all.map(e => e.message).join('\n')) ||
             'Нажмите "Сканировать" для начала...'}
          </Box>
        </Collapse>
      </Paper>

      {/* ── Sticky Bottom Action Bar ── */}
      <Paper
        elevation={0}
        sx={{
          position: 'sticky',
          bottom: 0,
          zIndex: 10,
          borderRadius: 2,
          borderTop: '1px solid',
          borderColor: (t) => t.palette.mode === 'dark' ? '#1f2937' : '#e2e8f0',
          bgcolor: (t) => t.palette.mode === 'dark' ? '#0f1629ee' : '#ffffffee',
          backdropFilter: 'blur(12px)',
          px: 2,
          py: 1.25,
          mt: 2,
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          {/* Status chip */}
          <Chip
            icon={selectedAggressiveCount > 0 ? <WarningIcon sx={{ fontSize: 16 }} /> : <CheckCircleIcon sx={{ fontSize: 16 }} />}
            label={riskLevel}
            size="small"
            color={selectedAggressiveCount > 0 ? 'warning' : 'success'}
            variant="outlined"
            sx={{ fontWeight: 600 }}
          />
          <Typography variant="caption" sx={{ opacity: 0.5, flex: 1 }}>
            {totalSelected > 0 ? `${totalSelected} категорий выбрано` : 'Выберите сценарий или категории'}
          </Typography>

          {/* Actions */}
          <Stack direction="row" spacing={1}>
            <Button
              variant="outlined"
              startIcon={<RefreshIcon />}
              onClick={handleScan}
              disabled={scanning || cleaning || categoriesLoading || totalSelected === 0}
              sx={{
                px: 3,
                borderRadius: 2.5,
                fontWeight: 700,
              }}
            >
              {scanning ? 'Анализ...' : 'Сканировать'}
            </Button>
            <Button
              variant="contained"
              color="error"
              startIcon={<DeleteIcon />}
              onClick={() => setConfirmDialog(true)}
              disabled={scanning || cleaning || !scanResult}
              sx={{
                px: 3,
                borderRadius: 2.5,
                fontWeight: 700,
                boxShadow: scanResult ? '0 4px 14px rgba(239, 68, 68, 0.4)' : 'none',
              }}
            >
              Очистить
            </Button>
          </Stack>
        </Box>
      </Paper>

      {/* ── Confirm Dialog ── */}
      <Dialog
        open={confirmDialog}
        onClose={() => setConfirmDialog(false)}
        PaperProps={{
          sx: {
            borderRadius: 3,
            minWidth: 400,
            bgcolor: 'background.paper',
          }
        }}
      >
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1.5, pb: 1 }}>
          <Box sx={{
            p: 1,
            bgcolor: (t) => t.palette.mode === 'dark' ? '#451a03' : '#fffbeb',
            borderRadius: 2,
            display: 'flex',
          }}>
            <WarningIcon color="warning" sx={{ fontSize: 24 }} />
          </Box>
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700 }}>Подтвердите удаление</Typography>
            <Typography variant="caption" sx={{ color: 'text.secondary' }}>
              Это действие нельзя отменить
            </Typography>
          </Box>
        </DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          {scanResult && (
            <Box sx={{
              p: 2,
              bgcolor: (t) => t.palette.mode === 'dark' ? '#0b1120' : '#f1f5f9',
              borderRadius: 2,
              mb: 2,
            }}>
              <Typography variant="body1" sx={{ fontWeight: 600 }}>
                Будет удалено: {formatBytes(scanResult.totalBytes)}
              </Typography>
              <Typography variant="body2" sx={{ color: 'text.secondary' }}>
                в {scanResult.totalFiles} файлах
              </Typography>
            </Box>
          )}
          {aggressive && (
            <Alert severity="warning" sx={{ borderRadius: 2 }}>
              Включён агрессивный режим! Это может удалить важные данные.
            </Alert>
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3, gap: 1 }}>
          <Button
            onClick={() => setConfirmDialog(false)}
            sx={{ borderRadius: 2, px: 3 }}
          >
            Отмена
          </Button>
          <Button
            onClick={handleClean}
            variant="contained"
            color="error"
            sx={{ borderRadius: 2, px: 3 }}
          >
            Удалить
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
