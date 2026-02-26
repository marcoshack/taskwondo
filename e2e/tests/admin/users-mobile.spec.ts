import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';

// Override storageState to inject admin token instead of the regular test user
const test = base.extend({
  storageState: async ({}, use) => {
    const adminToken = getAdminToken();
    const state = {
      cookies: [],
      origins: [
        {
          origin: process.env.BASE_URL || 'http://localhost:5173',
          localStorage: [
            { name: 'taskwondo_token', value: adminToken },
          ],
        },
      ],
    };
    await use(state as any);
  },
});

test.describe('Admin users page mobile', () => {
  test('users page header and controls are accessible on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/admin/users');

    // Title should be visible
    await expect(page.getByRole('heading', { name: 'Users' })).toBeVisible({ timeout: 10000 });

    // Description text should be visible and not cut off
    await expect(page.getByText('Manage user accounts')).toBeVisible();

    // The "New User" button should be visible and clickable
    const newUserBtn = page.getByRole('button', { name: /New User/i });
    await expect(newUserBtn).toBeVisible();

    // The status filter should be visible
    await expect(page.getByRole('button', { name: /Users/i }).first()).toBeVisible();
  });

  test('user row controls are visible on mobile without overlapping name', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/admin/users');

    // Wait for the user list to load by checking for a role combobox (non-admin users)
    const firstCombobox = page.getByRole('combobox').first();
    await expect(firstCombobox).toBeVisible({ timeout: 10000 });

    // The role combobox should be interactive
    const options = await firstCombobox.locator('option').allTextContents();
    expect(options).toContain('User');

    // The "Active" button should be visible alongside the role control
    await expect(page.getByRole('button', { name: 'Active' }).first()).toBeVisible();
  });

  test('expanding a user row shows project limit on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/admin/users');

    // Wait for a combobox to appear (non-admin user loaded)
    const firstCombobox = page.getByRole('combobox').first();
    await expect(firstCombobox).toBeVisible({ timeout: 10000 });

    // Click the chevron button of a non-admin user row to expand it
    // The row structure has a chevron > avatar > name on line 1, controls on line 2
    // Find the parent row and click it
    const userRow = firstCombobox.locator('xpath=ancestor::*[@class and contains(@class,"cursor")]').first();
    await userRow.click();

    // The expanded section should show a project limit input on mobile
    // (hidden on desktop, shown in expanded panel on mobile)
    const spinbuttons = page.getByRole('spinbutton');
    // There should be at least 2: the global "Max projects per user" + per-user limit
    await expect(spinbuttons.nth(1)).toBeVisible({ timeout: 5000 });
  });
});
