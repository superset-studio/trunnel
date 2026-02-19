import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [tailwindcss(), react()],
  server: {
    port: 3456,
    proxy: {
      '/api': 'http://localhost:9650',
      '/healthz': 'http://localhost:9650',
      '/readyz': 'http://localhost:9650',
    },
  },
})
