import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

test.describe('Milestone Dashboard', () => {
  test('navigates to dashboard from milestone list and shows all sections', async ({ request, testUser, testProject, page }) => {
    // Create milestone with due date
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Dashboard Test',
      due_date: '2026-06-30',
    });

    // Create work items of different types and priorities
    const task = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'A task item',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, task.item_number, {
      milestone_id: milestone.id,
      priority: 'high',
    });

    const bug = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'A bug item',
      type: 'bug',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, bug.item_number, {
      milestone_id: milestone.id,
      priority: 'critical',
    });

    const ticket = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'A ticket item',
      type: 'ticket',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, ticket.item_number, {
      milestone_id: milestone.id,
      priority: 'low',
    });

    // Close the bug: open → in_progress → in_review → done
    await api.updateWorkItem(request, testUser.token, testProject.key, bug.item_number, {
      status: 'in_progress',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, bug.item_number, {
      status: 'in_review',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, bug.item_number, {
      status: 'done',
    });

    // Navigate to milestones list
    await page.goto(`/d/projects/${testProject.key}/milestones`);
    await expect(page.getByText('Dashboard Test')).toBeVisible({ timeout: 10000 });

    // Click milestone name to navigate to dashboard
    await page.getByText('Dashboard Test').click();
    await page.waitForURL(/\/milestones\//);

    // Verify header section — use heading role to avoid breadcrumb ambiguity
    await expect(page.getByRole('heading', { name: 'Dashboard Test' })).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('1/3 items')).toBeVisible();
    await expect(page.getByText('33%')).toBeVisible();

    // Summary tab is the default and holds the counters and breakdown charts
    await expect(page.getByRole('button', { name: 'Summary', exact: true })).toBeVisible();
    await expect(page.getByText('By Type')).toBeVisible();
    await expect(page.getByText('By Priority')).toBeVisible();

    // Work items live behind their own tab
    await expect(page.getByText('A task item')).not.toBeVisible();
    await page.getByRole('button', { name: 'Work Items', exact: true }).click();
    await expect(page.getByText('A task item')).toBeVisible();
    await expect(page.getByText('A bug item')).toBeVisible();
    await expect(page.getByText('A ticket item')).toBeVisible();

    // The header and its progress bar stay visible across tabs
    await expect(page.getByRole('heading', { name: 'Dashboard Test' })).toBeVisible();
    await expect(page.getByText('1/3 items')).toBeVisible();

    // The breakdown charts are gone while the Work Items tab is active
    await expect(page.getByText('By Type')).not.toBeVisible();
  });

  test('shows empty state for milestone with no items', async ({ request, testUser, testProject, page }) => {
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Empty Milestone',
    });

    await page.goto(`/d/projects/${testProject.key}/milestones/${milestone.id}`);

    await expect(page.getByRole('heading', { name: 'Empty Milestone' })).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('No items')).toBeVisible();

    await page.getByRole('button', { name: 'Work Items', exact: true }).click();
    await expect(page.getByText('No work items in this milestone yet.')).toBeVisible();
  });

  test('shows time tracking when items have estimates and time entries', async ({ request, testUser, testProject, page }) => {
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Time Tracking Test',
    });

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Estimated task',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      milestone_id: milestone.id,
      estimated_seconds: 7200, // 2 hours
    });

    // Log 1 hour of time
    await api.createTimeEntry(request, testUser.token, testProject.key, item.item_number, {
      started_at: new Date().toISOString(),
      duration_seconds: 3600,
    });

    await page.goto(`/d/projects/${testProject.key}/milestones/${milestone.id}`);

    // Wait for the heading to confirm page loaded
    await expect(page.getByRole('heading', { name: 'Time Tracking Test' })).toBeVisible({ timeout: 10000 });

    // Verify time tracking section appears (use exact match to avoid matching milestone name)
    await expect(page.getByRole('heading', { name: 'Time Tracking', exact: true })).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Estimated', { exact: true })).toBeVisible();
    await expect(page.getByText('Spent', { exact: true })).toBeVisible();
    await expect(page.getByText('Remaining', { exact: true })).toBeVisible();
  });

  test('breadcrumb navigates back to milestones list', async ({ request, testUser, testProject, page }) => {
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Breadcrumb Test',
    });

    await page.goto(`/d/projects/${testProject.key}/milestones/${milestone.id}`);
    await expect(page.getByRole('heading', { name: 'Breadcrumb Test' })).toBeVisible({ timeout: 10000 });

    // Click breadcrumb back to milestones list
    await page.getByRole('link', { name: 'Milestones' }).first().click();
    await page.waitForURL(/\/milestones$/);
    await expect(page.getByText('Track progress toward goals')).toBeVisible({ timeout: 10000 });
  });

  test('edit button opens modal and changes persist', async ({ request, testUser, testProject, page }) => {
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Edit Test',
    });

    await page.goto(`/d/projects/${testProject.key}/milestones/${milestone.id}`);
    await expect(page.getByRole('heading', { name: 'Edit Test' })).toBeVisible({ timeout: 10000 });

    // Click edit button (pencil icon)
    await page.getByRole('button').filter({ has: page.locator('svg.lucide-pencil') }).click();

    // Verify modal opens
    await expect(page.getByText('Edit Milestone')).toBeVisible();

    // Change name
    const nameInput = page.getByRole('textbox').first();
    await nameInput.clear();
    await nameInput.fill('Updated Name');

    // Save
    await page.getByRole('button', { name: 'Save' }).click();

    // Verify name updated on the dashboard
    await expect(page.getByRole('heading', { name: 'Updated Name' })).toBeVisible({ timeout: 10000 });
  });

  test('active tab is kept in the URL and survives a reload', async ({ request, testUser, testProject, page }) => {
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Tab State Test',
    });

    await page.goto(`/d/projects/${testProject.key}/milestones/${milestone.id}`);
    await expect(page.getByRole('heading', { name: 'Tab State Test' })).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: 'Work Items', exact: true }).click();
    await page.waitForURL(/[?&]tab=workItems/);

    await page.reload();
    await expect(page.getByText('No work items in this milestone yet.')).toBeVisible({ timeout: 10000 });
  });

  test('header collapses to a single row and the choice applies to every milestone', async ({ request, testUser, testProject, page }) => {
    const first = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Collapse First',
      description: 'First milestone description',
    });
    const second = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Collapse Second',
      description: 'Second milestone description',
    });

    await page.goto(`/d/projects/${testProject.key}/milestones/${first.id}`);

    // Expanded by default: the description and the standalone progress row show
    await expect(page.getByText('First milestone description')).toBeVisible({ timeout: 10000 });

    await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('/user/preferences/milestone_header_collapsed') && r.request().method() === 'PUT',
      ),
      page.getByRole('button', { name: 'Collapse header' }).click(),
    ]);
    await expect(page.getByText('First milestone description')).not.toBeVisible();

    // The preference is global, so a different milestone opens collapsed too
    await page.goto(`/d/projects/${testProject.key}/milestones/${second.id}`);
    await expect(page.getByRole('heading', { name: 'Collapse Second' })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('button', { name: 'Expand header' })).toBeVisible();
    await expect(page.getByText('Second milestone description')).not.toBeVisible();

    // The items bar stays in the collapsed row, and the tabs still work
    await expect(page.getByText('No items')).toBeVisible();
    await page.getByRole('button', { name: 'Work Items', exact: true }).click();
    await expect(page.getByText('No work items in this milestone yet.')).toBeVisible();

    await page.getByRole('button', { name: 'Expand header' }).click();
    await expect(page.getByText('Second milestone description')).toBeVisible();
  });

  test('clicking the header row toggles the header', async ({ request, testUser, testProject, page }) => {
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Header Click Test',
      description: 'Click the row, not the button',
    });

    await page.goto(`/d/projects/${testProject.key}/milestones/${milestone.id}`);
    await expect(page.getByText('Click the row, not the button')).toBeVisible({ timeout: 10000 });

    // Click the milestone name — part of the header row, not a button
    await page.getByRole('heading', { name: 'Header Click Test' }).click();
    await expect(page.getByText('Click the row, not the button')).not.toBeVisible();

    await page.getByRole('heading', { name: 'Header Click Test' }).click();
    await expect(page.getByText('Click the row, not the button')).toBeVisible();

    // The edit button inside the row still edits instead of toggling: the modal
    // opens and the header stays expanded (its control still reads "Collapse header")
    await page.getByRole('button').filter({ has: page.locator('svg.lucide-pencil') }).click();
    await expect(page.getByText('Edit Milestone')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Collapse header' })).toBeVisible();
  });

  test('settings tab deletes the milestone and returns to the list', async ({ request, testUser, testProject, page }) => {
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Delete From Settings',
    });

    await page.goto(`/d/projects/${testProject.key}/milestones/${milestone.id}`);
    await expect(page.getByRole('heading', { name: 'Delete From Settings' })).toBeVisible({ timeout: 10000 });

    // The Settings tab only appears once the caller's project role has loaded
    const settingsTab = page.getByRole('button', { name: 'Settings', exact: true });
    await expect(settingsTab).toBeVisible({ timeout: 10000 });
    await settingsTab.click();
    await expect(page.getByText('Danger Zone')).toBeVisible();

    // Open the confirmation modal from the danger zone, then confirm
    await page.getByRole('button', { name: 'Delete Milestone', exact: true }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await dialog.getByRole('button', { name: 'Delete', exact: true }).click();

    await page.waitForURL(/\/milestones$/);
    await expect(page.getByText('Delete From Settings')).not.toBeVisible();
  });

  test('work item links navigate to detail page', async ({ request, testUser, testProject, page }) => {
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Link Test',
    });

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Linked item',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      milestone_id: milestone.id,
    });

    await page.goto(`/d/projects/${testProject.key}/milestones/${milestone.id}`);
    await expect(page.getByRole('heading', { name: 'Link Test' })).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: 'Work Items', exact: true }).click();
    await expect(page.getByText('Linked item')).toBeVisible();

    // Click item title to navigate to detail
    await page.getByText('Linked item').click();
    await page.waitForURL(/\/items\/\d+/);
  });

  test('stats endpoint returns correct type and priority breakdowns', async ({ request, testUser, testProject }) => {
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Stats API Test',
    });

    // Create items of different types
    const task = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Task 1',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, task.item_number, {
      milestone_id: milestone.id,
      priority: 'high',
    });

    const bug = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Bug 1',
      type: 'bug',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, bug.item_number, {
      milestone_id: milestone.id,
      priority: 'critical',
    });

    // Close the bug
    await api.updateWorkItem(request, testUser.token, testProject.key, bug.item_number, {
      status: 'in_progress',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, bug.item_number, {
      status: 'in_review',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, bug.item_number, {
      status: 'done',
    });

    // Call stats API directly
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';
    const res = await request.get(`${BASE_URL}/api/v1/default/projects/${testProject.key}/milestones/${milestone.id}/stats`, {
      headers: { Authorization: `Bearer ${testUser.token}` },
    });
    expect(res.ok()).toBeTruthy();

    const body = await res.json();
    const stats = body.data;

    // Verify type breakdown
    expect(stats.by_type.task.open).toBe(1);
    expect(stats.by_type.task.closed).toBe(0);
    expect(stats.by_type.bug.open).toBe(0);
    expect(stats.by_type.bug.closed).toBe(1);

    // Verify priority breakdown
    expect(stats.by_priority.high.open).toBe(1);
    expect(stats.by_priority.high.closed).toBe(0);
    expect(stats.by_priority.critical.open).toBe(0);
    expect(stats.by_priority.critical.closed).toBe(1);
  });
});
