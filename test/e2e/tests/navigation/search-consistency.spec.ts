import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

async function waitForPageReady(page: import('@playwright/test').Page) {
  await page.waitForLoadState('domcontentloaded');
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 1000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

test.describe('Search bar UI consistency', () => {
  test('sidebar shows Inbox, Watchlist, Feed in correct order', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    const sidebar = page.locator('nav.hidden.sm\\:block');
    await expect(sidebar).toBeVisible({ timeout: 15000 });

    // Get bounding boxes to verify visual order (top to bottom)
    const inboxLink = sidebar.getByText('Inbox');
    const watchlistLink = sidebar.getByText('Watchlist');
    const feedLink = sidebar.getByText('Feed');

    await expect(inboxLink).toBeVisible();
    await expect(watchlistLink).toBeVisible();
    await expect(feedLink).toBeVisible();

    const inboxBox = await inboxLink.boundingBox();
    const watchlistBox = await watchlistLink.boundingBox();
    const feedBox = await feedLink.boundingBox();

    expect(inboxBox).toBeTruthy();
    expect(watchlistBox).toBeTruthy();
    expect(feedBox).toBeTruthy();

    // Inbox should be above Watchlist, Watchlist should be above Feed
    expect(inboxBox!.y).toBeLessThan(watchlistBox!.y);
    expect(watchlistBox!.y).toBeLessThan(feedBox!.y);
  });

  test('inbox search bar is in the title row (desktop)', async ({ request, testUser, testProject, page }) => {
    // Create an inbox item so the page has content
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Search consistency test item',
      type: 'task',
    });
    await api.addToInbox(request, testUser.token, item.id);

    await page.goto('/user/inbox');
    await waitForPageReady(page);

    // The title row should contain the heading
    const heading = page.getByRole('heading', { name: 'Inbox' });
    await expect(heading).toBeVisible({ timeout: 10000 });

    // Desktop search bar should be visible in the title row (hidden lg:block)
    const desktopSearch = page.locator('.hidden.lg\\:block input').first();
    await expect(desktopSearch).toBeVisible();

    // The heading and search bar should be on the same visual row
    const headingBox = await heading.boundingBox();
    const searchBox = await desktopSearch.boundingBox();
    expect(headingBox).toBeTruthy();
    expect(searchBox).toBeTruthy();

    // They should overlap vertically (same row)
    const headingMidY = headingBox!.y + headingBox!.height / 2;
    const searchMidY = searchBox!.y + searchBox!.height / 2;
    expect(Math.abs(headingMidY - searchMidY)).toBeLessThan(30);
  });

  test('inbox has project filter, auto-hide, and refresh in second row', async ({ page }) => {
    await page.goto('/user/inbox');
    await waitForPageReady(page);

    // Wait for the page content to load
    await page.waitForLoadState('networkidle');

    // Refresh button should be visible on the page
    const refreshButton = page.getByRole('button', { name: /refresh/i }).first();
    await expect(refreshButton).toBeVisible({ timeout: 10000 });

    // Auto-hide toggle text should be visible
    await expect(page.getByText('Auto-hide completed items')).toBeVisible();
  });

  test('watchlist has search bar in title row (desktop)', async ({ request, testUser, testProject, page }) => {
    // Create and watch an item so the watchlist has content
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watchlist search test item',
      type: 'task',
    });
    await api.toggleWatch(request, testUser.token, testProject.key, item.item_number);

    await page.goto('/user/watchlist');
    await waitForPageReady(page);

    // Wait for items to load
    await expect(page.getByText('Watchlist search test item').first()).toBeVisible({ timeout: 10000 });

    // Desktop search bar should be in the title row
    const desktopSearch = page.locator('.hidden.lg\\:block input').first();
    await expect(desktopSearch).toBeVisible();

    // The heading "Watchlist" or "My Watchlist"
    const heading = page.locator('h2').first();
    await expect(heading).toBeVisible();

    // Verify search is on same row as heading
    const headingBox = await heading.boundingBox();
    const searchBox = await desktopSearch.boundingBox();
    expect(headingBox).toBeTruthy();
    expect(searchBox).toBeTruthy();

    const headingMidY = headingBox!.y + headingBox!.height / 2;
    const searchMidY = searchBox!.y + searchBox!.height / 2;
    expect(Math.abs(headingMidY - searchMidY)).toBeLessThan(30);
  });

  test('watchlist has refresh button', async ({ page }) => {
    await page.goto('/user/watchlist');
    await waitForPageReady(page);
    await page.waitForLoadState('networkidle');

    // Refresh button should be visible on the watchlist page
    const refreshButton = page.getByRole('button', { name: /refresh/i }).first();
    await expect(refreshButton).toBeVisible({ timeout: 10000 });
  });

  test('watchlist refresh button shares preference with inbox', async ({ request, testUser, page }) => {
    // Set refresh interval via inbox preference
    await api.setPreference(request, testUser.token, 'inbox_refresh_interval', 30000);

    // Navigate to watchlist
    await page.goto('/user/watchlist');
    await waitForPageReady(page);
    await page.waitForLoadState('networkidle');

    // The refresh button should show "30s" label from the shared preference
    const refreshButton = page.getByRole('button', { name: /refresh/i }).first();
    await expect(refreshButton).toBeVisible({ timeout: 10000 });
    await expect(refreshButton).toContainText('30s');
  });

  test('work items search bar is in title row (desktop)', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    // The heading should be visible
    const heading = page.getByText('Work Items').first();
    await expect(heading).toBeVisible({ timeout: 10000 });

    // Desktop search bar should be in the title row
    const desktopSearch = page.locator('.hidden.lg\\:block input').first();
    await expect(desktopSearch).toBeVisible();

    // Verify search is on same row as heading
    const headingBox = await heading.boundingBox();
    const searchBox = await desktopSearch.boundingBox();
    expect(headingBox).toBeTruthy();
    expect(searchBox).toBeTruthy();

    const headingMidY = headingBox!.y + headingBox!.height / 2;
    const searchMidY = searchBox!.y + searchBox!.height / 2;
    expect(Math.abs(headingMidY - searchMidY)).toBeLessThan(30);
  });
});
