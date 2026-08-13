import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig({
  base: './',
  plugins: [react()],
  build: { outDir: 'dist', sourcemap: true },
  test: { environment: 'jsdom', environmentOptions: { jsdom: { url: 'https://misaeldasilva123ms96-commits.github.io/BAIXAR-MP3/' } }, globals: true, setupFiles: './src/test/setup.ts', css: false }
});
