import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

const apiTarget = 'http://localhost:9000';

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
