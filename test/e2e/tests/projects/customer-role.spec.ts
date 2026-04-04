import { test, expect, getAdminToken } from '../../lib/fixtures';
import type { BrowserContext, Page } from '@playwright/test';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

const TEST_PASSWORD = 'TestPass123!';

/** Log in as a different user by clearing state and filling the login form. */
async function loginAs(page: Page, context: BrowserContext, email: string, password: string) {
  await page.goto('/login');
  await context.clearCookies();
  await page.evaluate(() => localStorage.clear());
  await page.goto('/login');
  await page.getByLabel('Email').fill(email);
  await page.getByLabel('Password').fill(password);
  await page.getByRole('button', { name: 'Sign in' }).click();
}

/**
 * Create a helper user who is set up (password changed, welcome dismissed).
 * Does NOT add them to any project.
 */
async function createReadyUser(
  request: import('@playwright/test').APIRequestContext,
  adminToken: string,
) {
  const uniqueId = randomUUID().slice(0, 8);
  const email = `cust-role-${uniqueId}@test.local`;
  const displayName = `CustRole ${uniqueId}`;
  const created = await api.createUser(request, adminToken, email, displayName);
  const tempLogin = await api.login(request, email, created.temporary_password);
  await api.changePassword(request, tempLogin.token, created.temporary_password, TEST_PASSWORD);
  const finalLogin = await api.login(request, email, TEST_PASSWORD);
  await api.setPreference(request, finalLogin.token, 'welcome_dismissed', true);
  return { id: finalLogin.user.id, email, displayName, token: finalLogin.token, password: TEST_PASSWORD };
}

test.describe('Customer role — project list and routing', () => {

  test('project list API returns member_role for each project', async ({ request, testUser, testProject }) => {
    const adminToken = getAdminToken();

    // Create a second project owned by admin, add testUser as customer
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const custProjKey = `C${suffix}`;
    const custProjName = `Cust Project ${suffix}`;
    await api.createProject(request, adminToken, custProjKey, custProjName);
    await api.addMember(request, adminToken, custProjKey, testUser.id, 'customer');

    // List projects via API
    const projects = await api.listProjects(request, testUser.token);

    const ownedProj = projects.find((p) => p.key === testProject.key);
    expect(ownedProj).toBeDefined();
    expect(ownedProj!.member_role).toBe('owner');

    const custProj = projects.find((p) => p.key === custProjKey);
    expect(custProj).toBeDefined();
    expect(custProj!.member_role).toBe('customer');
  });

  test('/me returns total_project_count and portal_projects for customer', async ({ request, testUser, testProject }) => {
    const adminToken = getAdminToken();

    // Create a second project owned by admin, add testUser as customer
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const custProjKey = `M${suffix}`;
    await api.createProject(request, adminToken, custProjKey, `Me Test ${suffix}`);
    await api.addMember(request, adminToken, custProjKey, testUser.id, 'customer');

    const me = await api.getMe(request, testUser.token);
    expect(me.total_project_count).toBe(2);
    expect(me.portal_projects).toBeDefined();
    expect(me.portal_projects!.some((pp) => pp.project_key === custProjKey)).toBe(true);
    // The owned project should NOT be in portal_projects
    expect(me.portal_projects!.some((pp) => pp.project_key === testProject.key)).toBe(false);
  });

  test('customer project in AppShell shows support-only sidebar', async ({ page, request, testUser, testProject }) => {
    const adminToken = getAdminToken();

    // Create a project owned by admin and add testUser as customer
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const custProjKey = `S${suffix}`;
    await api.createProject(request, adminToken, custProjKey, `Sidebar Test ${suffix}`);
    await api.addMember(request, adminToken, custProjKey, testUser.id, 'customer');

    // Ensure public queue exists (needed for portal tickets)
    await api.createQueue(request, adminToken, custProjKey, {
      name: 'Public Queue',
      queue_type: 'support',
      is_public: true,
    });

    // Navigate to the customer project in AppShell
    await page.goto(`/d/projects/${custProjKey}`);
    await page.waitForLoadState('domcontentloaded');

    // Should redirect to /support sub-route
    await expect(page).toHaveURL(new RegExp(`/d/projects/${custProjKey}/support`), { timeout: 10000 });

    // Sidebar should show "Support" but not "Overview" or "Items"
    const sidebar = page.locator('nav.hidden.sm\\:block');
    await expect(sidebar.getByText('Support')).toBeVisible({ timeout: 5000 });
    await expect(sidebar.getByText('Overview')).not.toBeVisible();
    await expect(sidebar.getByText('Items')).not.toBeVisible();
    await expect(sidebar.getByText('Settings')).not.toBeVisible();

    // Navigate to the owned project — should show full sidebar
    await page.goto(`/d/projects/${testProject.key}/items`);
    await page.waitForLoadState('domcontentloaded');
    await expect(sidebar.getByText('Overview')).toBeVisible({ timeout: 10000 });
    await expect(sidebar.getByText('Items')).toBeVisible();
    await expect(sidebar.getByText('Settings')).toBeVisible();
  });

  test('project list UI shows role column', async ({ page, request, testUser, testProject }) => {
    const adminToken = getAdminToken();

    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const custProjKey = `R${suffix}`;
    await api.createProject(request, adminToken, custProjKey, `Role Col ${suffix}`);
    await api.addMember(request, adminToken, custProjKey, testUser.id, 'customer');

    // Go to project list (desktop view for table)
    await page.setViewportSize({ width: 1280, height: 720 });
    await page.goto('/d/projects');
    await page.waitForLoadState('domcontentloaded');

    // Table should have a "Role" column header
    await expect(page.getByText('Role', { exact: true }).first()).toBeVisible({ timeout: 10000 });

    // The customer project row should show "Customer" role
    const custRow = page.locator('tr', { hasText: custProjKey });
    await expect(custRow.getByText(/customer/i)).toBeVisible({ timeout: 5000 });

    // The owned project row should show "Owner" role
    const ownedRow = page.locator('tr', { hasText: testProject.key });
    await expect(ownedRow.getByText(/owner/i)).toBeVisible();
  });

  test('clicking customer project in list navigates to support page', async ({ page, request, testUser, testProject }) => {
    const adminToken = getAdminToken();

    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const custProjKey = `N${suffix}`;
    await api.createProject(request, adminToken, custProjKey, `Nav Test ${suffix}`);
    await api.addMember(request, adminToken, custProjKey, testUser.id, 'customer');

    await page.setViewportSize({ width: 1280, height: 720 });
    await page.goto('/d/projects');
    await page.waitForLoadState('domcontentloaded');

    // Click on the customer project row
    await page.locator('tr', { hasText: custProjKey }).click();
    await expect(page).toHaveURL(new RegExp(`/projects/${custProjKey}/support`), { timeout: 10000 });
  });

  test('non-portal-only user redirected from /portal to AppShell support', async ({ page, request, testUser, testProject }) => {
    const adminToken = getAdminToken();

    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const custProjKey = `P${suffix}`;
    await api.createProject(request, adminToken, custProjKey, `Portal Guard ${suffix}`);
    await api.addMember(request, adminToken, custProjKey, testUser.id, 'customer');

    // testUser owns testProject AND is customer of custProjKey → NOT portal-only
    // Navigating to /portal should redirect to AppShell
    await page.goto(`/portal/d/projects/${custProjKey}/tickets`);
    await page.waitForLoadState('domcontentloaded');

    // Should be redirected away from /portal to the regular support page
    await expect(page).toHaveURL(new RegExp(`/d/projects/${custProjKey}/support`), { timeout: 10000 });
  });
});

test.describe('Customer role — project creation restriction', () => {

  test('customer-only user cannot create projects via API', async ({ request }) => {
    const adminToken = getAdminToken();

    // Create a fresh user
    const user = await createReadyUser(request, adminToken);

    // Create a project owned by admin and add user as customer
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const projKey = `A${suffix}`;
    await api.createProject(request, adminToken, projKey, `Admin Proj ${suffix}`);
    await api.addMember(request, adminToken, projKey, user.id, 'customer');

    // User attempts to create a project — should be forbidden
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';
    const res = await request.post(`${BASE_URL}/api/v1/default/projects`, {
      headers: { Authorization: `Bearer ${user.token}` },
      data: { key: `X${suffix}`, name: 'Should Fail' },
    });
    expect(res.status()).toBe(403);

    // Cleanup
    await api.deactivateUser(request, adminToken, user.id).catch(() => {});
  });

  test('portal-only user is redirected to portal on login and from AppShell', async ({ page, request }) => {
    const adminToken = getAdminToken();
    const user = await createReadyUser(request, adminToken);

    // Create a project owned by admin and add user as customer
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const projKey = `B${suffix}`;
    await api.createProject(request, adminToken, projKey, `Portal Redir ${suffix}`);
    await api.addMember(request, adminToken, projKey, user.id, 'customer');

    // Login as the portal-only user (single customer project, total_project_count=1)
    await loginAs(page, page.context(), user.email, user.password);

    // Should auto-redirect to the portal ticket list
    await page.waitForURL(/\/portal\/.*\/tickets/, { timeout: 15000 });
    expect(page.url()).toContain(projKey);

    // Attempting to navigate to AppShell should redirect back to portal
    await page.goto('/d/projects');
    await page.waitForURL(/\/portal\//, { timeout: 10000 });

    // Cleanup
    await api.deactivateUser(request, adminToken, user.id).catch(() => {});
  });
});
