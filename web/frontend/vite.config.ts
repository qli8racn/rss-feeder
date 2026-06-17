import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    // `npm run dev` 時、bin/web（デフォルト :8080）にAPIをプロキシする
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  build: {
    outDir: '../static',
    emptyOutDir: true,
  },
  test: {
    environment: 'jsdom',
  },
})
