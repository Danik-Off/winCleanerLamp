/**
 * System Info Panel Component
 * Displays large system files that cannot be auto-cleaned
 */
import React from 'react';
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
} from '@mui/material';
import { Info as InfoIcon } from '@mui/icons-material';
import { useSystemInfo } from '../hooks';

interface SystemInfoPanelProps {
  onError: (error: string) => void;
}

export function SystemInfoPanel({ onError }: SystemInfoPanelProps): JSX.Element {
  const { info, loading, error } = useSystemInfo();

  if (loading) {
    return (
      <Paper elevation={2} sx={{ p: 3, textAlign: 'center' }}>
        <CircularProgress />
        <Typography sx={{ mt: 2 }}>Загрузка системной информации...</Typography>
      </Paper>
    );
  }

  if (error) {
    onError(error);
    return (
      <Alert severity="error">
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
    <Paper elevation={2} sx={{ p: 3 }}>
      <Typography variant="h6" gutterBottom>
        <InfoIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
        Системная информация
      </Typography>

      <Typography variant="body2" color="text.secondary" gutterBottom>
        Эти файлы не удаляются автоматически, но занимают много места:
      </Typography>

      {info && (
        <>
          <List>
            {info.existingFiles.map((file) => (
              <ListItem key={file.name} divider>
                <ListItemText
                  primary={file.name}
                  secondary={file.path}
                  primaryTypographyProps={{ variant: 'subtitle1' }}
                />
                <Chip
                  label={formatBytes(file.sizeBytes)}
                  color={file.sizeBytes > 1024 ** 3 ? 'warning' : 'default'}
                  sx={{ ml: 2 }}
                />
              </ListItem>
            ))}
          </List>

          <Box sx={{ mt: 2, p: 2, bgcolor: 'background.paper', borderRadius: 1 }}>
            <Typography variant="subtitle2" gutterBottom>
              Итого системных файлов: {formatBytes(info.totalBytes)}
            </Typography>
          </Box>

          <Box sx={{ mt: 3 }}>
            <Typography variant="subtitle2" gutterBottom>Управление:</Typography>
            <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
              <Button
                variant="outlined"
                size="small"
                onClick={() => navigator.clipboard.writeText('powercfg /h off')}
              >
                Копировать: powercfg /h off
              </Button>
              <Button
                variant="outlined"
                size="small"
                onClick={() => navigator.clipboard.writeText('dism /Online /Cleanup-Image /StartComponentCleanup /ResetBase')}
              >
                Копировать: DISM cleanup
              </Button>
            </Box>
          </Box>
        </>
      )}

      <Alert severity="info" sx={{ mt: 2 }}>
        Для управления системными файлами используйте командную строку с правами администратора.
      </Alert>
    </Paper>
  );
}
