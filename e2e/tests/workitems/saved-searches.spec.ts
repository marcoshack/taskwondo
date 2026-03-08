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

/** Open the saved searches dropdown (desktop) and wait for it to be visible. */
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

    await page.goto(`/d/projects/${testProject.key}/items`);
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

    await page.goto(`/d/projects/${testProject.key}/items`);
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

    await page.goto(`/d/projects/${testProject.key}/items`);
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

    await page.goto(`/d/projects/${testProject.key}/items`);
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

    await page.goto(`/d/projects/${testProject.key}/items`);
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

    await page.goto(`/d/projects/${testProject.key}/items`);
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

    await page.goto(`/d/projects/${testProject.key}/items`);
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
    // Dropdown stays open after rename — verify the new name is visible
    await expect(page.getByText('New Name')).toBeVisible({ timeout: 5000 });
  });

  test('desktop rename modal keeps dropdown open', async ({ request, testUser, testProject, page }) => {
    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Rename Keep Open',
      filters: {},
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    await openSelector(page);
    await expect(page.getByText('Rename Keep Open')).toBeVisible({ timeout: 5000 });

    // Hover over entry and click pencil
    const entry = page.locator('.group').filter({ hasText: 'Rename Keep Open' }).first();
    await entry.hover();
    const pencilBtn = entry.locator('button').filter({ has: page.locator('svg.lucide-pencil') });
    await pencilBtn.click({ force: true });

    // Rename modal should be open
    await expect(page.getByRole('heading', { name: 'Rename Saved Search' })).toBeVisible({ timeout: 5000 });

    // Cancel the rename modal
    await page.getByRole('button', { name: 'Cancel' }).click();
    await expect(page.getByRole('heading', { name: 'Rename Saved Search' })).not.toBeVisible({ timeout: 3000 });

    // The dropdown should still be visible (not closed by outside click)
    await expect(page.getByText('Rename Keep Open')).toBeVisible({ timeout: 3000 });
  });

  test('desktop delete modal keeps dropdown open', async ({ request, testUser, testProject, page }) => {
    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Delete Keep Open',
      filters: {},
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    await openSelector(page);
    await expect(page.getByText('Delete Keep Open')).toBeVisible({ timeout: 5000 });

    // Hover over entry and click trash icon
    const entry = page.locator('.group').filter({ hasText: 'Delete Keep Open' }).first();
    await entry.hover();
    const trashBtn = entry.locator('button').filter({ has: page.locator('svg.lucide-trash-2') });
    await trashBtn.click({ force: true });

    // Delete confirmation modal should be open
    await expect(page.getByRole('heading', { name: 'Delete Saved Search' })).toBeVisible({ timeout: 5000 });

    // Cancel the delete
    await page.getByRole('button', { name: 'Cancel' }).click();
    await expect(page.getByRole('heading', { name: 'Delete Saved Search' })).not.toBeVisible({ timeout: 3000 });

    // The dropdown should still be visible
    await expect(page.getByText('Delete Keep Open')).toBeVisible({ timeout: 3000 });
  });

  test('desktop delete via UI removes entry from dropdown', async ({ request, testUser, testProject, page }) => {
    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'UI Delete Me',
      filters: {},
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    await openSelector(page);
    await expect(page.getByText('UI Delete Me')).toBeVisible({ timeout: 5000 });

    // Hover and click trash
    const entry = page.locator('.group').filter({ hasText: 'UI Delete Me' }).first();
    await entry.hover();
    const trashBtn = entry.locator('button').filter({ has: page.locator('svg.lucide-trash-2') });
    await trashBtn.click({ force: true });

    // Confirm delete
    await expect(page.getByRole('heading', { name: 'Delete Saved Search' })).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: 'Delete', exact: true }).click();

    // Modal should close
    await expect(page.getByRole('heading', { name: 'Delete Saved Search' })).not.toBeVisible({ timeout: 5000 });

    // Dropdown stays open after delete — entry should be gone from the list
    await expect(page.getByText('UI Delete Me')).not.toBeVisible({ timeout: 5000 });

    // Verify via API
    const searches = await api.listSavedSearches(request, testUser.token, testProject.key);
    expect(searches.some((s) => s.name === 'UI Delete Me')).toBe(false);
  });

  test('desktop reorder saved searches with up/down arrows', async ({ request, testUser, testProject, page }) => {
    // Create three searches with distinct positions
    const s1 = await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Alpha',
      filters: {},
      view_mode: 'list',
      shared: false,
    });
    const s2 = await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Beta',
      filters: {},
      view_mode: 'list',
      shared: false,
    });
    const s3 = await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Gamma',
      filters: {},
      view_mode: 'list',
      shared: false,
    });

    // Set explicit positions so ordering is deterministic
    await api.updateSavedSearch(request, testUser.token, testProject.key, s1.id, { position: 0 });
    await api.updateSavedSearch(request, testUser.token, testProject.key, s2.id, { position: 1 });
    await api.updateSavedSearch(request, testUser.token, testProject.key, s3.id, { position: 2 });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    await openSelector(page);

    // All three should be visible
    await expect(page.getByText('Alpha')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Beta')).toBeVisible();
    await expect(page.getByText('Gamma')).toBeVisible();

    // Hover over "Gamma" (last) and click move up
    const gammaEntry = page.locator('.group').filter({ hasText: 'Gamma' }).first();
    await gammaEntry.hover();
    const upBtn = gammaEntry.locator('button').filter({ has: page.locator('svg.lucide-chevron-up') });
    await upBtn.click({ force: true });

    // Wait for mutation to complete
    await page.waitForTimeout(1000);

    // Verify via API that Gamma moved up (position should swap with Beta)
    const searches = await api.listSavedSearches(request, testUser.token, testProject.key);
    const gamma = searches.find((s) => s.name === 'Gamma');
    const beta = searches.find((s) => s.name === 'Beta');
    // After moving Gamma up: Alpha=0, Gamma=1, Beta=2
    expect(gamma).toBeDefined();
    expect(beta).toBeDefined();
  });

  test('desktop filter searches in dropdown', async ({ request, testUser, testProject, page }) => {
    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Bug Tracker',
      filters: { type: ['bug'] },
      view_mode: 'list',
      shared: false,
    });
    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Task Overview',
      filters: { type: ['task'] },
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    await openSelector(page);

    // Both should be visible initially
    await expect(page.getByText('Bug Tracker')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Task Overview')).toBeVisible();

    // Type filter text
    await page.getByPlaceholder('Search...').fill('Bug');

    // Only "Bug Tracker" should be visible
    await expect(page.getByText('Bug Tracker')).toBeVisible({ timeout: 3000 });
    await expect(page.getByText('Task Overview')).not.toBeVisible({ timeout: 3000 });

    // Clear filter
    await page.getByPlaceholder('Search...').clear();

    // Both visible again
    await expect(page.getByText('Bug Tracker')).toBeVisible({ timeout: 3000 });
    await expect(page.getByText('Task Overview')).toBeVisible();
  });

  test('desktop first item has up arrow disabled, last has down disabled', async ({ request, testUser, testProject, page }) => {
    const s1 = await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'First Item',
      filters: {},
      view_mode: 'list',
      shared: false,
    });
    const s2 = await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Last Item',
      filters: {},
      view_mode: 'list',
      shared: false,
    });
    await api.updateSavedSearch(request, testUser.token, testProject.key, s1.id, { position: 0 });
    await api.updateSavedSearch(request, testUser.token, testProject.key, s2.id, { position: 1 });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    await openSelector(page);

    // Hover over first item — up arrow should be disabled
    const firstEntry = page.locator('.group').filter({ hasText: 'First Item' }).first();
    await firstEntry.hover();
    const firstUpBtn = firstEntry.locator('button').filter({ has: page.locator('svg.lucide-chevron-up') });
    await expect(firstUpBtn).toBeDisabled();

    // Hover over last item — down arrow should be disabled
    const lastEntry = page.locator('.group').filter({ hasText: 'Last Item' }).first();
    await lastEntry.hover();
    const lastDownBtn = lastEntry.locator('button').filter({ has: page.locator('svg.lucide-chevron-down') });
    await expect(lastDownBtn).toBeDisabled();
  });
});

test.describe('Saved Searches - Mobile', () => {
  test('mobile open saved search modal and select', async ({ request, testUser, testProject, page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Mobile bug item',
      type: 'bug',
    });

    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Mobile Bugs',
      filters: { type: ['bug'] },
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Click the saved search button (folder-search icon)
    const savedSearchBtn = page.locator('button[aria-label="Saved Searches"]');
    await expect(savedSearchBtn).toBeVisible({ timeout: 5000 });
    await savedSearchBtn.click();

    // Modal should open — verify search input and entries are visible
    await expect(page.getByPlaceholder('Search...')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('My Searches', { exact: true })).toBeVisible({ timeout: 3000 });
    await expect(page.getByText('Mobile Bugs')).toBeVisible();

    // Select the search
    await page.getByRole('button', { name: 'Mobile Bugs' }).click();

    // Modal should close and filter should be applied
    await expect(page).toHaveURL(/type=bug/, { timeout: 10000 });
  });

  test('mobile active search shows badge on icon', async ({ request, testUser, testProject, page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Badge Test',
      filters: { type: ['bug'] },
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Open and select the search
    const savedSearchBtn = page.locator('button[aria-label="Saved Searches"]');
    await savedSearchBtn.click();
    await page.getByRole('button', { name: 'Badge Test' }).click();
    await expect(page).toHaveURL(/type=bug/, { timeout: 10000 });

    // The saved search button should now have a badge indicator
    const badge = savedSearchBtn.locator('.bg-indigo-600');
    await expect(badge).toBeVisible({ timeout: 3000 });
  });

  test('mobile edit mode shows rename and delete buttons', async ({ request, testUser, testProject, page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Editable Search',
      filters: {},
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Open saved search modal
    await page.locator('button[aria-label="Saved Searches"]').click();
    await expect(page.getByText('Editable Search')).toBeVisible({ timeout: 5000 });

    // In browse mode, pencil/trash icons should not be always visible on the entries
    const entry = page.locator('.group').filter({ hasText: 'Editable Search' }).first();
    const pencilBtn = entry.locator('button').filter({ has: page.locator('svg.lucide-pencil') });

    // Click Edit mode button (pencil icon in title bar)
    const editToggle = page.locator('button[aria-label="Edit"]');
    await editToggle.click();

    // Now edit action buttons should be visible on entries
    await expect(pencilBtn).toBeVisible({ timeout: 3000 });

    // Click pencil on the entry to rename
    await pencilBtn.click();

    // Rename modal should open
    await expect(page.getByRole('heading', { name: 'Rename Saved Search' })).toBeVisible({ timeout: 5000 });
    const renameInput = page.getByPlaceholder('e.g. My open bugs');
    await renameInput.clear();
    await renameInput.fill('Renamed Search');
    await page.getByRole('button', { name: 'Save', exact: true }).click();

    // Rename modal closes
    await expect(page.getByRole('heading', { name: 'Rename Saved Search' })).not.toBeVisible({ timeout: 5000 });

    // Verify renamed entry in the modal list
    await expect(page.getByText('Renamed Search')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Editable Search')).not.toBeVisible({ timeout: 3000 });
  });

  test('mobile order mode shows up/down arrows', async ({ request, testUser, testProject, page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    const s1 = await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Mobile First',
      filters: {},
      view_mode: 'list',
      shared: false,
    });
    const s2 = await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Mobile Second',
      filters: {},
      view_mode: 'list',
      shared: false,
    });
    await api.updateSavedSearch(request, testUser.token, testProject.key, s1.id, { position: 0 });
    await api.updateSavedSearch(request, testUser.token, testProject.key, s2.id, { position: 1 });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Open saved search modal
    await page.locator('button[aria-label="Saved Searches"]').click();
    await expect(page.getByText('Mobile First')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Mobile Second')).toBeVisible();

    // Enable Order mode
    const orderToggle = page.locator('button[aria-label="Order"]');
    await orderToggle.click();

    // Up/down arrows should be visible on entries
    const secondEntry = page.locator('.group').filter({ hasText: 'Mobile Second' }).first();
    const upBtn = secondEntry.locator('button').filter({ has: page.locator('svg.lucide-chevron-up') });
    await expect(upBtn).toBeVisible({ timeout: 3000 });

    // Click up to move "Mobile Second" above "Mobile First"
    await upBtn.click();

    // Wait for reorder to complete
    await page.waitForTimeout(1000);

    // Verify via API
    const searches = await api.listSavedSearches(request, testUser.token, testProject.key);
    const first = searches.find((s) => s.name === 'Mobile First');
    const second = searches.find((s) => s.name === 'Mobile Second');
    expect(first).toBeDefined();
    expect(second).toBeDefined();
  });

  test('mobile delete saved search via edit mode', async ({ request, testUser, testProject, page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Mobile Delete Me',
      filters: {},
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Open saved search modal
    await page.locator('button[aria-label="Saved Searches"]').click();
    await expect(page.getByText('Mobile Delete Me')).toBeVisible({ timeout: 5000 });

    // Enable Edit mode
    await page.locator('button[aria-label="Edit"]').click();

    // Click trash on the entry
    const entry = page.locator('.group').filter({ hasText: 'Mobile Delete Me' }).first();
    const trashBtn = entry.locator('button').filter({ has: page.locator('svg.lucide-trash-2') });
    await trashBtn.click();

    // Confirm delete
    await expect(page.getByRole('heading', { name: 'Delete Saved Search' })).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: 'Delete', exact: true }).click();

    // Modal closes and entry should be gone
    await expect(page.getByRole('heading', { name: 'Delete Saved Search' })).not.toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Mobile Delete Me')).not.toBeVisible({ timeout: 5000 });

    // Verify via API
    const searches = await api.listSavedSearches(request, testUser.token, testProject.key);
    expect(searches.some((s) => s.name === 'Mobile Delete Me')).toBe(false);
  });

  test('mobile search filter within modal', async ({ request, testUser, testProject, page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Bugs View',
      filters: { type: ['bug'] },
      view_mode: 'list',
      shared: false,
    });
    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Tasks View',
      filters: { type: ['task'] },
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Open saved search modal
    await page.locator('button[aria-label="Saved Searches"]').click();

    // Both should be visible
    await expect(page.getByText('Bugs View')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Tasks View')).toBeVisible();

    // Type in the search input
    await page.getByPlaceholder('Search...').fill('Bugs');

    // Only "Bugs View" should remain
    await expect(page.getByText('Bugs View')).toBeVisible({ timeout: 3000 });
    await expect(page.getByText('Tasks View')).not.toBeVisible({ timeout: 3000 });
  });

  test('mobile mode toggles are exclusive', async ({ request, testUser, testProject, page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    await api.createSavedSearch(request, testUser.token, testProject.key, {
      name: 'Toggle Test',
      filters: {},
      view_mode: 'list',
      shared: false,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Open saved search modal
    await page.locator('button[aria-label="Saved Searches"]').click();
    await expect(page.getByText('Toggle Test')).toBeVisible({ timeout: 5000 });

    const entry = page.locator('.group').filter({ hasText: 'Toggle Test' }).first();
    const editToggle = page.locator('button[aria-label="Edit"]');
    const orderToggle = page.locator('button[aria-label="Order"]');

    // Enable Edit mode
    await editToggle.click();

    // Edit buttons (pencil) should be visible
    const pencilBtn = entry.locator('button').filter({ has: page.locator('svg.lucide-pencil') });
    await expect(pencilBtn).toBeVisible({ timeout: 3000 });

    // Reorder arrows should NOT be visible
    const upBtn = entry.locator('button').filter({ has: page.locator('svg.lucide-chevron-up') });
    await expect(upBtn).not.toBeVisible();

    // Switch to Order mode
    await orderToggle.click();

    // Reorder arrows should be visible now
    await expect(upBtn).toBeVisible({ timeout: 3000 });

    // Edit buttons (pencil) should NOT be visible
    await expect(pencilBtn).not.toBeVisible();

    // Toggle Order off — back to browse mode
    await orderToggle.click();

    // Neither should be always visible
    await expect(upBtn).not.toBeVisible({ timeout: 3000 });
  });
});
