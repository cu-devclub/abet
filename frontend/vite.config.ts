import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

const gateway = process.env.VITE_DEV_GATEWAY ?? 'http://localhost:8000'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    allowedHosts: true,
    // bind mounts from Windows/macOS hosts do not deliver file events into the container
    watch: {
      ignored: ['**/e2e/**', '**/test-results/**', '**/playwright-report/**'], // test output must not trigger HMR reloads
      ...(process.env.VITE_USE_POLLING ? { usePolling: true, interval: 300 } : {}),
    },
    hmr: {
      clientPort: Number(process.env.VITE_HMR_CLIENT_PORT ?? 8000),
    },
    // Used when vite runs outside docker; inside compose the gateway routes these paths itself.
    proxy: {
      '/public': { target: gateway, changeOrigin: true },
      '/auth': { target: gateway, changeOrigin: true },
    },
  },
  preview: {
    host: '0.0.0.0',
    allowedHosts: true,
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    css: false,
    testTimeout: 60_000, // jsdom UI tests are slow on busy CI/dev machines (default 5 s flaked under load)
    exclude: ['node_modules/**', 'e2e/**'],
  },
})
