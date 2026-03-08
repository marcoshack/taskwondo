import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

test.describe('Milestone time tracking', () => {
  test('shows estimated and spent time on milestone card', async ({ request, testUser, testProject, page }) => {
    // Create a milestone
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Time Tracking Milestone',
    });

    // Create two work items and assign them to the milestone
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Task with estimate and time',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, item1.item_number, {
      milestone_id: milestone.id,
      estimated_seconds: 7200, // 2h estimate
    });

    const item2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Task with only time logged',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, item2.item_number, {
      milestone_id: milestone.id,
      estimated_seconds: 3600, // 1h estimate
    });

    // Log time entries
    await api.createTimeEntry(request, testUser.token, testProject.key, item1.item_number, {
      started_at: new Date().toISOString(),
      duration_seconds: 3600, // 1h spent
    });
    await api.createTimeEntry(request, testUser.token, testProject.key, item2.item_number, {
      started_at: new Date().toISOString(),
      duration_seconds: 1800, // 30m spent
    });

    // Navigate to milestones page
    await page.goto(`/d/projects/${testProject.key}/milestones`);

    // Wait for the milestone to render
    await expect(page.getByText('Time Tracking Milestone')).toBeVisible({ timeout: 10000 });

    // Verify the milestone card shows time data
    // Total estimate: 2h + 1h = 3h, Total spent: 1h + 30m = 1h 30m
    await expect(page.getByText('Est: 3h')).toBeVisible();
    await expect(page.getByText('Spent: 1h 30m')).toBeVisible();
  });

  test('hides time section when no estimates or time logged', async ({ request, testUser, testProject, page }) => {
    // Create a milestone with a work item that has no estimate or time logged
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'No Time Milestone',
    });

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Task without time data',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      milestone_id: milestone.id,
    });

    await page.goto(`/d/projects/${testProject.key}/milestones`);

    // The milestone should be visible
    await expect(page.getByText('No Time Milestone')).toBeVisible({ timeout: 10000 });

    // But no time info should be shown (no "Est:" or "Spent:" text)
    await expect(page.getByText(/Est:/)).not.toBeVisible();
    await expect(page.getByText(/Spent:/)).not.toBeVisible();
  });

  test('shows over-budget indicator when spent exceeds estimate', async ({ request, testUser, testProject, page }) => {
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Over Budget Milestone',
    });

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Over budget task',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      milestone_id: milestone.id,
      estimated_seconds: 3600, // 1h estimate
    });

    // Log 2h (over the 1h estimate)
    await api.createTimeEntry(request, testUser.token, testProject.key, item.item_number, {
      started_at: new Date().toISOString(),
      duration_seconds: 7200, // 2h spent
    });

    await page.goto(`/d/projects/${testProject.key}/milestones`);

    // Wait for the milestone to render
    await expect(page.getByText('Over Budget Milestone')).toBeVisible({ timeout: 10000 });

    // Verify time data is visible
    await expect(page.getByText('Est: 1h')).toBeVisible();
    await expect(page.getByText('Spent: 2h')).toBeVisible();
  });
});
