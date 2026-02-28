import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

/** Transition a work item to done through valid workflow steps: open → in_progress → in_review → done */
async function completeWorkItem(
  request: import('@playwright/test').APIRequestContext,
  token: string,
  projectKey: string,
  itemNumber: number,
) {
  await api.updateWorkItem(request, token, projectKey, itemNumber, { status: 'in_progress' });
  await api.updateWorkItem(request, token, projectKey, itemNumber, { status: 'in_review' });
  await api.updateWorkItem(request, token, projectKey, itemNumber, { status: 'done' });
}

test.describe('Inbox', () => {
  test('add item to inbox via API and verify on inbox page', async ({ request, testUser, testProject, page }) => {
    // Create a work item
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Inbox test item',
      type: 'task',
    });

    // Add to inbox via API
    await api.addToInbox(request, testUser.token, item.id);

    // Verify count via API
    const count = await api.getInboxCount(request, testUser.token);
    expect(count).toBe(1);

    // Navigate to inbox page
    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    // Verify the item appears in the table
    await expect(page.getByRole('table').getByText('Inbox test item')).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('table').getByText(item.display_id)).toBeVisible();
  });

  test('remove item from inbox via API', async ({ request, testUser, testProject }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Remove test',
      type: 'task',
    });

    await api.addToInbox(request, testUser.token, item.id);

    // Get inbox item ID from list
    const list0 = await api.listInboxItems(request, testUser.token);
    const inboxItemId = list0.items[0].id;

    // Remove from inbox
    await api.removeFromInbox(request, testUser.token, inboxItemId);

    // Verify count is 0
    const count = await api.getInboxCount(request, testUser.token);
    expect(count).toBe(0);

    // Verify list is empty
    const list = await api.listInboxItems(request, testUser.token);
    expect(list.items).toHaveLength(0);
  });

  test('reorder inbox items via API', async ({ request, testUser, testProject }) => {
    // Create and add 3 items
    for (let i = 1; i <= 3; i++) {
      const item = await api.createWorkItem(request, testUser.token, testProject.key, {
        title: `Reorder item ${i}`,
        type: 'task',
      });
      await api.addToInbox(request, testUser.token, item.id);
    }

    // Get inbox items to find IDs
    const listBefore = await api.listInboxItems(request, testUser.token);
    // Items are ordered by position: item1 (1000), item2 (2000), item3 (3000)
    const item3InboxId = listBefore.items[2].id;

    // Move item 3 to position 500 (before item 1 at 1000)
    await api.reorderInboxItem(request, testUser.token, item3InboxId, 500);

    // Verify order: item 3 should be first
    const list = await api.listInboxItems(request, testUser.token);
    expect(list.items).toHaveLength(3);
    expect(list.items[0].title).toBe('Reorder item 3');
    expect(list.items[1].title).toBe('Reorder item 1');
    expect(list.items[2].title).toBe('Reorder item 2');
  });

  test('inbox count badge shows in navigation', async ({ request, testUser, testProject, page }) => {
    // Create and add 2 items to inbox
    for (let i = 1; i <= 2; i++) {
      const item = await api.createWorkItem(request, testUser.token, testProject.key, {
        title: `Badge item ${i}`,
        type: 'task',
      });
      await api.addToInbox(request, testUser.token, item.id);
    }

    // Navigate to any page
    await page.goto(`/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Verify the inbox button has a count badge with "2"
    const inboxBtn = page.getByRole('button', { name: 'Inbox' });
    await expect(inboxBtn).toBeVisible({ timeout: 10000 });
    await expect(inboxBtn.locator('span.rounded-full')).toHaveText('2', { timeout: 10000 });
  });

  test('remove item from inbox via UI', async ({ request, testUser, testProject, page }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'UI remove test',
      type: 'task',
    });
    await api.addToInbox(request, testUser.token, item.id);

    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    // Wait for the item to appear in the table
    const table = page.getByRole('table');
    await expect(table.getByText('UI remove test')).toBeVisible({ timeout: 10000 });

    // Hover over the row to reveal the remove button and click it
    const row = table.locator('tr').filter({ hasText: 'UI remove test' });
    await row.hover();
    // The remove button is the last button in the row
    const removeBtn = row.locator('button').last();
    await removeBtn.click();

    // Item should disappear
    await expect(table.getByText('UI remove test')).not.toBeVisible({ timeout: 5000 });

    // Verify via API
    const list = await api.listInboxItems(request, testUser.token);
    expect(list.items).toHaveLength(0);
  });

  test('completed items are excluded by default', async ({ request, testUser, testProject }) => {
    // Create a work item and add to inbox
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Complete me',
      type: 'task',
    });
    await api.addToInbox(request, testUser.token, item.id);

    // Mark the work item as done (via valid workflow transitions)
    await completeWorkItem(request, testUser.token, testProject.key, item.item_number);

    // List inbox without include_completed — should be empty
    const list = await api.listInboxItems(request, testUser.token);
    expect(list.items).toHaveLength(0);

    // List with include_completed — should have the item
    const listAll = await api.listInboxItems(request, testUser.token, { include_completed: true });
    expect(listAll.items).toHaveLength(1);
    expect(listAll.items[0].status_category).toBe('done');
  });

  test('clear completed removes done items', async ({ request, testUser, testProject }) => {
    // Create 2 items, complete one
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Keep me',
      type: 'task',
    });
    const item2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Complete and clear',
      type: 'task',
    });
    await api.addToInbox(request, testUser.token, item1.id);
    await api.addToInbox(request, testUser.token, item2.id);

    // Complete item2 (via valid workflow transitions)
    await completeWorkItem(request, testUser.token, testProject.key, item2.item_number);

    // Clear completed
    const removed = await api.clearCompletedInbox(request, testUser.token);
    expect(removed).toBe(1);

    // Only item1 should remain
    const list = await api.listInboxItems(request, testUser.token, { include_completed: true });
    expect(list.items).toHaveLength(1);
    expect(list.items[0].title).toBe('Keep me');
  });

  test('search inbox items', async ({ request, testUser, testProject }) => {
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Fix login bug',
      type: 'bug',
    });
    const item2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Add dashboard feature',
      type: 'task',
    });
    await api.addToInbox(request, testUser.token, item1.id);
    await api.addToInbox(request, testUser.token, item2.id);

    // Search for "login"
    const result = await api.listInboxItems(request, testUser.token, { search: 'login' });
    expect(result.items).toHaveLength(1);
    expect(result.items[0].title).toBe('Fix login bug');
  });

  test('inbox is per-user isolated', async ({ request, testUser, testProject }) => {
    // Create a work item and add to testUser's inbox
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'User A item',
      type: 'task',
    });
    await api.addToInbox(request, testUser.token, item.id);

    // Create a second user
    const adminToken = (await import('../../lib/fixtures')).getAdminToken();
    const uniqueId = require('crypto').randomUUID().slice(0, 8);
    const user2Data = await api.createUser(request, adminToken, `e2e-inbox-${uniqueId}@test.local`, `Inbox User ${uniqueId}`);
    const tempLogin = await api.login(request, `e2e-inbox-${uniqueId}@test.local`, user2Data.temporary_password);
    await api.changePassword(request, tempLogin.token, user2Data.temporary_password, 'TestPass123!');
    const user2Login = await api.login(request, `e2e-inbox-${uniqueId}@test.local`, 'TestPass123!');

    // User 2's inbox should be empty
    const user2List = await api.listInboxItems(request, user2Login.token);
    expect(user2List.items).toHaveLength(0);

    // User 1's inbox should have 1 item
    const user1List = await api.listInboxItems(request, testUser.token);
    expect(user1List.items).toHaveLength(1);

    // Cleanup
    await api.deactivateUser(request, adminToken, user2Login.user.id).catch(() => {});
  });

  test('duplicate add returns conflict error', async ({ request, testUser, testProject }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Duplicate test',
      type: 'task',
    });
    await api.addToInbox(request, testUser.token, item.id);

    // Try adding again — should fail
    const res = await request.post(`${process.env.BASE_URL || 'http://localhost:5173'}/api/v1/user/inbox`, {
      headers: { Authorization: `Bearer ${testUser.token}` },
      data: { work_item_id: item.id },
    });
    expect(res.status()).toBe(409);
  });

  test('sidebar navigation between Inbox, Feed, and Watchlist', async ({ page }) => {
    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    // Should be on inbox page
    await expect(page).toHaveURL(/\/user\/inbox/);

    // Click Feed in sidebar
    const sidebar = page.locator('nav.hidden.sm\\:block');
    await sidebar.getByText('Feed').click();
    await expect(page).toHaveURL(/\/user\/feed/);
    await expect(page.getByText('Feeds coming soon')).toBeVisible();

    // Click Watchlist in sidebar
    await sidebar.getByText('Watchlist').click();
    await expect(page).toHaveURL(/\/user\/watchlist/);
    // Watchlist is now a real page — verify the heading or empty state is visible
    await expect(page.getByText('Watchlist').first()).toBeVisible();

    // Click Inbox in sidebar to go back
    await sidebar.getByText('Inbox').click();
    await expect(page).toHaveURL(/\/user\/inbox/);
  });

  test('sidebar collapse and expand', async ({ page }) => {
    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    const sidebar = page.locator('nav.hidden.sm\\:block');
    await expect(sidebar).toBeVisible();

    // Sidebar should start expanded (w-48 class)
    await expect(sidebar).toHaveClass(/w-48/);

    // Click collapse button
    const collapseBtn = sidebar.getByRole('button', { name: /collapse/i });
    await collapseBtn.click();

    // Sidebar should be collapsed (w-14 class)
    await expect(sidebar).toHaveClass(/w-14/, { timeout: 3000 });

    // Click expand button
    const expandBtn = sidebar.getByRole('button', { name: /expand/i });
    await expandBtn.click();

    // Sidebar should be expanded again
    await expect(sidebar).toHaveClass(/w-48/, { timeout: 3000 });
  });

  test('keyboard shortcut [ toggles sidebar', async ({ page }) => {
    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    const sidebar = page.locator('nav.hidden.sm\\:block');

    // Sidebar should start expanded
    await expect(sidebar).toHaveClass(/w-48/);

    // Press [ to collapse
    await page.keyboard.press('[');
    await expect(sidebar).toHaveClass(/w-14/, { timeout: 3000 });

    // Press [ again to expand
    await page.keyboard.press('[');
    await expect(sidebar).toHaveClass(/w-48/, { timeout: 3000 });
  });

  test('reorder items via up/down arrow buttons', async ({ request, testUser, testProject, page }) => {
    // Create 3 items and add to inbox
    const items = [];
    for (let i = 1; i <= 3; i++) {
      const item = await api.createWorkItem(request, testUser.token, testProject.key, {
        title: `Order item ${i}`,
        type: 'task',
      });
      await api.addToInbox(request, testUser.token, item.id);
      items.push(item);
    }

    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    // Wait for all 3 items in the table
    const table = page.getByRole('table');
    await expect(table.getByText('Order item 1')).toBeVisible({ timeout: 10000 });
    await expect(table.getByText('Order item 3')).toBeVisible();

    const rows = table.locator('tbody tr');
    await expect(rows).toHaveCount(3);

    // Item 3 is last — click "Move up" to move it above item 2
    const moveUpBtn = rows.nth(2).getByRole('button', { name: 'Move up' });
    await moveUpBtn.click();

    // Wait a moment for reorder to complete
    await page.waitForTimeout(1000);

    // Verify new order via API: item 1, item 3, item 2
    const list = await api.listInboxItems(request, testUser.token);
    expect(list.items[0].title).toBe('Order item 1');
    expect(list.items[1].title).toBe('Order item 3');
    expect(list.items[2].title).toBe('Order item 2');
  });

  test('show/hide completed toggle', async ({ request, testUser, testProject, page }) => {
    // Create 2 items, add both to inbox
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Active item for toggle',
      type: 'task',
    });
    const item2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Completed item for toggle',
      type: 'task',
    });
    await api.addToInbox(request, testUser.token, item1.id);
    await api.addToInbox(request, testUser.token, item2.id);

    // Complete item2 via workflow transitions
    await completeWorkItem(request, testUser.token, testProject.key, item2.item_number);

    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    const table = page.getByRole('table');

    // With auto-remove ON (default), completed item should be hidden
    await expect(table.getByText('Active item for toggle')).toBeVisible({ timeout: 10000 });
    await expect(table.getByText('Completed item for toggle')).not.toBeVisible();

    // Toggle auto-remove OFF
    const toggle = page.getByRole('switch');
    await toggle.click();

    // Now both items should be visible (completed one shown)
    await expect(table.getByText('Completed item for toggle')).toBeVisible({ timeout: 5000 });
    await expect(table.getByText('Active item for toggle')).toBeVisible();
  });

  test('auto-remove toggle persists across page reloads', async ({ page }) => {
    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    // Toggle should be ON by default
    const toggle = page.getByRole('switch');
    await expect(toggle).toHaveAttribute('aria-checked', 'true');

    // Turn OFF
    await toggle.click();
    await expect(toggle).toHaveAttribute('aria-checked', 'false');

    // Reload page
    await page.reload();
    await dismissWelcomeModal(page);

    // Toggle should still be OFF
    const reloadedToggle = page.getByRole('switch');
    await expect(reloadedToggle).toHaveAttribute('aria-checked', 'false', { timeout: 10000 });
  });

  test('add item to inbox via mobile card view', async ({ request, testUser, testProject, page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    // Create work items
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Mobile inbox item 1',
      type: 'task',
    });
    const item2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Mobile inbox item 2',
      type: 'bug',
    });

    // Navigate to work item list (mobile card view)
    await page.goto(`/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Scope to the mobile card container (sm:hidden div) to avoid matching hidden desktop table rows
    const mobileCards = page.locator('.lg\\:hidden');

    // Wait for cards to render
    await expect(mobileCards.getByText('Mobile inbox item 1')).toBeVisible({ timeout: 10000 });
    await expect(mobileCards.getByText('Mobile inbox item 2')).toBeVisible();

    // Click the inbox button on item 1's card
    const card1 = mobileCards.locator('[role="button"]').filter({ hasText: 'Mobile inbox item 1' });
    const inboxBtn1 = card1.getByRole('button', { name: 'Send to inbox' });
    await expect(inboxBtn1).toBeVisible();
    await inboxBtn1.click();

    // Wait for the green checkmark feedback
    await expect(card1.locator('.text-green-500')).toBeVisible({ timeout: 5000 });

    // Click the inbox button on item 2's card
    const card2 = mobileCards.locator('[role="button"]').filter({ hasText: 'Mobile inbox item 2' });
    const inboxBtn2 = card2.getByRole('button', { name: 'Send to inbox' });
    await inboxBtn2.click();
    await expect(card2.locator('.text-green-500')).toBeVisible({ timeout: 5000 });

    // Verify both items are in the inbox via API
    const list = await api.listInboxItems(request, testUser.token);
    const titles = list.items.map((i: { title: string }) => i.title);
    expect(titles).toContain('Mobile inbox item 1');
    expect(titles).toContain('Mobile inbox item 2');

    // Navigate to inbox page and verify items are displayed (scope to mobile card container)
    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);
    const inboxCards = page.locator('.lg\\:hidden');
    await expect(inboxCards.getByText('Mobile inbox item 1')).toBeVisible({ timeout: 10000 });
    await expect(inboxCards.getByText('Mobile inbox item 2')).toBeVisible();
  });

  test('remove item from inbox via mobile card view toggle', async ({ request, testUser, testProject, page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    // Create a work item and add it to inbox
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Mobile remove inbox item',
      type: 'task',
    });
    await api.addToInbox(request, testUser.token, item.id);

    // Navigate to work item list (mobile card view)
    await page.goto(`/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Scope to mobile card container
    const mobileCards = page.locator('.lg\\:hidden');
    await expect(mobileCards.getByText('Mobile remove inbox item')).toBeVisible({ timeout: 10000 });

    const card = mobileCards.locator('[role="button"]').filter({ hasText: 'Mobile remove inbox item' });

    // The inbox button should be active (indigo) since item is in inbox — aria-label should say "Remove from inbox"
    const inboxBtn = card.getByRole('button', { name: 'Remove from inbox' });
    await expect(inboxBtn).toBeVisible({ timeout: 10000 });

    // Click to remove from inbox
    await inboxBtn.click();

    // Wait for the green checkmark feedback
    await expect(card.locator('.text-green-500')).toBeVisible({ timeout: 5000 });

    // After checkmark disappears, button should revert to "Send to inbox"
    await expect(card.getByRole('button', { name: 'Send to inbox' })).toBeVisible({ timeout: 5000 });

    // Verify via API that item is no longer in inbox
    const list = await api.listInboxItems(request, testUser.token);
    expect(list.items).toHaveLength(0);
  });

  test('clicking inbox item navigates to work item detail with back to inbox link', async ({ request, testUser, testProject, page }) => {
    // Create a work item and add to inbox
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Navigate detail test',
      type: 'task',
    });
    await api.addToInbox(request, testUser.token, item.id);

    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    // Click on the work item row (scope to table to avoid mobile card match)
    await page.getByRole('table').getByText('Navigate detail test').click();

    // Should navigate to work item detail page
    await expect(page).toHaveURL(new RegExp(`/projects/${testProject.key}/items/${item.item_number}`), { timeout: 10000 });

    // Should show "Back to inbox" link
    const backLink = page.getByText('Back to inbox');
    await expect(backLink).toBeVisible();

    // Click back to inbox
    await backLink.click();
    await expect(page).toHaveURL(/\/user\/inbox/, { timeout: 10000 });
  });

  test('mobile edit mode toggle shows and hides card controls', async ({ request, testUser, testProject, page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Edit mode test',
      type: 'task',
    });
    await api.addToInbox(request, testUser.token, item.id);

    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    const inboxCards = page.locator('.lg\\:hidden');
    await expect(inboxCards.getByText('Edit mode test')).toBeVisible({ timeout: 10000 });

    // Controls should not be visible before enabling edit mode
    await expect(inboxCards.getByRole('button', { name: 'Remove from inbox' })).not.toBeVisible();
    await expect(inboxCards.getByRole('button', { name: 'Move up' })).not.toBeVisible();
    await page.screenshot({ path: 'test-results/inbox-mobile-edit-toggle-1-before.png' });

    // Click the edit toggle button (pencil icon)
    const editBtn = page.getByRole('button', { name: 'Edit', exact: true });
    await editBtn.click();

    // Controls should now be visible
    await expect(inboxCards.getByRole('button', { name: 'Remove from inbox' })).toBeVisible({ timeout: 3000 });
    await page.screenshot({ path: 'test-results/inbox-mobile-edit-toggle-2-controls-visible.png' });

    // Toggle edit mode off
    await editBtn.click();

    // Controls should be hidden again
    await expect(inboxCards.getByRole('button', { name: 'Remove from inbox' })).not.toBeVisible({ timeout: 3000 });
    await page.screenshot({ path: 'test-results/inbox-mobile-edit-toggle-3-controls-hidden.png' });
  });

  test('mobile edit mode remove item from inbox', async ({ request, testUser, testProject, page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Mobile edit remove',
      type: 'task',
    });
    await api.addToInbox(request, testUser.token, item.id);

    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    const inboxCards = page.locator('.lg\\:hidden');
    await expect(inboxCards.getByText('Mobile edit remove')).toBeVisible({ timeout: 10000 });
    await page.screenshot({ path: 'test-results/inbox-mobile-edit-remove-1-before.png' });

    // Enable edit mode
    await page.getByRole('button', { name: 'Edit', exact: true }).click();

    // Click the remove button on the card
    const removeBtn = inboxCards.getByRole('button', { name: 'Remove from inbox' });
    await expect(removeBtn).toBeVisible({ timeout: 3000 });
    await page.screenshot({ path: 'test-results/inbox-mobile-edit-remove-2-edit-mode.png' });
    await removeBtn.click();

    // Item should disappear
    await expect(inboxCards.getByText('Mobile edit remove')).not.toBeVisible({ timeout: 5000 });
    await page.screenshot({ path: 'test-results/inbox-mobile-edit-remove-3-after.png' });

    // Verify via API
    const list = await api.listInboxItems(request, testUser.token);
    expect(list.items).toHaveLength(0);
  });

  test('create new item from inbox via desktop New Item button', async ({ request, testUser, testProject, page }) => {
    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    // Click the "New Item" button in the header
    await page.getByRole('button', { name: 'New Item' }).click();

    // Modal should open with "New Work Item" title
    await expect(page.getByRole('heading', { name: 'New Work Item' })).toBeVisible({ timeout: 5000 });

    // Select the project from the picker
    await page.getByText('Select project').click();
    await page.getByPlaceholder('Search projects...').fill(testProject.key);
    await page.getByText(testProject.name).last().click();

    // Select type (wait for it to be enabled after project selection)
    await page.getByLabel('Type').selectOption('task');

    // Fill title
    await page.getByLabel('Title').fill('Inbox created item desktop');

    // Click Create
    await page.getByRole('button', { name: 'Create' }).click();

    // Modal should close
    await expect(page.getByRole('heading', { name: 'New Work Item' })).not.toBeVisible({ timeout: 5000 });

    // Verify item was added to inbox via API
    const list = await api.listInboxItems(request, testUser.token);
    expect(list.items.some((i: { title: string }) => i.title === 'Inbox created item desktop')).toBe(true);

    // The inbox page should show the new item
    await expect(page.getByRole('table').getByText('Inbox created item desktop')).toBeVisible({ timeout: 10000 });
  });

  test('create new item from inbox via c keyboard shortcut', async ({ request, testUser, testProject, page }) => {
    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    // Click on the page body to ensure no input has focus
    await page.locator('h1').click();

    // Press 'c' to open the create modal
    await page.keyboard.press('c');

    // Modal should open
    await expect(page.getByRole('heading', { name: 'New Work Item' })).toBeVisible({ timeout: 5000 });

    // Select the project from the picker
    await page.getByText('Select project').click();
    await page.getByPlaceholder('Search projects...').fill(testProject.key);
    await page.getByText(testProject.name).last().click();

    // Select type and fill title
    await page.getByLabel('Type').selectOption('bug');
    await page.getByLabel('Title').fill('Inbox keyboard created bug');
    await page.getByRole('button', { name: 'Create' }).click();

    // Modal should close
    await expect(page.getByRole('heading', { name: 'New Work Item' })).not.toBeVisible({ timeout: 5000 });

    // Verify item was added to inbox
    const list = await api.listInboxItems(request, testUser.token);
    expect(list.items.some((i: { title: string }) => i.title === 'Inbox keyboard created bug')).toBe(true);
  });

  test('inbox new item modal has searchable project picker', async ({ request, testUser, testProject, page }) => {
    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    // Open the create modal
    await page.getByRole('button', { name: 'New Item' }).click();
    await expect(page.getByRole('heading', { name: 'New Work Item' })).toBeVisible({ timeout: 5000 });

    // Click the project picker button (shows "Select project" placeholder)
    await page.getByText('Select project').click();

    // Search input should appear
    const searchInput = page.getByPlaceholder('Search projects...');
    await expect(searchInput).toBeVisible({ timeout: 3000 });

    // Type to search by project key
    await searchInput.fill(testProject.key);

    // The project should be visible in results
    await expect(page.getByText(testProject.name).last()).toBeVisible();

    // Type nonsense — should show no results
    await searchInput.fill('ZZZZNONEXISTENT');
    await expect(page.getByText('No projects found')).toBeVisible({ timeout: 3000 });

    // Clear and select the project by clicking
    await searchInput.fill('');
    await page.getByText(testProject.name).last().click();

    // Project should be selected — picker button now shows project name
    await expect(page.locator('button').filter({ hasText: testProject.name })).toBeVisible();

    // Type field should now be enabled
    await expect(page.getByLabel('Type')).toBeEnabled();

    // Cancel to close modal
    await page.getByRole('button', { name: 'Cancel' }).click();
  });

  test('create new item from inbox on mobile', async ({ request, testUser, testProject, page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    // Click the "New" button (mobile shows short text)
    await page.getByRole('button', { name: 'New', exact: true }).click();

    // Modal should open
    await expect(page.getByRole('heading', { name: 'New Work Item' })).toBeVisible({ timeout: 5000 });

    // Select the project from the picker
    await page.getByText('Select project').click();
    await page.getByPlaceholder('Search projects...').fill(testProject.key);
    await page.getByText(testProject.name).last().click();

    // Select type
    await page.getByLabel('Type').selectOption('task');

    // Fill title
    await page.getByLabel('Title').fill('Inbox mobile created item');

    // Click Create
    await page.getByRole('button', { name: 'Create' }).click();

    // Modal should close
    await expect(page.getByRole('heading', { name: 'New Work Item' })).not.toBeVisible({ timeout: 5000 });

    // Verify item was added to inbox via API
    const list = await api.listInboxItems(request, testUser.token);
    expect(list.items.some((i: { title: string }) => i.title === 'Inbox mobile created item')).toBe(true);

    // The inbox page should show the new item in mobile cards
    const inboxCards = page.locator('.lg\\:hidden');
    await expect(inboxCards.getByText('Inbox mobile created item')).toBeVisible({ timeout: 10000 });
  });

  test('mobile edit mode reorder items', async ({ request, testUser, testProject, page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    // Create 3 items and add to inbox
    for (let i = 1; i <= 3; i++) {
      const item = await api.createWorkItem(request, testUser.token, testProject.key, {
        title: `Mobile order ${i}`,
        type: 'task',
      });
      await api.addToInbox(request, testUser.token, item.id);
    }

    await page.goto('/user/inbox');
    await dismissWelcomeModal(page);

    const inboxCards = page.locator('.lg\\:hidden');
    await expect(inboxCards.getByText('Mobile order 1')).toBeVisible({ timeout: 10000 });
    await expect(inboxCards.getByText('Mobile order 3')).toBeVisible();
    await page.screenshot({ path: 'test-results/inbox-mobile-edit-reorder-1-before.png' });

    // Enable edit mode
    await page.getByRole('button', { name: 'Edit', exact: true }).click();
    await page.screenshot({ path: 'test-results/inbox-mobile-edit-reorder-2-edit-mode.png' });

    // Find the last card (Mobile order 3) and click its "Move up" button
    const cards = inboxCards.locator('.rounded-lg');
    const lastCard = cards.filter({ hasText: 'Mobile order 3' });
    const moveUpBtn = lastCard.getByRole('button', { name: 'Move up' });
    await expect(moveUpBtn).toBeVisible({ timeout: 3000 });
    await moveUpBtn.click();

    // Wait for reorder to complete
    await page.waitForTimeout(1000);
    await page.screenshot({ path: 'test-results/inbox-mobile-edit-reorder-3-after.png' });

    // Verify new order via API: item 1, item 3, item 2
    const list = await api.listInboxItems(request, testUser.token);
    expect(list.items[0].title).toBe('Mobile order 1');
    expect(list.items[1].title).toBe('Mobile order 3');
    expect(list.items[2].title).toBe('Mobile order 2');
  });
});
