import { test as base, expect } from '@playwright/test';
import { randomUUID } from 'crypto';
import * as api from './api';
import path from 'path';
import fs from 'fs';

const ADMIN_AUTH_FILE = path.join(__dirname, '../.auth/admin.json');
const TEST_PASSWORD = 'TestPass123!';

// Read admin token saved by auth.setup.ts
export function getAdminToken(): string {
  const data = JSON.parse(fs.readFileSync(ADMIN_AUTH_FILE, 'utf-8'));
  return data.token;
}

export interface TestUser {
  id: string;
  email: string;
  displayName: string;
  password: string;
  token: string;
}

export interface TestProject {
  id: string;
  key: string;
  name: string;
}

// Extend the base test with `testUser` and `testProject` fixtures
export const test = base.extend<{ testUser: TestUser; testProject: TestProject }>({
  testUser: async ({ request }, use) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const email = `e2e-${uniqueId}@test.local`;
    const displayName = `E2E User ${uniqueId}`;

    // 1. Admin creates user → temporary password
    const created = await api.createUser(request, adminToken, email, displayName);

    // 2. Login with temporary password
    const tempLogin = await api.login(request, email, created.temporary_password);

    // 3. Change password to a known value
    await api.changePassword(request, tempLogin.token, created.temporary_password, TEST_PASSWORD);

    // 4. Login again with the final password to get a clean token
    const finalLogin = await api.login(request, email, TEST_PASSWORD);

    // 5. Dismiss the Welcome modal so it doesn't interfere with tests
    await api.setPreference(request, finalLogin.token, 'welcome_dismissed', true);

    const testUser: TestUser = {
      id: finalLogin.user.id,
      email,
      displayName,
      password: TEST_PASSWORD,
      token: finalLogin.token,
    };

    await use(testUser);

    // Cleanup: deactivate the test user
    await api.deactivateUser(request, adminToken, testUser.id).catch(() => {});
  },

  // Creates a project owned by the test user
  testProject: async ({ request, testUser }, use) => {
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const key = `E${suffix}`;
    const name = `E2E Project ${suffix}`;

    const project = await api.createProject(request, testUser.token, key, name);

    await use({ id: project.id, key: project.key, name: project.name });
  },

  // Override storageState so each test gets a fresh browser context
  // with the test user's token injected via localStorage
  storageState: async ({ testUser }, use) => {
    const state = {
      cookies: [],
      origins: [
        {
          origin: process.env.BASE_URL || 'http://localhost:5173',
          localStorage: [
            { name: 'taskwondo_token', value: testUser.token },
          ],
        },
      ],
    };
    await use(state as any);
  },
});

export { expect };
