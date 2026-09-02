import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';

// Admin context — brand_name is a system setting
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

const BRAND = 'Acme Tab Title';

test.describe('Brand tab title', () => {
  test.afterEach(async ({ request }) => {
    const adminToken = getAdminToken();
    // Re-sync the default namespace display name back, then drop the override
    await api.setSystemSetting(request, adminToken, 'brand_name', 'Taskwondo');
    await api.deleteSystemSetting(request, adminToken, 'brand_name');
  });

  test('tab title uses the configured brand name', async ({ page, request }) => {
    const adminToken = getAdminToken();
    await api.setSystemSetting(request, adminToken, 'brand_name', BRAND);

    // Base title on a plain page
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    await expect(page).toHaveTitle(BRAND);

    // Work item detail page composes display ID + title + brand
    const project = await api.createProject(request, adminToken, 'BT' + Date.now().toString(36).slice(-3).toUpperCase(), 'Brand Title Project');
    const item = await api.createWorkItem(request, adminToken, project.key, {
      title: 'Check the tab',
      type: 'task',
    });

    await page.goto(`/d/projects/${project.key}/items/${item.item_number}`);
    await expect(page.getByText('Check the tab').first()).toBeVisible();
    await expect(page).toHaveTitle(`${project.key}-${item.item_number} Check the tab - ${BRAND}`);

    // Leaving the detail page restores the base brand title
    await page.goto('/');
    await expect(page).toHaveTitle(BRAND);
  });

  test('tab title falls back to Taskwondo when no brand is set', async ({ page, request }) => {
    const adminToken = getAdminToken();
    await api.deleteSystemSetting(request, adminToken, 'brand_name');

    await page.goto('/');
    await page.waitForLoadState('networkidle');
    await expect(page).toHaveTitle('Taskwondo');
  });
});
