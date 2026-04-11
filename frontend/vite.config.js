import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [tailwindcss(), svelte()],
  base: './',
  build: {
    manifest: true,
    outDir: 'dist',
    assetsDir: 'assets',
  },
  publicDir: false,
});
