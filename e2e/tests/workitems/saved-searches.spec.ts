import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

/** Wait for the work items table to be visible. */
async function waitForTable(page: import('@playwright/test').Page) {
  await expect(page.getByRole('table')).toBeVisible({ timeout: 10000 });
}

/** Open the saved searches dropdown and wait for it to be visible. */
async function openSelector(page: import('@playwright/test').Page) {
  // Find the selector button by its chevron-down icon
  const selectorBtn = page.locator('[class*="relative"]').filter({ has: page.locator('svg.lucide-chevron-down') }).first();
  await selectorBtn.click();
  // Wait for the dropdown to appear
  await expect(page.getByPlaceholder('Search...')).toBeVisible({ timeout: 5000 });
}

test.describe('Saved Searches', () => {
  test('create personal saved search via API and see in selector', async ({ request, testUser, testProject, page }) => {
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Bug item',
      type: 'bug',
    });

    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'My Bugs',
      filters: { type: ['bug'] },
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Click the saved search selector
    await openSelector(page);

    // Verify sections and entry
    await expect(page.getByText('My Searches', { exact: true })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('My Bugs')).toBeVisible();
  });

  test('apply saved search filters', async ({ request, testUser, testProject, page }) => {
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Task item A',
      type: 'task',
    });
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Bug item B',
      type: 'bug',
    });

    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Only Bugs',
      filters: { type: ['bug'] },
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Wait for items to load in the table
    await waitForTable(page);
    await expect(page.getByRole('table').getByText('Bug item B')).toBeVisible({ timeout: 10000 });

    // Open selector and apply
    await openSelector(page);
    await page.getByRole('button', { name: 'Only Bugs' }).click();

    // Wait for URL to update with the type filter
    await expect(page).toHaveURL(/type=bug/, { timeout: 10000 });

    // After filtering, bug item should be visible
    await expect(page.getByRole('table').getByText('Bug item B')).toBeVisible({ timeout: 10000 });
  });

  test('save current filters via UI', async ({ request, testUser, testProject, page }) => {
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Test item for save',
      type: 'task',
    });

    await page.goto(`/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Wait for items to load in the table
    await waitForTable(page);
    await expect(page.getByRole('table').getByText('Test item for save')).toBeVisible({ timeout: 10000 });

    // Click save icon (desktop)
    await page.locator('button[aria-label="Save current filters"]').first().click();

    // Fill name in save modal (Modal doesn't use role="dialog", find by heading)
    await expect(page.getByRole('heading', { name: 'Save Search' })).toBeVisible({ timeout: 5000 });
    await page.getByPlaceholder('e.g. My open bugs').fill('Default View');
    await page.getByRole('button', { name: 'Save Search' }).click();

    // Modal should close
    await expect(page.getByRole('heading', { name: 'Save Search' })).not.toBeVisible({ timeout: 5000 });

    // The selector button should now show the active search name
    await expect(page.getByText('Default View').first()).toBeVisible({ timeout: 5000 });
  });

  test('clear filters resets to defaults', async ({ request, testUser, testProject, page }) => {
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Clearable item',
      type: 'bug',
    });

    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Bug Filter',
      filters: { type: ['bug'] },
      view_mode: 'board',
      shared: false,
    });

    await page.goto(`/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await waitForTable(page);
    await expect(page.getByRole('table').getByText('Clearable item')).toBeVisible({ timeout: 10000 });

    // Apply saved search
    await openSelector(page);
    await page.getByRole('button', { name: 'Bug Filter' }).click();

    // Verify board view is active (URL contains view=board)
    await expect(page).toHaveURL(/view=board/, { timeout: 10000 });

    // Click clear filters (desktop)
    await page.locator('button[aria-label="Clear all filters"]').first().click();

    // Verify board view is no longer active (cleared back to default list view)
    await expect(page).not.toHaveURL(/view=board/, { timeout: 10000 });
    // Also verify type filter was removed
    await expect(page).not.toHaveURL(/type=bug/, { timeout: 5000 });
  });

  test('delete saved search via API', async ({ request, testUser, testProject, page }) => {
    // Create via API
    const search = await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'To Delete',
      filters: {},
      view_mode: 'list',
      shared: false,
    });

    // Verify it exists
    const before = await api.listSavedSearches(request, testUser.token, testProject.key);
    expect(before.some((s) => s.name === 'To Delete')).toBe(true);

    // Delete via API
    await api.deleteSavedSearch(request, testUser.token, testProject.key, search.id);

    // Verify it's gone
    const after = await api.listSavedSearches(request, testUser.token, testProject.key);
    expect(after.some((s) => s.name === 'To Delete')).toBe(false);
  });

  test('shared saved search visible to members', async ({ request, testUser, testProject, page }) => {
    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Team Bugs',
      filters: { type: ['bug'] },
      view_mode: 'list',
      shared: true,
    });

    await page.goto(`/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    await openSelector(page);

    // Verify "Shared" section and entry
    await expect(page.getByText('Shared', { exact: true })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Team Bugs')).toBeVisible();
  });

  test('unsaved changes indicator shows on modified search', async ({ request, testUser, testProject, page }) => {
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Unsaved indicator item',
      type: 'task',
    });

    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Active Search',
      filters: {},
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await waitForTable(page);
    await expect(page.getByRole('table').getByText('Unsaved indicator item')).toBeVisible({ timeout: 10000 });

    // Apply saved search
    await openSelector(page);
    await page.getByRole('button', { name: 'Active Search' }).click();

    // Verify the selector shows the active search name (use first() as it may appear in both desktop and mobile)
    await expect(page.getByRole('button', { name: 'Active Search' }).first()).toBeVisible({ timeout: 5000 });

    // Type in search box to create unsaved changes
    await page.getByPlaceholder(/Search/).first().fill('modified');

    // The amber dot indicator should appear (on the save button)
    const saveBtn = page.locator('button[aria-label="Save current filters"]').first();
    const dot = saveBtn.locator('.bg-amber-500');
    await expect(dot).toBeVisible({ timeout: 5000 });
  });

  test('rename saved search via inline edit', async ({ request, testUser, testProject, page }) => {
    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Old Name',
      filters: {},
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Open selector
    await openSelector(page);
    await expect(page.getByText('Old Name')).toBeVisible({ timeout: 5000 });

    // Hover over the entry to make action icons visible, then click the pencil button
    const entry = page.locator('.group').filter({ hasText: 'Old Name' }).first();
    await entry.hover();
    // Click the pencil button (parent of the SVG icon)
    const pencilBtn = entry.locator('button').filter({ has: page.locator('svg.lucide-pencil') });
    await pencilBtn.click({ force: true });

    // Fill new name in rename modal (Modal doesn't use role="dialog", find by heading)
    await expect(page.getByRole('heading', { name: 'Rename Saved Search' })).toBeVisible({ timeout: 5000 });
    const renameInput = page.getByPlaceholder('e.g. My open bugs');
    await renameInput.clear();
    await renameInput.fill('New Name');
    await page.getByRole('button', { name: 'Save', exact: true }).click();

    // Verify rename
    await expect(page.getByRole('heading', { name: 'Rename Saved Search' })).not.toBeVisible({ timeout: 5000 });
    // Reopen selector to verify
    await openSelector(page);
    await expect(page.getByText('New Name')).toBeVisible({ timeout: 5000 });
  });
});
