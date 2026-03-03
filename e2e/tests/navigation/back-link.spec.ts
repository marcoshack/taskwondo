import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
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

    await page.goto(`/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    await page.getByRole('table').getByText('Back link list test').click();
    await expect(page).toHaveURL(new RegExp(`/projects/${testProject.key}/items/${item.item_number}`), { timeout: 10000 });

    const backLink = page.getByText('Back to items');
    await expect(backLink).toBeVisible();

    await backLink.click();
    await expect(page).toHaveURL(new RegExp(`/projects/${testProject.key}/items`), { timeout: 10000 });
  });

  test('from inbox shows "Back to inbox" and navigates back', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Back link inbox test',
      type: 'task',
    });
    await api.addToInbox(request, testUser.token, item.id);

    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    await page.getByRole('table').getByText('Back link inbox test').click();
    await expect(page).toHaveURL(new RegExp(`/projects/${testProject.key}/items/${item.item_number}`), { timeout: 10000 });

    const backLink = page.getByText('Back to inbox');
    await expect(backLink).toBeVisible();

    await backLink.click();
    await expect(page).toHaveURL(/\/user\/inbox/, { timeout: 10000 });
  });

  test('from watchlist shows "Back to watchlist" and navigates back', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Back link watchlist test',
      type: 'task',
    });
    await api.toggleWatch(request, testUser.token, testProject.key, item.item_number);

    await page.goto('/user/watchlist');
    await dismissWelcomeModal(page);

    await page.getByRole('table').getByText('Back link watchlist test').click();
    await expect(page).toHaveURL(new RegExp(`/projects/${testProject.key}/items/${item.item_number}`), { timeout: 10000 });

    const backLink = page.getByText('Back to watchlist');
    await expect(backLink).toBeVisible();

    await backLink.click();
    await expect(page).toHaveURL(/\/user\/watchlist/, { timeout: 10000 });
  });

  test('from milestone dashboard shows "Back to milestone" and navigates back', async ({ request, testUser, testProject, page }) => {
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

    await page.goto(`/projects/${testProject.key}/milestones/${milestone.id}`);
    await dismissWelcomeModal(page);

    await page.getByText('Back link milestone test').first().click();
    await expect(page).toHaveURL(new RegExp(`/projects/${testProject.key}/items/${item.item_number}`), { timeout: 10000 });

    const backLink = page.getByText('Back to milestone');
    await expect(backLink).toBeVisible();

    await backLink.click();
    await expect(page).toHaveURL(new RegExp(`/projects/${testProject.key}/milestones/${milestone.id}`), { timeout: 10000 });
  });
});
