/// <reference types="vitest/config" />
import adapter from '@sveltejs/adapter-static'
import { sveltekit } from '@sveltejs/kit/vite'
import { svelteTesting } from '@testing-library/svelte/vite'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    sveltekit({
      // The build lands in dist/, which the Go binary embeds. No page is
      // prerendered: the Go server answers every path that names no file
      // with 200.html, and the client router picks the page from there.
      adapter: adapter({ pages: 'dist', fallback: '200.html' }),
    }),
    svelteTesting(),
  ],
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
