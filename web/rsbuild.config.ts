import { defineConfig } from '@rsbuild/core';
import { pluginReact } from '@rsbuild/plugin-react';

export default defineConfig({
  server: {
    port: 8080,
    base: '/admin',
  },
  plugins: [pluginReact()],
  html: {
    title: 'Zhi-Server',
    favicon: './public/favicon.ico',
  },
  output: {
    assetPrefix: '/admin/',
  },
  dev: {
    assetPrefix: '/admin/',
  },
});
