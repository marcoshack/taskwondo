import { test, expect } from '../../lib/fixtures';

test.describe('Invite Admin Role Warning', () => {

  test('shows warning and sets defaults when Admin role is selected', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await page.waitForLoadState('networkidle');

    // Locate the create invite form section by its heading text
    const createInviteSection = page.locator('div.border').filter({ hasText: 'Create Invite Link' }).first();

    // Get the selects and input within the invite form section
    const roleSelect = createInviteSection.locator('select').first();
    const expirationSelect = createInviteSection.locator('select').nth(1);
    const maxUsesInput = createInviteSection.locator('input[type="number"]');

    // Verify initial defaults: member role, no expiration, empty max uses
    await expect(roleSelect).toHaveValue('member');
    await expect(expirationSelect).toHaveValue('');
    await expect(maxUsesInput).toHaveValue('');

    // Warning should NOT be visible for member role
    const warningText = page.getByText('Admin invites grant full project management access');
    await expect(warningText).not.toBeVisible();

    // Select Admin role
    await roleSelect.selectOption('admin');

    // Warning message should appear
    await expect(warningText).toBeVisible();

    // Expiration should be auto-set to 1d and max uses to 1
    await expect(expirationSelect).toHaveValue('1d');
    await expect(maxUsesInput).toHaveValue('1');
  });

  test('hides warning when switching away from Admin role', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await page.waitForLoadState('networkidle');

    const createInviteSection = page.locator('div.border').filter({ hasText: 'Create Invite Link' }).first();
    const roleSelect = createInviteSection.locator('select').first();
    const warningText = page.getByText('Admin invites grant full project management access');

    // Select Admin → warning visible
    await roleSelect.selectOption('admin');
    await expect(warningText).toBeVisible();

    // Switch to Viewer → warning hidden
    await roleSelect.selectOption('viewer');
    await expect(warningText).not.toBeVisible();
  });

  test('allows user to change auto-selected defaults for Admin invite', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await page.waitForLoadState('networkidle');

    const createInviteSection = page.locator('div.border').filter({ hasText: 'Create Invite Link' }).first();
    const roleSelect = createInviteSection.locator('select').first();
    const expirationSelect = createInviteSection.locator('select').nth(1);
    const maxUsesInput = createInviteSection.locator('input[type="number"]');
    const warningText = page.getByText('Admin invites grant full project management access');

    // Select Admin → defaults applied
    await roleSelect.selectOption('admin');
    await expect(expirationSelect).toHaveValue('1d');
    await expect(maxUsesInput).toHaveValue('1');

    // User can override the auto-selected values
    await expirationSelect.selectOption('7d');
    await expect(expirationSelect).toHaveValue('7d');

    await maxUsesInput.fill('5');
    await expect(maxUsesInput).toHaveValue('5');

    // Warning should still be visible
    await expect(warningText).toBeVisible();
  });

  test('creates Admin invite successfully with warning defaults', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await page.waitForLoadState('networkidle');

    const createInviteSection = page.locator('div.border').filter({ hasText: 'Create Invite Link' }).first();
    const roleSelect = createInviteSection.locator('select').first();

    // Select Admin role (auto-selects 1 use, 24h expiration)
    await roleSelect.selectOption('admin');

    // Click Create Invite Link button
    await page.getByRole('button', { name: /Create Invite Link/i }).click();

    // Should see a success checkmark
    await expect(createInviteSection.locator('.text-green-500')).toBeVisible({ timeout: 5000 });

    // The invite list should show the admin invite with an Admin badge
    // Use a Badge-specific locator to avoid matching hidden <option> elements
    const inviteList = page.locator('div.divide-y');
    await expect(inviteList.locator('span', { hasText: /^Admin$/ })).toBeVisible({ timeout: 5000 });
  });
});
