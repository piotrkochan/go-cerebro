import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

import packageJson from './package.json';

const apiTarget = process.env.VITE_API_TARGET ?? 'http://localhost:9000';
const appVersion = process.env.VITE_APP_VERSION ?? packageJson.version;
const apiProxy = {
  changeOrigin: true,
  target: apiTarget,
  xfwd: true,
};
const apiProxyPaths = [
  '/clusters',
  '/aliases',
  '/analysis',
  '/auth',
  '/cat',
  '/cluster_changes',
  '/cluster_settings',
  '/commons',
  '/connect',
  '/create_index',
  '/data_explorer',
  '/data_streams',
  '/docs',
  '/ilm',
  '/index_settings',
  '/navbar',
  '/login',
  '/nodes',
  '/openapi.json',
  '/openapi.yaml',
  '/overview',
  '/repositories',
  '/rest',
  '/schemas',
  '/snapshots',
  '/templates',
];

export default defineConfig({
  build: {
    emptyOutDir: false,
    outDir: 'internal/server/static',
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
    proxy: Object.fromEntries(apiProxyPaths.map((path) => [path, apiProxy])),
  },
});
