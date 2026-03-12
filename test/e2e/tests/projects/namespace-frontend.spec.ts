import { test as base, expect, test as userTest } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';

// Tests in this file share mutable global state (namespaces_enabled setting)
// so they must run serially to avoid race conditions.
base.describe.configure({ mode: 'serial' });

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

    // Ensure namespaces are disabled via API, then navigate to pick up the new state
    await api.disableNamespaces(request, adminToken);

    // Navigate to the admin features page (fresh load picks up API state)
    await page.goto('/admin/features');
    await page.waitForLoadState('networkidle');

    // Find the Namespaces section and its toggle
    const namespacesSection = page.locator('div').filter({ hasText: /^Namespaces/ }).first();
    const toggle = namespacesSection.locator('button[role="switch"]');

    // If a parallel test re-enabled namespaces between our API call and page load, turn it off via UI
    if (await toggle.getAttribute('aria-checked') === 'true') {
      await toggle.click();
      await expect(toggle).toHaveAttribute('aria-checked', 'false', { timeout: 10000 });
    }
    await expect(toggle).toHaveAttribute('aria-checked', 'false');
    await attach(page, testInfo, '00-toggle-off');

    // Click to enable
    await toggle.click();

    // Toggle should now be on
    await expect(toggle).toHaveAttribute('aria-checked', 'true', { timeout: 10000 });
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

    // Submit and wait for navigation to the namespace settings page
    await dialog.getByRole('button', { name: /^create$/i }).click();
    await page.waitForURL(new RegExp(`/${createSlug}/settings`), { timeout: 15000 });
    await page.waitForLoadState('networkidle');

    // Verify the namespace was created via API
    const ns = await api.getNamespace(request, adminToken, createSlug);
    expect(ns.display_name).toBe('Created via Modal');

    await attach(page, testInfo, '04-after-create');

    // Cleanup
    await api.deleteNamespace(request, adminToken, createSlug);
  });

  test('sidebar shows namespace badge immediately after creation', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();

    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');

    // Open the namespace switcher dropdown
    const switcher = page.getByTestId('namespace-switcher');
    await switcher.click();

    // Click "Create namespace" in the dropdown
    await page.getByText('Create namespace').click();

    // Fill in the form
    const createSlug = uniqueSlug();
    const nsName = 'Sidebar Badge NS';
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await dialog.getByLabel('Display Name', { exact: false }).fill(nsName);
    await dialog.getByLabel('Slug', { exact: false }).fill(createSlug);

    // Submit
    await dialog.getByRole('button', { name: /^create$/i }).click();

    // Wait for navigation to the namespace settings page
    await page.waitForURL(new RegExp(`/${createSlug}/settings`));
    await page.waitForLoadState('networkidle');

    // The sidebar should show the new namespace name in the banner button
    const nsBanner = page.getByRole('button', { name: /switch namespace/i }).filter({ hasText: nsName });
    await expect(nsBanner).toBeVisible();

    await attach(page, testInfo, 'sidebar-badge-after-create');

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

  test('new project modal auto-selects the current namespace', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const slug = uniqueSlug();
    const nsName = 'Auto Select NS';

    // Create a second namespace so the switcher and namespace picker appear
    await api.createNamespace(request, adminToken, slug, nsName);

    // Navigate to the default namespace project list
    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');

    // Switch to the new namespace
    const switcher = page.getByTestId('namespace-switcher');
    await switcher.click();
    await page.getByText(nsName).click();
    await page.waitForLoadState('networkidle');

    // Open the New Project modal
    await page.getByRole('button', { name: /new project/i }).click();

    // The namespace selector button inside the modal should show the current namespace
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await expect(dialog.getByText(nsName)).toBeVisible();
    await attach(page, testInfo, 'new-project-auto-select-ns');

    // Cleanup
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

      // Type project key to confirm
      const confirmInput = page.getByRole('dialog').locator(`input[placeholder="${xferKey}"]`);
      await confirmInput.fill(xferKey);

      // Select target namespace from custom dropdown
      const nsDropdown = page.getByRole('dialog').getByRole('button', { name: /select a namespace/i });
      if (await nsDropdown.isVisible().catch(() => false)) {
        await nsDropdown.click();
        await page.waitForTimeout(300);
        // Click the namespace option in the portaled dropdown
        const nsOption = page.locator(`button:has-text("${nsName}")`).last();
        await nsOption.click();
      }

      // Confirm transfer
      const confirmBtn = page.getByRole('dialog').getByRole('button', { name: /transfer project/i });
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

  test('namespace icon and color show in New Project modal namespace selector', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const slug = uniqueSlug();

    // Create a namespace with a custom icon and color
    const ns = await api.createNamespace(request, adminToken, slug, 'Icon Test NS');
    await api.updateNamespace(request, adminToken, slug, { icon: 'rocket', color: 'green' });

    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');

    // Open New Project modal
    await page.getByRole('button', { name: /new project/i }).click();
    const modal = page.getByRole('dialog');
    await expect(modal).toBeVisible();

    // The namespace selector button should show the NamespaceIcon (not a generic Globe/Building2)
    // The active namespace is default, which uses globe icon — click to open the picker
    const nsButton = modal.locator('button').filter({ hasText: /namespace/i }).first();
    // If namespace button isn't visible (single namespace), the selector field should be there
    const nsSelectorBtn = modal.locator('button[type="button"]').filter({ has: page.locator('.lucide') }).last();

    // Open the namespace picker
    await nsSelectorBtn.click();

    // A second modal with the namespace list should appear
    const pickerModal = page.getByRole('dialog').filter({ hasText: /select namespace/i });
    await expect(pickerModal).toBeVisible();

    // The custom namespace should show a rocket icon (lucide-rocket class)
    const nsRow = pickerModal.locator('button').filter({ hasText: 'Icon Test NS' });
    await expect(nsRow).toBeVisible();
    await expect(nsRow.locator('.lucide-rocket')).toBeVisible();

    await attach(page, testInfo, 'ns-icon-in-picker');

    // Select the custom namespace
    await nsRow.click();

    // The selector button in the form should now show the rocket icon too
    const selectedBtn = modal.locator('button[type="button"]').filter({ hasText: 'Icon Test NS' });
    await expect(selectedBtn.locator('.lucide-rocket')).toBeVisible();

    await attach(page, testInfo, 'ns-icon-in-selector');

    // Cleanup
    await api.deleteNamespace(request, adminToken, slug);
  });
});

// --- Regular (non-admin) user namespace tests ---

userTest.describe('Namespace Switcher for Regular Users', () => {
  userTest.beforeEach(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.enableNamespaces(request, adminToken);
  });

  userTest('created namespace appears in switcher for regular user', async ({ page, request, testUser }, testInfo) => {
    const adminToken = getAdminToken();
    const slug = uniqueSlug();
    const nsName = 'Regular User NS';

    // Create a namespace via API as the regular user
    const ns = await api.createNamespace(request, testUser.token, slug, nsName);
    expect(ns.slug).toBe(slug);

    // Verify the namespace appears in the list API for this user
    const namespaces = await api.listNamespaces(request, testUser.token);
    const found = namespaces.find(n => n.slug === slug);
    expect(found).toBeDefined();
    expect(found!.display_name).toBe(nsName);

    // Navigate to the app and verify the namespace appears in the switcher UI
    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');

    const switcher = page.getByTestId('namespace-switcher');
    await switcher.click();
    await expect(page.getByText(nsName)).toBeVisible();

    await attach(page, testInfo, 'regular-user-ns-in-switcher');

    // Cleanup
    await api.deleteNamespace(request, adminToken, slug);
  });

  userTest('namespace created via UI modal appears in switcher for regular user', async ({ page, request, testUser }, testInfo) => {
    const adminToken = getAdminToken();

    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');

    // Open the namespace switcher dropdown
    const switcher = page.getByTestId('namespace-switcher');
    await switcher.click();

    // Click "Create namespace"
    await page.getByText('Create namespace').click();

    // Fill in the form
    const createSlug = uniqueSlug();
    const nsName = 'UI Created NS';
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await dialog.getByLabel('Display Name', { exact: false }).fill(nsName);
    await dialog.getByLabel('Slug', { exact: false }).fill(createSlug);

    // Submit
    await dialog.getByRole('button', { name: /^create$/i }).click();

    // Wait for navigation to the namespace settings page
    await page.waitForURL(new RegExp(`/${createSlug}/settings`));
    await page.waitForLoadState('networkidle');

    // Now open the switcher and verify the new namespace appears in the dropdown
    const switcherAfter = page.getByTestId('namespace-switcher');
    await switcherAfter.click();
    // Scope to the dropdown container (sibling of the switcher button)
    const dropdown = switcherAfter.locator('..').locator('div.absolute');
    await expect(dropdown).toBeVisible();
    await expect(dropdown.getByText(nsName)).toBeVisible();

    await attach(page, testInfo, 'regular-user-ui-ns-in-switcher');

    // Verify via API that the list endpoint returns the namespace
    const namespaces = await api.listNamespaces(request, testUser.token);
    const found = namespaces.find(n => n.slug === createSlug);
    expect(found).toBeDefined();

    // Cleanup
    await api.deleteNamespace(request, adminToken, createSlug);
  });
});
