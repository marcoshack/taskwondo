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

test.describe('Team members', () => {
  test('add and remove team member via UI', async ({ request, testUser, testProject, page }) => {
    const adminToken = getAdminToken();
    const user2 = await createSecondUser(request, adminToken, testProject.key, testUser.token);

    const team = await api.createTeam(request, testUser.token, testProject.key, {
      name: 'Member Test Team',
    });

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);

    // Members tab is active by default — should show empty state
    await expect(page.getByText('No members in this team.')).toBeVisible({ timeout: 10000 });

    // Search for the second user to narrow the available list
    await page.getByPlaceholder(/search/i).fill(user2.displayName);

    // Click the user in the dropdown to add them
    await page.locator('ul').getByText(user2.displayName).click();

    // Verify the user appears in the members list
    await expect(page.getByText(user2.displayName)).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(user2.email)).toBeVisible();

    // Remove the member
    await page.getByRole('button', { name: '' }).filter({ has: page.locator('.text-red-500') }).first().click();

    // Confirm removal in modal
    await expect(page.getByRole('heading', { name: 'Remove Team Member' })).toBeVisible();
    await page.getByRole('button', { name: 'Remove', exact: true }).click();

    // Verify member is gone
    await expect(page.getByText('No members in this team.')).toBeVisible({ timeout: 10000 });
  });

  test('add team member via API and verify on page', async ({ request, testUser, testProject, page }) => {
    const adminToken = getAdminToken();
    const user2 = await createSecondUser(request, adminToken, testProject.key, testUser.token);

    const team = await api.createTeam(request, testUser.token, testProject.key, {
      name: 'API Member Team',
    });

    // Add member via API
    await api.addTeamMember(request, testUser.token, testProject.key, team.id, user2.id);

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);

    // Verify member is shown on Members tab
    await expect(page.getByText(user2.displayName)).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(user2.email)).toBeVisible();
  });

  test('member count badge updates on teams list page', async ({ request, testUser, testProject, page }) => {
    const adminToken = getAdminToken();
    const user2 = await createSecondUser(request, adminToken, testProject.key, testUser.token);

    const team = await api.createTeam(request, testUser.token, testProject.key, {
      name: 'Count Team',
    });

    // Add two members via API (owner + second user)
    await api.addTeamMember(request, testUser.token, testProject.key, team.id, testUser.id);
    await api.addTeamMember(request, testUser.token, testProject.key, team.id, user2.id);

    await page.goto(`/d/projects/${testProject.key}/settings?tab=teams`);

    // Verify the member count badge shows 2
    await expect(page.getByText('Count Team')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('2 members')).toBeVisible();
  });

  test('viewers are not available for team membership', async ({ request, testUser, testProject, page }) => {
    const adminToken = getAdminToken();
    const viewer = await createSecondUser(request, adminToken, testProject.key, testUser.token, 'viewer');

    const team = await api.createTeam(request, testUser.token, testProject.key, {
      name: 'Viewer Exclusion Team',
    });

    await page.goto(`/d/projects/${testProject.key}/teams/${team.id}`);
    await expect(page.getByText('No members in this team.')).toBeVisible({ timeout: 10000 });

    // Search for viewer — should not be available (viewers are excluded)
    await page.getByPlaceholder(/search/i).fill(viewer.displayName);
    await expect(page.getByText('No results')).toBeVisible({ timeout: 5000 });
  });

  test('team members CRUD via API only', async ({ request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    const user2 = await createSecondUser(request, adminToken, testProject.key, testUser.token);

    const team = await api.createTeam(request, testUser.token, testProject.key, {
      name: 'API CRUD Team',
    });

    // List should be empty
    let members = await api.listTeamMembers(request, testUser.token, testProject.key, team.id);
    expect(members).toHaveLength(0);

    // Add member
    await api.addTeamMember(request, testUser.token, testProject.key, team.id, user2.id);

    // List should have one member
    members = await api.listTeamMembers(request, testUser.token, testProject.key, team.id);
    expect(members).toHaveLength(1);
    expect(members[0].user_id).toBe(user2.id);

    // Remove member
    await api.removeTeamMember(request, testUser.token, testProject.key, team.id, user2.id);

    // List should be empty again
    members = await api.listTeamMembers(request, testUser.token, testProject.key, team.id);
    expect(members).toHaveLength(0);
  });
});
