import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

// Admin context — admin projects/namespaces inspection requires admin role
const test = base.extend({
  storageState: async ({}, use) => {
    const adminToken = getAdminToken();
    const state = {
      cookies: [],
      origins: [
        {
          origin: BASE_URL,
          localStorage: [{ name: 'taskwondo_token', value: adminToken }],
        },
      ],
    };
    await use(state as any);
  },
});

async function attach(page: any, testInfo: any, name: string) {
  const screenshot = await page.screenshot();
  await testInfo.attach(name, { body: screenshot, contentType: 'image/png' });
}

// ─── API Tests ───────────────────────────────────────────────────────────────

test.describe('Admin Projects & Namespaces — API', () => {
  test.describe.configure({ mode: 'serial' });

  test('admin projects API returns project list', async ({ request }) => {
    const adminToken = getAdminToken();
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const key = `A${suffix}`;
    const name = `Admin Inspect Project ${suffix}`;

    // Create a project via API
    const project = await api.createProject(request, adminToken, key, name);

    // Call the admin projects endpoint
    const res = await request.get(`${BASE_URL}/api/v1/admin/projects`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.status()).toBe(200);

    const body = await res.json();
    expect(body.data).toBeDefined();
    expect(Array.isArray(body.data)).toBe(true);
    expect(body.meta).toBeDefined();
    expect(typeof body.meta.has_more).toBe('boolean');

    // Verify the created project appears in the list
    const found = body.data.find((p: any) => p.id === project.id);
    expect(found).toBeTruthy();
    expect(found.key).toBe(key);
    expect(found.name).toBe(name);
    expect(typeof found.member_count).toBe('number');
    expect(typeof found.item_count).toBe('number');
    expect(typeof found.storage_bytes).toBe('number');
  });

  test('admin namespaces API returns namespace list', async ({ request }) => {
    const adminToken = getAdminToken();

    const res = await request.get(`${BASE_URL}/api/v1/admin/namespaces`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.status()).toBe(200);

    const body = await res.json();
    expect(body.data).toBeDefined();
    expect(Array.isArray(body.data)).toBe(true);
    expect(body.meta).toBeDefined();

    // At least the "default" namespace should exist
    const defaultNs = body.data.find((ns: any) => ns.slug === 'default');
    expect(defaultNs).toBeTruthy();
    expect(defaultNs.display_name).toBeTruthy();
    expect(defaultNs.is_default).toBe(true);
    expect(typeof defaultNs.project_count).toBe('number');
    expect(typeof defaultNs.member_count).toBe('number');
    expect(typeof defaultNs.storage_bytes).toBe('number');
  });

  test('admin projects API search filters results', async ({ request }) => {
    const adminToken = getAdminToken();
    const suffix1 = randomUUID().slice(0, 4).toUpperCase();
    const suffix2 = randomUUID().slice(0, 4).toUpperCase();

    // Create two projects with distinct names
    const proj1 = await api.createProject(request, adminToken, `X${suffix1}`, `Findable Alpha ${suffix1}`);
    await api.createProject(request, adminToken, `Y${suffix2}`, `Findable Beta ${suffix2}`);

    // Search for the first project by its unique suffix
    const res = await request.get(`${BASE_URL}/api/v1/admin/projects?search=${suffix1}`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.status()).toBe(200);

    const body = await res.json();
    const matches = body.data.filter((p: any) => p.id === proj1.id);
    expect(matches.length).toBe(1);
    expect(matches[0].key).toBe(`X${suffix1}`);

    // The second project should not appear when searching for the first suffix
    const other = body.data.find((p: any) => p.key === `Y${suffix2}`);
    expect(other).toBeUndefined();
  });

  test('admin projects API pagination works', async ({ request }) => {
    const adminToken = getAdminToken();

    // Create 3 projects with a common prefix for search isolation
    const prefix = randomUUID().slice(0, 6).toUpperCase();
    const keys: string[] = [];
    for (let i = 0; i < 3; i++) {
      const k = `P${prefix.slice(0, 3)}${i}`;
      await api.createProject(request, adminToken, k, `Paged Project ${prefix} ${i}`);
      keys.push(k);
    }

    // Request with limit=2, searching by the unique prefix
    const res1 = await request.get(`${BASE_URL}/api/v1/admin/projects?search=${prefix}&limit=2`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res1.status()).toBe(200);
    const body1 = await res1.json();
    expect(body1.data.length).toBe(2);
    expect(body1.meta.has_more).toBe(true);
    expect(body1.meta.cursor).toBeTruthy();

    // Request the next page using the cursor
    const res2 = await request.get(`${BASE_URL}/api/v1/admin/projects?search=${prefix}&limit=2&cursor=${encodeURIComponent(body1.meta.cursor)}`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res2.status()).toBe(200);
    const body2 = await res2.json();
    expect(body2.data.length).toBe(1);
    expect(body2.meta.has_more).toBe(false);
  });

  test('non-admin gets 403 on admin projects endpoint', async ({ request }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const email = `e2e-noadmin-proj-${uniqueId}@test.local`;

    // Create a regular (non-admin) user
    const created = await api.createUser(request, adminToken, email, `Non-Admin User ${uniqueId}`);
    const tempLogin = await api.login(request, email, created.temporary_password);
    await api.changePassword(request, tempLogin.token, created.temporary_password, 'TestPass123!');
    const userLogin = await api.login(request, email, 'TestPass123!');

    // Try to call the admin projects endpoint as a non-admin — should get 403
    const res = await request.get(`${BASE_URL}/api/v1/admin/projects`, {
      headers: { Authorization: `Bearer ${userLogin.token}` },
    });
    expect(res.status()).toBe(403);

    // Also try namespaces
    const nsRes = await request.get(`${BASE_URL}/api/v1/admin/namespaces`, {
      headers: { Authorization: `Bearer ${userLogin.token}` },
    });
    expect(nsRes.status()).toBe(403);

    // Cleanup
    await api.deactivateUser(request, adminToken, userLogin.user.id).catch(() => {});
  });
});

// ─── UI Tests ────────────────────────────────────────────────────────────────

test.describe('Admin Projects & Namespaces — UI', () => {
  test.describe.configure({ mode: 'serial' });

  test('admin projects page is accessible', async ({ page }, testInfo) => {
    await page.goto('/admin/project-overview');
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('heading', { name: 'Projects & Namespaces' })).toBeVisible({ timeout: 10000 });

    // Verify tabs are shown (use button role to distinguish from sidebar links)
    await expect(page.getByRole('button', { name: 'Projects' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Namespaces' })).toBeVisible();
    await attach(page, testInfo, '01-projects-page');
  });

  test('admin projects page shows projects in table', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const key = `T${suffix}`;
    const name = `UI Table Project ${suffix}`;

    // Create a project via API to ensure at least one exists
    await api.createProject(request, adminToken, key, name);

    await page.goto('/admin/project-overview');
    await page.waitForLoadState('networkidle');

    // Wait for the admin page to fully render
    await expect(page.getByRole('heading', { name: 'Projects & Namespaces' })).toBeVisible({ timeout: 10000 });

    // The DataTable should contain the project key
    await expect(page.getByText(key)).toBeVisible({ timeout: 10000 });
    await attach(page, testInfo, '01-projects-table');
  });

  test('admin namespaces tab shows namespaces', async ({ page }, testInfo) => {
    await page.goto('/admin/project-overview');
    await page.waitForLoadState('networkidle');

    // Wait for page to load then click the "Namespaces" tab
    await expect(page.getByRole('heading', { name: 'Projects & Namespaces' })).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: 'Namespaces' }).click();
    await page.waitForLoadState('networkidle');

    // Verify the "default" namespace appears in the table body
    await expect(page.locator('tbody').getByText('default', { exact: true })).toBeVisible({ timeout: 10000 });
    await attach(page, testInfo, '01-namespaces-tab');
  });
});
