'use strict';

/**
 * Собирает ядро текущей ОС в корень репозитория (npm run build:cli).
 * Раньше скрипт был захардкожен на wincli — на Linux/macOS GUI было не собрать.
 */
const { execSync } = require('child_process');
const { repoRoot, buildCommand } = require('./cli-target.cjs');

console.log('[winCleanerLamp]', buildCommand);
execSync(buildCommand, { cwd: repoRoot, stdio: 'inherit' });
