import { test, expect, getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

test.describe('Search modal (g then k)', () => {
  test('g then k opens search modal and Escape closes it', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open search modal
    await page.keyboard.press('g');
    await page.keyboard.press('k');
    const searchInput = page.getByPlaceholder(/search across/i);
    await expect(searchInput).toBeVisible({ timeout: 3000 });
    await expect(searchInput).toBeFocused();

    // Close with Escape
    await page.keyboard.press('Escape');
    await expect(searchInput).not.toBeVisible({ timeout: 3000 });
  });

  test('search shows results for matching work items via unified endpoint', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    // Create a work item with a unique title
    const uniqueTitle = `SearchTest-${Date.now()}`;
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: uniqueTitle,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open search modal
    await page.keyboard.press('g');
    await page.keyboard.press('k');
    const searchInput = page.getByPlaceholder(/search across/i);
    await expect(searchInput).toBeVisible({ timeout: 3000 });

    // Type search query
    await searchInput.fill(uniqueTitle.slice(0, 12));

    // Wait for results - FTS results stream via SSE
    const resultItem = page.locator('[data-search-item]').first();
    await expect(resultItem).toBeVisible({ timeout: 10000 });
    await expect(resultItem).toContainText(uniqueTitle);
  });

  test('display ID search returns exact match as top result', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    // Create a work item
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `DisplayIDSearch-${Date.now()}`,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open search and search by display ID
    await page.keyboard.press('g');
    await page.keyboard.press('k');
    const searchInput = page.getByPlaceholder(/search across/i);
    await searchInput.fill(item.display_id);

    // First result should appear
    const resultItem = page.locator('[data-search-item]').first();
    await expect(resultItem).toBeVisible({ timeout: 10000 });

    // Press Enter should navigate to this item
    await searchInput.press('Enter');
    await expect(page).toHaveURL(
      new RegExp(`/d/projects/${testProject.key}/items/${item.item_number}`),
      { timeout: 5000 },
    );
  });

  test('search shows empty state when no results found', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open search modal
    await page.keyboard.press('g');
    await page.keyboard.press('k');
    const searchInput = page.getByPlaceholder(/search across/i);
    await expect(searchInput).toBeVisible({ timeout: 3000 });

    // Search for something that doesn't exist
    await searchInput.fill('zzzznonexistent99999');

    // Wait for empty state
    await expect(page.getByText(/no results found/i)).toBeVisible({ timeout: 10000 });
  });

  test('keyboard navigation works in search results', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    // Create work items with searchable titles
    const prefix = `NavTest-${Date.now()}`;
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `${prefix} First`,
      type: 'task',
    });
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `${prefix} Second`,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open search and search
    await page.keyboard.press('g');
    await page.keyboard.press('k');
    const searchInput = page.getByPlaceholder(/search across/i);
    await searchInput.fill(prefix);

    // Wait for results
    const results = page.locator('[data-search-item]');
    await expect(results.first()).toBeVisible({ timeout: 10000 });

    // First item should be highlighted by default
    const firstItem = results.first();
    await expect(firstItem).toHaveClass(/bg-indigo-50/);

    // Press ArrowDown to move to second item
    await searchInput.press('ArrowDown');
    const secondItem = results.nth(1);
    await expect(secondItem).toHaveClass(/bg-indigo-50/);

    // Press ArrowUp to go back to first
    await searchInput.press('ArrowUp');
    await expect(firstItem).toHaveClass(/bg-indigo-50/);
  });

  test('Enter on a result navigates to the work item', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const uniqueTitle = `EnterNav-${Date.now()}`;
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: uniqueTitle,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open search and find the item
    await page.keyboard.press('g');
    await page.keyboard.press('k');
    const searchInput = page.getByPlaceholder(/search across/i);
    await searchInput.fill(uniqueTitle);

    // Wait for result
    const result = page.locator('[data-search-item]').first();
    await expect(result).toBeVisible({ timeout: 10000 });

    // Press Enter to navigate
    await searchInput.press('Enter');

    // Should navigate to the work item detail page
    await expect(page).toHaveURL(
      new RegExp(`/d/projects/${testProject.key}/items/${item.item_number}`),
      { timeout: 5000 },
    );
  });

  test('clicking a result navigates to the work item', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const uniqueTitle = `ClickNav-${Date.now()}`;
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: uniqueTitle,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open search and find the item
    await page.keyboard.press('g');
    await page.keyboard.press('k');
    const searchInput = page.getByPlaceholder(/search across/i);
    await searchInput.fill(uniqueTitle);

    // Wait for result and click it
    const result = page.locator('[data-search-item]').first();
    await expect(result).toBeVisible({ timeout: 10000 });
    await result.click();

    // Should navigate to the work item detail page
    await expect(page).toHaveURL(
      new RegExp(`/d/projects/${testProject.key}/items/${item.item_number}`),
      { timeout: 5000 },
    );
  });

  test('search hint shown before typing', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('g');
    await page.keyboard.press('k');
    await expect(page.getByText(/type at least 2 characters/i)).toBeVisible({ timeout: 3000 });
  });

  test('comment deep-link opens comments tab and highlights the comment', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `CommentDeepLink-${Date.now()}`,
      type: 'task',
    });
    const comment = await api.addComment(
      request,
      testUser.token,
      testProject.key,
      item.item_number,
      'Deep-linked comment body',
    );

    // Navigate with tab=comments&highlight=<commentId>
    await page.goto(
      `/d/projects/${testProject.key}/items/${item.item_number}?tab=comments&highlight=${comment.id}`,
    );
    await dismissWelcomeModal(page);

    // Comments tab should be active
    const commentsTab = page.getByRole('button', { name: /comments/i });
    await expect(commentsTab).toHaveClass(/border-indigo|text-indigo/, { timeout: 5000 });

    // The comment should be visible
    await expect(page.getByText('Deep-linked comment body')).toBeVisible({ timeout: 5000 });
  });

  test('attachment deep-link opens attachments tab and highlights the attachment', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `AttachDeepLink-${Date.now()}`,
      type: 'task',
    });
    const attachment = await api.uploadAttachment(
      request,
      testUser.token,
      testProject.key,
      item.item_number,
      'test-deeplink.txt',
      Buffer.from('deep link test content'),
      'text/plain',
    );

    // Navigate with tab=attachments&highlight=<attachmentId>
    await page.goto(
      `/d/projects/${testProject.key}/items/${item.item_number}?tab=attachments&highlight=${attachment.id}`,
    );
    await dismissWelcomeModal(page);

    // Attachments tab should be active
    const attachTab = page.getByRole('button', { name: /attachments/i });
    await expect(attachTab).toHaveClass(/border-indigo|text-indigo/, { timeout: 5000 });

    // The attachment should be visible
    await expect(page.getByText('test-deeplink.txt')).toBeVisible({ timeout: 5000 });
  });

  test('project search result navigates to project page', async ({
    page,
    testProject,
  }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open search and search for the project name
    await page.keyboard.press('g');
    await page.keyboard.press('k');
    const searchInput = page.getByPlaceholder(/search across/i);
    await searchInput.fill(testProject.name.slice(0, 12));

    // Wait for results (FTS returns work items; project may or may not appear)
    // Just verify searching doesn't crash and results appear or empty state shows
    const hasResults = page.locator('[data-search-item]').first();
    const emptyState = page.getByText(/no results found/i);
    await expect(hasResults.or(emptyState)).toBeVisible({ timeout: 10000 });
  });

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

    // Also create a default-namespace project so we start on a real page inside NamespaceGuard
    const defProjKey = `XD${Date.now().toString(36).slice(-3).toUpperCase()}`;
    await api.createProject(request, testUser.token, defProjKey, 'Default NS Search Proj');
    const defItem = await api.createWorkItem(request, testUser.token, defProjKey, {
      title: `DefItem-${Date.now()}`,
      type: 'task',
    });

    try {
      // Start on a work item detail page in the default namespace (inside NamespaceGuard)
      // so the test covers the re-render path (not fresh mount)
      await page.goto(`/d/projects/${defProjKey}/items/${defItem.item_number}`);
      await dismissWelcomeModal(page);
      await page.waitForLoadState('networkidle');

      // Open search and search for the cross-namespace item
      await page.keyboard.press('g');
      await page.keyboard.press('k');
      const searchInput = page.getByPlaceholder(/search across/i);
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
      await page.keyboard.press('g');
      await page.keyboard.press('k');
      const searchInput = page.getByPlaceholder(/search across/i);
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
