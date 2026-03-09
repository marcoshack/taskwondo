import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

/** Dismiss the welcome modal via API so it never appears. */
async function dismissWelcome(request: import('@playwright/test').APIRequestContext, token: string) {
  await api.setPreference(request, token, 'welcome_dismissed', true);
}

async function waitForTable(page: import('@playwright/test').Page) {
  await expect(page.getByRole('table')).toBeVisible({ timeout: 10000 });
}

async function openSelector(page: import('@playwright/test').Page) {
  const selectorBtn = page.locator('[class*="relative"]').filter({ has: page.locator('svg.lucide-chevron-down') }).first();
  await selectorBtn.click();
  await expect(page.getByPlaceholder('Search...')).toBeVisible({ timeout: 5000 });
}

/** Wait for the debounced view state save (500ms debounce + API round-trip). */
async function waitForSave(page: import('@playwright/test').Page) {
  await page.waitForTimeout(1500);
}

test.describe('Filter Persistence', () => {
  test.describe.configure({ mode: 'serial' });

  test('cleared filters persist after navigation', async ({ request, testUser, testProject, page }) => {
    await dismissWelcome(request, testUser.token);

    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Task for clear test',
      type: 'task',
    });
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Bug for clear test',
      type: 'bug',
    });

    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Bugs Only',
      filters: { type: ['bug'] },
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForTable(page);

    // Apply the saved search to set type=bug
    await openSelector(page);
    await page.getByRole('button', { name: 'Bugs Only' }).click();
    await expect(page).toHaveURL(/type=bug/, { timeout: 10000 });
    await waitForSave(page);

    // Clear all filters
    await page.locator('button[aria-label="Clear all filters"]').first().click();
    await expect(page).not.toHaveURL(/type=bug/, { timeout: 5000 });
    await waitForSave(page);

    // Navigate away
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await expect(page).toHaveURL(/settings/, { timeout: 5000 });

    // Navigate back to work items
    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForTable(page);

    // Verify type=bug filter is NOT in the URL (cleared state was persisted)
    await expect(page).not.toHaveURL(/type=bug/, { timeout: 10000 });

    // Both items should be visible (no type filter)
    await expect(page.getByRole('table').getByText('Task for clear test')).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('table').getByText('Bug for clear test')).toBeVisible({ timeout: 5000 });
  });

  test('filter state survives navigation to detail and back', async ({ request, testUser, testProject, page }) => {
    await dismissWelcome(request, testUser.token);

    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Task alpha',
      type: 'task',
    });
    const bug = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Bug beta',
      type: 'bug',
    });

    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Bug Filter',
      filters: { type: ['bug'] },
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForTable(page);
    await expect(page.getByRole('table').getByText('Bug beta')).toBeVisible({ timeout: 10000 });

    // Apply type=bug filter via saved search
    await openSelector(page);
    await page.getByRole('button', { name: 'Bug Filter' }).click();
    await expect(page).toHaveURL(/type=bug/, { timeout: 10000 });
    await waitForSave(page);

    // Navigate to work item detail
    await page.getByRole('table').getByText('Bug beta').click();
    await expect(page).toHaveURL(new RegExp(`items/${bug.item_number}`), { timeout: 10000 });

    // Navigate back to the list
    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForTable(page);

    // The type=bug filter should be restored
    await expect(page).toHaveURL(/type=bug/, { timeout: 10000 });

    // Only bug should be visible
    await expect(page.getByRole('table').getByText('Bug beta')).toBeVisible({ timeout: 10000 });
  });

  test('saved search selection survives navigation', async ({ request, testUser, testProject, page }) => {
    await dismissWelcome(request, testUser.token);

    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Item for search test',
      type: 'task',
    });

    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'My Task View',
      filters: { type: ['task'] },
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForTable(page);

    // Apply saved search
    await openSelector(page);
    await page.getByRole('button', { name: 'My Task View' }).click();
    await expect(page).toHaveURL(/type=task/, { timeout: 10000 });
    await expect(page.getByText('My Task View').first()).toBeVisible({ timeout: 5000 });
    await waitForSave(page);

    // Navigate away
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await expect(page).toHaveURL(/settings/, { timeout: 5000 });

    // Navigate back
    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForTable(page);

    // Saved search should still be selected
    await expect(page.getByText('My Task View').first()).toBeVisible({ timeout: 10000 });
    await expect(page).toHaveURL(/type=task/, { timeout: 5000 });
  });

  test('view mode and search text survive navigation', async ({ request, testUser, testProject, page }) => {
    await dismissWelcome(request, testUser.token);

    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Searchable item XYZ',
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForTable(page);
    await expect(page.getByRole('table').getByText('Searchable item XYZ')).toBeVisible({ timeout: 10000 });

    // Switch to board view
    await page.getByRole('button', { name: 'Board' }).click();
    await expect(page).toHaveURL(/view=board/, { timeout: 10000 });

    // Type in search box
    await page.getByPlaceholder(/Search/).first().fill('XYZ');
    await expect(page).toHaveURL(/q=XYZ/, { timeout: 10000 });
    await waitForSave(page);

    // Navigate away
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await expect(page).toHaveURL(/settings/, { timeout: 5000 });

    // Navigate back
    await page.goto(`/d/projects/${testProject.key}/items`);

    // Board view and search text should be restored
    await expect(page).toHaveURL(/view=board/, { timeout: 10000 });
    await expect(page).toHaveURL(/q=XYZ/, { timeout: 5000 });
  });

  test('unsaved changes to saved search survive navigation', async ({ request, testUser, testProject, page }) => {
    await dismissWelcome(request, testUser.token);

    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Task for unsaved',
      type: 'task',
    });

    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Base Search',
      filters: { type: ['task'] },
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForTable(page);

    // Apply saved search
    await openSelector(page);
    await page.getByRole('button', { name: 'Base Search' }).click();
    await expect(page).toHaveURL(/type=task/, { timeout: 10000 });

    // Modify the search (creates unsaved changes)
    await page.getByPlaceholder(/Search/).first().fill('unsaved');
    await expect(page).toHaveURL(/q=unsaved/, { timeout: 10000 });

    // Verify unsaved indicator appears
    const saveBtn = page.locator('button[aria-label="Save current filters"]').first();
    await expect(saveBtn.locator('.bg-amber-500')).toBeVisible({ timeout: 5000 });
    await waitForSave(page);

    // Navigate away
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await expect(page).toHaveURL(/settings/, { timeout: 5000 });

    // Navigate back
    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForTable(page);

    // Saved search should still be selected
    await expect(page.getByText('Base Search').first()).toBeVisible({ timeout: 10000 });

    // The modified search text should be restored
    await expect(page).toHaveURL(/q=unsaved/, { timeout: 5000 });

    // The unsaved changes indicator should still show
    const saveBtnAfter = page.locator('button[aria-label="Save current filters"]').first();
    await expect(saveBtnAfter.locator('.bg-amber-500')).toBeVisible({ timeout: 5000 });
  });

  test('filter state survives full page reload (tab close/reopen)', async ({ request, testUser, testProject, page }) => {
    await dismissWelcome(request, testUser.token);

    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Bug for reload',
      type: 'bug',
    });

    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Reload Test',
      filters: { type: ['bug'] },
      view_mode: 'board',
      shared: false,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForTable(page);

    // Apply saved search (sets type=bug, view=board)
    await openSelector(page);
    await page.getByRole('button', { name: 'Reload Test' }).click();
    await expect(page).toHaveURL(/type=bug/, { timeout: 10000 });
    await expect(page).toHaveURL(/view=board/, { timeout: 5000 });
    await waitForSave(page);

    // Hard reload clears all JS state
    await page.reload();

    // After reload, filters should be restored from server-side user settings
    await expect(page).toHaveURL(/type=bug/, { timeout: 10000 });
    await expect(page).toHaveURL(/view=board/, { timeout: 5000 });

    // Saved search name should still be visible
    await expect(page.getByText('Reload Test').first()).toBeVisible({ timeout: 10000 });
  });

  test('filter state survives logout and login', async ({ request, testUser, testProject, page }) => {
    await dismissWelcome(request, testUser.token);

    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Bug for auth',
      type: 'bug',
    });

    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Auth Test Search',
      filters: { type: ['bug'] },
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForTable(page);

    // Apply saved search
    await openSelector(page);
    await page.getByRole('button', { name: 'Auth Test Search' }).click();
    await expect(page).toHaveURL(/type=bug/, { timeout: 10000 });
    await waitForSave(page);

    // Logout: clear the token from localStorage and navigate to login
    await page.evaluate(() => localStorage.removeItem('taskwondo_token'));
    await page.goto('/login');
    await expect(page).toHaveURL(/login/, { timeout: 10000 });

    // Login again (Input component uses labels, not placeholders)
    await page.getByLabel(/email/i).fill(testUser.email);
    await page.getByLabel(/password/i).fill(testUser.password);
    await page.getByRole('button', { name: /sign in/i }).click();

    // Wait for redirect after login
    await expect(page).not.toHaveURL(/login/, { timeout: 10000 });

    // Navigate to work items
    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForTable(page);

    // Filter state should be restored from server-side settings
    await expect(page).toHaveURL(/type=bug/, { timeout: 10000 });

    // Saved search should still be selected
    await expect(page.getByText('Auth Test Search').first()).toBeVisible({ timeout: 10000 });
  });
});
