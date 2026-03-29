import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

async function waitForPageReady(page: import('@playwright/test').Page) {
  await page.waitForLoadState('networkidle');
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 1000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

test.describe('Back link navigation', () => {
  test('from work items list shows "Back to items" and navigates back', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Back link list test',
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    await page.getByRole('table').getByText('Back link list test').click();
    await expect(page).toHaveURL(new RegExp(`/d/projects/${testProject.key}/items/${item.item_number}`), { timeout: 10000 });

    // Browser tab title should show the work item display ID
    await expect(page).toHaveTitle(new RegExp(`${testProject.key}-${item.item_number}.*Taskwondo`));

    const backLink = page.getByText('Back to items');
    await expect(backLink).toBeVisible();

    await backLink.click();
    await expect(page).toHaveURL(new RegExp(`/d/projects/${testProject.key}/items`), { timeout: 10000 });
  });

  test('from inbox navigates back and highlights the previously opened item', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Back link inbox test',
      type: 'task',
    });
    await api.addToInbox(request, testUser.token, item.id);

    await page.goto('/user/inbox');
    await waitForPageReady(page);

    // Store active row key should be set after click
    await page.getByRole('table').getByText('Back link inbox test').click();
    await expect(page).toHaveURL(new RegExp(`/d/projects/${testProject.key}/items/${item.item_number}`), { timeout: 10000 });

    await page.getByText('Back to inbox').click();
    await expect(page).toHaveURL(/\/user\/inbox/, { timeout: 10000 });

    // The previously clicked row should be highlighted (InboxRow uses bg-indigo-50 for isActive)
    const highlightedRow = page.locator('tr.bg-indigo-50').filter({ hasText: 'Back link inbox test' });
    await expect(highlightedRow).toBeVisible({ timeout: 5000 });
  });

  test('from watchlist navigates back and highlights the previously opened item', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Back link watchlist test',
      type: 'task',
    });
    await api.toggleWatch(request, testUser.token, testProject.key, item.item_number);

    await page.goto('/user/watchlist');
    await waitForPageReady(page);

    await page.getByRole('table').getByText('Back link watchlist test').click();
    await expect(page).toHaveURL(new RegExp(`/d/projects/${testProject.key}/items/${item.item_number}`), { timeout: 10000 });

    await page.getByText('Back to watchlist').click();
    await expect(page).toHaveURL(/\/user\/watchlist/, { timeout: 10000 });

    // DataTable uses ring-indigo-500 for active row highlight
    const highlightedRow = page.locator('tr.ring-indigo-500').filter({ hasText: 'Back link watchlist test' });
    await expect(highlightedRow).toBeVisible({ timeout: 5000 });
  });

  test('from milestone dashboard navigates back and highlights the previously opened item', async ({ request, testUser, testProject, page }) => {
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Back link milestone',
    });
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Back link milestone test',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      milestone_id: milestone.id,
    });

    await page.goto(`/d/projects/${testProject.key}/milestones/${milestone.id}`);
    await waitForPageReady(page);

    await page.getByText('Back link milestone test').first().click();
    await expect(page).toHaveURL(new RegExp(`/d/projects/${testProject.key}/items/${item.item_number}`), { timeout: 10000 });

    await page.getByText('Back to milestone').click();
    await expect(page).toHaveURL(new RegExp(`/d/projects/${testProject.key}/milestones/${milestone.id}`), { timeout: 10000 });

    // MilestoneDashboardPage WorkItemRow uses bg-indigo-50 for active
    const highlightedRow = page.locator('.bg-indigo-50').filter({ hasText: 'Back link milestone test' });
    await expect(highlightedRow).toBeVisible({ timeout: 5000 });
  });
});
