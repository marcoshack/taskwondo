import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

test.describe('Team CRUD', () => {
  test('create team via UI and verify in list', async ({ request, testUser, testProject, page }) => {
    await page.goto(`/d/projects/${testProject.key}/settings?tab=teams`);

    // Empty state should be shown
    await expect(page.getByText('No teams yet.')).toBeVisible({ timeout: 10000 });

    // Open create modal
    await page.getByRole('button', { name: 'Create team' }).click();
    await expect(page.getByRole('heading', { name: 'Create Team' })).toBeVisible();

    // Fill form and submit
    await page.getByRole('textbox', { name: 'Name' }).fill('Backend Team');
    await page.locator('textarea').fill('Handles backend services');
    await page.getByRole('button', { name: 'Create', exact: true }).click();

    // Verify team appears in list
    await expect(page.getByText('Backend Team')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Handles backend services')).toBeVisible();
    await expect(page.getByText('0 members')).toBeVisible();
  });

  test('create team via API and verify on teams page', async ({ request, testUser, testProject, page }) => {
    await api.createTeam(request, testUser.token, testProject.key, {
      name: 'API Team',
      description: 'Created via API',
    });

    await page.goto(`/d/projects/${testProject.key}/settings?tab=teams`);
    await expect(page.getByText('API Team')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Created via API')).toBeVisible();
  });

  test('update team name and description via settings tab', async ({ request, testUser, testProject, page }) => {
    const team = await api.createTeam(request, testUser.token, testProject.key, {
      name: 'Old Name',
      description: 'Old description',
    });

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);

    // Navigate to settings tab
    await page.getByRole('button', { name: 'Settings', exact: true }).click();

    // The settings form is a controlled component whose inputs are seeded
    // from the team API response. Wait for the original values to land
    // before editing — otherwise a late re-render during initial page load
    // can overwrite what we typed, silently dropping the new name.
    const nameInput = page.getByRole('textbox', { name: 'Name' });
    const descTextarea = page.locator('textarea');
    await expect(nameInput).toHaveValue('Old Name');
    await expect(descTextarea).toHaveValue('Old description');

    // Update name and description, retrying until both values stick.
    await expect(async () => {
      await nameInput.fill('New Name');
      await descTextarea.fill('New description');
      await expect(nameInput).toHaveValue('New Name', { timeout: 1000 });
      await expect(descTextarea).toHaveValue('New description', { timeout: 1000 });
    }).toPass({ timeout: 15000 });

    await page.getByRole('button', { name: 'Save', exact: true }).click();

    // Verify save succeeded (green checkmark appears)
    await expect(page.locator('.text-green-500')).toBeVisible({ timeout: 5000 });

    // Verify via API
    const updated = await api.getTeam(request, testUser.token, testProject.key, team.id);
    expect(updated.name).toBe('New Name');
    expect(updated.description).toBe('New description');
  });

  test('delete team via UI from teams list', async ({ request, testUser, testProject, page }) => {
    await api.createTeam(request, testUser.token, testProject.key, {
      name: 'Doomed Team',
    });

    await page.goto(`/d/projects/${testProject.key}/settings?tab=teams`);
    await expect(page.getByText('Doomed Team')).toBeVisible({ timeout: 10000 });

    // Click the delete (trash) button on the team card
    const teamCard = page.locator('div.p-4').filter({ hasText: 'Doomed Team' });
    await teamCard.locator('button').filter({ has: page.locator('.text-red-500') }).click();

    // Confirm deletion in modal
    await expect(page.getByRole('heading', { name: 'Delete Team' })).toBeVisible();
    await page.getByRole('button', { name: 'Delete', exact: true }).click();

    // Verify the team is gone — check the link specifically to avoid matching the modal's <strong>
    await expect(page.getByRole('link', { name: 'Doomed Team' })).not.toBeVisible({ timeout: 5000 });
  });

  test('delete team via settings tab danger zone', async ({ request, testUser, testProject, page }) => {
    const team = await api.createTeam(request, testUser.token, testProject.key, {
      name: 'Delete From Settings',
    });

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);

    // Go to settings tab
    await page.getByRole('button', { name: 'Settings', exact: true }).click();

    // Click delete in danger zone
    await page.getByRole('button', { name: 'Delete team' }).click();

    // Confirm deletion
    await expect(page.getByRole('heading', { name: 'Delete Team' })).toBeVisible();
    await page.getByRole('button', { name: 'Delete', exact: true }).click();

    // Should redirect back to teams list (in settings tab)
    await expect(page).toHaveURL(/\/settings\?tab=teams/, { timeout: 10000 });
  });

  test('team detail page shows correct tabs', async ({ request, testUser, testProject, page }) => {
    const team = await api.createTeam(request, testUser.token, testProject.key, {
      name: 'Tab Test Team',
    });

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);

    // Verify all three tab buttons are present
    await expect(page.getByRole('button', { name: 'Members', exact: true })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('button', { name: 'On-Call', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Settings', exact: true })).toBeVisible();

    // Team name should be displayed in the header
    await expect(page.getByRole('heading', { name: 'Tab Test Team' })).toBeVisible();
  });
});
