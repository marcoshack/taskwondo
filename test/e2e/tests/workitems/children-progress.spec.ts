import { test, expect } from '../../lib/fixtures';
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

test.describe('Children work items progress', () => {
  test('shows progress bar in header when item has children', async ({ request, testUser, testProject, page }) => {
    // Create parent and two children
    const parent = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Parent with children',
      type: 'task',
    });
    const child1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Child one',
      type: 'task',
    });
    const child2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Child two',
      type: 'task',
    });

    // Create parent_of relations
    await api.createRelation(request, testUser.token, testProject.key, parent.item_number, {
      target_display_id: child1.display_id,
      relation_type: 'parent_of',
    });
    await api.createRelation(request, testUser.token, testProject.key, parent.item_number, {
      target_display_id: child2.display_id,
      relation_type: 'parent_of',
    });

    // Complete one child
    await completeWorkItem(request, testUser.token, testProject.key, child1.item_number);

    await page.goto(`/d/projects/${testProject.key}/items/${parent.item_number}`);
    await dismissWelcomeModal(page);

    // Verify progress text is shown in the header (desktop) — use .first() since text appears in both desktop and mobile
    await expect(page.getByText('1/2 completed').first()).toBeVisible({ timeout: 10000 });
  });

  test('relations tab separates children from other relations', async ({ request, testUser, testProject, page }) => {
    const parent = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Parent item',
      type: 'task',
    });
    const child = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Child task',
      type: 'task',
    });
    const related = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Related task',
      type: 'task',
    });

    // Create parent_of + relates_to relations
    await api.createRelation(request, testUser.token, testProject.key, parent.item_number, {
      target_display_id: child.display_id,
      relation_type: 'parent_of',
    });
    await api.createRelation(request, testUser.token, testProject.key, parent.item_number, {
      target_display_id: related.display_id,
      relation_type: 'relates_to',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${parent.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to Relations tab
    await page.getByRole('button', { name: 'Relations', exact: true }).click();

    // Verify Children section header is visible
    await expect(page.getByText('Children')).toBeVisible({ timeout: 10000 });

    // Verify Other Relations section header is visible
    await expect(page.getByText('Other Relations')).toBeVisible();

    // Verify both items are shown
    await expect(page.getByText(child.display_id)).toBeVisible();
    await expect(page.getByText(related.display_id)).toBeVisible();
  });

  test('completed children show done effect in relations tab', async ({ request, testUser, testProject, page }) => {
    const parent = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Parent for done effect',
      type: 'task',
    });
    const openChild = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Open child task',
      type: 'task',
    });
    const doneChild = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Done child task',
      type: 'task',
    });

    await api.createRelation(request, testUser.token, testProject.key, parent.item_number, {
      target_display_id: openChild.display_id,
      relation_type: 'parent_of',
    });
    await api.createRelation(request, testUser.token, testProject.key, parent.item_number, {
      target_display_id: doneChild.display_id,
      relation_type: 'parent_of',
    });

    // Complete one child
    await completeWorkItem(request, testUser.token, testProject.key, doneChild.item_number);

    await page.goto(`/d/projects/${testProject.key}/items/${parent.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to Relations tab
    await page.getByRole('button', { name: 'Relations', exact: true }).click();
    await expect(page.getByText(openChild.display_id)).toBeVisible({ timeout: 10000 });

    // Done child title should have line-through
    const doneTitle = page.getByRole('link', { name: 'Done child task' });
    await expect(doneTitle).toHaveCSS('text-decoration-line', 'line-through');

    // Open child title should NOT have line-through
    const openTitle = page.getByRole('link', { name: 'Open child task' });
    await expect(openTitle).not.toHaveCSS('text-decoration-line', 'line-through');
  });

  test('done effect applies to non-child relations too', async ({ request, testUser, testProject, page }) => {
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Source item',
      type: 'task',
    });
    const doneRelated = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Completed related item',
      type: 'task',
    });

    await api.createRelation(request, testUser.token, testProject.key, item1.item_number, {
      target_display_id: doneRelated.display_id,
      relation_type: 'relates_to',
    });

    await completeWorkItem(request, testUser.token, testProject.key, doneRelated.item_number);

    await page.goto(`/d/projects/${testProject.key}/items/${item1.item_number}`);
    await dismissWelcomeModal(page);

    await page.getByRole('button', { name: 'Relations', exact: true }).click();
    await expect(page.getByText(doneRelated.display_id)).toBeVisible({ timeout: 10000 });

    // Done related item title should have line-through
    const doneTitle = page.getByRole('link', { name: 'Completed related item' });
    await expect(doneTitle).toHaveCSS('text-decoration-line', 'line-through');
  });

  test('progress bar updates when all children are completed', async ({ request, testUser, testProject, page }) => {
    const parent = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Parent full progress',
      type: 'task',
    });
    const child = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Only child',
      type: 'task',
    });

    await api.createRelation(request, testUser.token, testProject.key, parent.item_number, {
      target_display_id: child.display_id,
      relation_type: 'parent_of',
    });

    await completeWorkItem(request, testUser.token, testProject.key, child.item_number);

    await page.goto(`/d/projects/${testProject.key}/items/${parent.item_number}`);
    await dismissWelcomeModal(page);

    // Should show 1/1 completed — use .first() since text appears in both desktop and mobile
    await expect(page.getByText('1/1 completed').first()).toBeVisible({ timeout: 10000 });
  });
});
