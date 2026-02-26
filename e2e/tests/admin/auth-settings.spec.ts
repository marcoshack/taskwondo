import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';

// Unauthenticated context — login page is public
const test = base.extend({
  storageState: async ({}, use) => {
    await use({ cookies: [], origins: [] } as any);
  },
});

test.describe.configure({ mode: 'serial' });

test.describe('Authentication Settings', () => {
  test('login page hides "Create account" link when registration is disabled', async ({ page, request }) => {
    const adminToken = getAdminToken();

    // Ensure registration is disabled
    await api.disableEmailRegistration(request, adminToken);

    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    await expect(page.getByText('Create an account')).not.toBeVisible();
  });

  test('login page shows "Create account" link when registration is enabled', async ({ page, request }) => {
    const adminToken = getAdminToken();

    // Enable registration
    await api.enableEmailAuth(request, adminToken);

    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    await expect(page.getByText('Create an account')).toBeVisible();

    // Disable again for clean state
    await api.disableEmailRegistration(request, adminToken);
  });
});
