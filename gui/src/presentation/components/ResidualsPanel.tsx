/**
 * ResidualsPanel — единая секция "Остатки и программы".
 *
 * Раньше это были два отдельных пункта навигации ("Остатки" и "Программы")
 * с почти одинаковым смыслом (поиск мусора от удалённых программ), но разным
 * визуальным оформлением — путало пользователя. Теперь один пункт навигации
 * с общим заголовком и вкладками для трёх разных способов поиска:
 *   - Эвристика   — широкий автоматический скан AppData/ProgramData/реестра
 *   - База программ — точная проверка по orphaned_apps.json + поиск новых кандидатов
 *   - Ярлыки      — битые .lnk на рабочем столе и в меню Пуск
 */
import { useState } from 'react';
import { Box, Paper, Typography, Tabs, Tab, Button, Alert } from '@mui/material';
import {
  FolderDelete as FolderDeleteIcon,
  Search as SearchIcon,
  FolderShared as OrphanIcon,
  LinkOff as ShortcutsIcon,
  FileDownload as ExportIcon,
} from '@mui/icons-material';
import { LeftoversPanel } from './LeftoversPanel';
import { OrphanPanel } from './OrphanPanel';
import { ShortcutsPanel } from './ShortcutsPanel';

interface ResidualsPanelProps {
  onError: (error: string) => void;
}

export function ResidualsPanel({ onError }: ResidualsPanelProps): JSX.Element {
  const [tab, setTab] = useState(0);
  const [exporting, setExporting] = useState(false);
  const [exportInfo, setExportInfo] = useState<string | null>(null);

  const handleExportAudit = async () => {
    setExporting(true);
    setExportInfo(null);
    try {
      const res = await window.electronAPI.exportAudit();
      if (res.canceled) return;
      if (!res.success) {
        onError(res.error || 'Не удалось сохранить снимок');
        return;
      }
      setExportInfo(
        `Сохранено: ${res.path} (неизвестных папок: ${res.unknownCount ?? 0}, установленных программ: ${res.installedCount ?? 0})`
      );
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Ошибка экспорта');
    } finally {
      setExporting(false);
    }
  };

  return (
    <Box>
      {/* Единый заголовок секции — общий для всех трёх способов поиска ниже */}
      <Paper
        elevation={2}
        sx={{
          p: 3,
          mb: 2,
          background: (theme) =>
            theme.palette.mode === 'dark'
              ? 'linear-gradient(135deg, #1a237e 0%, #311b92 100%)'
              : 'linear-gradient(135deg, #e3f2fd 0%, #ede7f6 100%)',
          borderRadius: 3,
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 2, flexWrap: 'wrap' }}>
          <Box>
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
              <FolderDeleteIcon sx={{ mr: 1.5, fontSize: 28 }} />
              <Typography variant="h6" sx={{ fontWeight: 700, color: 'text.primary' }}>
                Остатки и программы
              </Typography>
            </Box>
            <Typography variant="body2" sx={{ opacity: 0.75, maxWidth: 560 }}>
              Файлы, оставшиеся от удалённых программ, неизвестные папки и битые ярлыки —
              эвристический поиск, точная проверка по базе известных программ и ярлыки в одном месте.
            </Typography>
          </Box>
          <Button
            variant="outlined"
            size="small"
            onClick={handleExportAudit}
            disabled={exporting}
            startIcon={<ExportIcon sx={{ fontSize: 16 }} />}
            sx={{ borderRadius: 2, fontWeight: 600, whiteSpace: 'nowrap' }}
          >
            {exporting ? 'Экспорт...' : 'Экспорт для анализа'}
          </Button>
        </Box>
        {exportInfo && (
          <Alert severity="success" sx={{ mt: 2, borderRadius: 1.5 }} onClose={() => setExportInfo(null)}>
            {exportInfo}
          </Alert>
        )}
      </Paper>

      <Tabs
        value={tab}
        onChange={(_, v) => setTab(v)}
        variant="scrollable"
        scrollButtons="auto"
        sx={{
          mb: 2,
          bgcolor: (t) => (t.palette.mode === 'dark' ? '#1e293b' : '#ffffff'),
          border: '1px solid',
          borderColor: 'divider',
          borderRadius: 3,
          p: 0.5,
          '& .MuiTab-root': { textTransform: 'none', fontWeight: 600, minHeight: 40, borderRadius: 2, fontSize: '0.82rem' },
        }}
      >
        <Tab label="Эвристический поиск" icon={<SearchIcon sx={{ fontSize: 17 }} />} iconPosition="start" />
        <Tab label="База программ" icon={<OrphanIcon sx={{ fontSize: 17 }} />} iconPosition="start" />
        <Tab label="Ярлыки" icon={<ShortcutsIcon sx={{ fontSize: 17 }} />} iconPosition="start" />
      </Tabs>

      <Box sx={{ display: tab === 0 ? 'block' : 'none' }}>
        <LeftoversPanel onError={onError} />
      </Box>
      <Box sx={{ display: tab === 1 ? 'block' : 'none' }}>
        <OrphanPanel onError={onError} />
      </Box>
      <Box sx={{ display: tab === 2 ? 'block' : 'none' }}>
        <ShortcutsPanel onError={onError} />
      </Box>
    </Box>
  );
}
