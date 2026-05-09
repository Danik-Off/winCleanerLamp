/**
 * System Info Panel Component
 * Displays large system files that cannot be auto-cleaned - Professional UI
 */
import { useState } from 'react';
import {
  Paper,
  Typography,
  Button,
  Box,
  Chip,
  CircularProgress,
  Alert,
  Card,
  CardContent,
  Tooltip,
  Stack,
  Accordion,
  AccordionSummary,
  AccordionDetails,
} from '@mui/material';
import {
  Info as InfoIcon,
  ContentCopy as CopyIcon,
  Storage as StorageIcon,
  ExpandMore as ExpandMoreIcon,
  Warning as WarningIcon,
  CheckCircle as CheckCircleIcon,
  DeleteSweep as DeleteIcon,
} from '@mui/icons-material';
import { useSystemInfo } from '../hooks';

interface SystemInfoPanelProps {
  onError: (error: string) => void;
}

const sysFileDescriptions: Record<string, { 
  description: string; 
  command?: string; 
  commandLabel?: string;
  icon: React.ReactNode;
  color: string;
  impact: 'high' | 'medium' | 'low';
}> = {
  'hiberfil.sys': {
    description: 'Файл гибернации Windows. Содержит снимок оперативной памяти для быстрого пробуждения. Можно отключить, если не используете спящий режим.',
    command: 'powercfg /h off',
    commandLabel: 'Отключить гибернацию',
    icon: <StorageIcon sx={{ fontSize: 20 }} />,
    color: '#1e5a8a',
    impact: 'high',
  },
  'pagefile.sys': {
    description: 'Файл подкачки (виртуальная память). Windows использует его, когда не хватает ОЗУ. Размер можно уменьшить, но не рекомендуется отключать полностью.',
    command: 'SystemPropertiesPerformance.exe',
    commandLabel: 'Настройки виртуальной памяти',
    icon: <WarningIcon sx={{ fontSize: 20 }} />,
    color: '#f57c00',
    impact: 'medium',
  },
  'swapfile.sys': {
    description: 'Файл подкачки для UWP-приложений из Microsoft Store. Обычно небольшой (до 256 МБ). Удаляется вместе с pagefile.sys.',
    icon: <InfoIcon sx={{ fontSize: 20 }} />,
    color: '#7b1fa2',
    impact: 'low',
  },
  'WinSxS': {
    description: 'Хранилище компонентов Windows (Side-by-Side). Содержит все версии системных файлов для обновлений и восстановления. Можно безопасно сжать через DISM.',
    command: 'dism /Online /Cleanup-Image /StartComponentCleanup /ResetBase',
    commandLabel: 'Очистить хранилище',
    icon: <DeleteIcon sx={{ fontSize: 20 }} />,
    color: '#c62828',
    impact: 'high',
  },
  'System Volume Information': {
    description: 'Системная папка для точек восстановления, индексации и теневых копий. Управляется через настройки защиты системы.',
    icon: <InfoIcon sx={{ fontSize: 20 }} />,
    color: '#555',
    impact: 'medium',
  },
  'Windows.old': {
    description: 'Резервная копия предыдущей версии Windows после обновления. Можно безопасно удалить через "Очистку диска" после проверки стабильности.',
    icon: <DeleteIcon sx={{ fontSize: 20 }} />,
    color: '#2e7d32',
    impact: 'high',
  },
  '$WINDOWS.~BT': {
    description: 'Временные файлы обновления Windows. Автоматически удаляются через 10 дней после обновления.',
    icon: <InfoIcon sx={{ fontSize: 20 }} />,
    color: '#555',
    impact: 'low',
  },
};

export function SystemInfoPanel({ onError }: SystemInfoPanelProps): JSX.Element {
  const { info, loading, error } = useSystemInfo();
  const [expanded, setExpanded] = useState<string | false>(false);

  const handleAccordionChange = (panel: string) => (_event: React.SyntheticEvent, isExpanded: boolean) => {
    setExpanded(isExpanded ? panel : false);
  };

  if (loading) {
    return (
      <Paper
        elevation={3}
        sx={{
          p: 4, 
          textAlign: 'center', 
          borderRadius: 3,
          bgcolor: (theme) => theme.palette.mode === 'dark' ? '#1e293b' : '#ffffff',
        }}
      >
        <CircularProgress sx={{ mb: 2 }} />
        <Typography variant="body1" sx={{ opacity: 0.7 }}>
          Загрузка системной информации...
        </Typography>
      </Paper>
    );
  }

  if (error) {
    onError(error);
    return (
      <Alert severity="error" sx={{ borderRadius: 2 }}>
        {error}
      </Alert>
    );
  }

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getImpactColor = (impact: string) => {
    switch (impact) {
      case 'high': return (theme: any) => theme.palette.mode === 'dark' ? '#7f1d1d' : '#fef2f2';
      case 'medium': return (theme: any) => theme.palette.mode === 'dark' ? '#7c2d12' : '#fff7ed';
      default: return (theme: any) => theme.palette.mode === 'dark' ? '#1e3a5f' : '#eef2ff';
    }
  };

  const getImpactLabel = (size: number) => {
    if (size > 1024 ** 3) return 'Много места';
    if (size > 500 * 1024 ** 2) return 'Средний размер';
    return 'Мало места';
  };

  return (
    <Box>
      {/* Header Card */}
      <Card
        elevation={2}
        sx={{
          mb: 3,
          borderRadius: 3,
          background: (theme) =>
            theme.palette.mode === 'dark'
              ? 'linear-gradient(135deg, #0d47a1 0%, #1565c0 100%)'
              : 'linear-gradient(135deg, #e3f2fd 0%, #bbdefb 100%)',
          p: 1,
        }}
      >
        <Box sx={{ p: 2.5, display: 'flex', alignItems: 'center', gap: 2 }}>
          <Box sx={{
            p: 1.5,
            bgcolor: 'rgba(255,255,255,0.2)',
            borderRadius: 3,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}>
            <InfoIcon sx={{ color: 'white', fontSize: 32 }} />
          </Box>
          <Box sx={{ flexGrow: 1 }}>
            <Typography variant="h6" sx={{ fontWeight: 700, color: 'white' }}>
              Системная информация
            </Typography>
            <Typography variant="body2" sx={{ color: 'rgba(255,255,255,0.9)', mt: 0.5 }}>
              Файлы, которые занимают место, но не удаляются автоматически
            </Typography>
          </Box>
        </Box>
      </Card>

      {info && info.existingFiles.length > 0 && (
        <>
          {/* Total Stats Card */}
          <Card
            elevation={1}
            sx={{
              mb: 3,
              borderRadius: 3,
              background: (theme) =>
                theme.palette.mode === 'dark'
                  ? 'linear-gradient(135deg, #1b5e20 0%, #2e7d32 100%)'
                  : 'linear-gradient(135deg, #e8f5e9 0%, #c8e6c9 100%)',
            }}
          >
            <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 3, py: 2.5 }}>
              <Box sx={{
                p: 1.5,
                bgcolor: 'rgba(255,255,255,0.2)',
                borderRadius: 3,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}>
                <StorageIcon sx={{ color: 'white', fontSize: 32 }} />
              </Box>
              <Box sx={{ flexGrow: 1 }}>
                <Typography variant="h5" sx={{ fontWeight: 700, color: 'white' }}>
                  {formatBytes(info.totalBytes)}
                </Typography>
                <Typography variant="body2" sx={{ color: 'rgba(255,255,255,0.9)', mt: 0.5 }}>
                  всего системных файлов
                </Typography>
              </Box>
              <Box sx={{ textAlign: 'right' }}>
                <Chip
                  label={`${info.existingFiles.length} файлов`}
                  sx={{
                    bgcolor: 'rgba(255,255,255,0.2)',
                    color: 'white',
                    fontWeight: 600,
                    border: '1px solid rgba(255,255,255,0.3)',
                  }}
                />
              </Box>
            </CardContent>
          </Card>

          {/* Files List */}
          <Stack spacing={2}>
            {info.existingFiles.map((file, idx) => {
              const desc = sysFileDescriptions[file.name] || {
                description: 'Системный файл Windows',
                icon: <InfoIcon sx={{ fontSize: 20 }} />,
                color: '#555',
                impact: 'low' as const,
              };
              
              return (
                <Accordion
                  expanded={expanded === `panel-${idx}`}
                  onChange={handleAccordionChange(`panel-${idx}`)}
                  key={file.name}
                  elevation={1}
                  sx={{
                    borderRadius: 3,
                    bgcolor: (theme) => theme.palette.mode === 'dark' ? '#1e293b' : '#ffffff',
                    '&:before': { display: 'none' },
                  }}
                >
                  <AccordionSummary
                    expandIcon={<ExpandMoreIcon sx={{ color: '#1e5a8a' }} />}
                    sx={{ bgcolor: getImpactColor(desc.impact) }}
                  >
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flexGrow: 1 }}>
                      <Box sx={{
                        p: 0.75,
                        bgcolor: `${desc.color}20`,
                        borderRadius: 2,
                        display: 'flex',
                        alignItems: 'center',
                        color: desc.color,
                      }}>
                        {desc.icon}
                      </Box>
                      <Box sx={{ flexGrow: 1 }}>
                        <Typography variant="body1" sx={{ fontWeight: 600 }}>
                          {file.name}
                        </Typography>
                        <Typography variant="caption" sx={{ opacity: 0.7, display: 'block' }}>
                          {file.path}
                        </Typography>
                      </Box>
                      <Chip
                        label={file.sizeFormatted}
                        sx={{
                          fontWeight: 600,
                          bgcolor: (theme) => theme.palette.mode === 'dark' ? '#0f172a' : '#f8fafc',
                        }}
                      />
                    </Box>
                  </AccordionSummary>
                  <AccordionDetails sx={{ pt: 1.5 }}>
                    <Typography variant="body2" sx={{ mb: 2, lineHeight: 1.6 }}>
                      {desc.description}
                    </Typography>
                    
                    {desc.command && (
                      <Box sx={{ 
                        p: 2, 
                        bgcolor: (theme) => theme.palette.mode === 'dark' ? '#0f172a' : '#f1f5f9',
                        borderRadius: 2,
                        mb: 1.5,
                      }}>
                        <Typography variant="caption" sx={{ fontWeight: 600, display: 'block', mb: 1 }}>
                          Рекомендованное действие:
                        </Typography>
                        <Tooltip title="Скопировать команду">
                          <Button
                            variant="outlined"
                            size="small"
                            startIcon={<CopyIcon sx={{ fontSize: 16 }} />}
                            onClick={() => navigator.clipboard.writeText(desc.command!)}
                            sx={{
                              borderRadius: 2,
                              textTransform: 'none',
                              fontWeight: 600,
                              px: 2,
                              borderColor: desc.color,
                              color: desc.color,
                              '&:hover': {
                                bgcolor: `${desc.color}15`,
                                borderColor: desc.color,
                              },
                            }}
                          >
                            {desc.commandLabel}
                          </Button>
                        </Tooltip>
                        <Typography variant="caption" sx={{ display: 'block', mt: 1, opacity: 0.7 }}>
                          {desc.command}
                        </Typography>
                      </Box>
                    )}
                    
                    <Chip
                      label={getImpactLabel(file.sizeBytes)}
                      size="small"
                      sx={{
                        fontWeight: 600,
                        bgcolor: `${desc.color}15`,
                        color: desc.color,
                      }}
                    />
                  </AccordionDetails>
                </Accordion>
              );
            })}
          </Stack>
        </>
      )}

      {info && info.existingFiles.length === 0 && (
        <Paper
          elevation={2}
          sx={{
            p: 4,
            textAlign: 'center',
            borderRadius: 3,
            bgcolor: (theme) => theme.palette.mode === 'dark' ? '#1e293b' : '#ffffff',
          }}
        >
          <CheckCircleIcon sx={{ fontSize: 48, color: '#2e7d32', mb: 2 }} />
          <Typography variant="h6" sx={{ fontWeight: 600, mb: 1 }}>
            Системных файлов не найдено
          </Typography>
          <Typography variant="body2" sx={{ opacity: 0.7 }}>
            Все проверенные системные файлы отсутствуют или имеют нулевой размер
          </Typography>
        </Paper>
      )}

      <Alert 
        severity="info" 
        sx={{ 
          mt: 3, 
          borderRadius: 2,
          bgcolor: (theme) => theme.palette.mode === 'dark' ? '#1e3a5f' : '#e3f2fd',
        }}
      >
        <Typography variant="body2">
          <strong>Совет:</strong> Для управления системными файлами используйте командную строку с правами администратора.
          Изменения в этих файлах могут повлиять на работу системы.
        </Typography>
      </Alert>
    </Box>
  );
}
