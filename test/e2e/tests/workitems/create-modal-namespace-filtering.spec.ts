import { test, expect, getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

/**
 * These tests verify that the "New Item" modal on the Inbox and Watchlist
 * pages only shows projects where the user can actually create work items —
 * i.e. projects where their role is owner/admin/member. Projects where the
 * user is a customer in another namespace must be hidden, since the backend
 * rejects creation there and customers are expected to open tickets via the
 * Support page instead.
 *
 * Scenario for each test:
 *   - User is owner of a project in their own namespace
 *   - User is customer of a project in a different namespace
 *   - The project picker in the New Item modal must show the owned project
 *     and must NOT show the customer project.
 */

const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

interface Setup {
  ownNsSlug: string;
  ownProjKey: string;
  ownProjName: string;
  custNsSlug: string;
  custProjKey: string;
  custProjName: string;
}

/**
 * Creates two namespaces and two projects:
 *   - Own namespace + project where the user is owner (creatable)
 *   - Customer namespace + project where the user is customer (NOT creatable)
 */
async function setupTwoNamespaces(
  request: import('@playwright/test').APIRequestContext,
  userId: string,
): Promise<Setup> {
  const adminToken = getAdminToken();
  const uid = randomUUID().slice(0, 6);

  const ownNsSlug = `own-${uid}`;
  const custNsSlug = `cust-${uid}`;
  const ownProjKey = `O${uid.slice(0, 4).toUpperCase()}`;
  const custProjKey = `C${uid.slice(0, 4).toUpperCase()}`;
  const ownProjName = `Own NS Project ${uid}`;
  const custProjName = `Customer NS Project ${uid}`;

  // Namespace + project where the user is owner (admin creates, then promotes user)
  await api.createNamespace(request, adminToken, ownNsSlug, `Own NS ${uid}`);
  await api.createProject(request, adminToken, ownProjKey, ownProjName, ownNsSlug);
  await api.addMemberInNamespace(request, adminToken, ownProjKey, userId, 'owner', ownNsSlug);

  // Namespace + project where the user is customer
  await api.createNamespace(request, adminToken, custNsSlug, `Customer NS ${uid}`);
  await api.createProject(request, adminToken, custProjKey, custProjName, custNsSlug);
  await api.addMemberInNamespace(request, adminToken, custProjKey, userId, 'customer', custNsSlug);

  return { ownNsSlug, ownProjKey, ownProjName, custNsSlug, custProjKey, custProjName };
}

async function cleanupSetup(
  request: import('@playwright/test').APIRequestContext,
  setup: Setup,
) {
  const adminToken = getAdminToken();
  // Delete projects first — namespace deletion fails while projects still exist.
  await request.delete(`${BASE_URL}/api/v1/${setup.ownNsSlug}/projects/${setup.ownProjKey}`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  }).catch(() => {});
  await request.delete(`${BASE_URL}/api/v1/${setup.custNsSlug}/projects/${setup.custProjKey}`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  }).catch(() => {});
  await api.deleteNamespace(request, adminToken, setup.ownNsSlug).catch(() => {});
  await api.deleteNamespace(request, adminToken, setup.custNsSlug).catch(() => {});
}

async function waitForPageReady(page: import('@playwright/test').Page) {
  await page.waitForLoadState('networkidle');
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 1000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

/**
 * Opens the ProjectPicker inside the New Work Item modal. The modal opens
 * with no project selected, so the picker button shows the "Select project"
 * placeholder.
 */
async function openProjectPicker(page: import('@playwright/test').Page) {
  await expect(page.getByRole('heading', { name: 'New Work Item' })).toBeVisible({ timeout: 5000 });
  const dialog = page.getByRole('dialog');
  await dialog.getByRole('button', { name: /select project/i }).click();
  await expect(page.getByPlaceholder('Search projects...')).toBeVisible({ timeout: 3000 });
}

test.describe('Create Work Item modal — cross-namespace customer filtering', () => {
  test.describe.configure({ mode: 'serial' });

  test.beforeEach(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.enableNamespaces(request, adminToken);
  });

  test('inbox new item picker hides projects where the user is a customer', async ({ page, request, testUser }) => {
    const setup = await setupTwoNamespaces(request, testUser.id);

    try {
      // Sanity check via API: both projects are returned cross-namespace, with
      // the expected member_roles (so we know the UI filter — not a missing
      // fixture — is what drives the assertion below).
      const allProjectsRes = await request.get(`${BASE_URL}/api/v1/projects`, {
        headers: { Authorization: `Bearer ${testUser.token}` },
      });
      expect(allProjectsRes.ok()).toBe(true);
      const allProjects = (await allProjectsRes.json()).data as Array<{ key: string; member_role?: string }>;
      const ownRow = allProjects.find((p) => p.key === setup.ownProjKey);
      const custRow = allProjects.find((p) => p.key === setup.custProjKey);
      expect(ownRow?.member_role).toBe('owner');
      expect(custRow?.member_role).toBe('customer');

      await page.goto('/user/inbox');
      await waitForPageReady(page);

      await page.getByRole('button', { name: 'New Item' }).click();
      await openProjectPicker(page);

      const dropdown = page.locator('ul').filter({ has: page.locator('li') }).last();

      // Search for the customer project — must not appear
      const searchInput = page.getByPlaceholder('Search projects...');
      await searchInput.fill(setup.custProjKey);
      await expect(dropdown.getByText(setup.custProjName)).toHaveCount(0);
      await expect(page.getByText('No projects found')).toBeVisible({ timeout: 3000 });

      // Searching for the own project should find it
      await searchInput.fill('');
      await searchInput.fill(setup.ownProjKey);
      await expect(dropdown.locator('li').filter({ hasText: setup.ownProjName })).toBeVisible({ timeout: 3000 });

      // Also verify there is no <li> matching the customer project key anywhere
      // in the dropdown (covers case-insensitive partial matches).
      await searchInput.fill('');
      const rows = await dropdown.locator('li').allTextContents();
      expect(rows.some((t) => t.includes(setup.custProjKey))).toBe(false);
      expect(rows.some((t) => t.includes(setup.ownProjKey))).toBe(true);
    } finally {
      await cleanupSetup(request, setup);
    }
  });

  test('watchlist new item picker hides projects where the user is a customer', async ({ page, request, testUser }) => {
    const setup = await setupTwoNamespaces(request, testUser.id);

    try {
      await page.goto('/user/watchlist');
      await waitForPageReady(page);

      await page.getByRole('button', { name: 'New Item' }).click();
      await openProjectPicker(page);

      const dropdown = page.locator('ul').filter({ has: page.locator('li') }).last();
      const searchInput = page.getByPlaceholder('Search projects...');

      // Customer project must not appear
      await searchInput.fill(setup.custProjKey);
      await expect(dropdown.getByText(setup.custProjName)).toHaveCount(0);
      await expect(page.getByText('No projects found')).toBeVisible({ timeout: 3000 });

      // Own project must appear
      await searchInput.fill('');
      await searchInput.fill(setup.ownProjKey);
      await expect(dropdown.locator('li').filter({ hasText: setup.ownProjName })).toBeVisible({ timeout: 3000 });

      // Confirm the customer project is absent from the full unfiltered list.
      await searchInput.fill('');
      const rows = await dropdown.locator('li').allTextContents();
      expect(rows.some((t) => t.includes(setup.custProjKey))).toBe(false);
      expect(rows.some((t) => t.includes(setup.ownProjKey))).toBe(true);
    } finally {
      await cleanupSetup(request, setup);
    }
  });
});
