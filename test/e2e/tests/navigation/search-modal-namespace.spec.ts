import { test, expect, getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { openPalette, paletteInput } from '../../lib/palette';

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

// These tests toggle the global namespaces_enabled setting, so they run in the
// namespace project (serial) to avoid racing with other namespace tests.
test.describe('Command palette cross-namespace navigation', () => {
  test('cross-namespace search result navigates and fully loads work item', async ({
    page,
    request,
    testUser,
  }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';
    const nsSlug = `ns-${Date.now().toString(36)}`;
    const projKey = `XN${Date.now().toString(36).slice(-3).toUpperCase()}`;
    const uniqueTitle = `CrossNS-${Date.now()}`;

    // Enable namespaces and create a second namespace
    await api.enableNamespaces(request, adminToken);
    await api.createNamespace(request, adminToken, nsSlug, 'Cross NS Test');
    await api.addNamespaceMember(request, adminToken, nsSlug, testUser.id, 'member');

    // Create a project and work item in the new namespace
    const projRes = await request.post(`${BASE_URL}/api/v1/${nsSlug}/projects`, {
      headers: { Authorization: `Bearer ${testUser.token}` },
      data: { key: projKey, name: 'Cross NS Project' },
    });
    expect(projRes.ok()).toBeTruthy();

    const itemRes = await request.post(`${BASE_URL}/api/v1/${nsSlug}/projects/${projKey}/items`, {
      headers: { Authorization: `Bearer ${testUser.token}` },
      data: { title: uniqueTitle, type: 'task' },
    });
    expect(itemRes.ok()).toBeTruthy();
    const item = (await itemRes.json()).data;

    // Also create a default-namespace project so the default namespace has real
    // content behind the page we start on.
    const defProjKey = `XD${Date.now().toString(36).slice(-3).toUpperCase()}`;
    await api.createProject(request, testUser.token, defProjKey, 'Default NS Search Proj');
    await api.createWorkItem(request, testUser.token, defProjKey, {
      title: `DefItem-${Date.now()}`,
      type: 'task',
    });

    try {
      // Start on the default namespace's project list (inside NamespaceGuard).
      // This used to start on a work item detail page, but entity search is now
      // scoped to the active project (TF-432/434), and a cross-namespace hit is
      // only reachable while no project is active — which is the case here,
      // because this context has never opened a project page.
      await page.goto('/d/projects');
      await dismissWelcomeModal(page);
      await page.waitForLoadState('networkidle');

      // Open the palette and search for the cross-namespace item
      await openPalette(page);
      const searchInput = paletteInput(page);
      await expect(searchInput).toBeVisible({ timeout: 3000 });
      await searchInput.fill(uniqueTitle);

      // Wait for results
      const result = page.locator('[data-search-item]').first();
      await expect(result).toBeVisible({ timeout: 10000 });
      await expect(result).toContainText(uniqueTitle);

      // Click the result
      await result.click();

      // Should navigate to the work item in the OTHER namespace
      await expect(page).toHaveURL(
        new RegExp(`/${nsSlug}/projects/${projKey}/items/${item.item_number}`),
        { timeout: 10000 },
      );

      // The work item detail page should load fully (not stuck on spinner)
      await expect(page.getByRole('heading', { name: uniqueTitle })).toBeVisible({ timeout: 15000 });

      // Verify no "Project not found" error
      await expect(page.getByText('Project not found')).not.toBeVisible({ timeout: 2000 });
    } finally {
      // Cleanup
      await api.migrateProject(request, adminToken, nsSlug, projKey, 'default').catch(() => {});
      await request.delete(`${BASE_URL}/api/v1/default/projects/${projKey}`, {
        headers: { Authorization: `Bearer ${adminToken}` },
      }).catch(() => {});
      await request.delete(`${BASE_URL}/api/v1/default/projects/${defProjKey}`, {
        headers: { Authorization: `Bearer ${adminToken}` },
      }).catch(() => {});
      await api.deleteNamespace(request, adminToken, nsSlug).catch(() => {});
    }
  });

  test('cross-namespace search via Enter key also loads correctly', async ({
    page,
    request,
    testUser,
  }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';
    const nsSlug = `ns-enter-${Date.now().toString(36)}`;
    const projKey = `XE${Date.now().toString(36).slice(-3).toUpperCase()}`;
    const uniqueTitle = `CrossNSEnter-${Date.now()}`;

    await api.enableNamespaces(request, adminToken);
    await api.createNamespace(request, adminToken, nsSlug, 'Cross NS Enter Test');
    await api.addNamespaceMember(request, adminToken, nsSlug, testUser.id, 'member');

    const projRes = await request.post(`${BASE_URL}/api/v1/${nsSlug}/projects`, {
      headers: { Authorization: `Bearer ${testUser.token}` },
      data: { key: projKey, name: 'Cross NS Enter Project' },
    });
    expect(projRes.ok()).toBeTruthy();

    const itemRes = await request.post(`${BASE_URL}/api/v1/${nsSlug}/projects/${projKey}/items`, {
      headers: { Authorization: `Bearer ${testUser.token}` },
      data: { title: uniqueTitle, type: 'task' },
    });
    expect(itemRes.ok()).toBeTruthy();
    const item = (await itemRes.json()).data;

    try {
      // Start on a page in the default namespace
      await page.goto('/d/projects');
      await dismissWelcomeModal(page);
      await page.waitForLoadState('networkidle');

      // Open search, type, and press Enter
      await openPalette(page);
      const searchInput = paletteInput(page);
      await expect(searchInput).toBeVisible({ timeout: 3000 });
      await searchInput.fill(uniqueTitle);

      const result = page.locator('[data-search-item]').first();
      await expect(result).toBeVisible({ timeout: 10000 });

      // Navigate via Enter key
      await searchInput.press('Enter');

      await expect(page).toHaveURL(
        new RegExp(`/${nsSlug}/projects/${projKey}/items/${item.item_number}`),
        { timeout: 10000 },
      );

      // Page should fully load
      await expect(page.getByRole('heading', { name: uniqueTitle })).toBeVisible({ timeout: 15000 });
      await expect(page.getByText('Project not found')).not.toBeVisible({ timeout: 2000 });
    } finally {
      await api.migrateProject(request, adminToken, nsSlug, projKey, 'default').catch(() => {});
      await request.delete(`${BASE_URL}/api/v1/default/projects/${projKey}`, {
        headers: { Authorization: `Bearer ${adminToken}` },
      }).catch(() => {});
      await api.deleteNamespace(request, adminToken, nsSlug).catch(() => {});
    }
  });
});
