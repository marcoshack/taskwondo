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

/** Set up a team with two members. */
async function setupTeamWithMembers(
  request: import('@playwright/test').APIRequestContext,
  testUser: { id: string; token: string },
  testProject: { key: string },
) {
  const adminToken = getAdminToken();
  const user2 = await createSecondUser(request, adminToken, testProject.key, testUser.token);

  const team = await api.createTeam(request, testUser.token, testProject.key, {
    name: 'Schedule Team',
  });
  await api.addTeamMember(request, testUser.token, testProject.key, team.id, testUser.id);
  await api.addTeamMember(request, testUser.token, testProject.key, team.id, user2.id);

  return { team, user2 };
}

/** Return a future date string YYYY-MM-DD, offset days from now. */
function futureDate(offsetDays: number): string {
  const d = new Date();
  d.setDate(d.getDate() + offsetDays);
  return d.toISOString().slice(0, 10);
}

/** Return a future ISO timestamp, offset days from now. */
function futureISO(offsetDays: number): string {
  const d = new Date();
  d.setDate(d.getDate() + offsetDays);
  d.setHours(12, 0, 0, 0);
  return d.toISOString();
}

test.describe('Oncall schedule endpoint', () => {
  test('returns projected shifts for a date range', async ({ request, testUser, testProject }) => {
    const { team, user2 } = await setupTeamWithMembers(request, testUser, testProject);

    const startDate = futureDate(1);
    await api.createOncallRotation(request, testUser.token, testProject.key, team.id, {
      period_days: 7,
      rotation_time: '12:00:00',
      timezone: 'UTC',
      start_date: startDate,
      member_ids: [testUser.id, user2.id],
    });

    const queryStart = futureDate(1);
    const queryEnd = futureDate(30);
    const schedule = await api.getOncallSchedule(
      request, testUser.token, testProject.key, team.id,
      queryStart, queryEnd,
    );

    // Should have rotation config
    expect(schedule.period_days).toBe(7);
    expect(schedule.timezone).toBe('UTC');
    expect(schedule.members).toHaveLength(2);

    // Should have shifts covering the range — at least 4 shifts for ~30 days / 7-day period
    expect(schedule.shifts.length).toBeGreaterThanOrEqual(4);

    // Shifts should alternate between testUser and user2
    const userIds = schedule.shifts.map((s: { user_id: string }) => s.user_id);
    const uniqueUsers = [...new Set(userIds)];
    expect(uniqueUsers).toHaveLength(2);
    expect(uniqueUsers).toContain(testUser.id);
    expect(uniqueUsers).toContain(user2.id);

    // All shifts should have no override
    for (const shift of schedule.shifts) {
      expect(shift.is_override).toBe(false);
    }

    // Shifts should be contiguous (each shift ends where the next starts)
    for (let i = 1; i < schedule.shifts.length; i++) {
      expect(schedule.shifts[i].start_at).toBe(schedule.shifts[i - 1].end_at);
    }
  });

  test('applies overrides to the schedule', async ({ request, testUser, testProject }) => {
    const { team, user2 } = await setupTeamWithMembers(request, testUser, testProject);

    const startDate = futureDate(1);
    await api.createOncallRotation(request, testUser.token, testProject.key, team.id, {
      period_days: 7,
      rotation_time: '12:00:00',
      timezone: 'UTC',
      start_date: startDate,
      member_ids: [testUser.id, user2.id],
    });

    // Create override: user2 takes over 3-5 days from now
    await api.createOncallOverride(request, testUser.token, testProject.key, team.id, {
      override_user_id: user2.id,
      start_at: futureISO(3),
      end_at: futureISO(5),
    });

    const schedule = await api.getOncallSchedule(
      request, testUser.token, testProject.key, team.id,
      futureDate(1), futureDate(10),
    );

    // Should have an override shift
    const overrideShifts = schedule.shifts.filter((s: { is_override: boolean }) => s.is_override);
    expect(overrideShifts.length).toBeGreaterThanOrEqual(1);
    expect(overrideShifts[0].user_id).toBe(user2.id);

    // Overrides should be in the response
    expect(schedule.overrides).toHaveLength(1);
  });

  test('returns rotation config, members, and overrides in single response', async ({ request, testUser, testProject }) => {
    const { team, user2 } = await setupTeamWithMembers(request, testUser, testProject);

    const startDate = futureDate(1);
    await api.createOncallRotation(request, testUser.token, testProject.key, team.id, {
      period_days: 14,
      rotation_time: '09:00:00',
      timezone: 'America/New_York',
      start_date: startDate,
      member_ids: [testUser.id, user2.id],
    });

    const schedule = await api.getOncallSchedule(
      request, testUser.token, testProject.key, team.id,
      futureDate(1), futureDate(30),
    );

    // Verify rotation config
    expect(schedule.period_days).toBe(14);
    expect(schedule.timezone).toBe('America/New_York');
    expect(schedule.rotation_time).toContain('09:00:00');

    // Verify members
    expect(schedule.members).toHaveLength(2);
    expect(schedule.members[0].user_id).toBeDefined();
    expect(schedule.members[0].display_name).toBeDefined();

    // Verify current state
    expect(schedule.current_user_id).toBe(testUser.id);
    expect(schedule.next_rotation_at).toBeDefined();
  });

  test('calendar renders shifts from schedule endpoint', async ({ request, testUser, testProject, page }) => {
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

    // Calendar should render with day headers
    await expect(page.getByText('Mon')).toBeVisible({ timeout: 10000 });

    // Both members should appear in the calendar legend
    await expect(page.getByText(testUser.displayName, { exact: true }).last()).toBeVisible();
  });

  test('schedule endpoint validates date range', async ({ request, testUser, testProject }) => {
    const { team } = await setupTeamWithMembers(request, testUser, testProject);

    const today = new Date().toISOString().slice(0, 10);
    await api.createOncallRotation(request, testUser.token, testProject.key, team.id, {
      period_days: 7,
      rotation_time: '12:00:00',
      timezone: 'UTC',
      start_date: today,
      member_ids: [testUser.id],
    });

    // Missing params
    const res1 = await request.get(
      `${process.env.BASE_URL || 'http://localhost:5173'}/api/v1/default/projects/${testProject.key}/teams/${team.id}/oncall/schedule`,
      { headers: { Authorization: `Bearer ${testUser.token}` } },
    );
    expect(res1.status()).toBe(400);

    // Range > 90 days
    const res2 = await request.get(
      `${process.env.BASE_URL || 'http://localhost:5173'}/api/v1/default/projects/${testProject.key}/teams/${team.id}/oncall/schedule?start=2026-01-01&end=2026-12-31`,
      { headers: { Authorization: `Bearer ${testUser.token}` } },
    );
    expect(res2.status()).toBe(400);
  });
});
