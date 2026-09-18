'use strict';

/**
 * Какое ядро нужно GUI на текущей ОС: имя бинарника в корне репозитория
 * и Go-пакет, из которого оно собирается. Единственное место, где живёт
 * эта карта для сборочных скриптов (build-cli.cjs, verify-cli.cjs);
 * в electron/main.ts та же логика продублирована для рантайма.
 */
const path = require('path');

const TARGETS = {
  win32: { name: 'win-cleaner-lamp.exe', pkg: './wincli' },
  linux: { name: 'lin-cleaner-lamp', pkg: './lincli' },
  darwin: { name: 'mac-cleaner-lamp', pkg: './maccli' },
};

const target = TARGETS[process.platform];
if (!target) {
  console.error('[winCleanerLamp] Неподдерживаемая ОС для сборки GUI:', process.platform);
  process.exit(1);
}

const repoRoot = path.join(__dirname, '..');

module.exports = {
  repoRoot,
  name: target.name,
  pkg: target.pkg,
  exePath: path.join(repoRoot, target.name),
  buildCommand: `go build -ldflags "-s -w" -o ${target.name} ${target.pkg}`,
};
