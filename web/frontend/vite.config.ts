import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    // `npm run dev` 時、rss-feeder-web（デフォルト :8080）にAPIをプロキシする
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  build: {
    outDir: '../static',
    emptyOutDir: true,
  },
})
