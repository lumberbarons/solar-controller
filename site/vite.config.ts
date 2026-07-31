import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    // The Go binary embeds the frontend from site/build (see Makefile)
    outDir: 'build',
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/metrics': 'http://localhost:8080',
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/setupTests.ts'],
    // Components are asserted through the DOM they render, not their styles.
    css: false,
  },
})
