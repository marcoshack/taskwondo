import { defineConfig, devices } from '@playwright/test';
import dotenv from 'dotenv';
import path from 'path';

// Load .env from repo root so tests can use ADMIN_EMAIL / ADMIN_PASSWORD
dotenv.config({ path: path.resolve(__dirname, '../.env'), quiet: true });

export default defineConfig({
  testDir: './tests',
  outputDir: './test-results',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['html', { outputFolder: './playwright-report' }],
    ['list'],
  ],
  use: {
    baseURL: 'http://localhost:5173',
    screenshot: 'on',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'setup',
      testMatch: /auth\.setup\.ts/,
    },
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
      dependencies: ['setup'],
    },
    {
      name: 'cleanup',
      testMatch: /cleanup\.teardown\.ts/,
      dependencies: ['chromium'],
    },
  ],
});
