import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

// Admin context
const test = base.extend({
  storageState: async ({}, use) => {
    const adminToken = getAdminToken();
    const state = {
      cookies: [],
      origins: [
        {
          origin: process.env.BASE_URL || 'http://localhost:5173',
          localStorage: [{ name: 'taskwondo_token', value: adminToken }],
        },
      ],
    };
    await use(state as any);
  },
});

test.describe.configure({ mode: 'serial' });

test.describe('Namespace hide non-member preference', () => {
  test.beforeEach(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.enableNamespaces(request, adminToken);
    await api.setPreference(request, adminToken, 'hide_non_member_projects', false);
  });

  test('admin namespace list filters when hide preference is enabled', async ({ request }) => {
    const adminToken = getAdminToken();
    const suffix = randomUUID().slice(0, 6);

    // Create a namespace as admin (admin will be owner/member)
    const ownedNs = await api.createNamespace(request, adminToken, `owned-${suffix}`, `Owned ${suffix}`);

    // Create a second user who creates their own namespace (admin is NOT a member)
    const userEmail = `e2e-nshide-${Date.now()}@test.local`;
    const created = await api.createUser(request, adminToken, userEmail, 'NS Hide User');
    const tempLogin = await api.login(request, userEmail, created.temporary_password);
    await api.changePassword(request, tempLogin.token, created.temporary_password, 'NsHide123!');
    const userLogin = await api.login(request, userEmail, 'NsHide123!');

    const otherNs = await api.createNamespace(request, userLogin.token, `other-${suffix}`, `Other ${suffix}`);

    // Without preference: admin sees all namespaces
    const allNamespaces = await api.listNamespaces(request, adminToken);
    const allSlugs = allNamespaces.map((n: any) => n.slug);
    expect(allSlugs).toContain(ownedNs.slug);
    expect(allSlugs).toContain(otherNs.slug);

    // Enable hide preference
    await api.setPreference(request, adminToken, 'hide_non_member_projects', true);

    // Admin should only see namespaces they're a member of
    const filteredNamespaces = await api.listNamespaces(request, adminToken);
    const filteredSlugs = filteredNamespaces.map((n: any) => n.slug);
    expect(filteredSlugs).toContain(ownedNs.slug);
    expect(filteredSlugs).toContain('default'); // default namespace always visible
    expect(filteredSlugs).not.toContain(otherNs.slug);

    // Disable preference: admin sees all again
    await api.setPreference(request, adminToken, 'hide_non_member_projects', false);

    const restoredNamespaces = await api.listNamespaces(request, adminToken);
    const restoredSlugs = restoredNamespaces.map((n: any) => n.slug);
    expect(restoredSlugs).toContain(ownedNs.slug);
    expect(restoredSlugs).toContain(otherNs.slug);

    // Cleanup
    await api.deleteNamespace(request, adminToken, otherNs.slug).catch(() => {});
    await api.deleteNamespace(request, adminToken, ownedNs.slug).catch(() => {});
    await api.deactivateUser(request, adminToken, userLogin.user.id).catch(() => {});
  });

  test('description text mentions namespaces', async ({ page }) => {
    await page.goto('/preferences/general');
    await page.waitForLoadState('networkidle');

    // The description should mention both projects and namespaces
    await expect(page.getByText(/projects and namespaces/i)).toBeVisible();
  });
});
