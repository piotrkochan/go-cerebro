import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

import packageJson from './package.json';

const apiTarget = 'http://localhost:9000';
const appVersion = process.env.VITE_APP_VERSION ?? packageJson.version;

export default defineConfig({
  build: {
    emptyOutDir: false,
    outDir: 'public',
    rollupOptions: {
      output: {
        assetFileNames: 'assets/[name][extname]',
        chunkFileNames: 'assets/[name].js',
        entryFileNames: 'assets/[name].js',
      },
    },
  },
  define: {
    __APP_VERSION__: JSON.stringify(appVersion),
  },
  plugins: [react(), tailwindcss()],
  publicDir: false,
  server: {
    proxy: {
      '/aliases': apiTarget,
      '/analysis': apiTarget,
      '/auth': apiTarget,
      '/cat': apiTarget,
      '/cluster_changes': apiTarget,
      '/cluster_settings': apiTarget,
      '/commons': apiTarget,
      '/connect': apiTarget,
      '/create_index': apiTarget,
      '/docs': apiTarget,
      '/index_settings': apiTarget,
      '/navbar': apiTarget,
      '/nodes': apiTarget,
      '/openapi.json': apiTarget,
      '/openapi.yaml': apiTarget,
      '/overview': apiTarget,
      '/repositories': apiTarget,
      '/rest': apiTarget,
      '/snapshots': apiTarget,
      '/templates': apiTarget,
    },
  },
});
