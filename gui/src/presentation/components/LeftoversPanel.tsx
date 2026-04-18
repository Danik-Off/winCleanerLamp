/**
 * Leftovers Panel Component
 * Displays potentially orphaned folders from uninstalled programs
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
  Divider,
} from '@mui/material';
import {
  FolderDelete as FolderDeleteIcon,
  Refresh as RefreshIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import { useLeftovers } from '../hooks';

interface LeftoversPanelProps {
  onError: (error: string) => void;
}

export function LeftoversPanel({ onError }: LeftoversPanelProps): JSX.Element {
  const { scanning, result, error, scan } = useLeftovers();

  if (error) {
    onError(error);
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
        <FolderDeleteIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
        Остатки удалённых программ
      </Typography>

      <Alert severity="warning" sx={{ mb: 2 }} icon={<WarningIcon />}>
        ⚠️ Это эвристика. Перед удалением проверьте каждую папку вручную!
        Программа автоматически не удаляет эти файлы.
      </Alert>

      <Button
        variant="contained"
        onClick={scan}
        disabled={scanning}
        startIcon={scanning ? <CircularProgress size={20} /> : <RefreshIcon />}
        sx={{ mb: 2 }}
      >
        {scanning ? 'Сканирование...' : 'Сканировать остатки'}
      </Button>

      {result && result.items.length > 0 && (
        <Box sx={{ mb: 2 }}>
          <Typography variant="h6" color="primary">
            Найдено: {formatBytes(result.totalBytes)} в {result.folderCount} папках
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {result.totalFiles} файлов
          </Typography>
        </Box>
      )}

      {result && result.items.length > 0 ? (
        <Paper variant="outlined" sx={{ maxHeight: 400, overflow: 'auto' }}>
          <List dense>
            {result.sortedBySize.map((item) => (
              <React.Fragment key={item.path}>
                <ListItem>
                  <ListItemText
                    primary={item.directoryName}
                    secondary={item.path}
                    primaryTypographyProps={{
                      variant: 'body1',
                      sx: { fontWeight: 'bold' }
                    }}
                    secondaryTypographyProps={{
                      variant: 'caption',
                      sx: { wordBreak: 'break-all' }
                    }}
                  />
                  <Box sx={{ textAlign: 'right' }}>
                    <Chip
                      label={formatBytes(item.sizeBytes)}
                      color={item.sizeBytes > 1024 ** 3 ? 'warning' : 'default'}
                      size="small"
                    />
                    <Typography variant="caption" display="block" sx={{ mt: 0.5 }}>
                      {item.fileCount} файлов
                    </Typography>
                  </Box>
                </ListItem>
                <Divider />
              </React.Fragment>
            ))}
          </List>
        </Paper>
      ) : (
        !scanning && result && (
          <Alert severity="success">
            Остатков не найдено! Ваши AppData/ProgramData чисты.
          </Alert>
        )
      )}

      {result && result.topItems.length > 0 && (
        <Box sx={{ mt: 3, p: 2, bgcolor: 'background.paper', borderRadius: 1 }}>
          <Typography variant="subtitle2" gutterBottom>
            Топ-5 по размеру:
          </Typography>
          <List dense>
            {result.topItems.map((item, index) => (
              <ListItem key={item.path}>
                <Typography variant="body2" sx={{ mr: 1, minWidth: 24 }}>
                  {index + 1}.
                </Typography>
                <ListItemText
                  primary={item.directoryName}
                  secondary={formatBytes(item.sizeBytes)}
                />
              </ListItem>
            ))}
          </List>
        </Box>
      )}
    </Paper>
  );
}
