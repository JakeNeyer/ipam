import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte()],
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
