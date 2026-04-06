import { test, expect } from '../../lib/fixtures';

test.describe('Project settings tabs', () => {

  test('defaults to General tab showing project info and danger zone', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await page.waitForLoadState('networkidle');

    // Page title is visible
    await expect(page.getByRole('heading', { name: 'Settings', exact: true })).toBeVisible();

    // General tab is active by default
    const generalTab = page.getByRole('button', { name: 'General', exact: true });
    await expect(generalTab).toBeVisible();
    await expect(generalTab).toHaveClass(/border-indigo/);

    // General tab content is visible: project key and name in same row
    await expect(page.getByLabel('Project key')).toBeVisible();
    await expect(page.getByLabel('Project name')).toBeVisible();

    // Danger Zone is visible
    await expect(page.getByText('Danger Zone')).toBeVisible();

    // Other tab content is not visible
    await expect(page.getByText('Manage who has access')).not.toBeVisible();
  });

  test('all five tabs are present for project owners', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('button', { name: 'General', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Users', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Teams', exact: true })).toBeVisible();
    // Invites and Work Items tabs appear after permissions load
    await expect(page.getByRole('button', { name: 'Invites', exact: true })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('button', { name: 'Work Items', exact: true })).toBeVisible({ timeout: 10000 });
  });

  test('switching to Users tab shows members section', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: 'Users', exact: true }).click();

    // Users content is visible
    await expect(page.getByText('Manage who has access')).toBeVisible();
    await expect(page.getByText('Add user')).toBeVisible();

    // General content is hidden
    await expect(page.getByLabel('Project key')).not.toBeVisible();
    await expect(page.getByText('Danger Zone')).not.toBeVisible();
  });

  test('switching to Teams tab shows teams content', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: 'Teams', exact: true }).click();

    // Teams empty state or content is visible
    await expect(page.getByText('No teams yet.')).toBeVisible({ timeout: 10000 });

    // General content is hidden
    await expect(page.getByLabel('Project key')).not.toBeVisible();
  });

  test('Invites tab shows invite links section', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/settings?tab=invites`);
    await page.waitForLoadState('networkidle');

    // Wait for Invites tab to appear (depends on permissions loading)
    const invitesTab = page.getByRole('button', { name: 'Invites', exact: true });
    await expect(invitesTab).toBeVisible({ timeout: 10000 });
    await expect(invitesTab).toHaveClass(/border-indigo/);

    // Invite content is visible
    await expect(page.getByRole('heading', { name: 'Create Invite Link' })).toBeVisible({ timeout: 10000 });

    // General content is hidden
    await expect(page.getByLabel('Project key')).not.toBeVisible();
  });

  test('switching to Work Items tab shows complexity section', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await page.waitForLoadState('networkidle');

    // Wait for Work Items tab to appear (depends on permissions loading)
    await expect(page.getByRole('button', { name: 'Work Items', exact: true })).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: 'Work Items', exact: true }).click();

    // Work Items content is visible
    await expect(page.getByLabel('Allowed values')).toBeVisible();

    // General content is hidden
    await expect(page.getByLabel('Project key')).not.toBeVisible();
  });

  test('deep linking to tab via ?tab= query param', async ({ page, testProject }) => {
    // Navigate directly to teams tab
    await page.goto(`/d/projects/${testProject.key}/settings?tab=teams`);
    await page.waitForLoadState('networkidle');

    // Teams tab is active
    const teamsTab = page.getByRole('button', { name: 'Teams', exact: true });
    await expect(teamsTab).toHaveClass(/border-indigo/);

    // Teams content is visible
    await expect(page.getByText('No teams yet.')).toBeVisible({ timeout: 10000 });

    // General content is hidden
    await expect(page.getByLabel('Project key')).not.toBeVisible();
  });

  test('/teams URL redirects to settings?tab=teams', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/teams`);

    // Should redirect to settings with teams tab
    await expect(page).toHaveURL(/\/settings\?tab=teams/, { timeout: 10000 });

    // Teams content is visible
    await expect(page.getByText('No teams yet.')).toBeVisible({ timeout: 10000 });
  });

  test('switching back to General tab restores general content', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await page.waitForLoadState('networkidle');

    // Switch to Users
    await page.getByRole('button', { name: 'Users', exact: true }).click();
    await expect(page.getByText('Manage who has access')).toBeVisible();

    // Switch back to General
    await page.getByRole('button', { name: 'General', exact: true }).click();
    await expect(page.getByLabel('Project key')).toBeVisible();
    await expect(page.getByText('Danger Zone')).toBeVisible();

    // Users content hidden again
    await expect(page.getByText('Manage who has access')).not.toBeVisible();
  });

  test('role dropdown in Users tab shows role descriptions', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/settings?tab=users`);
    await page.waitForLoadState('networkidle');

    // Wait for the "Add user" section to load
    await expect(page.getByText('Add user')).toBeVisible({ timeout: 10000 });

    // Find the role dropdown button in the add user section (shows "Member" by default with an info icon)
    const addUserCard = page.locator('.border.rounded-lg').filter({ hasText: 'Add user' }).first();
    const roleButton = addUserCard.locator('button').filter({ hasText: 'Member' }).first();
    await expect(roleButton).toBeVisible({ timeout: 5000 });

    // Click to open role dropdown
    await roleButton.click();

    // Role descriptions should be visible in the dropdown
    await expect(page.getByText('Can manage members, settings, and all work items.')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Can create and edit work items assigned to them.')).toBeVisible();
  });

  test('General tab has project key and name in same row', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await page.waitForLoadState('networkidle');

    const keyInput = page.getByLabel('Project key');
    const nameInput = page.getByLabel('Project name');

    // Both should be visible
    await expect(keyInput).toBeVisible();
    await expect(nameInput).toBeVisible();

    // They should be in the same row (similar Y position)
    const keyBox = await keyInput.boundingBox();
    const nameBox = await nameInput.boundingBox();

    expect(keyBox).not.toBeNull();
    expect(nameBox).not.toBeNull();

    // Same row means similar Y position (within 5px tolerance)
    expect(Math.abs(keyBox!.y - nameBox!.y)).toBeLessThan(5);
  });
});
