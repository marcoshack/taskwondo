import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

test.describe('Time tracking', () => {
  test('log time entry via API and verify in Time tab', async ({ request, testUser, testProject, page }) => {
    // Create a work item
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Time tracking test',
      type: 'task',
    });

    // Log a time entry via API
    await api.createTimeEntry(request, testUser.token, testProject.key, item.item_number, {
      started_at: new Date().toISOString(),
      duration_seconds: 5400, // 1h 30m
      description: 'Worked on feature',
    });

    // Navigate to the work item detail page
    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to the Time tab
    await page.getByRole('button', { name: 'Time', exact: true }).click();

    // Verify the time entry appears in the entry list
    const entry = page.locator('.group\\/entry').first();
    await expect(entry.getByText('1h 30m')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Worked on feature')).toBeVisible();
  });

  test('log time entry via UI form', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'UI time log test',
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to the Time tab
    await page.getByRole('button', { name: 'Time', exact: true }).click();

    // Wait for the no-entries message to confirm tab is loaded
    await expect(page.getByText('No time entries yet.')).toBeVisible({ timeout: 10000 });

    // Fill in the log time form — natural duration format
    await page.getByPlaceholder('e.g. 30m, 2h').fill('2h 15m');

    // Fill description
    await page.getByPlaceholder('What did you work on?').first().fill('E2E test entry');

    // Submit
    await page.getByRole('button', { name: 'Log Time' }).click();

    // Verify the entry appears in the entry list
    const entry = page.locator('.group\\/entry').first();
    await expect(entry.getByText('2h 15m')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('E2E test entry')).toBeVisible();

    // Verify via API
    const timeData = await api.listTimeEntries(request, testUser.token, testProject.key, item.item_number);
    expect(timeData.entries).toHaveLength(1);
    expect(timeData.entries[0].duration_seconds).toBe(8100); // 2h 15m = 8100s
    expect(timeData.total_logged_seconds).toBe(8100);
  });

  test('delete time entry via UI', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Time entry deletion test',
      type: 'task',
    });

    // Create a time entry via API
    await api.createTimeEntry(request, testUser.token, testProject.key, item.item_number, {
      started_at: new Date().toISOString(),
      duration_seconds: 3600,
      description: 'Entry to delete',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to the Time tab
    await page.getByRole('button', { name: 'Time', exact: true }).click();

    // Wait for the entry to appear
    const entry = page.locator('.group\\/entry').first();
    await expect(entry.getByText('Entry to delete')).toBeVisible({ timeout: 10000 });

    // Hover over the entry to reveal the delete button
    await entry.hover();

    // Click the delete button (trash icon)
    const deleteBtn = entry.locator('button').filter({ has: page.locator('svg path[d*="M3 4.5h10"]') });
    await deleteBtn.click();

    // Confirm deletion in the modal
    const modal = page.locator('.fixed.inset-0.z-50');
    await expect(modal.getByRole('heading', { name: 'Delete Time Entry', exact: true })).toBeVisible({ timeout: 5000 });
    await modal.getByRole('button', { name: 'Delete', exact: true }).click();

    // Entry should disappear
    await expect(page.getByText('Entry to delete')).not.toBeVisible({ timeout: 5000 });

    // Verify via API
    const timeData = await api.listTimeEntries(request, testUser.token, testProject.key, item.item_number);
    expect(timeData.entries).toHaveLength(0);
    expect(timeData.total_logged_seconds).toBe(0);
  });

  test('set estimate and view time summary in sidebar', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Estimate test',
      type: 'task',
    });

    // Set estimate via API
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      estimated_seconds: 7200, // 2h
    });

    // Log some time
    await api.createTimeEntry(request, testUser.token, testProject.key, item.item_number, {
      started_at: new Date().toISOString(),
      duration_seconds: 3600, // 1h
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await dismissWelcomeModal(page);

    // The sidebar should show the estimate and logged time
    const sidebar = page.locator('.hidden.lg\\:block');
    await expect(sidebar.getByText('Estimate')).toBeVisible({ timeout: 10000 });
    await expect(sidebar.getByText('Logged')).toBeVisible();

    // The progress bar should be visible (50% — 1h out of 2h)
    const progressBar = sidebar.locator('.bg-indigo-500');
    await expect(progressBar).toBeVisible();
  });
});
