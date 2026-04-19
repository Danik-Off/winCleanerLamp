'use strict';

/**
 * Вызывается из npm lifecycle `version` после того, как npm уже поднял
 * версию в корневом package.json. Копирует её в gui/package.json и
 * добавляет файл в индекс, чтобы он попал в тот же коммит, что и релиз.
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const root = path.join(__dirname, '..');
const pkgPath = path.join(root, 'package.json');
const guiPath = path.join(root, 'gui', 'package.json');

const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
const gui = JSON.parse(fs.readFileSync(guiPath, 'utf8'));

if (gui.version !== pkg.version) {
  gui.version = pkg.version;
  fs.writeFileSync(guiPath, JSON.stringify(gui, null, 2) + '\n');
}

try {
  execSync('git add gui/package.json', { cwd: root, stdio: 'inherit' });
} catch (err) {
  console.warn('[sync-gui-version] git add:', err.message);
}
