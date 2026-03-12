import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

// Admin context — hide non-member projects is an admin feature
const test = base.extend({
  storageState: async ({}, use) => {
    const adminToken = getAdminToken();
    const state = {
      cookies: [],
      origins: [
        {
          origin: process.env.BASE_URL || 'http://localhost:5173',
          localStorage: [{ name: 'taskwondo_token', value: adminToken }],
        },
      ],
    };
    await use(state as any);
  },
});

test.describe.configure({ mode: 'serial' });

async function attach(page: any, testInfo: any, name: string) {
  const screenshot = await page.screenshot();
  await testInfo.attach(name, { body: screenshot, contentType: 'image/png' });
}

test.describe('General Preferences — Hide Non-Member Projects', () => {
  test.beforeEach(async ({ request }) => {
    // Reset preference to unchecked before each test
    const adminToken = getAdminToken();
    await api.setPreference(request, adminToken, 'hide_non_member_projects', false);
  });

  test('General page is accessible with toggle visible for admin', async ({ page }, testInfo) => {
    await page.goto('/preferences/general');
    await page.waitForLoadState('networkidle');

    // Page heading
    await expect(page.getByRole('heading', { name: 'General' })).toBeVisible();

    // Administrator section
    await expect(page.getByText('Administrator')).toBeVisible();

    // Toggle checkbox
    const checkbox = page.getByRole('checkbox', { name: /hide non-member/i });
    await expect(checkbox).toBeVisible();
    await expect(checkbox).not.toBeChecked();

    await attach(page, testInfo, '01-general-page');
  });

  test('toggle persists after page reload', async ({ page }, testInfo) => {
    await page.goto('/preferences/general');
    await page.waitForLoadState('networkidle');

    const checkbox = page.getByRole('checkbox', { name: /hide non-member/i });
    await expect(checkbox).not.toBeChecked();

    // Enable the toggle
    await checkbox.click();
    await expect(checkbox).toBeChecked();

    // Wait for the green check to appear (confirms save)
    await expect(page.locator('svg.text-green-500')).toBeVisible({ timeout: 3000 });
    await attach(page, testInfo, '02-toggle-enabled');

    // Reload and verify persistence
    await page.reload();
    await page.waitForLoadState('networkidle');
    const reloadedCheckbox = page.getByRole('checkbox', { name: /hide non-member/i });
    await expect(reloadedCheckbox).toBeChecked();

    await attach(page, testInfo, '03-toggle-persisted');
  });

  test('admin project list filters when hide preference is enabled', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const suffix = randomUUID().slice(0, 4).toUpperCase();

    // Create a project as admin (admin is owner/member)
    const adminProject = await api.createProject(request, adminToken, `A${suffix}`, `Admin Project ${suffix}`);

    // Create a separate user and project that admin is NOT a member of
    const userEmail = `e2e-hide-${Date.now()}@test.local`;
    const created = await api.createUser(request, adminToken, userEmail, 'Hide Test User');
    const tempLogin = await api.login(request, userEmail, created.temporary_password);
    await api.changePassword(request, tempLogin.token, created.temporary_password, 'HideTest123!');
    const userLogin = await api.login(request, userEmail, 'HideTest123!');

    const userSuffix = randomUUID().slice(0, 4).toUpperCase();
    const userProject = await api.createProject(request, userLogin.token, `U${userSuffix}`, `User Project ${userSuffix}`);

    const table = page.locator('table');

    // Without preference: admin sees both projects
    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');
    await expect(table.getByText(adminProject.name)).toBeVisible();
    await expect(table.getByText(userProject.name)).toBeVisible();
    await attach(page, testInfo, '04-all-projects-visible');

    // Enable hide preference
    await api.setPreference(request, adminToken, 'hide_non_member_projects', true);

    // Reload projects page
    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');

    // Admin should only see their own project
    await expect(table.getByText(adminProject.name)).toBeVisible();
    await expect(table.getByText(userProject.name)).not.toBeVisible();
    await attach(page, testInfo, '05-filtered-projects');

    // Disable preference again
    await api.setPreference(request, adminToken, 'hide_non_member_projects', false);

    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');
    await expect(table.getByText(userProject.name)).toBeVisible();
    await attach(page, testInfo, '06-all-projects-restored');

    // Cleanup
    await api.deactivateUser(request, adminToken, userLogin.user.id).catch(() => {});
  });

  test('non-admin user sees no-settings message on General page', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();

    // Create a regular user
    const email = `e2e-nonadmin-${Date.now()}@test.local`;
    const created = await api.createUser(request, adminToken, email, 'Non Admin');
    const tempLogin = await api.login(request, email, created.temporary_password);
    await api.changePassword(request, tempLogin.token, created.temporary_password, 'NonAdmin123!');
    const userLogin = await api.login(request, email, 'NonAdmin123!');
    await api.setPreference(request, userLogin.token, 'welcome_dismissed', true);

    // Create a new browser context with the regular user's token
    const context = await page.context().browser()!.newContext({
      storageState: {
        cookies: [],
        origins: [
          {
            origin: process.env.BASE_URL || 'http://localhost:5173',
            localStorage: [{ name: 'taskwondo_token', value: userLogin.token }],
          },
        ],
      },
    });
    const userPage = await context.newPage();

    await userPage.goto('/preferences/general');
    await userPage.waitForLoadState('networkidle');

    // Should see the page heading
    await expect(userPage.getByRole('heading', { name: 'General' })).toBeVisible();

    // Should NOT see the admin checkbox
    await expect(userPage.getByRole('checkbox', { name: /hide non-member/i })).not.toBeVisible();

    // Should see "no settings" message
    await expect(userPage.getByText(/no general settings/i)).toBeVisible();
    await attach(userPage, testInfo, '07-non-admin-general');

    await userPage.close();
    await context.close();

    // Cleanup
    await api.deactivateUser(request, adminToken, userLogin.user.id).catch(() => {});
  });
});
