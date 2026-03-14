import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

/** Transition a work item to done through valid workflow steps: backlog → open → in_progress → in_review → done */
async function completeWorkItem(
  request: import('@playwright/test').APIRequestContext,
  token: string,
  projectKey: string,
  itemNumber: number,
) {
  await api.updateWorkItem(request, token, projectKey, itemNumber, { status: 'open' });
  await api.updateWorkItem(request, token, projectKey, itemNumber, { status: 'in_progress' });
  await api.updateWorkItem(request, token, projectKey, itemNumber, { status: 'in_review' });
  await api.updateWorkItem(request, token, projectKey, itemNumber, { status: 'done' });
}

/** Set the work items view state to show all statuses (including done/cancelled). */
async function showAllStatuses(
  request: import('@playwright/test').APIRequestContext,
  token: string,
  projectKey: string,
) {
  await api.setProjectUserSetting(request, token, projectKey, 'workitem_view_state', {
    filter: { status: ['backlog', 'open', 'in_progress', 'in_review', 'done', 'cancelled'] },
    search: '',
    viewMode: 'list',
    sort: 'created_at',
    order: 'desc',
    activeSearchId: null,
    activeSearchSnapshot: null,
  });
}

async function waitForTable(page: import('@playwright/test').Page) {
  await expect(page.getByRole('table')).toBeVisible({ timeout: 10000 });
}

test.describe('Done effect on work item lists', () => {
  test('work items list shows done effect for completed items (desktop)', async ({ request, testUser, testProject, page }) => {
    // Create two items: one open, one completed
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Open task stays normal',
      type: 'task',
    });

    const doneItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Done task is muted',
      type: 'task',
    });
    await completeWorkItem(request, testUser.token, testProject.key, doneItem.item_number);

    // Set view state to show all statuses including done
    await showAllStatuses(request, testUser.token, testProject.key);

    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForTable(page);

    const table = page.locator('table');
    // Wait for items to load (scoped to table to avoid matching mobile cards)
    await expect(table.getByText('Open task stays normal')).toBeVisible({ timeout: 10000 });
    await expect(table.getByText('Done task is muted')).toBeVisible();

    // Desktop table: done item title should have line-through
    await expect(table.getByText('Done task is muted')).toHaveCSS('text-decoration-line', 'line-through');

    // Open item title should NOT have line-through
    await expect(table.getByText('Open task stays normal')).not.toHaveCSS('text-decoration-line', 'line-through');
  });

  test('board view shows done effect for completed items', async ({ request, testUser, testProject, page }) => {
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Board open task',
      type: 'task',
    });

    const doneItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Board done task',
      type: 'task',
    });
    await completeWorkItem(request, testUser.token, testProject.key, doneItem.item_number);

    // Set view state to show all statuses in board mode
    await api.setProjectUserSetting(request, testUser.token, testProject.key, 'workitem_view_state', {
      filter: { status: ['backlog', 'open', 'in_progress', 'in_review', 'done', 'cancelled'] },
      search: '',
      viewMode: 'board',
      sort: 'created_at',
      order: 'desc',
      activeSearchId: null,
      activeSearchSnapshot: null,
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await expect(page.getByText('Board open task')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Board done task')).toBeVisible();

    // Done item title should have line-through
    const doneTitle = page.getByText('Board done task');
    await expect(doneTitle).toHaveCSS('text-decoration-line', 'line-through');

    // Open item title should NOT have line-through
    const openTitle = page.getByText('Board open task');
    await expect(openTitle).not.toHaveCSS('text-decoration-line', 'line-through');
  });

  test('watchlist shows done effect for completed items', async ({ request, testUser, testProject, page }) => {
    const openItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watch open item',
      type: 'task',
    });
    await api.toggleWatch(request, testUser.token, testProject.key, openItem.item_number);

    const doneItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watch done item',
      type: 'task',
    });
    await api.toggleWatch(request, testUser.token, testProject.key, doneItem.item_number);
    await completeWorkItem(request, testUser.token, testProject.key, doneItem.item_number);

    await page.goto('/user/watchlist');
    const table = page.locator('table');
    // Wait for items - watchlist shows all statuses by default (scoped to table)
    await expect(table.getByText('Watch open item')).toBeVisible({ timeout: 10000 });
    await expect(table.getByText('Watch done item')).toBeVisible();

    // Done item title should have line-through
    await expect(table.getByText('Watch done item')).toHaveCSS('text-decoration-line', 'line-through');

    // Open item title should NOT have line-through
    await expect(table.getByText('Watch open item')).not.toHaveCSS('text-decoration-line', 'line-through');
  });

  test('milestone dashboard shows done effect for completed items', async ({ request, testUser, testProject, page }) => {
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Done Effect Test',
    });

    const openItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Milestone open item',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, openItem.item_number, {
      milestone_id: milestone.id,
    });

    const doneItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Milestone done item',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, doneItem.item_number, {
      milestone_id: milestone.id,
    });
    await completeWorkItem(request, testUser.token, testProject.key, doneItem.item_number);

    await page.goto(`/d/projects/${testProject.key}/milestones/${milestone.id}`);
    await expect(page.getByRole('heading', { name: 'Done Effect Test' })).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Milestone open item')).toBeVisible();
    await expect(page.getByText('Milestone done item')).toBeVisible();

    // Done item title link should have line-through
    const doneTitle = page.getByRole('link', { name: 'Milestone done item' });
    await expect(doneTitle).toHaveCSS('text-decoration-line', 'line-through');

    // Open item title should NOT have line-through
    const openTitle = page.getByRole('link', { name: 'Milestone open item' });
    await expect(openTitle).not.toHaveCSS('text-decoration-line', 'line-through');
  });

  test('mobile card shows done effect for completed items', async ({ request, testUser, testProject, page }) => {
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Mobile open item',
      type: 'task',
    });

    const doneItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Mobile done item',
      type: 'task',
    });
    await completeWorkItem(request, testUser.token, testProject.key, doneItem.item_number);

    // Set view state to show all statuses
    await showAllStatuses(request, testUser.token, testProject.key);

    // Set mobile viewport before navigation
    await page.setViewportSize({ width: 375, height: 812 });

    await page.goto(`/d/projects/${testProject.key}/items`);
    // Mobile cards use <p> for title; scope to <p> to avoid matching hidden table cells
    const mobileOpenTitle = page.locator('p').getByText('Mobile open item');
    const mobileDoneTitle = page.locator('p').getByText('Mobile done item');
    await expect(mobileOpenTitle).toBeVisible({ timeout: 10000 });
    await expect(mobileDoneTitle).toBeVisible();

    // Done item title should have line-through
    await expect(mobileDoneTitle).toHaveCSS('text-decoration-line', 'line-through');

    // Open item title should NOT have line-through
    await expect(mobileOpenTitle).not.toHaveCSS('text-decoration-line', 'line-through');
  });
});
