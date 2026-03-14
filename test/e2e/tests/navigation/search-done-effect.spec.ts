import { test, expect, getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';

/** Transition a work item to done through valid workflow steps. */
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

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

async function openSearchAndType(page: import('@playwright/test').Page, query: string) {
  await page.keyboard.press('g');
  await page.keyboard.press('k');
  const searchInput = page.getByPlaceholder(/search across/i);
  await expect(searchInput).toBeVisible({ timeout: 3000 });
  await searchInput.fill(query);
  return searchInput;
}

test.describe('Search modal done effect', () => {
  test('completed work items show strikethrough in search results', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const prefix = `SearchDone-${Date.now()}`;

    // Create an open item and a completed item with a shared prefix
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `${prefix} open item`,
      type: 'task',
    });

    const doneItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `${prefix} done item`,
      type: 'task',
    });
    await completeWorkItem(request, testUser.token, testProject.key, doneItem.item_number);

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open search and search for the prefix
    await openSearchAndType(page, prefix);

    // Wait for both results to appear
    const results = page.locator('[data-search-item]');
    await expect(results).toHaveCount(2, { timeout: 10000 });

    // The done item should have line-through on its title text
    const doneResult = results.filter({ hasText: `${prefix} done item` });
    await expect(doneResult.locator('span.truncate')).toHaveCSS('text-decoration-line', 'line-through');

    // The open item should NOT have line-through
    const openResult = results.filter({ hasText: `${prefix} open item` });
    await expect(openResult.locator('span.truncate')).not.toHaveCSS('text-decoration-line', 'line-through');
  });

  test('strikethrough in search results respects user preference', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const prefix = `SearchPref-${Date.now()}`;

    const doneItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `${prefix} completed task`,
      type: 'task',
    });
    await completeWorkItem(request, testUser.token, testProject.key, doneItem.item_number);

    // Disable the strikethrough preference
    await api.setPreference(request, testUser.token, 'strikethrough_completed', false);

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Search for the done item
    await openSearchAndType(page, prefix);

    const result = page.locator('[data-search-item]').first();
    await expect(result).toBeVisible({ timeout: 10000 });

    // With strikethrough disabled, title should NOT have line-through
    await expect(result.locator('span.truncate')).not.toHaveCSS('text-decoration-line', 'line-through');

    // Re-enable the preference for cleanup
    await api.setPreference(request, testUser.token, 'strikethrough_completed', true);
  });

  test('strikethrough works with per-type custom workflow', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const adminToken = getAdminToken();
    const prefix = `SearchCW-${Date.now()}`;

    // Create a custom workflow with a "closed" status (category: done)
    const customWf = await api.createWorkflow(request, adminToken, {
      name: `E2E Custom ${Date.now()}`,
      statuses: [
        { name: 'new', display_name: 'New', category: 'todo', position: 0 },
        { name: 'active', display_name: 'Active', category: 'in_progress', position: 1 },
        { name: 'closed', display_name: 'Closed', category: 'done', position: 2 },
      ],
      transitions: [
        { from_status: 'new', to_status: 'active' },
        { from_status: 'active', to_status: 'closed' },
      ],
    });

    try {
      // Assign the custom workflow to "bug" type in the test project
      await api.setProjectTypeWorkflow(
        request, testUser.token, testProject.key, 'bug', customWf.id,
      );

      // Create a bug and transition it to "closed" via the custom workflow
      const bugItem = await api.createWorkItem(request, testUser.token, testProject.key, {
        title: `${prefix} closed bug`,
        type: 'bug',
      });
      await api.updateWorkItem(request, testUser.token, testProject.key, bugItem.item_number, { status: 'active' });
      await api.updateWorkItem(request, testUser.token, testProject.key, bugItem.item_number, { status: 'closed' });

      // Create an open bug for comparison
      await api.createWorkItem(request, testUser.token, testProject.key, {
        title: `${prefix} open bug`,
        type: 'bug',
      });

      await page.goto(`/d/projects/${testProject.key}/items`);
      await dismissWelcomeModal(page);
      await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

      // Search for the prefix
      await openSearchAndType(page, prefix);

      const results = page.locator('[data-search-item]');
      await expect(results).toHaveCount(2, { timeout: 10000 });

      // The closed bug should have line-through
      const closedResult = results.filter({ hasText: `${prefix} closed bug` });
      await expect(closedResult.locator('span.truncate')).toHaveCSS('text-decoration-line', 'line-through');

      // The open bug should NOT have line-through
      const openResult = results.filter({ hasText: `${prefix} open bug` });
      await expect(openResult.locator('span.truncate')).not.toHaveCSS('text-decoration-line', 'line-through');
    } finally {
      // Cleanup: delete the custom workflow
      await api.deleteWorkflow(request, adminToken, customWf.id).catch(() => {});
    }
  });
});
