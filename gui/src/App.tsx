/**
 * Main Application Component
 * Clean Architecture presentation layer
 */
import React, { useState } from 'react';
import {
  Box,
  Typography,
  Container,
  Tabs,
  Tab,
  Alert,
  IconButton,
} from '@mui/material';
import {
  CleaningServices as CleaningServicesIcon,
  Speed as SpeedIcon,
  Storage as StorageIcon,
  FolderDelete as FolderDeleteIcon,
  ContentCopy as DupIcon,
  FolderOff as EmptyFolderIcon,
  Minimize as MinimizeIcon,
  CropSquare as MaximizeIcon,
  Close as CloseIcon,
} from '@mui/icons-material';

// Feature Components
import { CleanupPanel } from './presentation/components/CleanupPanel';
import { SystemInfoPanel } from './presentation/components/SystemInfoPanel';
import { LeftoversPanel } from './presentation/components/LeftoversPanel';
import { DuplicatesPanel } from './presentation/components/DuplicatesPanel';
import { EmptyFoldersPanel } from './presentation/components/EmptyFoldersPanel';

/**
 * Tab Panel Component
 */
interface TabPanelProps {
  children: React.ReactNode;
  value: number;
  index: number;
}

function TabPanel({ children, value, index }: TabPanelProps): JSX.Element | null {
  if (value !== index) {
    return null;
  }
  return (
    <Box role="tabpanel" sx={{ py: 2 }}>
      {children}
    </Box>
  );
}

/**
 * Main App Component
 */
function App(): JSX.Element {
  const [activeTab, setActiveTab] = useState(0);
  const [error, setError] = useState<string | null>(null);

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number): void => {
    setActiveTab(newValue);
    setError(null);
  };

  return (
    <Box sx={{ flexGrow: 1, height: '100vh', display: 'flex', flexDirection: 'column' }}>
      {/* Custom Title Bar */}
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          height: 38,
          bgcolor: (theme) =>
            theme.palette.mode === 'dark' ? '#0d0d0d' : '#e0e0e0',
          WebkitAppRegion: 'drag',
          userSelect: 'none',
          flexShrink: 0,
          px: 1,
        }}
      >
        <CleaningServicesIcon
          sx={{ fontSize: 18, mr: 1, opacity: 0.8 }}
        />
        <Typography
          variant="caption"
          sx={{ fontWeight: 600, opacity: 0.8, flexGrow: 1 }}
        >
          WinCleanerLamp
        </Typography>
        <Typography
          variant="caption"
          sx={{ opacity: 0.4, mr: 1 }}
        >
          v2.0.0
        </Typography>
        <Box sx={{ WebkitAppRegion: 'no-drag', display: 'flex' }}>
          <IconButton
            size="small"
            onClick={() => window.electronAPI?.windowMinimize()}
            sx={{
              borderRadius: 0,
              width: 36,
              height: 28,
              color: 'text.secondary',
              '&:hover': { bgcolor: 'action.hover' },
            }}
          >
            <MinimizeIcon sx={{ fontSize: 16 }} />
          </IconButton>
          <IconButton
            size="small"
            onClick={() => window.electronAPI?.windowMaximize()}
            sx={{
              borderRadius: 0,
              width: 36,
              height: 28,
              color: 'text.secondary',
              '&:hover': { bgcolor: 'action.hover' },
            }}
          >
            <MaximizeIcon sx={{ fontSize: 14 }} />
          </IconButton>
          <IconButton
            size="small"
            onClick={() => window.electronAPI?.windowClose()}
            sx={{
              borderRadius: 0,
              width: 36,
              height: 28,
              color: 'text.secondary',
              '&:hover': { bgcolor: 'error.main', color: 'white' },
            }}
          >
            <CloseIcon sx={{ fontSize: 16 }} />
          </IconButton>
        </Box>
      </Box>

      {/* Tab Navigation */}
      <Box
        sx={{
          bgcolor: (theme) =>
            theme.palette.mode === 'dark' ? '#1a1a2e' : '#f5f5f5',
          borderBottom: 1,
          borderColor: 'divider',
          flexShrink: 0,
        }}
      >
        <Tabs
          value={activeTab}
          onChange={handleTabChange}
          variant="fullWidth"
          sx={{
            minHeight: 52,
            '& .MuiTab-root': {
              minHeight: 52,
              textTransform: 'none',
              fontWeight: 600,
              fontSize: '0.85rem',
              transition: 'all 0.2s',
            },
          }}
        >
          <Tab icon={<SpeedIcon sx={{ fontSize: 20 }} />} iconPosition="start" label="Очистка" />
          <Tab icon={<StorageIcon sx={{ fontSize: 20 }} />} iconPosition="start" label="Система" />
          <Tab icon={<FolderDeleteIcon sx={{ fontSize: 20 }} />} iconPosition="start" label="Остатки" />
          <Tab icon={<DupIcon sx={{ fontSize: 20 }} />} iconPosition="start" label="Дубликаты" />
          <Tab icon={<EmptyFolderIcon sx={{ fontSize: 20 }} />} iconPosition="start" label="Пустые" />
        </Tabs>
      </Box>

      {/* Main Content */}
      <Container
        maxWidth="lg"
        sx={{
          mt: 2,
          mb: 2,
          flex: 1,
          overflow: 'auto',
          '&::-webkit-scrollbar': { width: 6 },
          '&::-webkit-scrollbar-thumb': {
            bgcolor: 'action.disabled',
            borderRadius: 3,
          },
        }}
      >
        {error && (
          <Alert
            severity="error"
            sx={{ mb: 2, borderRadius: 2 }}
            onClose={() => setError(null)}
          >
            {error}
          </Alert>
        )}

        <TabPanel value={activeTab} index={0}>
          <CleanupPanel onError={setError} />
        </TabPanel>

        <TabPanel value={activeTab} index={1}>
          <SystemInfoPanel onError={setError} />
        </TabPanel>

        <TabPanel value={activeTab} index={2}>
          <LeftoversPanel onError={setError} />
        </TabPanel>

        <TabPanel value={activeTab} index={3}>
          <DuplicatesPanel onError={setError} />
        </TabPanel>

        <TabPanel value={activeTab} index={4}>
          <EmptyFoldersPanel onError={setError} />
        </TabPanel>
      </Container>
    </Box>
  );
}

export default App;
