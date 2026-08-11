/**
 * Apps Panel Component — список установленных программ + деинсталляция.
 *
 * После успешного удаления программы, у которой есть запись в
 * orphaned_apps.json (inOrphanDB), панель сама проверяет остатки через
 * --orphan-scan и предлагает почистить найденный мусор (--orphan-clean),
 * не заставляя пользователя переходить на вкладку "Остатки" вручную.
 */
import { useState, useCallback, useEffect, useMemo } from 'react';
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
} from '@mui/material';
import {
  Apps as AppsIcon,
  Refresh as RefreshIcon,
  Search as SearchIcon,
  Clear as ClearIcon,
  DeleteForever as UninstallIcon,
  CleaningServices as CleanIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import { ScanningIndicator } from './ScanningIndicator';

interface AppsPanelProps {
  onError: (error: string) => void;
}

interface OrphanFoundPathDto {
  path: string;
  size: number;
  files: number;
}

interface OrphanScanResultDto {
  app: { displayName: string };
  foundPaths: OrphanFoundPathDto[] | null;
  totalSize: number;
  totalFiles: number;
}

function formatBytes(bytes: number): string {
  if (bytes <= 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

export function AppsPanel({ onError }: AppsPanelProps): JSX.Element {
  const [loading, setLoading] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [apps, setApps] = useState<InstalledAppDto[]>([]);
  const [filter, setFilter] = useState('');

  const [uninstallConfirm, setUninstallConfirm] = useState<InstalledAppDto | null>(null);
  const [uninstallingFor, setUninstallingFor] = useState<string | null>(null);
  const [uninstallInfo, setUninstallInfo] = useState<string | null>(null);

  // После удаления программы, у которой есть база остатков — найденный мусор,
  // предложенный на очистку. checking=true, пока идёт --orphan-scan.
  const [checkingLeftovers, setCheckingLeftovers] = useState(false);
  const [leftoverOffer, setLeftoverOffer] = useState<OrphanScanResultDto | null>(null);
  const [cleaningLeftovers, setCleaningLeftovers] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await window.electronAPI.getInstalledApps();
      if (res.error && res.apps.length === 0) {
        onError(res.error);
      }
      setApps(res.apps);
      setLoaded(true);
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Не удалось получить список программ');
    } finally {
      setLoading(false);
    }
  }, [onError]);

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const filtered = useMemo(() => {
    const lf = filter.trim().toLowerCase();
    if (!lf) return apps;
    return apps.filter(
      (a) => a.displayName.toLowerCase().includes(lf) || a.publisher.toLowerCase().includes(lf)
    );
  }, [apps, filter]);

  const checkLeftoversFor = useCallback(async (displayName: string) => {
    setCheckingLeftovers(true);
    try {
      const res = await window.electronAPI.orphanScan();
      if (res.code !== 0 || !res.output.trim()) return;
      const results: OrphanScanResultDto[] = JSON.parse(res.output);
      const match = results.find((r) => r.app.displayName.toLowerCase() === displayName.toLowerCase());
      if (match && (match.totalSize > 0 || (match.foundPaths?.length ?? 0) > 0)) {
        setLeftoverOffer(match);
      }
    } catch {
      // Мусор не найден/не распарсился — не критично, просто не показываем предложение.
    } finally {
      setCheckingLeftovers(false);
    }
  }, []);

  const handleUninstall = useCallback(async (app: InstalledAppDto) => {
    setUninstallConfirm(null);
    setUninstallingFor(app.displayName);
    setUninstallInfo(null);
    try {
      const res = await window.electronAPI.launchUninstaller(app.displayName);
      if (res.success) {
        setApps((prev) => prev.filter((a) => a.displayName !== app.displayName));
        setUninstallInfo(`«${app.displayName}» удалена.`);
        if (app.inOrphanDB) {
          await checkLeftoversFor(app.displayName);
        }
      } else {
        onError(res.error || 'Не удалось запустить деинсталлятор');
      }
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Ошибка запуска деинсталлятора');
    } finally {
      setUninstallingFor(null);
    }
  }, [onError, checkLeftoversFor]);

  const handleCleanLeftovers = useCallback(async () => {
    if (!leftoverOffer) return;
    setCleaningLeftovers(true);
    try {
      const res = await window.electronAPI.orphanClean({ names: leftoverOffer.app.displayName, cacheOnly: true });
      if (res.code === 0) {
        setUninstallInfo(`Мусор «${leftoverOffer.app.displayName}» очищен (~${formatBytes(leftoverOffer.totalSize)}).`);
        setLeftoverOffer(null);
      } else {
        onError(res.error || 'Не удалось очистить мусор');
      }
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Ошибка очистки мусора');
    } finally {
      setCleaningLeftovers(false);
    }
  }, [leftoverOffer, onError]);

  return (
    <Box>
      <Paper
        elevation={2}
        sx={{
          p: 3, mb: 2, borderRadius: 3,
          background: (t) => t.palette.mode === 'dark'
            ? 'linear-gradient(135deg, #1e3a8a 0%, #3b82f6 100%)'
            : 'linear-gradient(135deg, #dbeafe 0%, #bfdbfe 100%)',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
          <AppsIcon sx={{ mr: 1.5, fontSize: 28 }} />
          <Typography variant="h6" sx={{ fontWeight: 700 }}>Программы</Typography>
        </Box>
        <Alert severity="info" sx={{ mb: 2, borderRadius: 1.5 }}>
          Список установленных программ (реестр «Программы и компоненты»). Удаление запускает официальный
          деинсталлятор производителя — вы сами проходите его мастер. Если для программы известна база остатков,
          после удаления будет предложено сразу очистить её мусор.
        </Alert>
        <Button
          variant="contained"
          onClick={load}
          disabled={loading}
          startIcon={loading ? <CircularProgress size={20} /> : <RefreshIcon />}
          sx={{ borderRadius: 2, textTransform: 'none', fontWeight: 600 }}
        >
          {loading ? 'Загрузка...' : 'Обновить'}
        </Button>
        {loaded && (
          <Chip label={`${apps.length} программ`} size="small" sx={{ ml: 1.5, fontWeight: 600 }} />
        )}
      </Paper>

      <ScanningIndicator active={loading} />

      {uninstallInfo && (
        <Alert severity="success" sx={{ mb: 1.5, borderRadius: 2 }} onClose={() => setUninstallInfo(null)}>
          {uninstallInfo}
        </Alert>
      )}

      {checkingLeftovers && (
        <Alert severity="info" sx={{ mb: 1.5, borderRadius: 2 }} icon={<CircularProgress size={16} />}>
          Проверка остатков программы...
        </Alert>
      )}

      {loaded && apps.length > 0 && (
        <TextField
          fullWidth
          size="small"
          placeholder="Фильтр по имени или издателю..."
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

      {loaded && apps.length === 0 && (
        <Alert severity="warning" sx={{ borderRadius: 2 }}>Не удалось получить список программ.</Alert>
      )}

      {filtered.length > 0 && (
        <Paper variant="outlined" sx={{ maxHeight: 520, overflow: 'auto', borderRadius: 3, borderColor: 'divider' }}>
          <List dense disablePadding>
            {filtered.map((app, idx) => (
              <Box key={app.displayName}>
                <ListItem
                  sx={{ py: 1.25, px: 2 }}
                  secondaryAction={
                    <Tooltip title={app.canUninstall ? 'Удалить программу' : 'Нет команды удаления для этой записи'}>
                      <span>
                        <IconButton
                          edge="end"
                          color="error"
                          size="small"
                          disabled={!app.canUninstall || uninstallingFor === app.displayName}
                          onClick={() => setUninstallConfirm(app)}
                          sx={{ border: '1px solid', borderColor: 'error.main', borderRadius: 1.5, '&:hover': { bgcolor: 'error.main', color: 'white' } }}
                        >
                          {uninstallingFor === app.displayName ? <CircularProgress size={16} /> : <UninstallIcon fontSize="small" />}
                        </IconButton>
                      </span>
                    </Tooltip>
                  }
                >
                  <ListItemText
                    primary={
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        {app.displayName}
                        {app.inOrphanDB && (
                          <Tooltip title="Для этой программы известны типичные места остатков — после удаления будет предложено их почистить">
                            <Chip label="известны остатки" size="small" color="info" sx={{ height: 18, fontSize: '0.65rem', fontWeight: 600 }} />
                          </Tooltip>
                        )}
                      </Box>
                    }
                    secondary={app.publisher || app.installLocation || undefined}
                    primaryTypographyProps={{ variant: 'body2', sx: { fontWeight: 600 } }}
                    secondaryTypographyProps={{ variant: 'caption', sx: { opacity: 0.7 } }}
                  />
                </ListItem>
                {idx < filtered.length - 1 && <Divider />}
              </Box>
            ))}
          </List>
        </Paper>
      )}

      {loaded && apps.length > 0 && filtered.length === 0 && (
        <Alert severity="success" sx={{ borderRadius: 2 }}>Ничего не найдено по фильтру.</Alert>
      )}

      {/* Uninstall Confirm Dialog */}
      <Dialog open={!!uninstallConfirm} onClose={() => setUninstallConfirm(null)} maxWidth="sm" fullWidth PaperProps={{ sx: { borderRadius: 2 } }}>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <WarningIcon color="warning" />
          Удалить программу?
        </DialogTitle>
        <DialogContent>
          <Typography gutterBottom>
            Будет запущен официальный деинсталлятор программы «{uninstallConfirm?.displayName}» (тот же, что в
            «Программы и компоненты»).
          </Typography>
          <Alert severity="info" sx={{ borderRadius: 1.5 }}>
            Ничего не удаляется автоматически и без вашего подтверждения — откроется мастер удаления производителя,
            вы сами проходите его до конца.
          </Alert>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setUninstallConfirm(null)} sx={{ borderRadius: 1.5 }}>Отмена</Button>
          <Button
            onClick={() => uninstallConfirm && handleUninstall(uninstallConfirm)}
            variant="contained"
            color="error"
            startIcon={<UninstallIcon />}
            sx={{ borderRadius: 1.5 }}
          >
            Удалить
          </Button>
        </DialogActions>
      </Dialog>

      {/* Leftover Cleanup Offer Dialog */}
      <Dialog open={!!leftoverOffer} onClose={() => !cleaningLeftovers && setLeftoverOffer(null)} maxWidth="sm" fullWidth PaperProps={{ sx: { borderRadius: 2 } }}>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <CleanIcon color="info" />
          Найден мусор от «{leftoverOffer?.app.displayName}»
        </DialogTitle>
        <DialogContent>
          <Typography gutterBottom>
            После удаления остались файлы: ~{leftoverOffer ? formatBytes(leftoverOffer.totalSize) : ''}
            {leftoverOffer ? `, ${leftoverOffer.totalFiles} файлов` : ''}.
          </Typography>
          {leftoverOffer?.foundPaths && leftoverOffer.foundPaths.length > 0 && (
            <Paper variant="outlined" sx={{ p: 1.5, mb: 1.5, maxHeight: 160, overflow: 'auto', borderRadius: 1.5 }}>
              {leftoverOffer.foundPaths.map((p) => (
                <Typography key={p.path} variant="caption" sx={{ display: 'block', wordBreak: 'break-all', opacity: 0.8 }}>
                  {p.path}
                </Typography>
              ))}
            </Paper>
          )}
          <Alert severity="success" sx={{ borderRadius: 1.5 }}>
            Будет удалён только кеш программы (безопасно) — настройки и данные не затрагиваются.
          </Alert>
          {cleaningLeftovers && <LinearProgress sx={{ mt: 2 }} />}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setLeftoverOffer(null)} disabled={cleaningLeftovers} sx={{ borderRadius: 1.5 }}>Не сейчас</Button>
          <Button
            onClick={handleCleanLeftovers}
            variant="contained"
            color="info"
            disabled={cleaningLeftovers}
            startIcon={cleaningLeftovers ? <CircularProgress size={16} /> : <CleanIcon />}
            sx={{ borderRadius: 1.5 }}
          >
            {cleaningLeftovers ? 'Очистка...' : 'Очистить кеш'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
