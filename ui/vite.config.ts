import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// https://vitejs.dev/config/
export default defineConfig({
  base: '/ui/',
  envPrefix: 'SMOOTHDB',
  plugins: [svelte()],
  build: {
    sourcemap: 'inline',
    minify: false
  }
})
