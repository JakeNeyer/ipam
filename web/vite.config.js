import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  // Relative asset paths let the built SPA work at "/" or under a subpath.
  base: './',
  plugins: [tailwindcss(), svelte()],
  esbuild: {
    target: 'es2022',
  },
  optimizeDeps: {
    esbuildOptions: {
      target: 'es2022',
    },
  },
  build: {
    target: 'es2022',
    chunkSizeWarningLimit: 600,
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8011',
        changeOrigin: true,
        xfwd: true,
      },
      '/docs': {
        target: 'http://127.0.0.1:8011',
        changeOrigin: true,
        xfwd: true,
      },
    },
  },
})
