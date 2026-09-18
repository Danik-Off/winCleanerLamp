/**
 * Проверяет наличие ядра текущей ОС в корне репозитория перед electron-builder.
 * Запуск из каталога gui: node verify-cli.cjs
 */
const fs = require('fs');
const { exePath, buildCommand } = require('./cli-target.cjs');

if (!fs.existsSync(exePath)) {
  console.error('[winCleanerLamp] Нет файла:', exePath);
  console.error('Соберите ядро из корня репозитория:', buildCommand);
  process.exit(1);
}
