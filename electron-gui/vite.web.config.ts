import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// Web-only config (no Electron plugin)
export default defineConfig({
  plugins: [
    react()
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src')
    }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:9000',
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: 'dist'
  }
})
