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
  return `ns-nav-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 5)}`;
}

function uniqueKey() {
  return `N${randomUUID().slice(0, 4).toUpperCase()}`;
}

test.describe('Inbox — cross-namespace work item navigation', () => {
  let nsSlug: string;
  let defaultProjectKey: string;
  let nsProjectKey: string;
  let defaultItemTitle: string;
  let nsItemTitle: string;
  let nsItemNumber: number;

  test.beforeAll(async ({ request }) => {
    const adminToken = getAdminToken();

    await api.enableNamespaces(request, adminToken);

    nsSlug = uniqueSlug();
    await api.createNamespace(request, adminToken, nsSlug, `Nav NS ${nsSlug}`);

    defaultProjectKey = uniqueKey();
    await api.createProject(request, adminToken, defaultProjectKey, `Default Nav Proj ${defaultProjectKey}`);

    nsProjectKey = uniqueKey();
    await api.createProject(request, adminToken, nsProjectKey, `NS Nav Proj ${nsProjectKey}`, nsSlug);

    // Create work items and add to inbox
    defaultItemTitle = `Default nav item ${defaultProjectKey}`;
    const item1 = await api.createWorkItem(request, adminToken, defaultProjectKey, {
      title: defaultItemTitle,
      type: 'task',
    });
    await api.addToInbox(request, adminToken, item1.id);

    nsItemTitle = `NS nav item ${nsProjectKey}`;
    const item2 = await api.createWorkItem(request, adminToken, nsProjectKey, {
      title: nsItemTitle,
      type: 'task',
    }, nsSlug);
    nsItemNumber = item2.item_number;
    await api.addToInbox(request, adminToken, item2.id);

    await api.setPreference(request, adminToken, 'welcome_dismissed', true);
    await api.setPreference(request, adminToken, 'inbox_project_filter', []);
  });

  test.afterAll(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.deleteNamespace(request, adminToken, nsSlug).catch(() => {});
  });

  test('clicking inbox item from different namespace loads work item detail page', async ({ page }) => {
    // First, navigate to the default namespace to set it as active
    await page.goto(`/d/projects/${defaultProjectKey}`);
    await page.waitForLoadState('networkidle');

    // Now go to inbox — the active namespace context is "default"
    await page.goto('/user/inbox');
    await page.waitForLoadState('networkidle');

    const table = page.getByRole('table');
    await expect(table.getByText(nsItemTitle)).toBeVisible({ timeout: 10000 });

    // Click the work item from the other namespace
    await table.getByText(nsItemTitle).click();

    // Should navigate to the work item detail page in the other namespace
    await expect(page).toHaveURL(
      new RegExp(`/${nsSlug}/projects/${nsProjectKey}/items/${nsItemNumber}`),
      { timeout: 10000 },
    );

    // The work item detail page should load fully (not stuck on spinner).
    // Use heading locator to avoid strict-mode clash with the inbox table row.
    await expect(page.getByRole('heading', { name: nsItemTitle })).toBeVisible({ timeout: 15000 });

    // Verify the "Project not found" error does NOT appear
    await expect(page.getByText('Project not found')).not.toBeVisible({ timeout: 2000 });
  });

  test('navigating back and clicking default namespace item also works', async ({ page }) => {
    // Start on a page in the custom namespace
    await page.goto(`/${nsSlug}/projects/${nsProjectKey}`);
    await page.waitForLoadState('networkidle');

    // Navigate to inbox
    await page.goto('/user/inbox');
    await page.waitForLoadState('networkidle');

    const table = page.getByRole('table');
    await expect(table.getByText(defaultItemTitle)).toBeVisible({ timeout: 10000 });

    // Click the work item from the default namespace
    await table.getByText(defaultItemTitle).click();

    // Should navigate to the work item detail page in the default namespace
    await expect(page).toHaveURL(
      new RegExp(`/d/projects/${defaultProjectKey}/items/`),
      { timeout: 10000 },
    );

    // The work item detail page should load fully
    await expect(page.getByRole('heading', { name: defaultItemTitle })).toBeVisible({ timeout: 15000 });

    // No error
    await expect(page.getByText('Project not found')).not.toBeVisible({ timeout: 2000 });
  });
});
