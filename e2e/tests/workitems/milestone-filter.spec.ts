import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

test.describe('Milestone filter', () => {
  test('selecting all milestones including "no milestone" shows all items', async ({ request, testUser, testProject, page }) => {
    // Create a milestone
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Sprint 1',
    });

    // Create items: one with milestone, one without
    const withMilestone = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Item with milestone',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, withMilestone.item_number, {
      milestone_id: milestone.id,
    });

    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Item without milestone',
      type: 'task',
    });

    // Navigate to work items list
    await page.goto(`/projects/${testProject.key}/items`);

    // Both items should be visible initially (no filter)
    const table = page.getByRole('table');
    await expect(table.getByText('Item with milestone')).toBeVisible({ timeout: 10000 });
    await expect(table.getByText('Item without milestone')).toBeVisible();

    // Open milestone filter dropdown
    const milestoneBtn = page.getByRole('button', { name: /Milestone/i });
    await milestoneBtn.click();

    // Select "No milestone" checkbox
    const noMilestoneCheckbox = page.getByRole('checkbox', { name: /No milestone/i });
    await noMilestoneCheckbox.check();

    // Close dropdown by clicking the button again
    await milestoneBtn.click();

    // Only the item without milestone should be visible
    await expect(table.getByText('Item without milestone')).toBeVisible({ timeout: 10000 });
    await expect(table.getByText('Item with milestone')).not.toBeVisible({ timeout: 5000 });

    // Now also select the "Sprint 1" milestone (selecting ALL options)
    await milestoneBtn.click();
    const sprintCheckbox = page.getByRole('checkbox', { name: 'Sprint 1' });
    await sprintCheckbox.check();
    await milestoneBtn.click();

    // Both items should now be visible (all milestones + no milestone selected)
    await expect(table.getByText('Item with milestone')).toBeVisible({ timeout: 10000 });
    await expect(table.getByText('Item without milestone')).toBeVisible();
  });

  test('selecting only a specific milestone filters correctly', async ({ request, testUser, testProject, page }) => {
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Release 1.0',
    });

    const tagged = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Tagged item',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, tagged.item_number, {
      milestone_id: milestone.id,
    });

    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Untagged item',
      type: 'task',
    });

    await page.goto(`/projects/${testProject.key}/items`);

    const table = page.getByRole('table');

    // Both items visible initially
    await expect(table.getByText('Tagged item', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(table.getByText('Untagged item', { exact: true })).toBeVisible();

    // Select only the "Release 1.0" milestone
    const milestoneBtn = page.getByRole('button', { name: /Milestone/i });
    await milestoneBtn.click();
    const releaseCheckbox = page.getByRole('checkbox', { name: 'Release 1.0' });
    await releaseCheckbox.check();
    await milestoneBtn.click();

    // Only the tagged item should be visible
    await expect(table.getByText('Tagged item', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(table.getByText('Untagged item', { exact: true })).not.toBeVisible({ timeout: 5000 });
  });
});
