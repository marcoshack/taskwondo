import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

async function waitForPageReady(page: import('@playwright/test').Page) {
  await page.waitForLoadState('domcontentloaded');
  await expect(page.locator('nav.hidden.sm\\:block')).toBeVisible({ timeout: 15000 });
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 1000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

test.describe('Unified sidebar', () => {
  test('shows user items on project pages', async ({ page, testProject }) => {
    await page.goto(`/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    const sidebar = page.locator('nav.hidden.sm\\:block');
    await expect(sidebar).toBeVisible();

    // User section items should be visible
    await expect(sidebar.getByText('Inbox')).toBeVisible();
    await expect(sidebar.getByText('Feed')).toBeVisible();
    await expect(sidebar.getByText('Watchlist')).toBeVisible();

    // Project section items should also be visible
    await expect(sidebar.getByText('Overview')).toBeVisible();
    await expect(sidebar.getByText('Items')).toBeVisible();
  });

  test('shows project context with badge in sidebar and name in top bar', async ({ page, testProject }) => {
    await page.goto(`/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    const sidebar = page.locator('nav.hidden.sm\\:block');

    // Project key badge should be visible in the sidebar Projects link
    await expect(sidebar.getByText(testProject.key)).toBeVisible();

    // Project name should be visible in the top bar, not sidebar
    const topBar = page.locator('nav').first();
    await expect(topBar.getByText(testProject.name)).toBeVisible();
  });

  test('Projects link navigates to project list with sidebar', async ({ page, testProject }) => {
    await page.goto(`/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    const sidebar = page.locator('nav.hidden.sm\\:block');

    // Click "Projects" in sidebar
    await sidebar.getByText('Projects').click();
    await expect(page).toHaveURL(/\/projects$/, { timeout: 5000 });

    // Sidebar should still be visible on the project list page
    await expect(sidebar).toBeVisible();
    await expect(sidebar.getByText('Inbox')).toBeVisible();
    await expect(sidebar.getByText('Projects')).toBeVisible();
  });

  test('project list page remembers last project in sidebar', async ({ page, testProject }) => {
    // Visit a project first to set the "last project"
    await page.goto(`/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    // Navigate to project list
    const sidebar = page.locator('nav.hidden.sm\\:block');
    await sidebar.getByText('Projects').click();
    await expect(page).toHaveURL(/\/projects$/, { timeout: 5000 });

    // Sidebar should still show the last project context
    await expect(sidebar).toBeVisible();
    await expect(sidebar.getByText('Inbox')).toBeVisible();
    await expect(sidebar.getByText(testProject.key)).toBeVisible();
    await expect(sidebar.getByText('Overview')).toBeVisible();
  });

  test('switching projects updates sidebar and top bar', async ({ request, testUser, testProject, page }) => {
    // Create a second project
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const secondKey = `S${suffix}`;
    const secondName = `Second Project ${suffix}`;
    await api.createProject(request, testUser.token, secondKey, secondName);

    // Navigate to first project
    await page.goto(`/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    const sidebar = page.locator('nav.hidden.sm\\:block');
    const topBar = page.locator('nav').first();

    // Verify first project key in sidebar and name in top bar
    await expect(sidebar.getByText(testProject.key)).toBeVisible();
    await expect(topBar.getByText(testProject.name)).toBeVisible();

    // Navigate to second project
    await page.goto(`/projects/${secondKey}/items`);

    // Verify sidebar and top bar now show second project
    await expect(sidebar.getByText(secondKey)).toBeVisible({ timeout: 5000 });
    await expect(topBar.getByText(secondName)).toBeVisible({ timeout: 5000 });
  });

  test('sidebar remembers last project on user pages', async ({ page, testProject }) => {
    // Visit a project first to set the "last project"
    await page.goto(`/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    // Navigate to user inbox
    const sidebar = page.locator('nav.hidden.sm\\:block');
    await sidebar.getByText('Inbox').click();
    await expect(page).toHaveURL(/\/user\/inbox/, { timeout: 5000 });

    // Sidebar should still show the project section from the last visited project
    await expect(sidebar.getByText(testProject.key)).toBeVisible();
    await expect(sidebar.getByText('Overview')).toBeVisible();
    await expect(sidebar.getByText('Items')).toBeVisible();
  });

  test('sidebar state shared between user and project pages', async ({ page, testProject }) => {
    // Start on user page, collapse sidebar
    await page.goto('/user/inbox');
    await waitForPageReady(page);

    const sidebar = page.locator('nav.hidden.sm\\:block');
    await expect(sidebar).toHaveClass(/w-48/);

    // Collapse
    await sidebar.getByRole('button', { name: /collapse/i }).click();
    await expect(sidebar).toHaveClass(/w-14/, { timeout: 3000 });

    // Navigate to project page — sidebar should still be collapsed
    await page.goto(`/projects/${testProject.key}/items`);
    const projectSidebar = page.locator('nav.hidden.sm\\:block');
    await expect(projectSidebar).toHaveClass(/w-14/, { timeout: 3000 });
  });
});
