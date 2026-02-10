// @ts-check
// Playwright config for IPAM e2e tests.
// Run against full app (API + static): build web, then from repo root:
//   STATIC_DIR=web/dist go run . &
//   cd web && npm run e2e
// Or set BASE_URL (default http://localhost:8011).

import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: process.env.CI ? 'github' : 'list',
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:8011',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  timeout: 15000,
  expect: { timeout: 5000 },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
})
