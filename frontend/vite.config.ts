import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig, loadEnv } from 'vite'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const target = env.VITE_DEV_API_URL || 'http://127.0.0.1:8080'

  return {
    plugins: [react(), tailwindcss()],
    build: {
      outDir: '../internal/httpapi/webui/dist',
      emptyOutDir: true,
    },
    server: {
      proxy: {
        '/v1': { target, changeOrigin: true, secure: false },
        '/repo': { target, changeOrigin: true, secure: false },
        '/healthz': { target, changeOrigin: true, secure: false },
        '/readyz': { target, changeOrigin: true, secure: false },
      },
    },
  }
})
