import { test, expect, getAdminToken } from '../../lib/fixtures';
import type { Page, BrowserContext } from '@playwright/test';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

const TEST_PASSWORD = 'TestPass123!';

/** Log in as a different user by clearing state and filling the login form. */
async function loginAs(page: Page, context: BrowserContext, email: string, password: string) {
  // Navigate first so localStorage is accessible on the correct origin
  await page.goto('/login');
  await context.clearCookies();
  await page.evaluate(() => localStorage.clear());
  await page.goto('/login');
  await page.getByLabel('Email').fill(email);
  await page.getByLabel('Password').fill(password);
  await page.getByRole('button', { name: 'Sign in' }).click();
}

/**
 * Create a customer user and add them to the project with the "customer" role.
 * Returns the customer's auth info.
 */
async function createCustomerUser(
  request: import('@playwright/test').APIRequestContext,
  adminToken: string,
  projectKey: string,
  ownerToken: string,
) {
  const uniqueId = randomUUID().slice(0, 8);
  const email = `customer-${uniqueId}@test.local`;
  const displayName = `Customer ${uniqueId}`;
  const created = await api.createUser(request, adminToken, email, displayName);
  const tempLogin = await api.login(request, email, created.temporary_password);
  await api.changePassword(request, tempLogin.token, created.temporary_password, TEST_PASSWORD);
  const finalLogin = await api.login(request, email, TEST_PASSWORD);
  await api.setPreference(request, finalLogin.token, 'welcome_dismissed', true);
  await api.addMember(request, ownerToken, projectKey, finalLogin.user.id, 'customer');
  return { id: finalLogin.user.id, email, displayName, token: finalLogin.token, password: TEST_PASSWORD };
}

/**
 * Create a public queue in the project so portal ticket creation works.
 */
async function ensurePublicQueue(
  request: import('@playwright/test').APIRequestContext,
  token: string,
  projectKey: string,
) {
  return api.createQueue(request, token, projectKey, {
    name: 'Portal Support',
    queue_type: 'support',
    is_public: true,
  });
}

test.describe('Portal tickets — comments and attachments', () => {

  test('internal comment is not visible to customer', async ({ page, request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    await ensurePublicQueue(request, testUser.token, testProject.key);
    const customer = await createCustomerUser(request, adminToken, testProject.key, testUser.token);

    // Customer creates a ticket via API
    const ticket = await api.createPortalTicket(request, customer.token, testProject.key, {
      title: 'Internal comment test',
    });

    // Owner adds an internal comment (default visibility)
    await api.addComment(request, testUser.token, testProject.key, ticket.item_number, 'Internal note for team only');

    // Customer views the ticket — internal comment should NOT be visible
    const portalComments = await api.listPortalComments(request, customer.token, testProject.key, ticket.item_number);
    expect(portalComments.length).toBe(0);

    // Verify via UI — customer logs in and navigates to support ticket
    await loginAs(page, page.context(), customer.email, customer.password);
    await page.waitForURL(/\/projects/, { timeout: 15000 });

    // Navigate to the customer support view
    await page.goto(`/d/projects/${testProject.key}/support`);
    await page.getByText('Internal comment test').first().click();
    await page.waitForURL(/\/support\/\d+/);

    // Comments section should show no comments
    await expect(page.getByText('Internal note for team only')).not.toBeVisible();
  });

  test('public comment is visible to customer', async ({ page, request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    await ensurePublicQueue(request, testUser.token, testProject.key);
    const customer = await createCustomerUser(request, adminToken, testProject.key, testUser.token);

    const ticket = await api.createPortalTicket(request, customer.token, testProject.key, {
      title: 'Public comment test',
    });

    // Owner adds a public comment
    await api.addCommentWithVisibility(request, testUser.token, testProject.key, ticket.item_number, 'Hello customer, we are looking into it.', 'public');

    // Customer views — public comment should be visible
    const portalComments = await api.listPortalComments(request, customer.token, testProject.key, ticket.item_number);
    expect(portalComments.length).toBe(1);
    expect(portalComments[0].body).toBe('Hello customer, we are looking into it.');

    // Verify via UI
    await loginAs(page, page.context(), customer.email, customer.password);
    await page.waitForURL(/\/projects/, { timeout: 15000 });
    await page.goto(`/d/projects/${testProject.key}/support`);
    await page.getByText('Public comment test').first().click();
    await page.waitForURL(/\/support\/\d+/);

    await expect(page.getByText('Hello customer, we are looking into it.')).toBeVisible();
  });

  test('internal comment edited to public becomes visible to customer', async ({ page, request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    await ensurePublicQueue(request, testUser.token, testProject.key);
    const customer = await createCustomerUser(request, adminToken, testProject.key, testUser.token);

    const ticket = await api.createPortalTicket(request, customer.token, testProject.key, {
      title: 'Visibility change test',
    });

    // Owner adds internal comment
    const comment = await api.addCommentWithVisibility(request, testUser.token, testProject.key, ticket.item_number, 'Draft response', 'internal');

    // Not visible to customer yet
    let portalComments = await api.listPortalComments(request, customer.token, testProject.key, ticket.item_number);
    expect(portalComments.length).toBe(0);

    // Owner edits comment to public visibility
    await api.updateCommentWithVisibility(request, testUser.token, testProject.key, ticket.item_number, comment.id, 'Final response to customer', 'public');

    // Now visible to customer
    portalComments = await api.listPortalComments(request, customer.token, testProject.key, ticket.item_number);
    expect(portalComments.length).toBe(1);
    expect(portalComments[0].body).toBe('Final response to customer');

    // Verify via UI — owner opens the work item and sees portal badge
    await page.goto(`/d/projects/${testProject.key}/items`);
    await page.getByText('Visibility change test').first().click();
    await expect(page.getByText('Final response to customer')).toBeVisible();
    await expect(page.getByText('portal').first()).toBeVisible();
  });

  test('customer can upload attachment to their ticket', async ({ page, request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    await ensurePublicQueue(request, testUser.token, testProject.key);
    const customer = await createCustomerUser(request, adminToken, testProject.key, testUser.token);

    const ticket = await api.createPortalTicket(request, customer.token, testProject.key, {
      title: 'Customer attachment test',
    });

    // Customer logs in and navigates to support view
    await loginAs(page, page.context(), customer.email, customer.password);
    await page.waitForURL(/\/projects/, { timeout: 15000 });
    await page.goto(`/d/projects/${testProject.key}/support`);
    await page.getByText('Customer attachment test').first().click();
    await page.waitForURL(/\/support\/\d+/);

    // Switch to Attachments tab
    await page.getByRole('button', { name: /Attachments/ }).click();

    // Upload a file
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles({
      name: 'customer-file.txt',
      mimeType: 'text/plain',
      buffer: Buffer.from('Hello from customer'),
    });

    // Wait for file to appear in the list
    await expect(page.getByText('customer-file.txt')).toBeVisible({ timeout: 10000 });
  });

  test('owner can upload attachment to portal ticket', async ({ page, request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    await ensurePublicQueue(request, testUser.token, testProject.key);
    const customer = await createCustomerUser(request, adminToken, testProject.key, testUser.token);

    const ticket = await api.createPortalTicket(request, customer.token, testProject.key, {
      title: 'Owner attachment test',
    });

    // Owner uploads attachment via API (through regular endpoint, elevated)
    await api.uploadAttachment(request, testUser.token, testProject.key, ticket.item_number, 'report.pdf', Buffer.from('PDF content'), 'application/pdf');

    // Owner opens the work item detail in the regular UI
    await page.goto(`/d/projects/${testProject.key}/items`);
    await page.getByText('Owner attachment test').first().click();
    await page.getByRole('button', { name: /Attachments/ }).click();

    await expect(page.getByText('report.pdf')).toBeVisible({ timeout: 10000 });
  });

  test('user can change comments sort order', async ({ page, request, testUser, testProject }) => {
    await ensurePublicQueue(request, testUser.token, testProject.key);

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Sort comments test',
      type: 'ticket',
    });

    // Add two comments
    await api.addComment(request, testUser.token, testProject.key, item.item_number, 'First comment');
    await api.addComment(request, testUser.token, testProject.key, item.item_number, 'Second comment');

    await page.goto(`/d/projects/${testProject.key}/items`);
    await page.getByText('Sort comments test').first().click();

    // Default is newest first — Second comment should appear before First
    const comments = page.locator('.border-b').filter({ hasText: /comment/ });
    await expect(comments.first()).toContainText('Second comment');

    // Click sort toggle
    await page.getByRole('button', { name: /Newest first|Oldest first/ }).click();

    // Now oldest first — First comment should appear before Second
    await expect(comments.first()).toContainText('First comment');
  });

  test('user can change attachments sort order', async ({ page, request, testUser, testProject }) => {
    await ensurePublicQueue(request, testUser.token, testProject.key);

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Sort attachments test',
      type: 'ticket',
    });

    // Upload two files
    await api.uploadAttachment(request, testUser.token, testProject.key, item.item_number, 'alpha.txt', Buffer.from('A'), 'text/plain');
    // Small delay to ensure different timestamps
    await new Promise((r) => setTimeout(r, 100));
    await api.uploadAttachment(request, testUser.token, testProject.key, item.item_number, 'beta.txt', Buffer.from('B'), 'text/plain');

    await page.goto(`/d/projects/${testProject.key}/items`);
    await page.getByText('Sort attachments test').first().click();
    await page.getByRole('button', { name: /Attachments/ }).click();

    // Default is newest first — beta should appear before alpha
    const attachments = page.locator('.border-b').filter({ hasText: /\.txt/ });
    await expect(attachments.first()).toContainText('beta.txt');

    // Click sort toggle
    await page.getByRole('button', { name: /Newest first|Oldest first/ }).click();

    // Now oldest first — alpha should appear first
    await expect(attachments.first()).toContainText('alpha.txt');
  });

});

test.describe('Portal tickets — hide completed and auto-refresh', () => {

  test('desktop: hide completed toggle filters out resolved tickets', async ({ page, request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    await ensurePublicQueue(request, testUser.token, testProject.key);
    const customer = await createCustomerUser(request, adminToken, testProject.key, testUser.token);

    // Create two tickets
    const openTicket = await api.createPortalTicket(request, customer.token, testProject.key, {
      title: 'Open ticket for filter',
    });
    const doneTicket = await api.createPortalTicket(request, customer.token, testProject.key, {
      title: 'Done ticket for filter',
    });

    // Resolve the second ticket via the owner
    // Transition through workflow: new → investigating → resolved
    await api.updateWorkItem(request, testUser.token, testProject.key, doneTicket.item_number, { status: 'investigating' });
    await api.updateWorkItem(request, testUser.token, testProject.key, doneTicket.item_number, { status: 'resolved' });

    // Customer logs in and navigates to support view — desktop viewport
    await page.setViewportSize({ width: 1280, height: 720 });
    await loginAs(page, page.context(), customer.email, customer.password);
    await page.waitForURL(/\/projects/, { timeout: 15000 });
    await page.goto(`/d/projects/${testProject.key}/support`);

    // Both tickets should be visible initially
    await expect(page.getByText('Open ticket for filter').first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Done ticket for filter').first()).toBeVisible();

    // Toggle "Hide completed"
    await page.getByRole('switch').click();

    // Done ticket should disappear, open ticket should remain
    await expect(page.getByText('Done ticket for filter').first()).not.toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Open ticket for filter').first()).toBeVisible();

    // Toggle back off
    await page.getByRole('switch').click();

    // Both should be visible again
    await expect(page.getByText('Done ticket for filter').first()).toBeVisible({ timeout: 5000 });
  });

  test('mobile: hide completed toggle in settings modal', async ({ page, request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    await ensurePublicQueue(request, testUser.token, testProject.key);
    const customer = await createCustomerUser(request, adminToken, testProject.key, testUser.token);

    const openTicket = await api.createPortalTicket(request, customer.token, testProject.key, {
      title: 'Mobile open ticket',
    });
    const doneTicket = await api.createPortalTicket(request, customer.token, testProject.key, {
      title: 'Mobile done ticket',
    });

    // Transition through workflow: new → investigating → resolved
    await api.updateWorkItem(request, testUser.token, testProject.key, doneTicket.item_number, { status: 'investigating' });
    await api.updateWorkItem(request, testUser.token, testProject.key, doneTicket.item_number, { status: 'resolved' });

    // Mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await loginAs(page, page.context(), customer.email, customer.password);
    await page.waitForURL(/\/projects/, { timeout: 15000 });
    await page.goto(`/d/projects/${testProject.key}/support`);

    await expect(page.getByText('Mobile open ticket').last()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Mobile done ticket').last()).toBeVisible();

    // Open settings modal via gear icon
    await page.getByRole('button', { name: 'Settings' }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog.getByText('Hide completed')).toBeVisible({ timeout: 3000 });

    // Toggle hide completed on
    await dialog.getByRole('switch').click();

    // Close modal by clicking the X button
    await dialog.locator('button').first().click();

    // Done ticket should be hidden
    await expect(page.getByText('Mobile done ticket').last()).not.toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Mobile open ticket').last()).toBeVisible();
  });

  test('desktop: hide completed persists across page reloads', async ({ page, request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    await ensurePublicQueue(request, testUser.token, testProject.key);
    const customer = await createCustomerUser(request, adminToken, testProject.key, testUser.token);

    const openTicket = await api.createPortalTicket(request, customer.token, testProject.key, {
      title: 'Persist open ticket',
    });
    const doneTicket = await api.createPortalTicket(request, customer.token, testProject.key, {
      title: 'Persist done ticket',
    });

    // Transition through workflow: new → investigating → resolved
    await api.updateWorkItem(request, testUser.token, testProject.key, doneTicket.item_number, { status: 'investigating' });
    await api.updateWorkItem(request, testUser.token, testProject.key, doneTicket.item_number, { status: 'resolved' });

    await page.setViewportSize({ width: 1280, height: 720 });
    await loginAs(page, page.context(), customer.email, customer.password);
    await page.waitForURL(/\/projects/, { timeout: 15000 });
    await page.goto(`/d/projects/${testProject.key}/support`);

    await expect(page.getByText('Persist open ticket').first()).toBeVisible({ timeout: 10000 });

    // Enable hide completed
    await page.getByRole('switch').click();
    await expect(page.getByText('Persist done ticket').first()).not.toBeVisible({ timeout: 5000 });

    // Reload the page
    await page.reload();

    // Hide completed should still be active after reload
    await expect(page.getByText('Persist open ticket').first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Persist done ticket').first()).not.toBeVisible();
  });

  test('desktop: auto-refresh dropdown opens and allows selecting interval', async ({ page, request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    await ensurePublicQueue(request, testUser.token, testProject.key);
    const customer = await createCustomerUser(request, adminToken, testProject.key, testUser.token);

    await api.createPortalTicket(request, customer.token, testProject.key, {
      title: 'Auto-refresh test ticket',
    });

    await page.setViewportSize({ width: 1280, height: 720 });
    await loginAs(page, page.context(), customer.email, customer.password);
    await page.waitForURL(/\/projects/, { timeout: 15000 });
    await page.goto(`/d/projects/${testProject.key}/support`);

    await expect(page.getByText('Auto-refresh test ticket').first()).toBeVisible({ timeout: 10000 });

    // Click the dropdown chevron
    const dropdownToggle = page.getByRole('button', { name: 'Auto-refresh' });
    await expect(dropdownToggle).toBeVisible();
    await dropdownToggle.click();

    // Dropdown should show interval options
    await expect(page.getByRole('button', { name: 'Off' })).toBeVisible({ timeout: 3000 });
    await expect(page.getByRole('button', { name: '30s' })).toBeVisible();

    // Select 30s
    await page.getByRole('button', { name: '30s' }).click();

    // Dropdown should close and label should update
    await expect(page.getByRole('button', { name: 'Off' })).not.toBeVisible({ timeout: 3000 });
    await expect(page.getByRole('button', { name: 'Refresh', exact: true }).getByText('30s')).toBeVisible();
  });

  test('desktop: auto-refresh interval persists across page reloads', async ({ page, request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    await ensurePublicQueue(request, testUser.token, testProject.key);
    const customer = await createCustomerUser(request, adminToken, testProject.key, testUser.token);

    await api.createPortalTicket(request, customer.token, testProject.key, {
      title: 'Persist refresh ticket',
    });

    await page.setViewportSize({ width: 1280, height: 720 });
    await loginAs(page, page.context(), customer.email, customer.password);
    await page.waitForURL(/\/projects/, { timeout: 15000 });
    await page.goto(`/d/projects/${testProject.key}/support`);

    await expect(page.getByText('Persist refresh ticket').first()).toBeVisible({ timeout: 10000 });

    // Set interval to 1m
    await page.getByRole('button', { name: 'Auto-refresh' }).click();
    await page.getByRole('button', { name: '1m', exact: true }).click();

    // Reload
    await page.reload();

    // Should still show 1m
    await expect(page.getByText('Persist refresh ticket').first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('button', { name: 'Refresh', exact: true }).getByText('1m', { exact: true })).toBeVisible({ timeout: 5000 });
  });

  test('mobile: auto-refresh works on mobile viewport', async ({ page, request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    await ensurePublicQueue(request, testUser.token, testProject.key);
    const customer = await createCustomerUser(request, adminToken, testProject.key, testUser.token);

    await api.createPortalTicket(request, customer.token, testProject.key, {
      title: 'Mobile refresh ticket',
    });

    await page.setViewportSize({ width: 375, height: 667 });
    await loginAs(page, page.context(), customer.email, customer.password);
    await page.waitForURL(/\/projects/, { timeout: 15000 });
    await page.goto(`/d/projects/${testProject.key}/support`);

    await expect(page.getByText('Mobile refresh ticket').last()).toBeVisible({ timeout: 10000 });

    // RefreshButton should be visible on mobile
    const dropdownToggle = page.getByRole('button', { name: 'Auto-refresh' });
    await expect(dropdownToggle).toBeVisible();
    await dropdownToggle.click();

    // Select 10s
    await expect(page.getByRole('button', { name: '10s' })).toBeVisible({ timeout: 3000 });
    await page.getByRole('button', { name: '10s' }).click();

    // Dropdown should close
    await expect(page.getByRole('button', { name: 'Off' })).not.toBeVisible({ timeout: 3000 });
  });

});
