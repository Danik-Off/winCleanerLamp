/**
 * Проверяет наличие wincleanerlamp.exe в корне репозитория перед electron-builder.
 * Запуск из каталога gui: node verify-cli.cjs
 */
const fs = require('fs');
const path = require('path');

const repoRoot = path.join(__dirname, '..');
const exe = path.join(repoRoot, 'wincleanerlamp.exe');

if (!fs.existsSync(exe)) {
  console.error('[winCleanerLamp] Нет файла:', exe);
  console.error('Соберите CLI из корня репозитория: go build -ldflags "-s -w" -o wincleanerlamp.exe .');
  process.exit(1);
}
