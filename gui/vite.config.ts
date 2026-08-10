/**
 * Vite Configuration
 * TypeScript config for Electron + React build
 */
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import { execSync } from 'child_process';
import { readFileSync } from 'fs';

// Получаем версию из package.json
const pkg = JSON.parse(readFileSync(path.resolve(__dirname, 'package.json'), 'utf8'));
const appVersion = pkg.version;

// Получаем хеш текущего коммита
let buildHash = 'unknown';
try {
  buildHash = execSync('git rev-parse --short HEAD', { 
    encoding: 'utf8', 
    stdio: 'pipe' 
  }).trim();
} catch {
  // Git может быть недоступен при сборке
}

export default defineConfig({
  plugins: [react()],
  base: './',
  define: {
    'import.meta.env.VITE_APP_VERSION': JSON.stringify(appVersion),
    'import.meta.env.VITE_BUILD_HASH': JSON.stringify(buildHash),
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      input: {
        main: path.resolve(__dirname, 'index.html'),
      },
    },
  },
  resolve: {
    alias: {
      '@domain': path.resolve(__dirname, './src/domain'),
      '@application': path.resolve(__dirname, './src/application'),
      '@infrastructure': path.resolve(__dirname, './src/infrastructure'),
      '@presentation': path.resolve(__dirname, './src/presentation'),
      '@container': path.resolve(__dirname, './src/container'),
      '@shared': path.resolve(__dirname, './src/shared'),
    },
  },
  server: {
    port: 3000,
  },
});
