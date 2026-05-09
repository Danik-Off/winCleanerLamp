/**
 * Main Application Component
 * WinCleaner GUI - Clean Architecture with Material Design
 */
import React, { useState } from 'react';
import {
  Box,
  Typography,
  Alert,
  IconButton,
  CssBaseline,
  ThemeProvider,
  createTheme,
  ButtonBase,
  Tooltip,
} from '@mui/material';
import {
  CleaningServices as CleaningServicesIcon,
  Speed as SpeedIcon,
  Storage as StorageIcon,
  FolderDelete as FolderDeleteIcon,
  ContentCopy as DupIcon,
  FolderOff as EmptyFolderIcon,
  Info as InfoIcon,
  Minimize as MinimizeIcon,
  CropSquare as MaximizeIcon,
  Close as CloseIcon,
  DarkMode as DarkModeIcon,
  LightMode as LightModeIcon,
  ChevronLeft as ChevronLeftIcon,
  ChevronRight as ChevronRightIcon,
  RocketLaunch as RocketIcon,
} from '@mui/icons-material';

// Feature Components
import { CleanupPanel } from './presentation/components/CleanupPanel';
import { SystemInfoPanel } from './presentation/components/SystemInfoPanel';
import { LeftoversPanel } from './presentation/components/LeftoversPanel';
import { DuplicatesPanel } from './presentation/components/DuplicatesPanel';
import { EmptyFoldersPanel } from './presentation/components/EmptyFoldersPanel';
import { HeroPanel } from './presentation/components/HeroPanel';
import { AboutPanel } from './presentation/components/AboutPanel';

const TITLEBAR_H = 36;
const SIDEBAR_W = 220;
const SIDEBAR_W_COLLAPSED = 56;

interface TabPanelProps {
  children: React.ReactNode;
  value: number;
  index: number;
}

function TabPanel({ children, value, index }: TabPanelProps): JSX.Element {
  return (
    <Box
      role="tabpanel"
      id={`tabpanel-${index}`}
      aria-labelledby={`tab-${index}`}
      hidden={value !== index}
      sx={{
        display: 'block',
        opacity: value === index ? 1 : 0,
        visibility: value === index ? 'visible' : 'hidden',
        position: value === index ? 'relative' : 'absolute',
        width: '100%',
        transition: 'opacity 0.15s ease',
      }}
    >
      {children}
    </Box>
  );
}

function App(): JSX.Element {
  const [activeTab, setActiveTab] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [darkMode, setDarkMode] = useState(true);
  const [sidebarOpen, setSidebarOpen] = useState(true);

  const currentSidebarW = sidebarOpen ? SIDEBAR_W : SIDEBAR_W_COLLAPSED;

  const theme = React.useMemo(
    () =>
      createTheme({
        shape: { borderRadius: 12 },
        palette: {
          mode: darkMode ? 'dark' : 'light',
          primary: { main: '#3b82f6' },
          background: {
            default: darkMode ? '#0b1120' : '#f1f5f9',
            paper: darkMode ? '#151d2e' : '#ffffff',
          },
        },
        typography: {
          fontFamily: '"Inter", "Segoe UI", "Roboto", "Helvetica", "Arial", sans-serif',
          h6: { fontWeight: 700 },
        },
        components: {
          MuiButton: {
            styleOverrides: {
              root: { textTransform: 'none', fontWeight: 600, borderRadius: 10 },
            },
          },
        },
      }),
    [darkMode]
  );

  const navItems = [
    { id: 0, label: 'Главная', icon: <RocketIcon sx={{ fontSize: 18 }} />, accent: '#8b5cf6' },
    { id: 1, label: 'Очистка', icon: <SpeedIcon sx={{ fontSize: 18 }} />, accent: '#3b82f6' },
    { id: 2, label: 'Система', icon: <StorageIcon sx={{ fontSize: 18 }} />, accent: '#06b6d4' },
    { id: 3, label: 'Остатки', icon: <FolderDeleteIcon sx={{ fontSize: 18 }} />, accent: '#8b5cf6' },
    { id: 4, label: 'Дубликаты', icon: <DupIcon sx={{ fontSize: 18 }} />, accent: '#f59e0b' },
    { id: 5, label: 'Пустые папки', icon: <EmptyFolderIcon sx={{ fontSize: 18 }} />, accent: '#ef4444' },
    { id: 6, label: 'О приложении', icon: <InfoIcon sx={{ fontSize: 18 }} />, accent: '#10b981' },
  ];

  const activeAccent = navItems[activeTab]?.accent || '#3b82f6';

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ height: '100vh', display: 'flex', flexDirection: 'column', bgcolor: 'background.default', overflow: 'hidden' }}>

        {/* ═══ Title Bar ═══ */}
        <Box
          sx={{
            height: TITLEBAR_H,
            minHeight: TITLEBAR_H,
            bgcolor: darkMode ? '#0b1120' : '#ffffff',
            borderBottom: `1px solid ${darkMode ? '#1f2937' : '#e2e8f0'}`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            px: 1.5,
            '-webkit-app-region': 'drag' as any,
            zIndex: 1300,
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, '-webkit-app-region': 'no-drag' as any }}>
            <Box
              sx={{
                width: 22,
                height: 22,
                borderRadius: 1,
                background: `linear-gradient(135deg, ${activeAccent}, ${activeAccent}88)`,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                transition: 'background 0.3s',
              }}
            >
              <CleaningServicesIcon sx={{ color: 'white', fontSize: 13 }} />
            </Box>
            <Typography
              variant="caption"
              sx={{
                fontWeight: 700,
                color: darkMode ? '#94a3b8' : '#64748b',
                userSelect: 'none',
                fontSize: '0.72rem',
              }}
            >
              WinCleaner Pro
            </Typography>
          </Box>

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.25, '-webkit-app-region': 'no-drag' as any }}>
            <IconButton
              size="small"
              onClick={() => window.electronAPI?.windowMinimize()}
              sx={{ width: 28, height: 28, borderRadius: 1, color: darkMode ? '#475569' : '#94a3b8', '&:hover': { bgcolor: darkMode ? '#1e293b' : '#f1f5f9' } }}
            >
              <MinimizeIcon sx={{ fontSize: 14 }} />
            </IconButton>
            <IconButton
              size="small"
              onClick={() => window.electronAPI?.windowMaximize()}
              sx={{ width: 28, height: 28, borderRadius: 1, color: darkMode ? '#475569' : '#94a3b8', '&:hover': { bgcolor: darkMode ? '#1e293b' : '#f1f5f9' } }}
            >
              <MaximizeIcon sx={{ fontSize: 14 }} />
            </IconButton>
            <IconButton
              size="small"
              onClick={() => window.electronAPI?.windowClose()}
              sx={{ width: 28, height: 28, borderRadius: 1, color: darkMode ? '#475569' : '#94a3b8', '&:hover': { bgcolor: '#ef4444', color: 'white' } }}
            >
              <CloseIcon sx={{ fontSize: 14 }} />
            </IconButton>
          </Box>
        </Box>

        {/* ═══ Body: Sidebar + Content ═══ */}
        <Box sx={{ display: 'flex', flex: 1, overflow: 'hidden' }}>

          {/* ─── Left Sidebar ─── */}
          <Box
            sx={{
              width: currentSidebarW,
              minWidth: currentSidebarW,
              bgcolor: darkMode ? '#0f1629' : '#ffffff',
              borderRight: `1px solid ${darkMode ? '#1f2937' : '#e2e8f0'}`,
              display: 'flex',
              flexDirection: 'column',
              overflow: 'hidden',
              transition: 'width 0.2s ease, min-width 0.2s ease',
            }}
          >
            {/* Nav items */}
            <Box sx={{ flex: 1, py: 1.5, px: sidebarOpen ? 1 : 0.5, display: 'flex', flexDirection: 'column', gap: 0.25 }}>
              {navItems.map((item) => {
                const isActive = activeTab === item.id;
                return (
                  <Tooltip key={item.id} title={sidebarOpen ? '' : item.label} placement="right" arrow>
                    <ButtonBase
                      onClick={() => setActiveTab(item.id)}
                      sx={{
                        width: '100%',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: sidebarOpen ? 'flex-start' : 'center',
                        gap: sidebarOpen ? 1.25 : 0,
                        py: 1,
                        px: sidebarOpen ? 1.5 : 1,
                        borderRadius: 2,
                        transition: 'all 0.15s ease',
                        position: 'relative',
                        color: isActive
                          ? item.accent
                          : (darkMode ? '#64748b' : '#94a3b8'),
                        bgcolor: isActive
                          ? (darkMode ? `${item.accent}15` : `${item.accent}0d`)
                          : 'transparent',
                        '&:hover': {
                          bgcolor: isActive
                            ? (darkMode ? `${item.accent}20` : `${item.accent}12`)
                            : (darkMode ? '#1e293b' : '#f8fafc'),
                          color: isActive ? item.accent : (darkMode ? '#e2e8f0' : '#334155'),
                        },
                        '&::before': isActive ? {
                          content: '""',
                          position: 'absolute',
                          left: 0,
                          top: '25%',
                          bottom: '25%',
                          width: 3,
                          borderRadius: 4,
                          bgcolor: item.accent,
                        } : {},
                      }}
                    >
                      {item.icon}
                      {sidebarOpen && (
                        <Typography
                          variant="body2"
                          sx={{
                            fontWeight: isActive ? 700 : 500,
                            fontSize: '0.82rem',
                            lineHeight: 1.2,
                            whiteSpace: 'nowrap',
                            overflow: 'hidden',
                          }}
                        >
                          {item.label}
                        </Typography>
                      )}
                    </ButtonBase>
                  </Tooltip>
                );
              })}
            </Box>

            {/* Bottom: collapse + theme */}
            <Box
              sx={{
                px: sidebarOpen ? 1.5 : 0.5,
                py: 1,
                borderTop: `1px solid ${darkMode ? '#1f2937' : '#e2e8f0'}`,
                display: 'flex',
                alignItems: 'center',
                justifyContent: sidebarOpen ? 'space-between' : 'center',
                gap: 0.5,
              }}
            >
              {sidebarOpen && (
                <Tooltip title={darkMode ? 'Светлая тема' : 'Тёмная тема'}>
                  <IconButton
                    size="small"
                    onClick={() => setDarkMode(!darkMode)}
                    sx={{
                      width: 28,
                      height: 28,
                      color: darkMode ? '#64748b' : '#94a3b8',
                      '&:hover': { bgcolor: darkMode ? '#1e293b' : '#f1f5f9' },
                    }}
                  >
                    {darkMode ? <LightModeIcon sx={{ fontSize: 15 }} /> : <DarkModeIcon sx={{ fontSize: 15 }} />}
                  </IconButton>
                </Tooltip>
              )}
              <Tooltip title={sidebarOpen ? 'Свернуть' : 'Развернуть'} placement="right">
                <IconButton
                  size="small"
                  onClick={() => setSidebarOpen(!sidebarOpen)}
                  sx={{
                    width: 28,
                    height: 28,
                    color: darkMode ? '#475569' : '#94a3b8',
                    '&:hover': { bgcolor: darkMode ? '#1e293b' : '#f1f5f9' },
                  }}
                >
                  {sidebarOpen ? <ChevronLeftIcon sx={{ fontSize: 16 }} /> : <ChevronRightIcon sx={{ fontSize: 16 }} />}
                </IconButton>
              </Tooltip>
            </Box>
          </Box>

          {/* ─── Main Content ─── */}
          <Box
            sx={{
              flex: 1,
              overflowY: 'auto',
              overflowX: 'hidden',
              p: 2.5,
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
              <HeroPanel onError={setError} />
            </TabPanel>
            <TabPanel value={activeTab} index={1}>
              <CleanupPanel onError={setError} />
            </TabPanel>
            <TabPanel value={activeTab} index={2}>
              <SystemInfoPanel onError={setError} />
            </TabPanel>
            <TabPanel value={activeTab} index={3}>
              <LeftoversPanel onError={setError} />
            </TabPanel>
            <TabPanel value={activeTab} index={4}>
              <DuplicatesPanel onError={setError} />
            </TabPanel>
            <TabPanel value={activeTab} index={5}>
              <EmptyFoldersPanel onError={setError} />
            </TabPanel>
            <TabPanel value={activeTab} index={6}>
              <AboutPanel />
            </TabPanel>
          </Box>
        </Box>
      </Box>
    </ThemeProvider>
  );
}

export default App;
