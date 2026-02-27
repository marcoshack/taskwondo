import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';

// Admin context — feature toggles require admin role
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

test.describe('Feature Toggle — Activity Graph', () => {
  test.beforeEach(async ({ request }) => {
    // Reset to default (enabled) by deleting the setting
    const adminToken = getAdminToken();
    await api.setSystemSetting(request, adminToken, 'feature_stats_timeline', true);
  });

  test('features page displays activity graph toggle', async ({ page }, testInfo) => {
    await page.goto('/admin/features');
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('heading', { name: 'Features' })).toBeVisible();
    await expect(page.getByText('Activity Graph')).toBeVisible();

    await attach(page, testInfo, '01-features-page');
  });

  test('activity graph is visible on project overview when enabled', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();

    // Ensure feature is enabled
    await api.setSystemSetting(request, adminToken, 'feature_stats_timeline', true);

    // Create a project to visit its overview
    const project = await api.createProject(request, adminToken, 'FT' + Date.now().toString(36).slice(-3).toUpperCase(), 'Feature Toggle Test');

    await page.goto(`/projects/${project.key}`);
    await page.waitForLoadState('networkidle');

    // The activity section heading should be visible
    await expect(page.getByText('Activity', { exact: false }).first()).toBeVisible();

    await attach(page, testInfo, '02-chart-visible');
  });

  test('activity graph is hidden on project overview when disabled', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();

    // Disable the feature
    await api.setSystemSetting(request, adminToken, 'feature_stats_timeline', false);

    // Create a project to visit its overview
    const project = await api.createProject(request, adminToken, 'FT' + Date.now().toString(36).slice(-3).toUpperCase(), 'Feature Toggle Test 2');

    await page.goto(`/projects/${project.key}`);
    await page.waitForLoadState('networkidle');

    // The "About" section should be visible (page loaded) but Activity section should not
    await expect(page.getByText('About', { exact: false }).first()).toBeVisible();

    // The activity chart heading should not be present
    const activityHeading = page.locator('h2', { hasText: 'Activity' });
    await expect(activityHeading).not.toBeVisible();

    await attach(page, testInfo, '03-chart-hidden');

    // Re-enable for clean state
    await api.setSystemSetting(request, adminToken, 'feature_stats_timeline', true);
  });

  test('toggling feature on admin page updates project overview', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();

    // Create a project first
    const project = await api.createProject(request, adminToken, 'FT' + Date.now().toString(36).slice(-3).toUpperCase(), 'Feature Toggle Test 3');

    // 1. Go to admin features page and disable via UI toggle
    await page.goto('/admin/features');
    await page.waitForLoadState('networkidle');

    const toggle = page.getByRole('switch');
    await expect(toggle).toBeVisible();

    // Toggle should be ON by default
    await expect(toggle).toHaveAttribute('aria-checked', 'true');

    // Click to disable
    await toggle.click();
    await expect(toggle).toHaveAttribute('aria-checked', 'false');

    await attach(page, testInfo, '04-toggle-off');

    // 2. Navigate to project overview — chart should be hidden
    await page.goto(`/projects/${project.key}`);
    await page.waitForLoadState('networkidle');

    await expect(page.getByText('About', { exact: false }).first()).toBeVisible();
    const activityHeading = page.locator('h2', { hasText: 'Activity' });
    await expect(activityHeading).not.toBeVisible();

    await attach(page, testInfo, '05-chart-hidden-after-toggle');

    // 3. Go back to features page and re-enable
    await page.goto('/admin/features');
    await page.waitForLoadState('networkidle');

    await toggle.click();
    await expect(toggle).toHaveAttribute('aria-checked', 'true');

    // 4. Navigate back — chart should be visible again
    await page.goto(`/projects/${project.key}`);
    await page.waitForLoadState('networkidle');

    await expect(page.getByText('Activity', { exact: false }).first()).toBeVisible();

    await attach(page, testInfo, '06-chart-visible-after-re-enable');
  });
});
