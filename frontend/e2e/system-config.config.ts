import { defineConfig, devices } from '@playwright/test'

// Like act-as.config.ts: no globalSetup (no seed.sql), safe on a dev DB holding real data.
// Run: npx playwright test -c e2e/system-config.config.ts   (SYSCFG_SHOTS=<dir> saves screenshots)
export default defineConfig({
  testDir: '.',
  testMatch: 'system-config.spec.ts',
  timeout: 120_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [['list']],
  use: { baseURL: process.env.BASE_URL ?? 'http://localhost:8000', trace: 'retain-on-failure' },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } } }],
})
