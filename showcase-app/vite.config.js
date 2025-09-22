import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    // Enable CORS for development
    cors: true,
    // Optionally add proxy for local FaultLine API
    proxy: {
      '/faultline-api': {
        target: 'http://localhost:8081',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/faultline-api/, '')
      }
    }
  }
})
