import { defineConfig, devices } from '@playwright/test'

// Same as playwright.config.ts but WITHOUT globalSetup (which loads deploy/dev/seed.sql): the act-as spec
// creates and removes its own data, so it is safe on a dev DB holding real data.
// Run: npx playwright test -c e2e/act-as.config.ts
export default defineConfig({
  testDir: '.',
  testMatch: 'act-as.spec.ts',
  timeout: 90_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [['list']],
  use: { baseURL: process.env.BASE_URL ?? 'http://localhost:8000', trace: 'retain-on-failure' },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } } }],
})
