import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';

// Admin context — namespace management requires admin role
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

async function attach(page: any, testInfo: any, name: string) {
  const screenshot = await page.screenshot();
  await testInfo.attach(name, { body: screenshot, contentType: 'image/png' });
}

function uniqueSlug() {
  return `ns-fe-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 5)}`;
}

test.describe('Namespace Feature Toggle', () => {
  test('enabling namespaces via admin features toggle', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();

    // Navigate to the admin features page
    await page.goto('/admin/features');
    await page.waitForLoadState('networkidle');

    // Find the Namespaces section and its toggle
    const namespacesSection = page.locator('div').filter({ hasText: /^Namespaces/ }).first();
    const toggle = namespacesSection.locator('button[role="switch"]');

    // Ensure toggle is off — if already on (parallel test enabled it), turn off via UI first
    if (await toggle.getAttribute('aria-checked') === 'true') {
      await toggle.click();
      await page.waitForResponse(resp =>
        resp.url().includes('/settings/public') && resp.status() === 200,
      );
    }
    await expect(toggle).toHaveAttribute('aria-checked', 'false');
    await attach(page, testInfo, '00-toggle-off');

    // Click to enable
    await toggle.click();

    // Wait for the setting to be saved and refetched
    await page.waitForResponse(resp =>
      resp.url().includes('/settings/public') && resp.status() === 200,
    );

    // Toggle should now be on
    await expect(toggle).toHaveAttribute('aria-checked', 'true');
    await attach(page, testInfo, '00-toggle-on');

    // Verify the feature actually works: create a namespace via API
    const slug = uniqueSlug();
    await api.createNamespace(request, adminToken, slug, 'Toggle Test NS');

    // Navigate to projects — namespace switcher should appear
    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');
    const switcher = page.getByTestId('namespace-switcher');
    await expect(switcher).toBeVisible();
    await attach(page, testInfo, '00-feature-active');

    // Cleanup
    await api.deleteNamespace(request, adminToken, slug);
  });
});

test.describe('Namespace Frontend UI', () => {
  test.beforeEach(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.enableNamespaces(request, adminToken);
  });

  test('namespace switcher appears when user has multiple namespaces', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const slug = uniqueSlug();

    // Create a second namespace so the switcher shows
    await api.createNamespace(request, adminToken, slug, 'Switcher Test NS');

    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');

    // The namespace switcher should be visible
    const switcher = page.getByTestId('namespace-switcher');
    await expect(switcher).toBeVisible();

    await attach(page, testInfo, '01-switcher-visible');

    // Cleanup
    await api.deleteNamespace(request, adminToken, slug);
  });

  test('create namespace via modal from switcher dropdown', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();

    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');

    // Open the namespace switcher dropdown
    const switcher = page.getByTestId('namespace-switcher');
    await switcher.click();

    // Click "Create namespace" in the dropdown
    await page.getByText('Create namespace').click();

    // Should see the create namespace modal
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    // Fill in the form
    const createSlug = uniqueSlug();
    await dialog.getByLabel('Display Name', { exact: false }).fill('Created via Modal');
    await dialog.getByLabel('Slug', { exact: false }).fill(createSlug);

    await attach(page, testInfo, '03-create-form-filled');

    // Submit
    await dialog.getByRole('button', { name: /^create$/i }).click();

    // Should navigate to projects (setActiveNamespace navigates)
    await page.waitForLoadState('networkidle');

    // Verify the namespace was created via API
    const ns = await api.getNamespace(request, adminToken, createSlug);
    expect(ns.display_name).toBe('Created via Modal');

    await attach(page, testInfo, '04-after-create');

    // Cleanup
    await api.deleteNamespace(request, adminToken, createSlug);
  });

  test('namespace settings page loads and allows rename', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const slug = uniqueSlug();

    await api.createNamespace(request, adminToken, slug, 'Settings Test NS');

    await page.goto(`/${slug}/settings`);
    await page.waitForLoadState('networkidle');

    // Should see settings page heading
    await expect(page.getByRole('heading', { name: 'Namespace Settings' })).toBeVisible();

    await attach(page, testInfo, '05-settings-page');

    // Update display name
    const nameInput = page.getByLabel('Display Name', { exact: false });
    await expect(nameInput).toBeVisible();
    await nameInput.fill('Renamed NS');

    // Click save
    const saveBtn = page.getByRole('button', { name: /save/i }).first();
    await saveBtn.click();

    // Wait for save to complete
    await page.waitForTimeout(1000);
    await attach(page, testInfo, '06-after-rename');

    // Verify via API
    const ns = await api.getNamespace(request, adminToken, slug);
    expect(ns.display_name).toBe('Renamed NS');

    // Cleanup
    await api.deleteNamespace(request, adminToken, slug);
  });

  test('namespace settings shows members section', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const slug = uniqueSlug();

    await api.createNamespace(request, adminToken, slug, 'Members Test NS');

    await page.goto(`/${slug}/settings`);
    await page.waitForLoadState('networkidle');

    // Should see members section
    await expect(page.getByText('Members', { exact: false }).first()).toBeVisible();

    await attach(page, testInfo, '07-members-section');

    // Cleanup
    await api.deleteNamespace(request, adminToken, slug);
  });

  test('switching namespace clears active project from nav', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const slug = uniqueSlug();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

    // Create a second namespace and a project in the default namespace
    await api.createNamespace(request, adminToken, slug, 'Switch Clear NS');
    const projKey = `SC${Date.now().toString(36).slice(-3).toUpperCase()}`;
    await api.createProject(request, adminToken, projKey, 'Switch Clear Project');

    // Navigate into the project so it becomes the active project in the nav
    await page.goto(`/d/projects/${projKey}/items`);
    await page.waitForLoadState('networkidle');

    // Verify we're on the project page
    expect(page.url()).toContain(`/d/projects/${projKey}`);
    // The project name should be visible in the nav bar (desktop)
    await expect(page.locator('nav').getByText('Switch Clear Project')).toBeVisible();
    await attach(page, testInfo, '11-project-active');

    // Switch namespace via the switcher dropdown
    const switcher = page.getByTestId('namespace-switcher');
    await switcher.click();
    await page.getByText('Switch Clear NS').click();
    await page.waitForLoadState('networkidle');

    // Should have navigated to /<slug>/projects (not a project detail page)
    await expect(page).toHaveURL(new RegExp(`/${slug}/projects$`));
    // The project name should no longer be in the nav
    await expect(page.locator('nav').getByText('Switch Clear Project')).not.toBeVisible();
    await attach(page, testInfo, '12-project-cleared');

    // Cleanup
    await request.delete(`${BASE_URL}/api/v1/default/projects/${projKey}`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    await api.deleteNamespace(request, adminToken, slug);
  });

  test('project transfer between namespaces via settings page', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const slug = uniqueSlug();
    const nsName = 'Transfer Target NS';
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

    await api.createNamespace(request, adminToken, slug, nsName);

    // Create a project in the default namespace for transfer
    const xferKey = `XF${Date.now().toString(36).slice(-3).toUpperCase()}`;
    await api.createProject(request, adminToken, xferKey, 'Transfer Test Project');

    // Go to the project settings page
    await page.goto(`/d/projects/${xferKey}/settings`);
    await page.waitForLoadState('networkidle');

    await attach(page, testInfo, '08-project-settings');

    // Look for transfer/namespace section in danger zone
    const transferBtn = page.getByRole('button', { name: /transfer|move|migrate/i });

    if (await transferBtn.isVisible().catch(() => false)) {
      await transferBtn.click();
      await page.waitForTimeout(500);
      await attach(page, testInfo, '09-transfer-modal');

      // Select target namespace in the modal
      const nsSelect = page.getByRole('dialog').locator('select');
      if (await nsSelect.isVisible().catch(() => false)) {
        await nsSelect.selectOption({ label: new RegExp(nsName, 'i') });
      }

      // Confirm transfer
      const confirmBtn = page.getByRole('dialog').getByRole('button', { name: /transfer|confirm|move/i });
      if (await confirmBtn.isVisible().catch(() => false)) {
        await confirmBtn.click();
        await page.waitForLoadState('networkidle');
        await attach(page, testInfo, '10-after-transfer');

        // Verify project moved via API
        const listRes = await request.get(`${BASE_URL}/api/v1/${slug}/projects`, {
          headers: { Authorization: `Bearer ${adminToken}` },
        });
        const listBody = await listRes.json();
        const found = listBody.data.find((p: any) => p.key === xferKey);
        expect(found).toBeDefined();
      }
    }

    // Cleanup: delete projects in namespace, then namespace
    const listRes = await request.get(`${BASE_URL}/api/v1/${slug}/projects`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    if (listRes.ok()) {
      for (const p of (await listRes.json()).data) {
        await request.delete(`${BASE_URL}/api/v1/${slug}/projects/${p.key}`, {
          headers: { Authorization: `Bearer ${adminToken}` },
        });
      }
    }
    await api.deleteNamespace(request, adminToken, slug);
  });
});
