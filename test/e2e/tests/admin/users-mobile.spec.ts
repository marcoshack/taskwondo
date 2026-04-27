import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

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

test.describe('Admin directory — Users tab on mobile', () => {
  // Ensure a non-admin user exists for tests that expect a role combobox
  test.beforeAll(async ({ request }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    await api.createUser(request, adminToken, `mobile-${uniqueId}@e2e.local`, `Mobile User ${uniqueId}`);
  });

  test('directory page header and Users tab controls are accessible on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/admin/directory');

    // Title should be visible (Directory page heading)
    await expect(page.getByRole('heading', { name: 'Directory' })).toBeVisible({ timeout: 10000 });

    // The "New User" button on the Users tab should be visible and clickable
    const newUserBtn = page.getByRole('button', { name: /New User/i });
    await expect(newUserBtn).toBeVisible();

    // The search bar specific to the Users tab should be visible
    await expect(page.getByPlaceholder(/Search users/i)).toBeVisible();
  });

  test('user row controls are visible on mobile without overlapping name', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/admin/directory');

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
    await page.goto('/admin/directory');

    // Wait for a combobox to appear (non-admin user loaded)
    const firstCombobox = page.getByRole('combobox').first();
    await expect(firstCombobox).toBeVisible({ timeout: 10000 });

    // Click the chevron button of a non-admin user row to expand it
    // The row structure has a chevron > avatar > name on line 1, controls on line 2
    // Find the parent row and click it
    const userRow = firstCombobox.locator('xpath=ancestor::*[@class and contains(@class,"cursor")]').first();
    await userRow.click();

    // The expanded section should show a project limit input on mobile
    // (hidden on desktop, shown in expanded panel on mobile). The Users tab
    // no longer shows the global default panels — those moved to the Settings
    // tab — so the per-user limit is the first spinbutton on the page.
    const spinbuttons = page.getByRole('spinbutton');
    await expect(spinbuttons.first()).toBeVisible({ timeout: 5000 });
  });

  test('Users tab search filters the user list', async ({ page, request }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    // Create two users with very distinct display names so we can test the filter
    const adminToken = getAdminToken();
    const tag = randomUUID().slice(0, 6);
    const matchEmail = `match-${tag}@e2e.local`;
    const otherEmail = `other-${tag}@e2e.local`;
    await api.createUser(request, adminToken, matchEmail, `Findable Person ${tag}`);
    await api.createUser(request, adminToken, otherEmail, `Hidden Person ${tag}`);

    await page.goto('/admin/directory');
    await expect(page.getByRole('heading', { name: 'Directory' })).toBeVisible({ timeout: 10000 });

    const searchInput = page.getByPlaceholder(/Search users/i);
    await searchInput.fill(`Findable Person ${tag}`);

    // Wait past the debounce window. The user list renders both the desktop
    // and mobile variants in the DOM (one hidden via CSS), so `.locator('visible=true').first()`
    // narrows to whichever is shown for the current viewport.
    await expect(
      page.getByText(`Findable Person ${tag}`).locator('visible=true').first(),
    ).toBeVisible({ timeout: 5000 });
    await expect(page.getByText(`Hidden Person ${tag}`)).toHaveCount(0);
  });
});
