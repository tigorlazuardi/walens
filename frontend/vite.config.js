import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [svelte()],
  base: './',
  build: {
    manifest: true,
    outDir: 'dist',
    assetsDir: 'assets',
  },
  publicDir: false,
});
