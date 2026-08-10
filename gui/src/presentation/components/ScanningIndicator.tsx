/**
 * ScanningIndicator — индикатор длительной операции с "живым" статусом.
 *
 * Раньше долгие сканирования (десятки тысяч файлов в %TEMP%, например) на
 * этом этапе показывали только голую индетерминированную LinearProgress —
 * визуально неотличимо от зависшего процесса, особенно на середине/ближе
 * к концу. Здесь под полосой крутится текст:
 * - если передан liveMessage (реальные данные от CLI через PROGRESS-события
 *   на stderr, см. gui/electron/main.ts) — показываем именно его;
 * - иначе — циклически меняющиеся, но честные (без выдуманных процентов)
 *   сообщения о том, что процесс идёт.
 */
import { useEffect, useState } from 'react';
import { Box, LinearProgress, Typography, Fade } from '@mui/material';

const GENERIC_MESSAGES = [
  'Сканирование...',
  'Проверяем файлы...',
  'Это может занять время на больших дисках или папках...',
  'Считаем размер...',
  'Ещё немного...',
];

interface ScanningIndicatorProps {
  active: boolean;
  liveMessage?: string | null;
  /** Заголовок над полосой (например, "Сканирование категорий"). */
  label?: string;
}

export function ScanningIndicator({ active, liveMessage, label }: ScanningIndicatorProps): JSX.Element | null {
  const [genericIdx, setGenericIdx] = useState(0);

  useEffect(() => {
    if (!active || liveMessage) return;
    const timer = setInterval(() => {
      setGenericIdx((i) => (i + 1) % GENERIC_MESSAGES.length);
    }, 1800);
    return () => clearInterval(timer);
  }, [active, liveMessage]);

  useEffect(() => {
    if (active) setGenericIdx(0);
  }, [active]);

  if (!active) return null;

  const text = liveMessage || GENERIC_MESSAGES[genericIdx];

  return (
    <Box sx={{ mb: 2 }}>
      {label && (
        <Typography variant="caption" sx={{ display: 'block', mb: 0.5, opacity: 0.6, fontWeight: 600 }}>
          {label}
        </Typography>
      )}
      <LinearProgress
        sx={{
          borderRadius: 2,
          height: 6,
          mb: 0.75,
          '& .MuiLinearProgress-bar': { borderRadius: 2 },
        }}
      />
      <Fade in key={text} timeout={400}>
        <Typography
          variant="caption"
          sx={{
            display: 'block',
            opacity: 0.65,
            fontFamily: liveMessage ? '"JetBrains Mono", "Fira Code", "Consolas", monospace' : undefined,
            fontSize: liveMessage ? '0.72rem' : undefined,
          }}
        >
          {text}
        </Typography>
      </Fade>
    </Box>
  );
}
