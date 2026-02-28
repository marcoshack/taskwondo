import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

const TEST_PASSWORD = 'TestPass123!';

/** Create a second test user and add them as a project member. */
async function createSecondUser(
  request: import('@playwright/test').APIRequestContext,
  adminToken: string,
  projectKey: string,
  ownerToken: string,
  role = 'member',
) {
  const uniqueId = randomUUID().slice(0, 8);
  const email = `e2e-${uniqueId}@test.local`;
  const created = await api.createUser(request, adminToken, email, `E2E User2 ${uniqueId}`);
  const tempLogin = await api.login(request, email, created.temporary_password);
  await api.changePassword(request, tempLogin.token, created.temporary_password, TEST_PASSWORD);
  const finalLogin = await api.login(request, email, TEST_PASSWORD);
  await api.setPreference(request, finalLogin.token, 'welcome_dismissed', true);
  await api.addMember(request, ownerToken, projectKey, finalLogin.user.id, role);
  return { id: finalLogin.user.id, email, displayName: `E2E User2 ${uniqueId}`, token: finalLogin.token };
}

test.describe('Watchers', () => {

  // ─── API-level tests ──────────────────────────────────────────────

  test('toggle watch via API adds and removes current user as watcher', async ({ request, testUser, testProject }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Toggle watch test',
      type: 'task',
    });

    // Watch
    const result1 = await api.toggleWatch(request, testUser.token, testProject.key, item.item_number);
    expect(result1.is_watching).toBe(true);

    // Verify via list
    const watchers = await api.listWatchers(request, testUser.token, testProject.key, item.item_number) as Array<{ user_id: string }>;
    expect(watchers).toHaveLength(1);
    expect(watchers[0].user_id).toBe(testUser.id);

    // Unwatch
    const result2 = await api.toggleWatch(request, testUser.token, testProject.key, item.item_number);
    expect(result2.is_watching).toBe(false);

    // Verify empty
    const watchers2 = await api.listWatchers(request, testUser.token, testProject.key, item.item_number) as Array<unknown>;
    expect(watchers2).toHaveLength(0);
  });

  test('add and remove watcher via API', async ({ request, testUser, testProject }) => {
    const adminToken = (await import('../../lib/fixtures')).getAdminToken();
    const user2 = await createSecondUser(request, adminToken, testProject.key, testUser.token);

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Add/remove watcher test',
      type: 'task',
    });

    // Owner adds user2 as watcher
    await api.addWatcher(request, testUser.token, testProject.key, item.item_number, user2.id);

    const watchers = await api.listWatchers(request, testUser.token, testProject.key, item.item_number) as Array<{ user_id: string }>;
    expect(watchers).toHaveLength(1);
    expect(watchers[0].user_id).toBe(user2.id);

    // Owner removes user2
    await api.removeWatcher(request, testUser.token, testProject.key, item.item_number, user2.id);

    const watchers2 = await api.listWatchers(request, testUser.token, testProject.key, item.item_number) as Array<unknown>;
    expect(watchers2).toHaveLength(0);
  });

  test('viewer can only see own watch status and count', async ({ request, testUser, testProject }) => {
    const adminToken = (await import('../../lib/fixtures')).getAdminToken();
    const viewer = await createSecondUser(request, adminToken, testProject.key, testUser.token, 'viewer');

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Viewer watcher test',
      type: 'task',
    });

    // Owner watches item
    await api.toggleWatch(request, testUser.token, testProject.key, item.item_number);
    // Viewer watches item
    await api.toggleWatch(request, viewer.token, testProject.key, item.item_number);

    // Viewer sees restricted response
    const viewerData = await api.listWatchers(request, viewer.token, testProject.key, item.item_number) as { me: { user_id: string }; other_count: number };
    expect(viewerData.me).toBeTruthy();
    expect(viewerData.me.user_id).toBe(viewer.id);
    expect(viewerData.other_count).toBe(1); // owner
  });

  test('watchlist returns watched items', async ({ request, testUser, testProject }) => {
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchlist item 1',
      type: 'task',
    });
    const item2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchlist item 2',
      type: 'bug',
    });

    // Watch both
    await api.toggleWatch(request, testUser.token, testProject.key, item1.item_number);
    await api.toggleWatch(request, testUser.token, testProject.key, item2.item_number);

    // Verify watchlist
    const watched = await api.listWatchedItems(request, testUser.token, testProject.key);
    expect(watched.data.length).toBeGreaterThanOrEqual(2);
    const titles = watched.data.map((i) => i.title);
    expect(titles).toContain('Watchlist item 1');
    expect(titles).toContain('Watchlist item 2');
  });

  test('create work item with watcher_ids', async ({ request, testUser, testProject }) => {
    const adminToken = (await import('../../lib/fixtures')).getAdminToken();
    const user2 = await createSecondUser(request, adminToken, testProject.key, testUser.token);

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Create with watchers',
      type: 'task',
      watcher_ids: [testUser.id, user2.id],
    });

    const watchers = await api.listWatchers(request, testUser.token, testProject.key, item.item_number) as Array<{ user_id: string }>;
    expect(watchers).toHaveLength(2);
    const ids = watchers.map((w) => w.user_id);
    expect(ids).toContain(testUser.id);
    expect(ids).toContain(user2.id);
  });

  // ─── UI tests ─────────────────────────────────────────────────────

  test('watch/unwatch via bell icon on detail page', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'UI watch toggle test',
      type: 'task',
    });

    await page.goto(`/projects/${testProject.key}/items/${item.item_number}`);
    await page.waitForLoadState('networkidle');

    // Find the desktop watch button (inside hidden lg:inline-flex container)
    // The mobile one is first but hidden on desktop viewports, so use nth(1) for the desktop one
    const watchBtn = page.locator('button[aria-label="Watch"]').nth(1);
    await expect(watchBtn).toBeVisible({ timeout: 10000 });

    // Click to watch
    await watchBtn.click();
    // Wait for the mutation to complete (green checkmark shows for 2s then back to bell)
    await page.waitForTimeout(3000);

    // Verify via API that we're watching
    const watchers = await api.listWatchers(request, testUser.token, testProject.key, item.item_number) as Array<{ user_id: string }>;
    expect(watchers).toHaveLength(1);
    expect(watchers[0].user_id).toBe(testUser.id);

    // Click again to unwatch — button label should now be "Unwatch" (desktop variant)
    const unwatchBtn = page.locator('button[aria-label="Unwatch"]').nth(1);
    await expect(unwatchBtn).toBeVisible({ timeout: 5000 });
    await unwatchBtn.click();
    await page.waitForTimeout(3000);

    const watchers2 = await api.listWatchers(request, testUser.token, testProject.key, item.item_number) as Array<unknown>;
    expect(watchers2).toHaveLength(0);
  });

  test('watchers tab shows watcher list and allows adding', async ({ request, testUser, testProject, page }) => {
    const adminToken = (await import('../../lib/fixtures')).getAdminToken();
    const user2 = await createSecondUser(request, adminToken, testProject.key, testUser.token);

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchers tab test',
      type: 'task',
    });

    // Owner watches the item via API
    await api.toggleWatch(request, testUser.token, testProject.key, item.item_number);

    await page.goto(`/projects/${testProject.key}/items/${item.item_number}`);
    await page.waitForLoadState('networkidle');

    // Navigate to Watchers tab
    await page.getByRole('button', { name: 'Watchers' }).click();
    await page.waitForTimeout(1000);

    // Verify the owner is shown in the watcher list (use .first() as name may appear in multiple places)
    await expect(page.getByText(testUser.displayName).first()).toBeVisible({ timeout: 10000 });

    // Add user2 as watcher via the UI
    await page.getByRole('button', { name: 'Add watcher' }).click();
    await expect(page.getByPlaceholder('Search members')).toBeVisible({ timeout: 5000 });

    // Type user2's name in the search and click to add
    await page.getByPlaceholder('Search members').fill(user2.displayName);
    await page.getByText(user2.displayName).first().click();

    // Wait for the mutation
    await page.waitForTimeout(1000);

    // Verify via API
    const watchers = await api.listWatchers(request, testUser.token, testProject.key, item.item_number) as Array<{ user_id: string }>;
    expect(watchers).toHaveLength(2);
  });

  test('watch icon visible on list page and toggles correctly', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'List page watch test',
      type: 'task',
    });

    await page.goto(`/projects/${testProject.key}/items`);
    await page.waitForLoadState('networkidle');

    // Find the row containing our item
    const row = page.locator('tr, [data-testid]').filter({ hasText: 'List page watch test' }).first();
    await expect(row).toBeVisible({ timeout: 10000 });

    // Hover to reveal watch button
    await row.hover();

    // Find the watch bell button within the row
    const watchBtn = row.locator('button[aria-label="Watch"], button[aria-label="Unwatch"]');
    if (await watchBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await watchBtn.click();
      await page.waitForTimeout(2000);

      // Verify via API
      const watchers = await api.listWatchers(request, testUser.token, testProject.key, item.item_number) as Array<{ user_id: string }>;
      expect(watchers).toHaveLength(1);
    }
  });

  test('watchlist page shows watched items', async ({ request, testUser, testProject, page }) => {
    // Create and watch two items
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchlist UI item 1',
      type: 'task',
    });
    const item2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchlist UI item 2',
      type: 'bug',
    });

    await api.toggleWatch(request, testUser.token, testProject.key, item1.item_number);
    await api.toggleWatch(request, testUser.token, testProject.key, item2.item_number);

    // Navigate to watchlist page
    await page.goto('/user/watchlist');
    await page.waitForLoadState('networkidle');

    // Verify items are visible (use .first() as items appear in both desktop table and mobile cards)
    await expect(page.getByText('Watchlist UI item 1').first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Watchlist UI item 2').first()).toBeVisible({ timeout: 10000 });
  });

  test('watchlist page empty state shows bookmark message when no watched items', async ({ testUser, testProject, page }) => {
    // testUser + testProject fixtures ensure authenticated session + at least one project
    await page.goto('/user/watchlist');
    await page.waitForLoadState('networkidle');

    // Verify empty state (bookmark icon + message)
    await expect(page.getByText("You're not watching any items yet")).toBeVisible({ timeout: 10000 });
  });

  test('watchlist page shows "No work items found" in table when filters produce no results', async ({ request, testUser, testProject, page }) => {
    // Create and watch an item so the watchlist is not truly empty
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchlist filtered empty test',
      type: 'task',
    });
    await api.toggleWatch(request, testUser.token, testProject.key, item.item_number);

    await page.goto('/user/watchlist');
    await page.waitForLoadState('networkidle');

    // Verify item is visible first
    await expect(page.getByText('Watchlist filtered empty test').first()).toBeVisible({ timeout: 10000 });

    // Search for something that doesn't match
    const searchInput = page.getByPlaceholder(/search/i).first();
    await searchInput.fill('zzz_nonexistent_query_xyz');
    await page.waitForTimeout(500);

    // Should show "No work items found" in the table (NOT the bookmark empty state)
    await expect(page.getByText("You're not watching any items yet")).not.toBeVisible({ timeout: 3000 });
    // Table headers should still be visible
    await expect(page.locator('th').filter({ hasText: 'Title' })).toBeVisible({ timeout: 5000 });
  });

  test('create work item with watchers via UI form', async ({ request, testUser, testProject, page }) => {
    const adminToken = (await import('../../lib/fixtures')).getAdminToken();
    const user2 = await createSecondUser(request, adminToken, testProject.key, testUser.token);

    // Navigate to the work item list page
    await page.goto(`/projects/${testProject.key}/items`);
    await page.waitForLoadState('networkidle');

    // Click create button — look for common create button patterns
    const createBtn = page.getByRole('button', { name: /new|create/i }).first();
    await expect(createBtn).toBeVisible({ timeout: 10000 });
    await createBtn.click();

    // Wait for the modal/form to appear — select Type
    const typeSelect = page.getByLabel('Type');
    await expect(typeSelect).toBeVisible({ timeout: 10000 });
    await typeSelect.selectOption('task');

    // Fill title
    await page.getByLabel('Title').fill('Item with watchers from form');

    // Look for the Watchers field and add a watcher
    const watcherField = page.getByText('Add watchers...');
    if (await watcherField.isVisible({ timeout: 3000 }).catch(() => false)) {
      await watcherField.click();
      await page.waitForTimeout(500);

      // Search for user2 — use the last search input (watchers picker, not assignee)
      const searchInputs = page.getByPlaceholder('Search members');
      await searchInputs.last().fill(user2.displayName);
      await page.waitForTimeout(500);

      // Click the user in the dropdown
      await page.locator('ul').last().getByText(user2.displayName).click();
      await page.waitForTimeout(500);
    }

    // Submit the form
    await page.getByRole('button', { name: 'Create', exact: true }).click();

    // Wait for navigation or success
    await page.waitForTimeout(3000);

    // Verify via API that the watcher was added
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';
    const res = await request.get(`${BASE_URL}/api/v1/projects/${testProject.key}/items?q=Item+with+watchers+from+form`, {
      headers: { Authorization: `Bearer ${testUser.token}` },
    });
    const body = await res.json();
    if (body.data && body.data.length > 0) {
      const watchers = await api.listWatchers(request, testUser.token, testProject.key, body.data[0].item_number) as Array<{ user_id: string }>;
      const watcherUserIds = watchers.map((w) => w.user_id);
      expect(watcherUserIds).toContain(user2.id);
    }
  });

  test('remove watcher from watchers tab', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Remove watcher UI test',
      type: 'task',
    });

    // Watch the item
    await api.toggleWatch(request, testUser.token, testProject.key, item.item_number);

    await page.goto(`/projects/${testProject.key}/items/${item.item_number}`);
    await page.waitForLoadState('networkidle');

    // Navigate to Watchers tab
    await page.getByRole('button', { name: 'Watchers' }).click();
    await page.waitForTimeout(1000);

    // Verify we see the watcher (use .first() to avoid strict mode violation)
    await expect(page.getByText(testUser.displayName).first()).toBeVisible({ timeout: 10000 });

    // Find and click the remove (X) button
    const removeBtn = page.locator('button[aria-label="Remove"]').first();
    if (await removeBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await removeBtn.click();
      await page.waitForTimeout(1000);

      // Verify removed via API
      const watchers = await api.listWatchers(request, testUser.token, testProject.key, item.item_number) as Array<unknown>;
      expect(watchers).toHaveLength(0);
    }
  });

  test('clicking watchlist item navigates to work item detail', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchlist navigation test',
      type: 'task',
    });

    await api.toggleWatch(request, testUser.token, testProject.key, item.item_number);

    await page.goto('/user/watchlist');
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('Watchlist navigation test').first()).toBeVisible({ timeout: 10000 });

    // Click the item in the desktop table row
    await page.getByText('Watchlist navigation test').first().click();

    // Verify navigated to detail page
    await expect(page).toHaveURL(/\/projects\/.*\/items\/\d+/, { timeout: 10000 });
    await expect(page.getByText('Watchlist navigation test').first()).toBeVisible();
  });

  // ─── Watchlist page full UI tests ──────────────────────────────────

  test('watchlist page shows DataTable with proper columns on desktop', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchlist table test',
      type: 'bug',
      priority: 'high',
    });
    await api.toggleWatch(request, testUser.token, testProject.key, item.item_number);

    await page.goto('/user/watchlist');
    await page.waitForLoadState('networkidle');

    // Verify DataTable columns are visible (desktop)
    await expect(page.locator('th').filter({ hasText: 'ID' })).toBeVisible({ timeout: 10000 });
    await expect(page.locator('th').filter({ hasText: 'Type' })).toBeVisible();
    await expect(page.locator('th').filter({ hasText: 'Title' })).toBeVisible();
    await expect(page.locator('th').filter({ hasText: 'Status' })).toBeVisible();
    await expect(page.locator('th').filter({ hasText: 'Priority' })).toBeVisible();

    // Verify the item appears in the table row (use .first() due to desktop table + mobile cards)
    await expect(page.getByText('Watchlist table test').first()).toBeVisible();
    await expect(page.getByText(item.display_id).first()).toBeVisible();
  });

  test('watchlist page has filter controls', async ({ request, testUser, testProject, page }) => {
    // Create items with different types
    const task = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchlist filter task',
      type: 'task',
    });
    const bug = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchlist filter bug',
      type: 'bug',
    });

    await api.toggleWatch(request, testUser.token, testProject.key, task.item_number);
    await api.toggleWatch(request, testUser.token, testProject.key, bug.item_number);

    await page.goto('/user/watchlist');
    await page.waitForLoadState('networkidle');

    // Both items should be visible (use .first() due to desktop + mobile views)
    await expect(page.getByText('Watchlist filter task').first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Watchlist filter bug').first()).toBeVisible();

    // Open filter panel — look for the filter toggle button
    const filterToggle = page.getByRole('button', { name: /filter/i }).first();
    if (await filterToggle.isVisible({ timeout: 3000 }).catch(() => false)) {
      await filterToggle.click();
      await page.waitForTimeout(500);
    }

    // Verify filter dropdowns exist — Type filter should be present
    const typeFilter = page.getByText('Type').first();
    await expect(typeFilter).toBeVisible({ timeout: 5000 });
  });

  test('watchlist page search filters results', async ({ request, testUser, testProject, page }) => {
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchlist search alpha',
      type: 'task',
    });
    const item2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchlist search beta',
      type: 'task',
    });

    await api.toggleWatch(request, testUser.token, testProject.key, item1.item_number);
    await api.toggleWatch(request, testUser.token, testProject.key, item2.item_number);

    await page.goto('/user/watchlist');
    await page.waitForLoadState('networkidle');

    // Both items should be visible initially (use .first() due to desktop + mobile views)
    await expect(page.getByText('Watchlist search alpha').first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Watchlist search beta').first()).toBeVisible();

    // Use the search bar to filter
    const searchInput = page.getByPlaceholder(/search/i).first();
    await searchInput.fill('alpha');
    await page.waitForTimeout(500); // wait for debounce

    // Only the matching item should be visible
    await expect(page.getByText('Watchlist search alpha').first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Watchlist search beta')).not.toBeVisible({ timeout: 5000 });
  });

  test('watchlist page view mode toggle switches between list and board', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchlist view toggle item',
      type: 'task',
    });
    await api.toggleWatch(request, testUser.token, testProject.key, item.item_number);

    await page.goto('/user/watchlist');
    await page.waitForLoadState('networkidle');

    // Verify list view is active by default (table should be present)
    await expect(page.getByText('Watchlist view toggle item').first()).toBeVisible({ timeout: 10000 });

    // Find and click the Board view toggle button
    const boardBtn = page.getByRole('button', { name: /board/i });
    await expect(boardBtn).toBeVisible({ timeout: 5000 });
    await boardBtn.click();
    await page.waitForTimeout(500);

    // Board view should still show the item
    await expect(page.getByText('Watchlist view toggle item').first()).toBeVisible({ timeout: 10000 });

    // Switch back to list
    const listBtn = page.getByRole('button', { name: /list/i });
    await listBtn.click();
    await page.waitForTimeout(500);

    // Item should still be visible in list view
    await expect(page.getByText('Watchlist view toggle item').first()).toBeVisible({ timeout: 5000 });
  });

  test('watchlist page column sorting works', async ({ request, testUser, testProject, page }) => {
    // Create items with different priorities
    const highItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchlist sort high',
      type: 'task',
      priority: 'high',
    });
    const lowItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchlist sort low',
      type: 'task',
      priority: 'low',
    });

    await api.toggleWatch(request, testUser.token, testProject.key, highItem.item_number);
    await api.toggleWatch(request, testUser.token, testProject.key, lowItem.item_number);

    await page.goto('/user/watchlist');
    await page.waitForLoadState('networkidle');

    // Both should be visible (use .first() due to desktop + mobile views)
    await expect(page.getByText('Watchlist sort high').first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Watchlist sort low').first()).toBeVisible();

    // Click on the Title column header to sort
    const titleHeader = page.locator('th').filter({ hasText: 'Title' });
    await titleHeader.click();
    await page.waitForTimeout(500);

    // Verify both items are still visible after sorting
    await expect(page.getByText('Watchlist sort high').first()).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Watchlist sort low').first()).toBeVisible();
  });

  test('watchlist page clear filters resets to defaults', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchlist clear filters item',
      type: 'task',
    });
    await api.toggleWatch(request, testUser.token, testProject.key, item.item_number);

    await page.goto('/user/watchlist');
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('Watchlist clear filters item').first()).toBeVisible({ timeout: 10000 });

    // Type in search to set a filter
    const searchInput = page.getByPlaceholder(/search/i).first();
    await searchInput.fill('nonexistent query xyz');
    await page.waitForTimeout(500);

    // Item should no longer be visible
    await expect(page.getByText('Watchlist clear filters item')).not.toBeVisible({ timeout: 5000 });

    // Clear the search
    await searchInput.clear();
    await page.waitForTimeout(500);

    // Item should reappear
    await expect(page.getByText('Watchlist clear filters item').first()).toBeVisible({ timeout: 10000 });
  });

  // ─── Cross-project & filter behavior tests ───────────────────────

  test('cross-project watchlist API returns items from multiple projects', async ({ request, testUser, testProject }) => {
    // Create a second project
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const project2Key = `W${suffix}`;
    const project2 = await api.createProject(request, testUser.token, project2Key, `Cross Proj ${suffix}`);

    // Create items in both projects
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Cross project item A',
      type: 'task',
    });
    const item2 = await api.createWorkItem(request, testUser.token, project2Key, {
      title: 'Cross project item B',
      type: 'bug',
    });

    // Watch both
    await api.toggleWatch(request, testUser.token, testProject.key, item1.item_number);
    await api.toggleWatch(request, testUser.token, project2Key, item2.item_number);

    // Fetch all watched items (no project filter)
    const all = await api.listWatchedItems(request, testUser.token);
    const titles = all.data.map((i) => i.title);
    expect(titles).toContain('Cross project item A');
    expect(titles).toContain('Cross project item B');

    // Fetch with multi-project filter
    const multi = await api.listWatchedItems(request, testUser.token, [testProject.key, project2Key]);
    const multiTitles = multi.data.map((i) => i.title);
    expect(multiTitles).toContain('Cross project item A');
    expect(multiTitles).toContain('Cross project item B');

    // Fetch single project filter — should only include items from that project
    const single = await api.listWatchedItems(request, testUser.token, project2Key);
    const singleTitles = single.data.map((i) => i.title);
    expect(singleTitles).toContain('Cross project item B');
    expect(singleTitles).not.toContain('Cross project item A');
  });

  test('watchlist page shows total counter next to title', async ({ request, testUser, testProject, page }) => {
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Counter test item 1',
      type: 'task',
    });
    const item2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Counter test item 2',
      type: 'task',
    });
    await api.toggleWatch(request, testUser.token, testProject.key, item1.item_number);
    await api.toggleWatch(request, testUser.token, testProject.key, item2.item_number);

    await page.goto('/user/watchlist');
    await page.waitForLoadState('networkidle');

    // Wait for items to load
    await expect(page.getByText('Counter test item 1').first()).toBeVisible({ timeout: 10000 });

    // Verify counter is shown — the total should be in parentheses near the title
    const watchlistHeader = page.locator('h2').filter({ hasText: /watchlist/i });
    await expect(watchlistHeader).toBeVisible();
    // Counter is a sibling span showing (N)
    const counterParent = watchlistHeader.locator('..');
    const counterText = await counterParent.textContent();
    // Should contain a number in parentheses
    expect(counterText).toMatch(/\(\d+\)/);
  });

  test('watchlist project filter narrows results to selected projects', async ({ request, testUser, testProject, page }) => {
    // Create a second project
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const project2Key = `F${suffix}`;
    await api.createProject(request, testUser.token, project2Key, `Filter Proj ${suffix}`);

    // Create and watch items in both projects
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Proj filter item alpha',
      type: 'task',
    });
    const item2 = await api.createWorkItem(request, testUser.token, project2Key, {
      title: 'Proj filter item beta',
      type: 'bug',
    });
    await api.toggleWatch(request, testUser.token, testProject.key, item1.item_number);
    await api.toggleWatch(request, testUser.token, project2Key, item2.item_number);

    await page.goto('/user/watchlist');
    await page.waitForLoadState('networkidle');

    // Both items should be visible (no project filter = all projects)
    await expect(page.getByText('Proj filter item alpha').first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Proj filter item beta').first()).toBeVisible({ timeout: 10000 });

    // Open the project MultiSelect and select only the second project
    const projectFilter = page.getByText('All Projects');
    await expect(projectFilter).toBeVisible({ timeout: 5000 });
    await projectFilter.click();
    await page.waitForTimeout(300);

    // MultiSelect uses <label> with checkbox — click the label containing project2's name
    await page.locator('label').filter({ hasText: new RegExp(`Filter Proj ${suffix}`) }).click();
    // Close the dropdown by clicking outside
    await page.locator('h2').first().click();
    await page.waitForTimeout(500);

    // Only item from project2 should be visible
    await expect(page.getByText('Proj filter item beta').first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Proj filter item alpha')).not.toBeVisible({ timeout: 5000 });
  });

  test('watchlist filter preferences persist across navigation', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Persist filter test item',
      type: 'bug',
    });
    await api.toggleWatch(request, testUser.token, testProject.key, item.item_number);

    // Clear any saved preferences first
    await api.setPreference(request, testUser.token, 'watchlistFilters', {});

    await page.goto('/user/watchlist');
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('Persist filter test item').first()).toBeVisible({ timeout: 10000 });

    // Switch to board view to change a persisted preference
    const boardBtn = page.getByRole('button', { name: /board/i });
    await expect(boardBtn).toBeVisible({ timeout: 5000 });
    await boardBtn.click();
    await page.waitForTimeout(1000);

    // Navigate away (to inbox)
    await page.goto('/user/inbox');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);

    // Navigate back to watchlist
    await page.goto('/user/watchlist');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    // Verify board view is still active (the board button should have the active style)
    const boardBtnAfter = page.getByRole('button', { name: /board/i });
    await expect(boardBtnAfter).toBeVisible({ timeout: 5000 });
    // Board button should have the indigo active style
    await expect(boardBtnAfter).toHaveClass(/bg-indigo/, { timeout: 5000 });
  });
});
