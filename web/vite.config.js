import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  // Relative asset paths let the built SPA work at "/" or under a subpath.
  base: './',
  plugins: [tailwindcss(), svelte()],
  build: {
    chunkSizeWarningLimit: 600,
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8011',
        changeOrigin: true,
      },
      '/docs': {
        target: 'http://localhost:8011',
        changeOrigin: true,
      },
    },
  },
})
