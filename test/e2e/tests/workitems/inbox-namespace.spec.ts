import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

// These tests toggle namespaces_enabled — run serially.
base.describe.configure({ mode: 'serial' });

// Use admin context since namespace creation requires admin role.
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

function uniqueSlug() {
  return `ns-ib-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 5)}`;
}

function uniqueKey() {
  return `I${randomUUID().slice(0, 4).toUpperCase()}`;
}

test.describe('Inbox — cross-namespace project filter', () => {
  let nsSlug: string;
  let defaultProjectKey: string;
  let nsProjectKey: string;

  test.beforeAll(async ({ request }) => {
    const adminToken = getAdminToken();

    await api.enableNamespaces(request, adminToken);

    nsSlug = uniqueSlug();
    await api.createNamespace(request, adminToken, nsSlug, `Inbox NS ${nsSlug}`);

    defaultProjectKey = uniqueKey();
    await api.createProject(request, adminToken, defaultProjectKey, `Default Inbox Proj ${defaultProjectKey}`);

    nsProjectKey = uniqueKey();
    await api.createProject(request, adminToken, nsProjectKey, `NS Inbox Proj ${nsProjectKey}`, nsSlug);

    // Create work items and add to inbox
    const item1 = await api.createWorkItem(request, adminToken, defaultProjectKey, {
      title: `Dflt inbox ${defaultProjectKey}`,
      type: 'task',
    });
    await api.addToInbox(request, adminToken, item1.id);

    const item2 = await api.createWorkItem(request, adminToken, nsProjectKey, {
      title: `NS inbox ${nsProjectKey}`,
      type: 'task',
    }, nsSlug);
    await api.addToInbox(request, adminToken, item2.id);

    await api.setPreference(request, adminToken, 'welcome_dismissed', true);
    await api.setPreference(request, adminToken, 'inbox_project_filter', []);
  });

  test.afterAll(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.deleteNamespace(request, adminToken, nsSlug).catch(() => {});
  });

  test('desktop project filter shows projects from all namespaces with key badge and namespace info', async ({ page }) => {
    await page.goto('/user/inbox');
    await page.waitForLoadState('networkidle');

    // Use unique item titles to avoid strict mode violations from retries
    const table = page.getByRole('table');
    await expect(table.getByText(`Dflt inbox ${defaultProjectKey}`)).toBeVisible({ timeout: 10000 });
    await expect(table.getByText(`NS inbox ${nsProjectKey}`)).toBeVisible();

    // Open the project filter dropdown
    const projectFilter = page.locator('.hidden.lg\\:flex').getByText('Projects');
    await projectFilter.click();

    // Both projects should be listed as checkboxes (labels include key, name, and namespace)
    await expect(page.getByLabel(new RegExp(defaultProjectKey))).toBeVisible({ timeout: 5000 });
    await expect(page.getByLabel(new RegExp(nsProjectKey))).toBeVisible();

    // Namespace slug should be visible in the project options
    const defaultLabel = page.getByLabel(new RegExp(defaultProjectKey));
    await expect(defaultLabel.locator('..').getByText('default', { exact: true })).toBeVisible();
    const nsLabel = page.getByLabel(new RegExp(nsProjectKey));
    await expect(nsLabel.locator('..').getByText(nsSlug, { exact: true })).toBeVisible();

    // Filter by the namespace project only
    await page.getByText('None').click();
    await page.getByLabel(new RegExp(nsProjectKey)).check();
    await page.locator('h1').click();

    // Only the namespace project item should be visible
    await expect(table.getByText(`NS inbox ${nsProjectKey}`)).toBeVisible({ timeout: 10000 });
    await expect(table.getByText(`Dflt inbox ${defaultProjectKey}`)).not.toBeVisible({ timeout: 5000 });
  });

  test('mobile project filter shows projects from all namespaces with key badge and namespace info', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    // Clear filter from previous test
    const adminToken = getAdminToken();
    await api.setPreference(page.request, adminToken, 'inbox_project_filter', []);

    await page.goto('/user/inbox');
    await page.waitForLoadState('networkidle');

    const cards = page.locator('.lg\\:hidden');
    await expect(cards.getByText(`Dflt inbox ${defaultProjectKey}`)).toBeVisible({ timeout: 10000 });
    await expect(cards.getByText(`NS inbox ${nsProjectKey}`)).toBeVisible();

    // Open the mobile project filter modal
    await page.getByRole('button', { name: 'Filter by project' }).click();
    await expect(page.getByRole('heading', { name: 'Filter by project' })).toBeVisible({ timeout: 5000 });

    // Both projects should be listed as checkboxes with namespace info
    await expect(page.getByLabel(new RegExp(defaultProjectKey))).toBeVisible();
    await expect(page.getByLabel(new RegExp(nsProjectKey))).toBeVisible();

    // Namespace info should be visible in the modal
    const modal = page.getByRole('dialog');
    const defaultLabel = modal.getByLabel(new RegExp(defaultProjectKey));
    await expect(defaultLabel.locator('..').getByText('default', { exact: true })).toBeVisible();
    const nsLabel = modal.getByLabel(new RegExp(nsProjectKey));
    await expect(nsLabel.locator('..').getByText(nsSlug, { exact: true })).toBeVisible();
  });
});

test.describe('CreateWorkItemModal — cross-namespace project picker', () => {
  let nsSlug: string;
  let defaultProjectKey: string;
  let nsProjectKey: string;

  test.beforeAll(async ({ request }) => {
    const adminToken = getAdminToken();

    await api.enableNamespaces(request, adminToken);

    nsSlug = uniqueSlug();
    await api.createNamespace(request, adminToken, nsSlug, `Create NS ${nsSlug}`);

    defaultProjectKey = uniqueKey();
    await api.createProject(request, adminToken, defaultProjectKey, `Default Create Proj ${defaultProjectKey}`);

    nsProjectKey = uniqueKey();
    await api.createProject(request, adminToken, nsProjectKey, `NS Create Proj ${nsProjectKey}`, nsSlug);

    await api.setPreference(request, adminToken, 'welcome_dismissed', true);
    // Clear any remembered last project so the picker shows "Select project"
    await api.setPreference(request, adminToken, 'taskwondo_last_project_key', '');
  });

  test.afterAll(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.deleteNamespace(request, adminToken, nsSlug).catch(() => {});
  });

  test('new item modal project picker shows projects from all namespaces with decorated display', async ({ page }) => {
    await page.goto('/user/inbox');
    await page.waitForLoadState('networkidle');

    // Open the Create Work Item modal
    await page.getByRole('button', { name: /new item/i }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog.getByRole('heading', { name: 'New Work Item' })).toBeVisible({ timeout: 5000 });

    // Click the project picker button to open the dropdown
    await dialog.getByRole('button', { name: /select project/i }).click();

    // Both projects should be visible as buttons in the picker dropdown
    const defaultBtn = dialog.getByRole('button', { name: new RegExp(defaultProjectKey) });
    const nsBtn = dialog.getByRole('button', { name: new RegExp(nsProjectKey) });
    await expect(defaultBtn).toBeVisible({ timeout: 5000 });
    await expect(nsBtn).toBeVisible();

    // Namespace info should be visible in the project buttons
    await expect(defaultBtn.getByText('default', { exact: true })).toBeVisible();
    await expect(nsBtn.getByText(nsSlug, { exact: true })).toBeVisible();

    // Search filters across namespace projects (search by namespace slug)
    const searchInput = dialog.getByPlaceholder(/search project/i);
    await searchInput.fill(nsSlug);
    await expect(nsBtn).toBeVisible({ timeout: 3000 });
    await expect(defaultBtn).not.toBeVisible({ timeout: 3000 });
  });
});
