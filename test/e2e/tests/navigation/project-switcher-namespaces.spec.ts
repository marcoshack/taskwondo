import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

// These tests share the namespaces_enabled setting, so run serially.
base.describe.configure({ mode: 'serial' });

// Use admin context since namespace creation requires admin role
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
  return `ns-ps-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 5)}`;
}

function uniqueKey() {
  return `P${randomUUID().slice(0, 4).toUpperCase()}`;
}

test.describe('Project switcher — cross-namespace', () => {
  let nsSlug: string;
  let defaultProjectKey: string;
  let nsProjectKey: string;

  test.beforeAll(async ({ request }) => {
    const adminToken = getAdminToken();

    // Enable namespaces
    await api.enableNamespaces(request, adminToken);

    // Create a second namespace
    nsSlug = uniqueSlug();
    await api.createNamespace(request, adminToken, nsSlug, `Test NS ${nsSlug}`);

    // Create a project in the default namespace
    defaultProjectKey = uniqueKey();
    await api.createProject(request, adminToken, defaultProjectKey, `Default Project ${defaultProjectKey}`);

    // Create a project in the second namespace
    nsProjectKey = uniqueKey();
    await api.createProject(request, adminToken, nsProjectKey, `NS Project ${nsProjectKey}`, nsSlug);

    // Dismiss welcome modal
    await api.setPreference(request, adminToken, 'welcome_dismissed', true);
  });

  test.afterAll(async ({ request }) => {
    const adminToken = getAdminToken();
    // Clean up — ignore errors if resources are already gone
    await api.deleteNamespace(request, adminToken, nsSlug).catch(() => {});
  });

  test('project switcher shows projects from all namespaces with namespace info', async ({ page }) => {
    // Navigate to the default namespace projects page
    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');

    // Open project switcher via keyboard shortcut
    await page.keyboard.press('g');
    await page.keyboard.press('p');

    // The switcher modal should be visible
    const modal = page.getByRole('dialog');
    await expect(modal.getByRole('heading', { name: 'Switch Project' })).toBeVisible({ timeout: 5000 });

    // Both projects should be visible in the modal (match by button containing the key)
    const defaultRow = modal.getByRole('button', { name: new RegExp(defaultProjectKey) });
    const nsRow = modal.getByRole('button', { name: new RegExp(nsProjectKey) });

    await expect(defaultRow).toBeVisible({ timeout: 5000 });
    await expect(nsRow).toBeVisible({ timeout: 5000 });

    // The namespace slugs should be visible in each row
    await expect(defaultRow.getByText('default', { exact: true })).toBeVisible();
    await expect(nsRow.getByText(nsSlug, { exact: true })).toBeVisible();
  });

  test('search filters by namespace slug', async ({ page }) => {
    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');

    // Open project switcher
    await page.keyboard.press('g');
    await page.keyboard.press('p');

    const modal = page.getByRole('dialog');
    const searchInput = modal.getByRole('textbox', { name: /search projects/i });
    await expect(searchInput).toBeVisible({ timeout: 5000 });

    // Search by namespace slug — only the NS project should remain
    await searchInput.fill(nsSlug);
    await expect(modal.getByRole('button', { name: new RegExp(nsProjectKey) })).toBeVisible({ timeout: 3000 });
    await expect(modal.getByRole('button', { name: new RegExp(defaultProjectKey) })).not.toBeVisible({ timeout: 3000 });
  });

  test('selecting a project from another namespace navigates correctly', async ({ page }) => {
    // Start on the default namespace
    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');

    // Open project switcher
    await page.keyboard.press('g');
    await page.keyboard.press('p');

    const modal = page.getByRole('dialog');
    const searchInput = modal.getByRole('textbox', { name: /search projects/i });
    await expect(searchInput).toBeVisible({ timeout: 5000 });

    // Search for the NS project and select it
    await searchInput.fill(nsProjectKey);
    const nsProjectRow = modal.getByRole('button', { name: new RegExp(nsProjectKey, 'i') });
    await expect(nsProjectRow).toBeVisible({ timeout: 3000 });
    await nsProjectRow.click();

    // Should navigate to the project in the other namespace
    await expect(page).toHaveURL(new RegExp(`/${nsSlug}/projects/${nsProjectKey}`), { timeout: 5000 });
  });

  test('no "Project not found" flash when switching namespaces via project switcher', async ({ page }) => {
    // First, visit the default project to populate the TanStack Query cache
    await page.goto(`/d/projects/${defaultProjectKey}`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(`Default Project ${defaultProjectKey}`)).toBeVisible({ timeout: 5000 });

    // Open project switcher and navigate to project in the other namespace
    await page.keyboard.press('g');
    await page.keyboard.press('p');

    const modal = page.getByRole('dialog');
    const searchInput = modal.getByRole('textbox', { name: /search projects/i });
    await expect(searchInput).toBeVisible({ timeout: 5000 });

    await searchInput.fill(nsProjectKey);
    const nsProjectRow = modal.getByRole('button', { name: new RegExp(nsProjectKey, 'i') });
    await expect(nsProjectRow).toBeVisible({ timeout: 3000 });
    await nsProjectRow.click();

    // Wait for navigation to complete
    await expect(page).toHaveURL(new RegExp(`/${nsSlug}/projects/${nsProjectKey}`), { timeout: 5000 });

    // The project overview should load — "Project not found." must never appear
    await expect(page.getByText('Project not found.')).not.toBeVisible({ timeout: 3000 });

    // The project name should appear in the overview
    await expect(page.getByText(`NS Project ${nsProjectKey}`)).toBeVisible({ timeout: 5000 });
  });
});
