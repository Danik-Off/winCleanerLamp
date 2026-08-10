/**
 * Проверяет наличие win-cleaner-lamp.exe в корне репозитория перед electron-builder.
 * Запуск из каталога gui: node verify-cli.cjs
 */
const fs = require('fs');
const path = require('path');

const repoRoot = path.join(__dirname, '..');
const exe = path.join(repoRoot, 'win-cleaner-lamp.exe');

if (!fs.existsSync(exe)) {
  console.error('[winCleanerLamp] Нет файла:', exe);
  console.error('Соберите CLI из корня репозитория: go build -ldflags "-s -w" -o win-cleaner-lamp.exe .');
  process.exit(1);
}
