import { test, expect, getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

const TEST_PASSWORD = 'TestPass123!';

/** Create a second test user and add them as a project member. */
async function createSecondUser(
  request: import('@playwright/test').APIRequestContext,
  adminToken: string,
  projectKey: string,
  ownerToken: string,
  role = 'member',
) {
  const uniqueId = randomUUID().slice(0, 8);
  const email = `e2e-${uniqueId}@test.local`;
  const created = await api.createUser(request, adminToken, email, `E2E User2 ${uniqueId}`);
  const tempLogin = await api.login(request, email, created.temporary_password);
  await api.changePassword(request, tempLogin.token, created.temporary_password, TEST_PASSWORD);
  const finalLogin = await api.login(request, email, TEST_PASSWORD);
  await api.setPreference(request, finalLogin.token, 'welcome_dismissed', true);
  await api.addMember(request, ownerToken, projectKey, finalLogin.user.id, role);
  return { id: finalLogin.user.id, email, displayName: `E2E User2 ${uniqueId}`, token: finalLogin.token };
}

/** Set up a team with two members (the test user + a second user). Returns team and user2. */
async function setupTeamWithMembers(
  request: import('@playwright/test').APIRequestContext,
  testUser: { id: string; token: string },
  testProject: { key: string },
) {
  const adminToken = getAdminToken();
  const user2 = await createSecondUser(request, adminToken, testProject.key, testUser.token);

  const team = await api.createTeam(request, testUser.token, testProject.key, {
    name: 'Oncall Team',
  });
  await api.addTeamMember(request, testUser.token, testProject.key, team.id, testUser.id);
  await api.addTeamMember(request, testUser.token, testProject.key, team.id, user2.id);

  return { team, user2 };
}

test.describe('Oncall rotation', () => {
  test('shows empty state when no rotation is configured', async ({ request, testUser, testProject, page }) => {
    const team = await api.createTeam(request, testUser.token, testProject.key, {
      name: 'No Oncall Team',
    });

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);

    // Switch to On-Call tab
    await page.getByRole('button', { name: 'On-Call', exact: true }).click();

    // Verify empty state
    await expect(page.getByText('No on-call rotation configured for this team.')).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('button', { name: 'Set Up On-Call' })).toBeVisible();
  });

  test('create oncall rotation via UI', async ({ request, testUser, testProject, page }) => {
    const { team, user2 } = await setupTeamWithMembers(request, testUser, testProject);

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);
    await page.getByRole('button', { name: 'On-Call', exact: true }).click();

    // Click set up
    await page.getByRole('button', { name: 'Set Up On-Call' }).click();
    await expect(page.getByRole('heading', { name: 'Create Rotation' })).toBeVisible();

    // Select both participants
    const checkboxes = page.locator('input[type="checkbox"]');
    const count = await checkboxes.count();
    for (let i = 0; i < count; i++) {
      await checkboxes.nth(i).check();
    }

    // Set period to 7 days (it defaults to 7, but clear and set explicitly)
    const periodInput = page.getByLabel('Period (days)');
    await periodInput.clear();
    await periodInput.fill('7');

    // Submit
    await page.getByRole('button', { name: 'Create', exact: true }).click();

    // Verify rotation details appear in sidebar
    await expect(page.getByText('Every 7 days')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Active', { exact: true })).toBeVisible();
  });

  test('edit oncall rotation via UI', async ({ request, testUser, testProject, page }) => {
    const { team, user2 } = await setupTeamWithMembers(request, testUser, testProject);

    const today = new Date().toISOString().slice(0, 10);
    await api.createOncallRotation(request, testUser.token, testProject.key, team.id, {
      period_days: 7,
      rotation_time: '12:00:00',
      timezone: 'UTC',
      start_date: today,
      member_ids: [testUser.id, user2.id],
    });

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);
    await page.getByRole('button', { name: 'On-Call', exact: true }).click();
    await expect(page.getByText('Every 7 days')).toBeVisible({ timeout: 10000 });

    // Open Edit Rotation modal
    await page.getByRole('button', { name: 'Edit Rotation' }).click();
    await expect(page.getByRole('heading', { name: 'Edit Rotation' })).toBeVisible();

    // Change period to 14 days
    const periodInput = page.getByLabel('Period (days)');
    await periodInput.clear();
    await periodInput.fill('14');

    // Save
    await page.getByRole('button', { name: 'Save', exact: true }).click();

    // Verify updated
    await expect(page.getByText('Every 14 days')).toBeVisible({ timeout: 10000 });
  });

  test('create oncall rotation via API and verify on page', async ({ request, testUser, testProject, page }) => {
    const { team, user2 } = await setupTeamWithMembers(request, testUser, testProject);

    const today = new Date().toISOString().slice(0, 10);
    await api.createOncallRotation(request, testUser.token, testProject.key, team.id, {
      period_days: 14,
      rotation_time: '09:00:00',
      timezone: 'UTC',
      start_date: today,
      member_ids: [testUser.id, user2.id],
    });

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);
    await page.getByRole('button', { name: 'On-Call', exact: true }).click();

    // Verify rotation details in sidebar
    await expect(page.getByText('Every 14 days')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('UTC')).toBeVisible();
  });

  test('delete oncall rotation via UI', async ({ request, testUser, testProject, page }) => {
    const { team, user2 } = await setupTeamWithMembers(request, testUser, testProject);

    const today = new Date().toISOString().slice(0, 10);
    await api.createOncallRotation(request, testUser.token, testProject.key, team.id, {
      period_days: 7,
      rotation_time: '12:00:00',
      timezone: 'UTC',
      start_date: today,
      member_ids: [testUser.id, user2.id],
    });

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);
    await page.getByRole('button', { name: 'On-Call', exact: true }).click();
    await expect(page.getByText('Every 7 days')).toBeVisible({ timeout: 10000 });

    // Open Edit Rotation modal
    await page.getByRole('button', { name: 'Edit Rotation' }).click();
    await expect(page.getByRole('heading', { name: 'Edit Rotation' })).toBeVisible();

    // Click the delete (trash icon) button inside the edit modal form footer
    const modal = page.getByRole('dialog')
    await modal.locator('button').filter({ has: page.locator('.text-red-500') }).click();

    // Confirm deletion
    await expect(page.getByRole('heading', { name: 'Delete On-Call Rotation' })).toBeVisible();
    await page.getByRole('button', { name: 'Delete', exact: true }).click();

    // Verify empty state returns
    await expect(page.getByText('No on-call rotation configured for this team.')).toBeVisible({ timeout: 10000 });
  });

  test('oncall rotation history appears after creation', async ({ request, testUser, testProject, page }) => {
    const { team, user2 } = await setupTeamWithMembers(request, testUser, testProject);

    const today = new Date().toISOString().slice(0, 10);
    await api.createOncallRotation(request, testUser.token, testProject.key, team.id, {
      period_days: 7,
      rotation_time: '12:00:00',
      timezone: 'UTC',
      start_date: today,
      member_ids: [testUser.id, user2.id],
    });

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);
    await page.getByRole('button', { name: 'On-Call', exact: true }).click();

    // History section should exist with entries
    await expect(page.getByText('Rotation History')).toBeVisible({ timeout: 10000 });
    // The first member should have a history entry showing a date range with "Current"
    await expect(page.getByText(/- Current$/)).toBeVisible();
  });

  test('oncall calendar is visible', async ({ request, testUser, testProject, page }) => {
    const { team, user2 } = await setupTeamWithMembers(request, testUser, testProject);

    const today = new Date().toISOString().slice(0, 10);
    await api.createOncallRotation(request, testUser.token, testProject.key, team.id, {
      period_days: 7,
      rotation_time: '12:00:00',
      timezone: 'UTC',
      start_date: today,
      member_ids: [testUser.id, user2.id],
    });

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);
    await page.getByRole('button', { name: 'On-Call', exact: true }).click();

    // Calendar should be visible with day headers
    await expect(page.getByText('Mon')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Tue')).toBeVisible();
    await expect(page.getByText('Wed')).toBeVisible();

    // Today button should be visible
    await expect(page.getByRole('button', { name: 'Today' })).toBeVisible();
  });

  test('oncall rotation create, get, delete via API', async ({ request, testUser, testProject }) => {
    const { team, user2 } = await setupTeamWithMembers(request, testUser, testProject);

    const today = new Date().toISOString().slice(0, 10);

    // Create
    const rotation = await api.createOncallRotation(request, testUser.token, testProject.key, team.id, {
      period_days: 7,
      rotation_time: '09:00:00',
      timezone: 'America/New_York',
      start_date: today,
      member_ids: [testUser.id, user2.id],
    });
    expect(rotation.members).toHaveLength(2);
    expect(rotation.current_user_id).toBe(testUser.id);

    // Get
    const fetched = await api.getOncallRotation(request, testUser.token, testProject.key, team.id);
    expect(fetched.period_days).toBe(7);
    expect(fetched.timezone).toBe('America/New_York');
    expect(fetched.members).toHaveLength(2);

    // History
    const history = await api.getOncallHistory(request, testUser.token, testProject.key, team.id);
    expect(history.length).toBeGreaterThanOrEqual(1);
    expect(history[0].user_id).toBe(testUser.id);

    // Delete
    await api.deleteOncallRotation(request, testUser.token, testProject.key, team.id);

    // Get should return 404
    const res = await request.get(
      `${process.env.BASE_URL || 'http://localhost:5173'}/api/v1/default/projects/${testProject.key}/teams/${team.id}/oncall`,
      { headers: { Authorization: `Bearer ${testUser.token}` } },
    );
    expect(res.status()).toBe(404);
  });

  test('oncall rotation submit is disabled with fewer than 2 members', async ({ request, testUser, testProject, page }) => {
    // Create team with only one member
    const team = await api.createTeam(request, testUser.token, testProject.key, {
      name: 'Single Member Team',
    });
    await api.addTeamMember(request, testUser.token, testProject.key, team.id, testUser.id);

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);
    await page.getByRole('button', { name: 'On-Call', exact: true }).click();
    await page.getByRole('button', { name: 'Set Up On-Call' }).click();

    // Check only the one available member
    await page.locator('input[type="checkbox"]').first().check();

    // Create button should be disabled because < 2 participants
    await expect(page.getByRole('button', { name: 'Create', exact: true })).toBeDisabled();
  });
});
