import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

test.describe('Milestone on mobile work item detail', () => {
  test('milestone name is visible on mobile detail second row when set', async ({ request, testUser, testProject, page }) => {
    // Create a milestone
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Sprint 42',
    });

    // Create a work item and assign it to the milestone
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Milestone mobile test item',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      milestone_id: milestone.id,
    });

    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    // Navigate to the work item detail page
    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await page.waitForLoadState('networkidle');

    // Verify milestone name is visible in the mobile metadata row (not in sidebar select)
    const metadataRow = page.locator('main span.inline-flex').filter({ hasText: 'Sprint 42' });
    await expect(metadataRow).toBeVisible({ timeout: 10000 });
  });

  test('milestone is not shown on mobile detail when not set', async ({ request, testUser, testProject, page }) => {
    // Create a milestone (but don't assign to item)
    await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Unused Sprint',
    });

    // Create a work item WITHOUT a milestone
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'No milestone mobile test',
      type: 'task',
    });

    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    // Navigate to the work item detail page
    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await page.waitForLoadState('networkidle');

    // Confirm the page loaded by checking the work item title
    await expect(page.getByText('No milestone mobile test')).toBeVisible({ timeout: 10000 });

    // The milestone name should NOT appear in the mobile metadata row
    const milestoneInRow = page.locator('main span.inline-flex').filter({ hasText: 'Unused Sprint' });
    await expect(milestoneInRow).not.toBeVisible();
  });
});
