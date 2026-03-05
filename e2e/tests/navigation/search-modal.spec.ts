import { test, expect } from '../../lib/fixtures';
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
    await page.goto(`/projects/${testProject.key}/items`);
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

  test('search shows results for matching work items (FTS fallback)', async ({
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

    await page.goto(`/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open search modal
    await page.keyboard.press('g');
    await page.keyboard.press('k');
    const searchInput = page.getByPlaceholder(/search across/i);
    await expect(searchInput).toBeVisible({ timeout: 3000 });

    // Type search query
    await searchInput.fill(uniqueTitle.slice(0, 12));

    // Wait for results - should show the work item (FTS mode since semantic search is disabled)
    const resultItem = page.locator('[data-search-item]').first();
    await expect(resultItem).toBeVisible({ timeout: 10000 });
    await expect(resultItem).toContainText(uniqueTitle);
  });

  test('search shows empty state when no results found', async ({ page, testProject }) => {
    await page.goto(`/projects/${testProject.key}/items`);
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

    await page.goto(`/projects/${testProject.key}/items`);
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

    await page.goto(`/projects/${testProject.key}/items`);
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
      new RegExp(`/projects/${testProject.key}/items/${item.item_number}`),
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

    await page.goto(`/projects/${testProject.key}/items`);
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
      new RegExp(`/projects/${testProject.key}/items/${item.item_number}`),
      { timeout: 5000 },
    );
  });

  test('search hint shown before typing', async ({ page, testProject }) => {
    await page.goto(`/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('g');
    await page.keyboard.press('k');
    await expect(page.getByText(/type at least 2 characters/i)).toBeVisible({ timeout: 3000 });
  });
});
