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

/** Set up a team with two members and a rotation. */
async function setupTeamWithRotation(
  request: import('@playwright/test').APIRequestContext,
  testUser: { id: string; token: string },
  testProject: { key: string },
) {
  const adminToken = getAdminToken();
  const user2 = await createSecondUser(request, adminToken, testProject.key, testUser.token);

  const team = await api.createTeam(request, testUser.token, testProject.key, {
    name: 'Override Team',
  });
  await api.addTeamMember(request, testUser.token, testProject.key, team.id, testUser.id);
  await api.addTeamMember(request, testUser.token, testProject.key, team.id, user2.id);

  const today = new Date().toISOString().slice(0, 10);
  const rotation = await api.createOncallRotation(request, testUser.token, testProject.key, team.id, {
    period_days: 7,
    rotation_time: '12:00:00',
    timezone: 'UTC',
    start_date: today,
    member_ids: [testUser.id, user2.id],
  });

  return { team, user2, rotation };
}

test.describe('Oncall overrides', () => {
  test('create, list, and delete override via API', async ({ request, testUser, testProject }) => {
    const { team, user2 } = await setupTeamWithRotation(request, testUser, testProject);

    const startAt = new Date(Date.now() + 1 * 60 * 60 * 1000).toISOString(); // +1h
    const endAt = new Date(Date.now() + 25 * 60 * 60 * 1000).toISOString();   // +25h

    // Create override
    const override = await api.createOncallOverride(request, testUser.token, testProject.key, team.id, {
      override_user_id: user2.id,
      start_at: startAt,
      end_at: endAt,
      reason: 'Vacation coverage',
    });
    expect(override.override_user_id).toBe(user2.id);
    expect(override.reason).toBe('Vacation coverage');

    // List overrides
    const overrides = await api.listOncallOverrides(request, testUser.token, testProject.key, team.id);
    expect(overrides.length).toBeGreaterThanOrEqual(1);
    const found = overrides.find((o) => o.id === override.id);
    expect(found).toBeTruthy();
    expect(found!.override_user_name).toBeTruthy();

    // Delete override
    await api.deleteOncallOverride(request, testUser.token, testProject.key, team.id, override.id);

    // Verify deleted
    const afterDelete = await api.listOncallOverrides(request, testUser.token, testProject.key, team.id);
    expect(afterDelete.find((o) => o.id === override.id)).toBeFalsy();
  });

  test('active override appears in rotation response', async ({ request, testUser, testProject }) => {
    const { team, user2 } = await setupTeamWithRotation(request, testUser, testProject);

    // Create an active override (starts now)
    const startAt = new Date(Date.now() - 5 * 60 * 1000).toISOString(); // -5min (active now)
    const endAt = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(); // +24h

    await api.createOncallOverride(request, testUser.token, testProject.key, team.id, {
      override_user_id: user2.id,
      start_at: startAt,
      end_at: endAt,
    });

    // Get rotation — should include is_override flag and overrides array
    const rotation = await api.getOncallRotation(request, testUser.token, testProject.key, team.id);
    expect(rotation.is_override).toBe(true);
    expect(rotation.overrides.length).toBeGreaterThanOrEqual(1);
    expect(rotation.overrides.some((o: any) => o.override_user_id === user2.id)).toBe(true);
  });

  test('member can create override for themselves', async ({ request, testUser, testProject }) => {
    const { team, user2 } = await setupTeamWithRotation(request, testUser, testProject);

    // user2 (regular member) creates an override for themselves
    const startAt = new Date(Date.now() + 1 * 60 * 60 * 1000).toISOString();
    const endAt = new Date(Date.now() + 25 * 60 * 60 * 1000).toISOString();

    const override = await api.createOncallOverride(request, user2.token, testProject.key, team.id, {
      override_user_id: user2.id,
      start_at: startAt,
      end_at: endAt,
    });
    expect(override.override_user_id).toBe(user2.id);
  });

  test('member cannot create override for someone else', async ({ request, testUser, testProject }) => {
    const { team, user2 } = await setupTeamWithRotation(request, testUser, testProject);

    const startAt = new Date(Date.now() + 1 * 60 * 60 * 1000).toISOString();
    const endAt = new Date(Date.now() + 25 * 60 * 60 * 1000).toISOString();

    // user2 (member) tries to create override for testUser (owner) — should fail
    const res = await request.post(
      `${process.env.BASE_URL || 'http://localhost:5173'}/api/v1/default/projects/${testProject.key}/teams/${team.id}/oncall/overrides`,
      {
        headers: { Authorization: `Bearer ${user2.token}` },
        data: {
          override_user_id: testUser.id,
          start_at: startAt,
          end_at: endAt,
        },
      },
    );
    expect(res.status()).toBe(403);
  });

  test('create override via UI', async ({ request, testUser, testProject, page }) => {
    const { team, user2 } = await setupTeamWithRotation(request, testUser, testProject);

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);
    await page.getByRole('button', { name: 'On-Call', exact: true }).click();
    await expect(page.getByText('Every 7 days')).toBeVisible({ timeout: 10000 });

    // Click "Add Override" button
    await page.getByRole('button', { name: 'Add Override' }).click();
    await expect(page.getByRole('heading', { name: 'Create On-Call Override' })).toBeVisible();

    // Select a covering member via UserPicker dropdown
    await page.getByText('Unassigned').click();
    await page.locator('ul').getByText(user2.displayName).click();

    // Set start and end times
    const now = new Date();
    const start = new Date(now.getTime() + 60 * 60 * 1000);
    const end = new Date(now.getTime() + 25 * 60 * 60 * 1000);

    const formatForInput = (d: Date) => {
      const y = d.getFullYear();
      const m = String(d.getMonth() + 1).padStart(2, '0');
      const day = String(d.getDate()).padStart(2, '0');
      const h = String(d.getHours()).padStart(2, '0');
      const min = String(d.getMinutes()).padStart(2, '0');
      return `${y}-${m}-${day}T${h}:${min}`;
    };

    await page.getByLabel('Start').fill(formatForInput(start));
    await page.getByLabel('End').fill(formatForInput(end));

    // Optionally add a reason
    await page.getByLabel('Reason (optional)').fill('Vacation coverage');

    // Submit
    await page.getByRole('button', { name: 'Create', exact: true }).click();

    // Verify override appears in the overrides panel
    await expect(page.getByText('Vacation coverage')).toBeVisible({ timeout: 10000 });
  });

  test('cancel override via UI', async ({ request, testUser, testProject, page }) => {
    const { team, user2 } = await setupTeamWithRotation(request, testUser, testProject);

    // Create an override via API
    const startAt = new Date(Date.now() + 1 * 60 * 60 * 1000).toISOString();
    const endAt = new Date(Date.now() + 25 * 60 * 60 * 1000).toISOString();

    await api.createOncallOverride(request, testUser.token, testProject.key, team.id, {
      override_user_id: user2.id,
      start_at: startAt,
      end_at: endAt,
      reason: 'To be cancelled',
    });

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);
    await page.getByRole('button', { name: 'On-Call', exact: true }).click();
    await expect(page.getByText('To be cancelled')).toBeVisible({ timeout: 10000 });

    // Click the cancel (X) button on the override entry
    const overrideEntry = page.locator('div').filter({ hasText: 'To be cancelled' }).first();
    await overrideEntry.locator('button').filter({ has: page.locator('.text-red-500') }).click();

    // Confirm cancellation
    await expect(page.getByRole('heading', { name: 'Cancel Override' })).toBeVisible();
    await page.getByRole('button', { name: 'Delete', exact: true }).click();

    // Verify the override is gone
    await expect(page.getByText('To be cancelled')).not.toBeVisible({ timeout: 10000 });
  });

  test('edit override via UI', async ({ request, testUser, testProject, page }) => {
    const { team, user2 } = await setupTeamWithRotation(request, testUser, testProject);

    const startAt = new Date(Date.now() + 1 * 60 * 60 * 1000).toISOString();
    const endAt = new Date(Date.now() + 25 * 60 * 60 * 1000).toISOString();

    await api.createOncallOverride(request, testUser.token, testProject.key, team.id, {
      override_user_id: user2.id,
      start_at: startAt,
      end_at: endAt,
      reason: 'Original reason',
    });

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);
    await page.getByRole('button', { name: 'On-Call', exact: true }).click();
    await expect(page.getByText('Original reason')).toBeVisible({ timeout: 10000 });

    // Click the edit (pencil) button — it's the first button in the action group (before trash)
    const actionGroup = page.locator('.gap-0\\.5').filter({ has: page.locator('.text-red-500') }).first();
    await actionGroup.locator('button').first().click();

    // Edit modal should open
    await expect(page.getByRole('heading', { name: 'Edit Override' })).toBeVisible({ timeout: 10000 });

    // Change the reason
    const reasonInput = page.getByLabel('Reason (optional)');
    await reasonInput.clear();
    await reasonInput.fill('Updated reason');

    // Save
    await page.getByRole('button', { name: 'Save', exact: true }).click();

    // Verify updated
    await expect(page.getByText('Updated reason')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Original reason')).not.toBeVisible();
  });

  test('immediate override sets is_override flag', async ({ request, testUser, testProject }) => {
    const { team, user2 } = await setupTeamWithRotation(request, testUser, testProject);

    // Create an override that started 5 min ago (immediately active)
    const startAt = new Date(Date.now() - 5 * 60 * 1000).toISOString();
    const endAt = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();

    await api.createOncallOverride(request, testUser.token, testProject.key, team.id, {
      override_user_id: user2.id,
      start_at: startAt,
      end_at: endAt,
    });

    const rotation = await api.getOncallRotation(request, testUser.token, testProject.key, team.id);
    expect(rotation.is_override).toBe(true);
    expect(rotation.current_user_id).toBe(user2.id);
  });

  test('deleting active override restores scheduled user', async ({ request, testUser, testProject }) => {
    // Use a custom rotation with rotation_time=00:00:00 to ensure the epoch
    // is in the past regardless of what time the test runs.
    const adminToken = getAdminToken();
    const user2 = await createSecondUser(request, adminToken, testProject.key, testUser.token);
    const team = await api.createTeam(request, testUser.token, testProject.key, { name: 'Delete Override Team' });
    await api.addTeamMember(request, testUser.token, testProject.key, team.id, testUser.id);
    await api.addTeamMember(request, testUser.token, testProject.key, team.id, user2.id);
    const today = new Date().toISOString().slice(0, 10);
    await api.createOncallRotation(request, testUser.token, testProject.key, team.id, {
      period_days: 7,
      rotation_time: '00:00:00',
      timezone: 'UTC',
      start_date: today,
      member_ids: [testUser.id, user2.id],
    });

    // Create an active override (starts 5 min ago)
    const startAt = new Date(Date.now() - 5 * 60 * 1000).toISOString();
    const endAt = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();

    const override = await api.createOncallOverride(request, testUser.token, testProject.key, team.id, {
      override_user_id: user2.id,
      start_at: startAt,
      end_at: endAt,
    });

    // Verify override is active
    const rotationBefore = await api.getOncallRotation(request, testUser.token, testProject.key, team.id);
    expect(rotationBefore.is_override).toBe(true);
    expect(rotationBefore.current_user_id).toBe(user2.id);

    // Delete the override
    await api.deleteOncallOverride(request, testUser.token, testProject.key, team.id, override.id);

    // Verify scheduled user is restored
    const rotationAfter = await api.getOncallRotation(request, testUser.token, testProject.key, team.id);
    expect(rotationAfter.is_override).toBe(false);
    expect(rotationAfter.current_user_id).toBe(testUser.id);
  });

  test('invalid time range rejected', async ({ request, testUser, testProject }) => {
    const { team, user2 } = await setupTeamWithRotation(request, testUser, testProject);

    // end_at before start_at
    const res = await request.post(
      `${process.env.BASE_URL || 'http://localhost:5173'}/api/v1/default/projects/${testProject.key}/teams/${team.id}/oncall/overrides`,
      {
        headers: { Authorization: `Bearer ${testUser.token}` },
        data: {
          override_user_id: user2.id,
          start_at: new Date(Date.now() + 25 * 60 * 60 * 1000).toISOString(),
          end_at: new Date(Date.now() + 1 * 60 * 60 * 1000).toISOString(),
        },
      },
    );
    expect(res.status()).toBe(400);
  });
});
