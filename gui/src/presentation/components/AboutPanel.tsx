/**
 * About Panel Component - Clean & Minimal
 */
import {
  Box,
  Typography,
  Card,
  CardContent,
  Grid,
  Paper,
  useTheme,
} from '@mui/material';
import {
  CleaningServices as Icon,
  Info as InfoIcon,
  Code as CodeIcon,
  Security as SecurityIcon,
} from '@mui/icons-material';

export function AboutPanel(): JSX.Element {
  const theme = useTheme();

  const appVersion = import.meta.env.VITE_APP_VERSION || '1.0.0';
  const buildHash = import.meta.env.VITE_BUILD_HASH || 'unknown';

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', p: 3, maxWidth: 700, mx: 'auto' }}>
      
      {/* Header */}
      <Box sx={{ textAlign: 'center', mb: 4 }}>
        <Box sx={{ 
          display: 'inline-flex', 
          alignItems: 'center', 
          justifyContent: 'center',
          width: 80, 
          height: 80, 
          borderRadius: 3,
          background: `linear-gradient(135deg, #6366f1, #8b5cf6)`,
          mb: 2,
        }}>
          <Icon sx={{ color: 'white', fontSize: 40 }} />
        </Box>
        <Typography variant="h4" sx={{ fontWeight: 800, mb: 0.5 }}>
          WinCleaner Pro
        </Typography>
        <Typography variant="body2" sx={{ opacity: 0.6 }}>
          Профессиональная очистка системы
        </Typography>
      </Box>

      {/* Info Cards */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6}>
          <Card sx={{ bgcolor: theme.palette.mode === 'dark' ? 'rgba(99, 102, 241, 0.08)' : 'rgba(99, 102, 241, 0.05)', border: 'none' }}>
            <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', width: 44, height: 44, borderRadius: 2, bgcolor: 'rgba(99, 102, 241, 0.2)' }}>
                <InfoIcon sx={{ fontSize: 22, color: '#6366f1' }} />
              </Box>
              <Box>
                <Typography variant="caption" sx={{ opacity: 0.6, display: 'block' }}>Версия</Typography>
                <Typography variant="h6" sx={{ fontWeight: 700 }}>{appVersion}</Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        
        <Grid item xs={12} sm={6}>
          <Card sx={{ bgcolor: theme.palette.mode === 'dark' ? 'rgba(139, 92, 246, 0.08)' : 'rgba(139, 92, 246, 0.05)', border: 'none' }}>
            <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', width: 44, height: 44, borderRadius: 2, bgcolor: 'rgba(139, 92, 246, 0.2)' }}>
                <CodeIcon sx={{ fontSize: 22, color: '#8b5cf6' }} />
              </Box>
              <Box>
                <Typography variant="caption" sx={{ opacity: 0.6, display: 'block' }}>Коммит</Typography>
                <Typography variant="h6" sx={{ fontWeight: 700, fontFamily: 'monospace' }}>{buildHash.slice(0, 8)}</Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Features */}
      <Paper sx={{ 
        p: 3,
        borderRadius: 3,
        bgcolor: theme.palette.mode === 'dark' ? 'rgba(16, 185, 129, 0.05)' : 'rgba(16, 185, 129, 0.03)',
        border: `1px solid ${theme.palette.mode === 'dark' ? 'rgba(16, 185, 129, 0.1)' : 'rgba(16, 185, 129, 0.05)'}`,
        mb: 3,
      }}>
        <Typography variant="subtitle1" sx={{ fontWeight: 700, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
          <SecurityIcon sx={{ fontSize: 18, color: '#10b981' }} />
          Безопасность
        </Typography>
        <Typography variant="body2" sx={{ opacity: 0.8, lineHeight: 1.7 }}>
          Все операции очистки выполняются безопасно. Временные файлы удаляются в Корзину, 
          а не навсегда. Критические системные файлы защищены от удаления.
        </Typography>
      </Paper>

      {/* Footer */}
      <Typography variant="caption" sx={{ opacity: 0.4, textAlign: 'center', display: 'block' }}>
        © 2024 WinCleaner Pro • Clean Architecture • Material Design
      </Typography>
    </Box>
  );
}
