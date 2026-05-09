/**
 * Hero Panel Component - Clean & Minimal
 * Professional one-button interface
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
import {
  CheckCircle as CheckIcon,
  DeleteSweep as CleanIcon,
  Search as SearchIcon,
  AutoFixHigh as MagicIcon,
  Refresh as RefreshIcon,
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

  // Update progress during scanning/cleaning
  useEffect(() => {
    if (phase === 'scanning') {
      const interval = setInterval(() => {
        setProgress(prev => {
          if (prev >= 45) {
            clearInterval(interval);
            return 45;
          }
          return prev + 5;
        });
      }, 200);
      return () => clearInterval(interval);
    }
    
    if (phase === 'cleaning') {
      const interval = setInterval(() => {
        setProgress(prev => {
          if (prev >= 95) {
            clearInterval(interval);
            return 95;
          }
          return prev + 3;
        });
      }, 200);
      return () => clearInterval(interval);
    }
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
        return { 
          icon: <MagicIcon />,
          color: '#6366f1',
          bgColor: theme.palette.mode === 'dark' ? 'rgba(99, 102, 241, 0.1)' : 'rgba(99, 102, 241, 0.08)',
        };
      case 'scanning':
        return { 
          icon: <SearchIcon />,
          color: '#3b82f6',
          bgColor: theme.palette.mode === 'dark' ? 'rgba(59, 130, 246, 0.1)' : 'rgba(59, 130, 246, 0.08)',
        };
      case 'cleaning':
        return { 
          icon: <CleanIcon />,
          color: '#f59e0b',
          bgColor: theme.palette.mode === 'dark' ? 'rgba(245, 158, 11, 0.1)' : 'rgba(245, 158, 11, 0.08)',
        };
      case 'completed':
        return { 
          icon: <CheckIcon />,
          color: '#10b981',
          bgColor: theme.palette.mode === 'dark' ? 'rgba(16, 185, 129, 0.1)' : 'rgba(16, 185, 129, 0.08)',
        };
      default:
        return { icon: <MagicIcon />, color: '#6366f1', bgColor: 'transparent' };
    }
  };

  const status = getStatus();

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
      <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', width: '100%', maxWidth: 600 }}>
        
        {/* Main Button */}
        <Box sx={{ position: 'relative', mb: 4 }}>
          {/* Progress ring - centered */}
          {(phase === 'scanning' || phase === 'cleaning') && (
            <Box sx={{ 
              position: 'absolute',
              top: '50%',
              left: '50%',
              transform: 'translate(-50%, -50%)',
              zIndex: 0,
            }}>
              <CircularProgress
                variant="determinate"
                value={progress}
                size={220}
                thickness={4}
                sx={{ color: status.color }}
              />
            </Box>
          )}
          
          <Button
            variant="contained"
            onClick={phase === 'idle' ? handleScan : undefined}
            disabled={phase !== 'idle'}
            sx={{
              width: 200,
              height: 200,
              borderRadius: '50%',
              bgcolor: status.color,
              border: `4px solid ${status.bgColor}`,
              boxShadow: phase === 'idle' 
                ? `0 8px 30px ${status.color}44`
                : 'none',
              transition: 'all 0.3s ease',
              position: 'relative',
              zIndex: 1,
              '&:hover': {
                bgcolor: status.color,
                boxShadow: `0 12px 40px ${status.color}66`,
                transform: phase === 'idle' ? 'scale(1.05)' : 'none',
              },
              '&:disabled': {
                bgcolor: theme.palette.mode === 'dark' ? '#1e293b' : '#f1f5f9',
                color: status.color,
                boxShadow: 'none',
              },
            }}
          >
            <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
              <Box sx={{ mb: 1, color: phase === 'idle' ? 'white' : status.color }}>
                {React.cloneElement(status.icon as React.ReactElement, { sx: { fontSize: 64 } })}
              </Box>
              <Typography variant="h6" sx={{ fontWeight: 700, color: phase === 'idle' ? 'white' : status.color, textTransform: 'uppercase', letterSpacing: '0.1em' }}>
                {phase === 'idle' && 'Старт'}
                {phase === 'scanning' && 'Скан'}
                {phase === 'cleaning' && 'Чистка'}
                {phase === 'completed' && 'Готово'}
              </Typography>
              {phase === 'idle' && (
                <Typography variant="caption" sx={{ color: 'rgba(255,255,255,0.8)', mt: 0.5 }}>
                  Очистка
                </Typography>
              )}
              {(phase === 'scanning' || phase === 'cleaning') && (
                <Typography variant="h6" sx={{ fontWeight: 700, color: theme.palette.mode === 'dark' ? 'white' : '#1e293b', mt: 1 }}>
                  {Math.round(progress)}%
                </Typography>
              )}
            </Box>
          </Button>
        </Box>

        {/* Steps */}
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
                  width: 12,
                  height: 12,
                  borderRadius: '50%',
                  bgcolor: isActive ? status.color : theme.palette.mode === 'dark' ? '#374151' : '#d1d5db',
                  transition: 'all 0.3s',
                }} />
                <Typography 
                  variant="caption" 
                  sx={{ 
                    ml: 0.75, 
                    fontWeight: isActive ? 600 : 400,
                    opacity: isActive ? 1 : 0.4,
                  }}
                >
                  {step.label}
                </Typography>
                {idx < 2 && (
                  <Box sx={{ 
                    width: 40, 
                    height: 2, 
                    mx: 1.5, 
                    bgcolor: isActive ? status.color : theme.palette.mode === 'dark' ? '#374151' : '#d1d5db',
                    transition: 'all 0.3s',
                  }} />
                )}
              </Box>
            );
          })}
        </Stack>

        {/* Result */}
        {phase === 'completed' && (
          <Paper sx={{ 
            p: 3,
            borderRadius: 3,
            bgcolor: theme.palette.mode === 'dark' ? 'rgba(16, 185, 129, 0.08)' : 'rgba(16, 185, 129, 0.05)',
            border: `1px solid ${theme.palette.mode === 'dark' ? 'rgba(16, 185, 129, 0.2)' : 'rgba(16, 185, 129, 0.1)'}`,
          }}>
            <Grid container spacing={3}>
              <Grid item xs={6}>
                <Card sx={{ bgcolor: theme.palette.mode === 'dark' ? 'rgba(16, 185, 129, 0.1)' : 'rgba(16, 185, 129, 0.05)', border: 'none' }}>
                  <CardContent sx={{ textAlign: 'center', py: 2 }}>
                    <Typography variant="h4" sx={{ fontWeight: 800, color: '#10b981' }}>
                      {formatBytes(bytesCleaned)}
                    </Typography>
                    <Typography variant="caption" sx={{ opacity: 0.7 }}>Освобождено</Typography>
                  </CardContent>
                </Card>
              </Grid>
              <Grid item xs={6}>
                <Card sx={{ bgcolor: theme.palette.mode === 'dark' ? 'rgba(99, 102, 241, 0.1)' : 'rgba(99, 102, 241, 0.05)', border: 'none' }}>
                  <CardContent sx={{ textAlign: 'center', py: 2 }}>
                    <Typography variant="h4" sx={{ fontWeight: 800, color: '#6366f1' }}>
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
                  sx={{ py: 1.5, fontWeight: 600 }}
                >
                  Ещё раз
                </Button>
              </Grid>
            </Grid>
          </Paper>
        )}

        {/* Info */}
        {phase === 'idle' && (
          <Typography variant="caption" sx={{ opacity: 0.5, mt: 3, textAlign: 'center', display: 'block' }}>
            Безопасная очистка временных файлов, кеша и мусора
          </Typography>
        )}
      </Box>
    </Box>
  );
}
