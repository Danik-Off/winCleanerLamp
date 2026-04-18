/**
 * React Application Entry Point
 */
import React from 'react';
import ReactDOM from 'react-dom/client';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import App from './App';

// Determine system theme preference
const prefersDarkMode = window.matchMedia?.('(prefers-color-scheme: dark)').matches ?? true;

// Create theme based on system preference
const theme = createTheme({
  palette: {
    mode: prefersDarkMode ? 'dark' : 'light',
    primary: {
      main: prefersDarkMode ? '#90caf9' : '#1976d2',
    },
    secondary: {
      main: prefersDarkMode ? '#f48fb1' : '#dc004e',
    },
    background: {
      default: prefersDarkMode ? '#121212' : '#fafafa',
      paper: prefersDarkMode ? '#1e1e1e' : '#ffffff',
    },
  },
  typography: {
    fontFamily: 'Roboto, Arial, sans-serif',
  },
});

// Render the application
const container = document.getElementById('root');

if (!container) {
  throw new Error('Root container not found');
}

ReactDOM.createRoot(container).render(
  <React.StrictMode>
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <App />
    </ThemeProvider>
  </React.StrictMode>
);
