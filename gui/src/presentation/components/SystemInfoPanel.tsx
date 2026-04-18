/**
 * System Info Panel Component
 * Displays large system files that cannot be auto-cleaned
 */
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
  Card,
  CardContent,
  Tooltip,
  Divider,
} from '@mui/material';
import {
  Info as InfoIcon,
  ContentCopy as CopyIcon,
  Storage as StorageIcon,
} from '@mui/icons-material';
import { useSystemInfo } from '../hooks';

interface SystemInfoPanelProps {
  onError: (error: string) => void;
}

const sysFileDescriptions: Record<string, { description: string; command?: string; commandLabel?: string }> = {
  'hiberfil.sys': {
    description: 'Файл гибернации Windows. Содержит снимок оперативной памяти для быстрого пробуждения. Можно отключить, если не используете спящий режим.',
    command: 'powercfg /h off',
    commandLabel: 'Отключить гибернацию',
  },
  'pagefile.sys': {
    description: 'Файл подкачки (виртуальная память). Windows использует его, когда не хватает ОЗУ. Размер можно уменьшить, но не рекомендуется отключать полностью.',
    command: 'SystemPropertiesPerformance.exe',
    commandLabel: 'Настройки виртуальной памяти',
  },
  'swapfile.sys': {
    description: 'Файл подкачки для UWP-приложений из Microsoft Store. Обычно небольшой (до 256 МБ). Удаляется вместе с pagefile.sys.',
  },
  'WinSxS': {
    description: 'Хранилище компонентов Windows (Side-by-Side). Содержит все версии системных файлов для обновлений и восстановления. Можно безопасно сжать через DISM.',
    command: 'dism /Online /Cleanup-Image /StartComponentCleanup /ResetBase',
    commandLabel: 'Очистить хранилище компонентов',
  },
  'System Volume Information': {
    description: 'Системная папка для точек восстановления, индексации и теневых копий. Управляется через настройки защиты системы.',
  },
  'Windows.old': {
    description: 'Резервная копия предыдущей версии Windows после обновления. Можно безопасно удалить через "Очистку диска" после проверки стабильности.',
  },
  '$WINDOWS.~BT': {
    description: 'Временные файлы обновления Windows. Автоматически удаляются через 10 дней после обновления.',
  },
};

export function SystemInfoPanel({ onError }: SystemInfoPanelProps): JSX.Element {
  const { info, loading, error } = useSystemInfo();

  if (loading) {
    return (
      <Paper elevation={3} sx={{ p: 4, textAlign: 'center', borderRadius: 2 }}>
        <CircularProgress />
        <Typography sx={{ mt: 2, opacity: 0.7 }}>Загрузка системной информации...</Typography>
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

  return (
    <Box>
      {/* Header */}
      <Paper
        elevation={3}
        sx={{
          p: 3,
          mb: 2,
          borderRadius: 2,
          background: (theme) =>
            theme.palette.mode === 'dark'
              ? 'linear-gradient(135deg, #0d47a1 0%, #1565c0 100%)'
              : 'linear-gradient(135deg, #e3f2fd 0%, #bbdefb 100%)',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
          <InfoIcon sx={{ mr: 1.5, fontSize: 28 }} />
          <Typography variant="h6" sx={{ fontWeight: 700 }}>
            Системная информация
          </Typography>
        </Box>
        <Typography variant="body2" sx={{ opacity: 0.8 }}>
          Эти файлы не удаляются автоматически, но занимают много места
        </Typography>
      </Paper>

      {info && (
        <>
          {/* Total Card */}
          <Card
            elevation={2}
            sx={{
              mb: 2,
              borderRadius: 2,
              background: (theme) =>
                theme.palette.mode === 'dark'
                  ? 'linear-gradient(135deg, #b71c1c 0%, #c62828 100%)'
                  : 'linear-gradient(135deg, #ffebee 0%, #ffcdd2 100%)',
            }}
          >
            <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2, py: 2, '&:last-child': { pb: 2 } }}>
              <StorageIcon sx={{ fontSize: 36, opacity: 0.8 }} />
              <Box>
                <Typography variant="h5" sx={{ fontWeight: 700 }}>
                  {formatBytes(info.totalBytes)}
                </Typography>
                <Typography variant="caption" sx={{ opacity: 0.8 }}>
                  Итого системных файлов
                </Typography>
              </Box>
            </CardContent>
          </Card>

          {/* File List */}
          <Paper variant="outlined" sx={{ borderRadius: 2, mb: 2 }}>
            <List disablePadding>
              {info.existingFiles.map((file, idx) => {
                const desc = sysFileDescriptions[file.name];
                return (
                  <Box key={file.name}>
                    <ListItem sx={{ py: 1.5, px: 2, alignItems: 'flex-start' }}>
                      <ListItemText
                        primary={
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            <Typography variant="body1" sx={{ fontWeight: 600 }}>{file.name}</Typography>
                            <Chip
                              label={file.sizeFormatted}
                              color={
                                file.sizeUnknown ? 'info'
                                  : file.sizeBytes > 1024 ** 3 ? 'error'
                                  : file.sizeBytes > 500 * 1024 ** 2 ? 'warning'
                                  : 'default'
                              }
                              size="small"
                              sx={{ fontWeight: 600 }}
                            />
                          </Box>
                        }
                        secondary={
                          <Box component="span" sx={{ display: 'block' }}>
                            <Typography variant="caption" component="span" sx={{ opacity: 0.6, wordBreak: 'break-all', display: 'block' }}>
                              {file.path}
                            </Typography>
                            {desc && (
                              <Typography variant="caption" component="span" sx={{ display: 'block', mt: 0.5, opacity: 0.85, lineHeight: 1.4 }}>
                                {desc.description}
                              </Typography>
                            )}
                            {desc?.command && (
                              <Tooltip title="Скопировать команду в буфер обмена">
                                <Button
                                  variant="outlined"
                                  size="small"
                                  startIcon={<CopyIcon sx={{ fontSize: 14 }} />}
                                  onClick={() => navigator.clipboard.writeText(desc.command!)}
                                  sx={{ mt: 0.5, borderRadius: 1.5, textTransform: 'none', fontWeight: 600, fontSize: '0.7rem', py: 0.2 }}
                                >
                                  {desc.commandLabel || desc.command}
                                </Button>
                              </Tooltip>
                            )}
                          </Box>
                        }
                        secondaryTypographyProps={{ component: 'div' }}
                      />
                    </ListItem>
                    {idx < info.existingFiles.length - 1 && <Divider />}
                  </Box>
                );
              })}
            </List>
          </Paper>
        </>
      )}

      <Alert severity="info" sx={{ borderRadius: 2 }}>
        Для управления системными файлами используйте командную строку с правами администратора.
      </Alert>
    </Box>
  );
}
