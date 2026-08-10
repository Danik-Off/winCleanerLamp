/**
 * About Panel Component - Clean & Minimal
 */
import { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Typography,
  Card,
  CardContent,
  Grid,
  Paper,
  Button,
  Chip,
  CircularProgress,
  LinearProgress,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Alert,
  useTheme,
} from '@mui/material';
import {
  CleaningServices as Icon,
  Info as InfoIcon,
  Code as CodeIcon,
  Security as SecurityIcon,
  SystemUpdateAlt as UpdateIcon,
  CheckCircle as CheckIcon,
} from '@mui/icons-material';

type UpdateState = 'idle' | 'checking' | 'up-to-date' | 'available' | 'downloading' | 'downloaded' | 'error';

export function AboutPanel(): JSX.Element {
  const theme = useTheme();

  // Версия берётся из package.json через VITE_APP_VERSION
  // При сборке в CI/CD автоматически подставляется из package.json
  const appVersion = import.meta.env.VITE_APP_VERSION || '1.0.0';
  // Хеш коммита подставляется при сборке
  const buildHash = import.meta.env.VITE_BUILD_HASH || 'unknown';
  const currentYear = new Date().getFullYear();
  const author = 'Danik Off';

  const [updateState, setUpdateState] = useState<UpdateState>('idle');
  const [newVersion, setNewVersion] = useState('');
  const [releaseNotes, setReleaseNotes] = useState('');
  const [downloadPercent, setDownloadPercent] = useState(0);
  const [updateError, setUpdateError] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);

  useEffect(() => {
    window.electronAPI.onUpdateAvailable((info) => {
      setNewVersion(info.version);
      setReleaseNotes(info.releaseNotes || '');
      setUpdateState('available');
      setDialogOpen(true);
    });
    window.electronAPI.onUpdateNotAvailable(() => {
      setUpdateState('up-to-date');
    });
    window.electronAPI.onUpdateError((message) => {
      setUpdateError(message);
      setUpdateState('error');
    });
    window.electronAPI.onUpdateDownloadProgress((percent) => {
      setDownloadPercent(percent);
      setUpdateState('downloading');
    });
    window.electronAPI.onUpdateDownloaded(() => {
      setUpdateState('downloaded');
    });
    return () => {
      window.electronAPI.removeAllListeners('update-available');
      window.electronAPI.removeAllListeners('update-not-available');
      window.electronAPI.removeAllListeners('update-error');
      window.electronAPI.removeAllListeners('update-download-progress');
      window.electronAPI.removeAllListeners('update-downloaded');
    };
  }, []);

  const handleCheck = useCallback(async () => {
    setUpdateState('checking');
    setUpdateError('');
    const res = await window.electronAPI.checkForUpdates();
    if (!res.success) {
      setUpdateError(res.error || 'Не удалось проверить обновления');
      setUpdateState('error');
    }
    // При успехе состояние обновится через события update-available/update-not-available.
  }, []);

  const handleInstall = useCallback(async () => {
    const res = await window.electronAPI.installUpdate();
    if (!res.success) {
      setUpdateError(res.error || 'Не удалось скачать обновление');
      setUpdateState('error');
      setDialogOpen(false);
    }
    // Успех -> события update-download-progress / update-downloaded.
  }, []);

  const updateStatusLabel: Record<UpdateState, string> = {
    idle: '',
    checking: 'Проверка...',
    'up-to-date': 'Установлена последняя версия',
    available: `Доступна версия ${newVersion}`,
    downloading: `Скачивание... ${downloadPercent}%`,
    downloaded: 'Установка и перезапуск...',
    error: updateError || 'Ошибка проверки обновлений',
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', p: 3, maxWidth: 700, mx: 'auto' }}>

      {/* Header */}
      <Box sx={{ textAlign: 'center', mb: 4 }}>
        <Box sx={{
          display: 'inline-flex',
          alignItems: 'center',
          justifyContent: 'center',
          width: 80,
          height: 80,
          borderRadius: 3,
          background: `linear-gradient(135deg, #6366f1, #8b5cf6)`,
          mb: 2,
        }}>
          <Icon sx={{ color: 'white', fontSize: 40 }} />
        </Box>
        <Typography variant="h4" sx={{ fontWeight: 800, mb: 0.5 }}>
          WinCleaner Pro
        </Typography>
        <Typography variant="body2" sx={{ opacity: 0.6 }}>
          Профессиональная очистка системы
        </Typography>
      </Box>

      {/* Info Cards */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6}>
          <Card sx={{ bgcolor: theme.palette.mode === 'dark' ? 'rgba(99, 102, 241, 0.08)' : 'rgba(99, 102, 241, 0.05)', border: 'none' }}>
            <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', width: 44, height: 44, borderRadius: 2, bgcolor: 'rgba(99, 102, 241, 0.2)' }}>
                <InfoIcon sx={{ fontSize: 22, color: '#6366f1' }} />
              </Box>
              <Box>
                <Typography variant="caption" sx={{ opacity: 0.6, display: 'block' }}>Версия</Typography>
                <Typography variant="h6" sx={{ fontWeight: 700 }}>{appVersion}</Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6}>
          <Card sx={{ bgcolor: theme.palette.mode === 'dark' ? 'rgba(139, 92, 246, 0.08)' : 'rgba(139, 92, 246, 0.05)', border: 'none' }}>
            <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', width: 44, height: 44, borderRadius: 2, bgcolor: 'rgba(139, 92, 246, 0.2)' }}>
                <CodeIcon sx={{ fontSize: 22, color: '#8b5cf6' }} />
              </Box>
              <Box>
                <Typography variant="caption" sx={{ opacity: 0.6, display: 'block' }}>Коммит</Typography>
                <Typography variant="h6" sx={{ fontWeight: 700, fontFamily: 'monospace' }}>{buildHash.slice(0, 8)}</Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Updates */}
      <Paper sx={{
        p: 3,
        width: '100%',
        borderRadius: 3,
        bgcolor: theme.palette.mode === 'dark' ? 'rgba(59, 130, 246, 0.05)' : 'rgba(59, 130, 246, 0.03)',
        border: `1px solid ${theme.palette.mode === 'dark' ? 'rgba(59, 130, 246, 0.12)' : 'rgba(59, 130, 246, 0.08)'}`,
        mb: 3,
      }}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 1.5 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <UpdateIcon sx={{ fontSize: 20, color: '#3b82f6' }} />
            <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>Обновления</Typography>
            {updateState !== 'idle' && (
              <Chip
                size="small"
                label={updateStatusLabel[updateState]}
                color={updateState === 'up-to-date' ? 'success' : updateState === 'error' ? 'error' : 'default'}
                icon={updateState === 'up-to-date' ? <CheckIcon sx={{ fontSize: 14 }} /> : undefined}
                sx={{ fontWeight: 600 }}
              />
            )}
          </Box>
          <Button
            variant="outlined"
            size="small"
            onClick={handleCheck}
            disabled={updateState === 'checking' || updateState === 'downloading' || updateState === 'downloaded'}
            startIcon={updateState === 'checking' ? <CircularProgress size={14} /> : <UpdateIcon sx={{ fontSize: 16 }} />}
            sx={{ borderRadius: 2, textTransform: 'none', fontWeight: 600 }}
          >
            Проверить обновления
          </Button>
        </Box>
        {updateState === 'error' && (
          <Alert severity="warning" sx={{ mt: 2, borderRadius: 1.5 }}>{updateError}</Alert>
        )}
      </Paper>

      {/* Features */}
      <Paper sx={{
        p: 3,
        borderRadius: 3,
        bgcolor: theme.palette.mode === 'dark' ? 'rgba(16, 185, 129, 0.05)' : 'rgba(16, 185, 129, 0.03)',
        border: `1px solid ${theme.palette.mode === 'dark' ? 'rgba(16, 185, 129, 0.1)' : 'rgba(16, 185, 129, 0.05)'}`,
        mb: 3,
      }}>
        <Typography variant="subtitle1" sx={{ fontWeight: 700, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
          <SecurityIcon sx={{ fontSize: 18, color: '#10b981' }} />
          Безопасность
        </Typography>
        <Typography variant="body2" sx={{ opacity: 0.8, lineHeight: 1.7 }}>
          Все операции очистки выполняются безопасно. Временные файлы удаляются в Корзину,
          а не навсегда. Критические системные файлы защищены от удаления.
        </Typography>
      </Paper>

      {/* Footer */}
      <Box sx={{ textAlign: 'center', mt: 2 }}>
        <Typography variant="caption" sx={{ opacity: 0.6, display: 'block', mb: 0.5 }}>
          © {currentYear} {author} • WinCleaner Pro
        </Typography>
        <Typography variant="caption" sx={{ opacity: 0.4, display: 'block' }}>
          Clean Architecture • Material Design
        </Typography>
      </Box>

      {/* Update dialog: info + Install/Cancel, затем прогресс скачивания */}
      <Dialog open={dialogOpen && (updateState === 'available' || updateState === 'downloading' || updateState === 'downloaded')} maxWidth="xs" fullWidth>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1, fontWeight: 700 }}>
          <UpdateIcon color="primary" />
          Доступно обновление {newVersion}
        </DialogTitle>
        <DialogContent>
          {updateState === 'available' && (
            <>
              <Typography variant="body2" sx={{ mb: releaseNotes ? 1.5 : 0, opacity: 0.8 }}>
                Текущая версия: {appVersion} → {newVersion}
              </Typography>
              {releaseNotes && (
                <Paper variant="outlined" sx={{ p: 1.5, maxHeight: 180, overflow: 'auto', borderRadius: 1.5 }}>
                  <Typography variant="caption" sx={{ whiteSpace: 'pre-wrap' }}>{releaseNotes}</Typography>
                </Paper>
              )}
            </>
          )}
          {updateState === 'downloading' && (
            <Box>
              <Typography variant="body2" sx={{ mb: 1 }}>Скачивание обновления...</Typography>
              <LinearProgress variant="determinate" value={downloadPercent} sx={{ borderRadius: 2, height: 6, mb: 0.5 }} />
              <Typography variant="caption" sx={{ opacity: 0.6 }}>{downloadPercent}%</Typography>
            </Box>
          )}
          {updateState === 'downloaded' && (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
              <CircularProgress size={20} />
              <Typography variant="body2">Готово. Приложение перезапустится для установки...</Typography>
            </Box>
          )}
        </DialogContent>
        {updateState === 'available' && (
          <DialogActions sx={{ px: 3, pb: 2.5 }}>
            <Button onClick={() => setDialogOpen(false)} sx={{ borderRadius: 1.5 }}>Отмена</Button>
            <Button variant="contained" onClick={handleInstall} sx={{ borderRadius: 1.5, fontWeight: 700 }}>
              Установить
            </Button>
          </DialogActions>
        )}
      </Dialog>
    </Box>
  );
}
