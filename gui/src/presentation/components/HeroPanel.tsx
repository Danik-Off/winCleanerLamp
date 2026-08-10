/**
 * Hero Panel Component — главный экран
 * Один большой круг запускает "скан → авточистка безопасных категорий".
 *
 * Намеренно минималистично: никаких зацикленных фоновых анимаций (пульс,
 * плавающие пятна) — они не несли информации и только отвлекали. Внутри
 * круга во время работы — ровно два элемента (процент + короткая строка
 * статуса), а не три вперемешку с крупной подписью фазы, которая физически
 * не помещалась в круг на словах вроде "ЧИСТКА".
 */
import React, { useState, useCallback, useEffect, useRef } from 'react';
import {
  Box,
  Typography,
  Button,
  CircularProgress,
  Alert,
  Grid,
  Card,
  CardContent,
  Paper,
  Stack,
  useTheme,
} from '@mui/material';
import { alpha } from '@mui/material/styles';
import {
  CheckCircle as CheckIcon,
  DeleteSweep as CleanIcon,
  Search as SearchIcon,
  AutoFixHigh as MagicIcon,
  Refresh as RefreshIcon,
  Shield as ShieldIcon,
  Bolt as BoltIcon,
  PowerSettingsNew as AutostartIcon,
} from '@mui/icons-material';
import { useCategories, useScan, useClean } from '../hooks';
import { CategorySelection } from '../../domain/entities/Category';

interface HeroPanelProps {
  onError: (error: string) => void;
}

type Phase = 'idle' | 'scanning' | 'cleaning' | 'completed';

export function HeroPanel({ onError }: HeroPanelProps): JSX.Element {
  const [phase, setPhase] = useState<Phase>('idle');
  const [progress, setProgress] = useState(0);
  const [shouldClean, setShouldClean] = useState(false);
  const animationRef = useRef<number | null>(null);
  const theme = useTheme();
  const dark = theme.palette.mode === 'dark';
  const { categories } = useCategories();

  const { scanning, error: scanError, scan, clear: clearScan, result: scanResult } = useScan();
  const { bytesCleaned, filesCleaned, error: cleanError, clean, clear: clearClean } = useClean();

  // Start clean after scan completes
  useEffect(() => {
    if (!scanning && scanResult && phase === 'scanning') {
      setProgress(50);
      setShouldClean(true);
    }
  }, [scanning, scanResult, phase]);

  useEffect(() => {
    if (shouldClean && phase === 'scanning') {
      setPhase('cleaning');
      setShouldClean(false);
      handleAutoClean();
    }
  }, [shouldClean, phase]);

  // Живой прогресс от CLI (см. gui/electron/main.ts: PROGRESS-строки на stderr).
  // Пока не пришло ни одного реального события — процент плавно (но без
  // остановки) подкрадывается к мягкому потолку фазы; как только событие
  // пришло — считаем реальную долю "[i/N]" внутри диапазона фазы.
  const [liveText, setLiveText] = useState<string | null>(null);
  const liveFracRef = useRef<number | null>(null);

  useEffect(() => {
    if (phase !== 'scanning' && phase !== 'cleaning') {
      setLiveText(null);
      liveFracRef.current = null;
      return;
    }
    const handler = (msg: string) => {
      setLiveText(msg);
      const m = msg.match(/^\[(\d+)\/(\d+)\]/);
      liveFracRef.current = m ? Math.min(1, parseInt(m[1], 10) / Math.max(1, parseInt(m[2], 10))) : null;
    };
    if (phase === 'scanning') window.electronAPI.onScanProgress(handler);
    if (phase === 'cleaning') window.electronAPI.onCleanProgress(handler);
    return () => {
      window.electronAPI.removeAllListeners('scan-progress');
      window.electronAPI.removeAllListeners('clean-progress');
    };
  }, [phase]);

  useEffect(() => {
    if (phase !== 'scanning' && phase !== 'cleaning') return;
    const [base, softCap] = phase === 'scanning' ? [10, 49] : [50, 99];
    const range = phase === 'scanning' ? 40 : 49;

    const tick = setInterval(() => {
      setProgress((prev) => {
        if (liveFracRef.current !== null) {
          return base + liveFracRef.current * range;
        }
        return prev + (softCap - prev) * 0.12;
      });
    }, 200);
    return () => clearInterval(tick);
  }, [phase]);

  const handleAutoClean = useCallback(async () => {
    try {
      let sel = CategorySelection.empty();
      categories.safe.forEach(c => { sel = sel.select(c.id); });
      await clean(false, sel);
      setPhase('completed');
      setProgress(100);
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Ошибка очистки');
      setPhase('idle');
      setProgress(0);
    }
  }, [clean, onError, categories.safe]);

  const handleScan = useCallback(async () => {
    clearScan();
    clearClean();
    setPhase('scanning');
    setProgress(10);
    setShouldClean(false);

    try {
      let sel = CategorySelection.empty();
      categories.safe.forEach(c => { sel = sel.select(c.id); });
      await scan(false, sel);
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Ошибка сканирования');
      setPhase('idle');
      setProgress(0);
    }
  }, [scan, clearScan, clearClean, onError, categories.safe]);

  const handleReset = useCallback(() => {
    clearScan();
    clearClean();
    setPhase('idle');
    setProgress(0);
    setShouldClean(false);
  }, [clearScan, clearClean]);

  const formatBytes = (b: number): string => {
    if (b === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(b) / Math.log(k));
    return parseFloat((b / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  useEffect(() => {
    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, []);

  const getStatus = () => {
    switch (phase) {
      case 'idle':
        return { icon: <MagicIcon />, color: '#8b5cf6', label: 'Начать', sublabel: 'Проверка и очистка' };
      case 'scanning':
        return { icon: <SearchIcon />, color: '#3b82f6', label: 'Сканирование', sublabel: '' };
      case 'cleaning':
        return { icon: <CleanIcon />, color: '#f59e0b', label: 'Очистка', sublabel: '' };
      case 'completed':
        return { icon: <CheckIcon />, color: '#10b981', label: 'Готово', sublabel: '' };
      default:
        return { icon: <MagicIcon />, color: '#8b5cf6', label: '', sublabel: '' };
    }
  };

  const status = getStatus();
  const isIdle = phase === 'idle';
  const isBusy = phase === 'scanning' || phase === 'cleaning';

  if (scanError || cleanError) {
    return (
      <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', minHeight: 400, p: 3 }}>
        <Alert severity="error" sx={{ mb: 3, maxWidth: 500 }} onClose={handleReset}>
          {scanError || cleanError}
        </Alert>
        <Button variant="outlined" onClick={handleReset} startIcon={<RefreshIcon />}>
          Попробовать снова
        </Button>
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', minHeight: 'calc(100vh - 200px)', p: 3 }}>
      <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', width: '100%', maxWidth: 640 }}>

        {/* Заголовок — только в состоянии покоя, одна строка */}
        {isIdle && (
          <Typography variant="body2" sx={{ opacity: 0.55, mb: 4 }}>
            Одно нажатие — сканирование и безопасная очистка
          </Typography>
        )}

        {/* Кнопка */}
        <Box sx={{ position: 'relative', mb: 4 }}>
          {isBusy && (
            <Box sx={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)' }}>
              <CircularProgress
                variant="determinate"
                value={100}
                size={220}
                thickness={3}
                sx={{ color: dark ? alpha('#ffffff', 0.06) : alpha('#000000', 0.06), position: 'absolute' }}
              />
              <CircularProgress
                variant="determinate"
                value={progress}
                size={220}
                thickness={3}
                sx={{
                  color: status.color,
                  '& .MuiCircularProgress-circle': { strokeLinecap: 'round', transition: 'stroke-dashoffset 0.3s ease' },
                }}
              />
            </Box>
          )}

          <Button
            variant="contained"
            onClick={isIdle ? handleScan : undefined}
            disabled={!isIdle}
            disableRipple={!isIdle}
            sx={{
              width: 200,
              height: 200,
              borderRadius: '50%',
              bgcolor: status.color,
              minWidth: 0,
              boxShadow: isIdle ? `0 8px 28px ${alpha(status.color, 0.35)}` : 'none',
              transition: 'box-shadow 0.25s ease, transform 0.25s ease, background-color 0.25s ease',
              position: 'relative',
              '&:hover': {
                bgcolor: status.color,
                boxShadow: isIdle ? `0 10px 32px ${alpha(status.color, 0.45)}` : 'none',
                transform: isIdle ? 'scale(1.03)' : 'none',
              },
              '&:disabled': {
                bgcolor: status.color,
                opacity: isBusy ? 1 : 0.55,
              },
            }}
          >
            {isBusy ? (
              <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
                <Typography sx={{ fontWeight: 700, color: 'white', fontSize: '2.5rem', lineHeight: 1, fontVariantNumeric: 'tabular-nums' }}>
                  {Math.round(progress)}%
                </Typography>
                <Typography
                  variant="caption"
                  sx={{
                    display: 'block',
                    mt: 1,
                    maxWidth: 148,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                    opacity: 0.85,
                    color: 'white',
                  }}
                >
                  {liveText || status.label}
                </Typography>
              </Box>
            ) : (
              <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
                <Box sx={{ color: 'white', mb: 1 }}>
                  {React.cloneElement(status.icon as React.ReactElement, { sx: { fontSize: 52 } })}
                </Box>
                <Typography sx={{ fontWeight: 700, color: 'white', fontSize: '1.1rem' }}>
                  {status.label}
                </Typography>
                {status.sublabel && (
                  <Typography variant="caption" sx={{ color: 'rgba(255,255,255,0.8)' }}>
                    {status.sublabel}
                  </Typography>
                )}
              </Box>
            )}
          </Button>
        </Box>

        {/* Шаги */}
        <Stack direction="row" spacing={3} sx={{ mb: 4, alignItems: 'center' }}>
          {[
            { label: 'Сканирование', phase: 'scanning' },
            { label: 'Очистка', phase: 'cleaning' },
            { label: 'Готово', phase: 'completed' },
          ].map((step, idx) => {
            const isActive = phase === step.phase ||
                            (idx === 0 && phase === 'cleaning') ||
                            (idx <= 1 && phase === 'completed');

            return (
              <Box key={idx} sx={{ display: 'flex', alignItems: 'center' }}>
                <Box sx={{
                  width: 10,
                  height: 10,
                  borderRadius: '50%',
                  bgcolor: isActive ? status.color : dark ? '#374151' : '#d1d5db',
                  transition: 'background-color 0.25s',
                }} />
                <Typography
                  variant="caption"
                  sx={{ ml: 0.75, fontWeight: isActive ? 700 : 400, opacity: isActive ? 1 : 0.4 }}
                >
                  {step.label}
                </Typography>
                {idx < 2 && (
                  <Box sx={{
                    width: 36,
                    height: 2,
                    mx: 1.5,
                    bgcolor: isActive ? status.color : dark ? '#374151' : '#d1d5db',
                    transition: 'background-color 0.25s',
                  }} />
                )}
              </Box>
            );
          })}
        </Stack>

        {/* Результат */}
        {phase === 'completed' && (
          <Paper sx={{
            p: 3,
            width: '100%',
            borderRadius: 4,
            bgcolor: dark ? 'rgba(16, 185, 129, 0.08)' : 'rgba(16, 185, 129, 0.05)',
            border: `1px solid ${dark ? 'rgba(16, 185, 129, 0.2)' : 'rgba(16, 185, 129, 0.1)'}`,
          }}>
            <Grid container spacing={3}>
              <Grid item xs={6}>
                <Card sx={{ bgcolor: dark ? 'rgba(16, 185, 129, 0.1)' : 'rgba(16, 185, 129, 0.05)', border: 'none' }}>
                  <CardContent sx={{ textAlign: 'center', py: 2 }}>
                    <Typography variant="h4" sx={{ fontWeight: 800, color: '#10b981' }}>
                      {formatBytes(bytesCleaned)}
                    </Typography>
                    <Typography variant="caption" sx={{ opacity: 0.7 }}>Освобождено</Typography>
                  </CardContent>
                </Card>
              </Grid>
              <Grid item xs={6}>
                <Card sx={{ bgcolor: dark ? 'rgba(139, 92, 246, 0.1)' : 'rgba(139, 92, 246, 0.05)', border: 'none' }}>
                  <CardContent sx={{ textAlign: 'center', py: 2 }}>
                    <Typography variant="h4" sx={{ fontWeight: 800, color: '#8b5cf6' }}>
                      {filesCleaned}
                    </Typography>
                    <Typography variant="caption" sx={{ opacity: 0.7 }}>Файлов</Typography>
                  </CardContent>
                </Card>
              </Grid>
              <Grid item xs={12}>
                <Button
                  variant="outlined"
                  onClick={handleReset}
                  startIcon={<RefreshIcon />}
                  fullWidth
                  sx={{ py: 1.5, fontWeight: 700, borderRadius: 2.5 }}
                >
                  Ещё раз
                </Button>
              </Grid>
            </Grid>
          </Paper>
        )}

        {/* Возможности — только в состоянии покоя */}
        {isIdle && (
          <Grid container spacing={1.5} sx={{ mt: 1 }}>
            {[
              { icon: <ShieldIcon sx={{ fontSize: 18 }} />, label: 'Безопасно', desc: 'системные файлы защищены', color: '#10b981' },
              { icon: <BoltIcon sx={{ fontSize: 18 }} />, label: 'Быстро', desc: 'параллельное сканирование', color: '#3b82f6' },
              { icon: <AutostartIcon sx={{ fontSize: 18 }} />, label: 'Автозагрузка', desc: 'обратимое управление', color: '#a855f7' },
            ].map((f) => (
              <Grid item xs={12} sm={4} key={f.label}>
                <Paper
                  elevation={0}
                  sx={{
                    p: 1.75,
                    borderRadius: 3,
                    textAlign: 'center',
                    border: '1px solid',
                    borderColor: dark ? '#1f2937' : '#e2e8f0',
                    bgcolor: 'background.paper',
                    transition: 'border-color 0.2s ease',
                    '&:hover': { borderColor: alpha(f.color, 0.4) },
                  }}
                >
                  <Box sx={{ color: f.color, mb: 0.5 }}>{f.icon}</Box>
                  <Typography variant="caption" sx={{ display: 'block', fontWeight: 700 }}>{f.label}</Typography>
                  <Typography variant="caption" sx={{ display: 'block', opacity: 0.5, fontSize: '0.68rem' }}>{f.desc}</Typography>
                </Paper>
              </Grid>
            ))}
          </Grid>
        )}
      </Box>
    </Box>
  );
}
