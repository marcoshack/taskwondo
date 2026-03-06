import { test, expect } from '../../lib/fixtures';

async function waitForPageReady(page: import('@playwright/test').Page) {
  await page.waitForLoadState('networkidle');
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 1000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

test.describe('Mobile search icon and top bar layout', () => {
  test('search icon is visible on mobile and opens search modal', async ({ page, testProject }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    // Search icon button should be visible on mobile
    const searchButton = page.getByRole('button', { name: /^search$/i });
    await expect(searchButton).toBeVisible({ timeout: 5000 });

    // Click it to open search modal
    await searchButton.click();
    const searchInput = page.getByPlaceholder(/search across/i);
    await expect(searchInput).toBeVisible({ timeout: 3000 });
    await expect(searchInput).toBeFocused();

    // Close with Escape
    await page.keyboard.press('Escape');
    await expect(searchInput).not.toBeVisible({ timeout: 3000 });
  });

  test('search icon is visible on desktop', async ({ page, testProject }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto(`/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    // Search icon button should be visible on all screen sizes
    const searchButton = page.getByRole('button', { name: /^search$/i });
    await expect(searchButton).toBeVisible({ timeout: 5000 });
  });

  test('search icon is visible on tablet', async ({ page, testProject }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto(`/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    const searchButton = page.getByRole('button', { name: /^search$/i });
    await expect(searchButton).toBeVisible({ timeout: 5000 });
  });

  test('mobile top bar shows home icon and project key when project is active', async ({
    page,
    testProject,
  }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    // Home button should be visible
    const homeButton = page.getByRole('button', { name: /^home$/i });
    await expect(homeButton).toBeVisible({ timeout: 5000 });

    // Project key badge should be visible in the nav
    const nav = page.locator('nav');
    await expect(nav.getByText(testProject.key).first()).toBeVisible();
  });

  test('mobile top bar shows home icon on preferences page when project exists', async ({
    page,
    testProject,
  }) => {
    // First visit a project page to set lastProjectKey
    await page.goto(`/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/preferences');

    // Home button should be visible (project is remembered)
    const homeButton = page.getByRole('button', { name: /^home$/i });
    await expect(homeButton).toBeVisible({ timeout: 5000 });

    // Project key badge visible in nav
    const nav = page.locator('nav');
    await expect(nav.getByText(testProject.key).first()).toBeVisible();
  });

  test('mobile top bar shows home icon on admin page when project exists', async ({
    page,
    testProject,
  }) => {
    // First visit a project page to set lastProjectKey
    await page.goto(`/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/admin');

    // Home button should be visible (project is remembered)
    const homeButton = page.getByRole('button', { name: /^home$/i });
    await expect(homeButton).toBeVisible({ timeout: 5000 });
  });

  test('home icon navigates to projects list', async ({ page, testProject }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    const homeButton = page.getByRole('button', { name: /^home$/i });
    await expect(homeButton).toBeVisible({ timeout: 5000 });
    await homeButton.click();

    await expect(page).toHaveURL(/\/projects\/?$/, { timeout: 5000 });
  });

  test('project key badge opens project switcher on mobile', async ({ page, testProject }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    // Click the project key badge in the top-left area
    const topBar = page.locator('nav');
    const projectBadge = topBar.getByText(testProject.key).first();
    await projectBadge.click();

    // Project switcher modal should open
    await expect(page.getByPlaceholder(/search projects/i)).toBeVisible({ timeout: 3000 });
  });

  test('search icon is visible on mobile preferences page', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/preferences');
    await waitForPageReady(page);

    // Search icon should be available on all pages
    const searchButton = page.getByRole('button', { name: /^search$/i });
    await expect(searchButton).toBeVisible({ timeout: 5000 });
  });

  test('mobile icon order: search before inbox before menu', async ({ page, testProject }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    // Get bounding boxes to verify left-to-right order
    const searchBox = await page.getByRole('button', { name: /^search$/i }).boundingBox();
    const inboxBox = await page.getByRole('button', { name: /inbox/i }).boundingBox();
    const menuBox = await page.getByRole('button', { name: /menu/i }).boundingBox();

    expect(searchBox).toBeTruthy();
    expect(inboxBox).toBeTruthy();
    expect(menuBox).toBeTruthy();

    // Search should be to the left of Inbox
    expect(searchBox!.x).toBeLessThan(inboxBox!.x);
    // Inbox should be to the left of Menu
    expect(inboxBox!.x).toBeLessThan(menuBox!.x);
  });

  test('desktop top bar is unchanged with project active', async ({ page, testProject }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto(`/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    // Brand name should be visible on desktop
    const nav = page.locator('nav');
    await expect(nav.getByText('Taskwondo').first()).toBeVisible({ timeout: 5000 });

    // Project name should be visible (desktop shows key + name)
    await expect(nav.getByText(testProject.name)).toBeVisible();

    // Home button should NOT be visible on desktop
    const homeButton = page.getByRole('button', { name: /^home$/i });
    await expect(homeButton).not.toBeVisible();
  });
});
