/**
 * Main Application Component
 * Clean Architecture presentation layer
 */
import React, { useState } from 'react';
import {
  Box,
  AppBar,
  Toolbar,
  Typography,
  Container,
  Paper,
  Tabs,
  Tab,
  Alert,
} from '@mui/material';
import {
  CleaningServices as CleaningServicesIcon,
  Speed as SpeedIcon,
  Storage as StorageIcon,
  FolderDelete as FolderDeleteIcon,
} from '@mui/icons-material';

// Feature Components
import { CleanupPanel } from './presentation/components/CleanupPanel';
import { SystemInfoPanel } from './presentation/components/SystemInfoPanel';
import { LeftoversPanel } from './presentation/components/LeftoversPanel';

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
      {/* Header */}
      <AppBar position="static" elevation={0}>
        <Toolbar>
          <CleaningServicesIcon sx={{ mr: 2 }} />
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            WinCleanerLamp GUI
          </Typography>
          <Typography variant="caption" sx={{ opacity: 0.7 }}>
            v2.0.0 TypeScript
          </Typography>
        </Toolbar>
        <Tabs
          value={activeTab}
          onChange={handleTabChange}
          variant="scrollable"
          scrollButtons="auto"
        >
          <Tab icon={<SpeedIcon />} label="Очистка" />
          <Tab icon={<StorageIcon />} label="Система" />
          <Tab icon={<FolderDeleteIcon />} label="Остатки программ" />
        </Tabs>
      </AppBar>

      {/* Main Content */}
      <Container maxWidth="lg" sx={{ mt: 3, mb: 3, flex: 1, overflow: 'auto' }}>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
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
      </Container>
    </Box>
  );
}

export default App;
