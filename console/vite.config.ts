import path from 'node:path';

import tailwindcss from '@tailwindcss/vite';
import { TanStackRouterVite } from '@tanstack/router-plugin/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';


export default defineConfig({
  plugins: [
    TanStackRouterVite({
      quoteStyle: 'single',
      routeFileIgnorePattern: '.*\\.test\\.(ts|tsx)$',
    }),
    react(),
    tailwindcss(),
  ],
  envDir: path.resolve(__dirname, '..'),
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
  },
});
