import { defineConfig, devices } from '@playwright/test';
import dotenv from 'dotenv';
import path from 'path';

// Load .env from repo root so tests can use ADMIN_EMAIL / ADMIN_PASSWORD
dotenv.config({ path: path.resolve(__dirname, '../.env'), quiet: true });

const isDocker = !!process.env.BASE_URL && !process.env.BASE_URL.includes('localhost');

export default defineConfig({
  testDir: './tests',
  outputDir: './test-results',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : isDocker ? 2 : 0,
  workers: process.env.CI ? 1 : isDocker ? 4 : undefined,
  reporter: [
    ['html', { outputFolder: './playwright-report' }],
    ['list'],
  ],
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:5173',
    screenshot: 'on',
    trace: 'on-first-retry',
    actionTimeout: isDocker ? 15000 : 10000,
  },
  projects: [
    {
      name: 'setup',
      testMatch: /auth\.setup\.ts/,
    },
    {
      // Admin tests mutate global state (SMTP, auth settings) — run them
      // first so other tests that depend on that state don't race.
      name: 'admin',
      testMatch: /tests\/admin\//,
      use: { ...devices['Desktop Chrome'] },
      dependencies: ['setup'],
    },
    {
      // After admin tests: configure SMTP (Mailpit) and enable auth providers
      // so chromium tests can use email registration without touching settings.
      name: 'chromium-setup',
      testMatch: /chromium\.setup\.ts/,
      dependencies: ['admin'],
    },
    {
      name: 'chromium',
      testIgnore: [/tests\/admin\//, /\.setup\.ts$/],
      use: { ...devices['Desktop Chrome'] },
      dependencies: ['setup', 'chromium-setup'],
    },
    {
      name: 'cleanup',
      testMatch: /cleanup\.teardown\.ts/,
      dependencies: ['chromium'],
    },
  ],
});
