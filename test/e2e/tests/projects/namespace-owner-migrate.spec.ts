import { test, expect, getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

const TEST_PASSWORD = 'TestPass123!';

/**
 * Create a user who is set up (password changed, welcome dismissed).
 */
async function createReadyUser(
  request: import('@playwright/test').APIRequestContext,
  adminToken: string,
) {
  const uniqueId = randomUUID().slice(0, 8);
  const email = `ns-mig-${uniqueId}@test.local`;
  const displayName = `NsMig ${uniqueId}`;
  const created = await api.createUser(request, adminToken, email, displayName);
  const tempLogin = await api.login(request, email, created.temporary_password);
  await api.changePassword(request, tempLogin.token, created.temporary_password, TEST_PASSWORD);
  const finalLogin = await api.login(request, email, TEST_PASSWORD);
  await api.setPreference(request, finalLogin.token, 'welcome_dismissed', true);
  return { id: finalLogin.user.id, email, token: finalLogin.token };
}

test.describe('Namespace — project owner migration and soft-delete cleanup', () => {
  test.describe.configure({ mode: 'serial' });

  test.beforeEach(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.enableNamespaces(request, adminToken);
  });

  test('project owner (non-namespace-admin) can migrate their project', async ({ request }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

    const srcSlug = `ns-omig-s-${Date.now().toString(36)}`;
    const dstSlug = `ns-omig-d-${Date.now().toString(36)}`;
    const projKey = `OM${Date.now().toString(36).slice(-3).toUpperCase()}`;

    // Create two namespaces
    await api.createNamespace(request, adminToken, srcSlug, 'Owner Migrate Source');
    await api.createNamespace(request, adminToken, dstSlug, 'Owner Migrate Dest');

    // Create a regular (non-admin) user
    const user = await createReadyUser(request, adminToken);

    // Admin creates project in source namespace
    await request.post(`${BASE_URL}/api/v1/${srcSlug}/projects`, {
      headers: { Authorization: `Bearer ${adminToken}` },
      data: { key: projKey, name: 'Owner Migrate Test' },
    });

    // Add user as project owner
    await api.addMemberInNamespace(request, adminToken, projKey, user.id, 'owner', srcSlug);

    // Add user as admin of target namespace (required for target permission)
    await api.addNamespaceMember(request, adminToken, dstSlug, user.id, 'admin');

    // User is NOT an admin/owner of the source namespace — only a project owner.
    // With the new change, they should still be able to migrate their project.
    await api.migrateProject(request, user.token, srcSlug, projKey, dstSlug);

    // Verify project is now in the destination namespace
    const listRes = await request.get(`${BASE_URL}/api/v1/${dstSlug}/projects`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    const projects = (await listRes.json()).data;
    expect(projects.some((p: any) => p.key === projKey)).toBe(true);

    // Verify project is no longer in source namespace
    const srcListRes = await request.get(`${BASE_URL}/api/v1/${srcSlug}/projects`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    const srcProjects = (await srcListRes.json()).data;
    expect(srcProjects.some((p: any) => p.key === projKey)).toBe(false);

    // Cleanup
    await api.migrateProject(request, adminToken, dstSlug, projKey, 'default').catch(() => {});
    await api.deleteNamespace(request, adminToken, srcSlug).catch(() => {});
    await api.deleteNamespace(request, adminToken, dstSlug).catch(() => {});
    await api.deactivateUser(request, adminToken, user.id).catch(() => {});
  });

  test('non-owner member cannot migrate project from namespace', async ({ request }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

    const srcSlug = `ns-nomig-${Date.now().toString(36)}`;
    const dstSlug = `ns-nomig-d-${Date.now().toString(36)}`;
    const projKey = `NM${Date.now().toString(36).slice(-3).toUpperCase()}`;

    await api.createNamespace(request, adminToken, srcSlug, 'NoMig Source');
    await api.createNamespace(request, adminToken, dstSlug, 'NoMig Dest');

    const user = await createReadyUser(request, adminToken);

    // Admin creates project, adds user as regular member (not owner)
    await request.post(`${BASE_URL}/api/v1/${srcSlug}/projects`, {
      headers: { Authorization: `Bearer ${adminToken}` },
      data: { key: projKey, name: 'NoMig Test' },
    });
    await api.addMemberInNamespace(request, adminToken, projKey, user.id, 'member', srcSlug);
    await api.addNamespaceMember(request, adminToken, dstSlug, user.id, 'admin');

    // User is a regular member of the project (not owner) and not a namespace admin.
    // Migration should be rejected.
    const migRes = await request.post(`${BASE_URL}/api/v1/namespaces/${srcSlug}/projects/${projKey}/migrate`, {
      headers: { Authorization: `Bearer ${user.token}` },
      data: { target_namespace: dstSlug },
    });
    expect(migRes.status()).toBe(403);

    // Cleanup
    await api.migrateProject(request, adminToken, srcSlug, projKey, 'default').catch(() => {});
    await api.deleteNamespace(request, adminToken, srcSlug).catch(() => {});
    await api.deleteNamespace(request, adminToken, dstSlug).catch(() => {});
    await api.deactivateUser(request, adminToken, user.id).catch(() => {});
  });

  test('namespace delete succeeds when only soft-deleted projects remain', async ({ request }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

    const slug = `ns-softdel-${Date.now().toString(36)}`;
    const projKey = `SD${Date.now().toString(36).slice(-3).toUpperCase()}`;

    // Create namespace and project in it
    await api.createNamespace(request, adminToken, slug, 'SoftDel NS');
    await request.post(`${BASE_URL}/api/v1/${slug}/projects`, {
      headers: { Authorization: `Bearer ${adminToken}` },
      data: { key: projKey, name: 'Will Be Deleted' },
    });

    // Soft-delete the project
    await api.deleteProject(request, adminToken, projKey, slug);

    // Namespace should no longer report active projects — delete should succeed
    await api.deleteNamespace(request, adminToken, slug);

    // Verify namespace is gone
    const listRes = await request.get(`${BASE_URL}/api/v1/namespaces`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    const namespaces = (await listRes.json()).data;
    expect(namespaces.some((ns: any) => ns.slug === slug)).toBe(false);
  });
});
