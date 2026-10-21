/// <reference types="vitest/config" />
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { svelteTesting } from '@testing-library/svelte/vite'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [svelte(), svelteTesting()],
  server: {
    // During development Vite serves the frontend and forwards API calls
    // to the Go backend, so fetch('/api/...') works on both setups.
    proxy: {
      '/api': 'http://localhost:8686',
    },
  },
  test: {
    environment: 'jsdom',
  },
})
