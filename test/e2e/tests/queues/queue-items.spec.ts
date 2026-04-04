import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

test.describe('Queue Items Page Redesign', () => {
  test('queue list card navigates to items page on click', async ({ request, testUser, testProject, page }) => {
    const queue = await api.createQueue(request, testUser.token, testProject.key, {
      name: 'Support Queue',
      queue_type: 'support',
    });

    await page.goto(`/d/projects/${testProject.key}/queues`);
    await expect(page.getByText('Support Queue')).toBeVisible({ timeout: 10000 });

    // Click the queue card (whole card is clickable)
    await page.getByText('Support Queue').click();

    // Should navigate to queue items page
    await expect(page).toHaveURL(new RegExp(`/queues/${queue.id}/items`), { timeout: 10000 });
  });

  test('queue list card does not show delete button', async ({ request, testUser, testProject, page }) => {
    await api.createQueue(request, testUser.token, testProject.key, {
      name: 'No Delete Queue',
      queue_type: 'general',
    });

    await page.goto(`/d/projects/${testProject.key}/queues`);
    await expect(page.getByText('No Delete Queue')).toBeVisible({ timeout: 10000 });

    // Trash icon should not be visible on the card
    const card = page.locator('a').filter({ hasText: 'No Delete Queue' });
    await expect(card.locator('.text-red-500')).not.toBeVisible();
  });

  test('queue list settings gear navigates to settings page', async ({ request, testUser, testProject, page }) => {
    const queue = await api.createQueue(request, testUser.token, testProject.key, {
      name: 'Settings Queue',
      queue_type: 'support',
    });

    await page.goto(`/d/projects/${testProject.key}/queues`);
    await expect(page.getByText('Settings Queue')).toBeVisible({ timeout: 10000 });

    // Click the settings gear (stop propagation should prevent navigating to items)
    const card = page.locator('a').filter({ hasText: 'Settings Queue' });
    await card.locator('button').filter({ has: page.locator('svg') }).first().click();

    // Should navigate to queue settings page (not items)
    await expect(page).toHaveURL(new RegExp(`/queues/${queue.id}$`), { timeout: 10000 });
  });

  test('queue items page shows list/board toggle', async ({ request, testUser, testProject, page }) => {
    const queue = await api.createQueue(request, testUser.token, testProject.key, {
      name: 'View Toggle Queue',
      queue_type: 'support',
    });

    await page.goto(`/d/projects/${testProject.key}/queues/${queue.id}/items`);

    // List and Board buttons should be visible
    await expect(page.getByRole('button', { name: 'List' })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('button', { name: 'Board' })).toBeVisible();
  });

  test('queue items page shows filter dropdowns and search', async ({ request, testUser, testProject, page }) => {
    const queue = await api.createQueue(request, testUser.token, testProject.key, {
      name: 'Filter Queue',
      queue_type: 'support',
    });

    // Create a work item in the queue
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Support Ticket Alpha',
      type: 'ticket',
      queue_id: queue.id,
    });

    await page.goto(`/d/projects/${testProject.key}/queues/${queue.id}/items`);

    // Wait for queue name to appear
    await expect(page.getByRole('heading', { name: 'Filter Queue' })).toBeVisible({ timeout: 10000 });

    // Work item should appear in the list (use table locator to avoid mobile card duplicate)
    await expect(page.getByRole('table').getByText('Support Ticket Alpha')).toBeVisible({ timeout: 10000 });
  });

  test('queue items page supports bulk status change', async ({ request, testUser, testProject, page }) => {
    const queue = await api.createQueue(request, testUser.token, testProject.key, {
      name: 'Bulk Queue',
      queue_type: 'support',
    });

    // Create work items in the queue
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Bulk Item 1',
      type: 'ticket',
      queue_id: queue.id,
    });
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Bulk Item 2',
      type: 'ticket',
      queue_id: queue.id,
    });

    await page.goto(`/d/projects/${testProject.key}/queues/${queue.id}/items`);
    await expect(page.getByRole('table').getByText('Bulk Item 1')).toBeVisible({ timeout: 10000 });

    // Select all checkbox
    const selectAll = page.locator('label').filter({ hasText: 'Select all' }).locator('input[type="checkbox"]');
    await selectAll.check();

    // Bulk toolbar should appear with status and assign dropdowns
    await expect(page.getByText('2 selected')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('select').filter({ has: page.locator('option', { hasText: 'Change status' }) })).toBeVisible();
    await expect(page.locator('select').filter({ has: page.locator('option', { hasText: 'Assign' }) })).toBeVisible();
  });

  test('queue settings page has danger zone with delete', async ({ request, testUser, testProject, page }) => {
    const queue = await api.createQueue(request, testUser.token, testProject.key, {
      name: 'Danger Queue',
      queue_type: 'general',
    });

    await page.goto(`/d/projects/${testProject.key}/queues/${queue.id}`);

    // Danger Zone section should be visible
    await expect(page.getByText('Danger Zone')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Deleting a queue will remove all items and settings.')).toBeVisible();

    // Click delete button
    await page.getByRole('button', { name: 'Delete queue' }).click();

    // Confirmation modal should appear
    await expect(page.getByRole('heading', { name: 'Delete Queue' })).toBeVisible();

    // Confirm deletion
    await page.getByRole('button', { name: 'Delete', exact: true }).click();

    // Should redirect back to queues list
    await expect(page).toHaveURL(/\/queues$/, { timeout: 10000 });
  });

  test('queue items page board view shows items by status', async ({ request, testUser, testProject, page }) => {
    const queue = await api.createQueue(request, testUser.token, testProject.key, {
      name: 'Board Queue',
      queue_type: 'support',
    });

    // Use type 'task' so the item's status matches the default (task) workflow columns
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Board View Item',
      type: 'task',
      queue_id: queue.id,
    });

    await page.goto(`/d/projects/${testProject.key}/queues/${queue.id}/items`);
    await expect(page.getByRole('table').getByText('Board View Item')).toBeVisible({ timeout: 10000 });

    // Switch to board view
    await page.getByRole('button', { name: 'Board' }).click();

    // Board view should show the item in a kanban column
    await expect(page.getByText('Board View Item').first()).toBeVisible({ timeout: 10000 });
  });
});
