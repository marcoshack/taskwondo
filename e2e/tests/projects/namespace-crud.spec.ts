import { test, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';

test.describe('Namespace CRUD', () => {
  test.describe.configure({ mode: 'serial' });

  test.beforeEach(async ({ request }) => {
    // Ensure namespaces feature is enabled for these tests
    const adminToken = getAdminToken();
    await api.enableNamespaces(request, adminToken);
  });

  test('create, get, update, and delete a namespace', async ({ request }) => {
    const adminToken = getAdminToken();
    const slug = `ns-${Date.now().toString(36)}`;

    // Create
    const ns = await api.createNamespace(request, adminToken, slug, 'Test NS');
    expect(ns.slug).toBe(slug);
    expect(ns.display_name).toBe('Test NS');
    expect(ns.is_default).toBe(false);

    // Get
    const fetched = await api.getNamespace(request, adminToken, slug);
    expect(fetched.id).toBe(ns.id);
    expect(fetched.slug).toBe(slug);

    // Update display name
    const newName = 'Updated NS';
    const updated = await api.updateNamespace(request, adminToken, slug, { display_name: newName });
    expect(updated.display_name).toBe(newName);

    // Update slug
    const newSlug = `${slug}-v2`;
    const reslug = await api.updateNamespace(request, adminToken, slug, { slug: newSlug });
    expect(reslug.slug).toBe(newSlug);

    // Delete (no projects = should succeed)
    await api.deleteNamespace(request, adminToken, newSlug);

    // Verify deleted — should 404
    const res = await request.get(`${process.env.BASE_URL || 'http://localhost:5173'}/api/v1/namespaces/${newSlug}`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.status()).toBe(404);
  });

  test('list namespaces includes default', async ({ request }) => {
    const adminToken = getAdminToken();
    const namespaces = await api.listNamespaces(request, adminToken);
    expect(namespaces.length).toBeGreaterThanOrEqual(1);
    const defaultNs = namespaces.find(ns => ns.is_default);
    expect(defaultNs).toBeDefined();
    expect(defaultNs!.slug).toBe('default');
  });

  test('cannot create namespace with reserved slug', async ({ request }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';
    for (const slug of ['default', 'api', 'admin', 'system']) {
      const res = await request.post(`${BASE_URL}/api/v1/namespaces`, {
        headers: { Authorization: `Bearer ${adminToken}` },
        data: { slug, display_name: 'Reserved' },
      });
      expect(res.status()).toBe(400);
    }
  });

  test('cannot create namespace with invalid slug format', async ({ request }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';
    for (const slug of ['A', '1abc', 'has spaces', 'TOO-UPPER']) {
      const res = await request.post(`${BASE_URL}/api/v1/namespaces`, {
        headers: { Authorization: `Bearer ${adminToken}` },
        data: { slug, display_name: 'Invalid' },
      });
      expect(res.status()).toBe(400);
    }
  });

  test('cannot delete default namespace', async ({ request }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';
    const res = await request.delete(`${BASE_URL}/api/v1/namespaces/default`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.status()).toBe(403);
  });

  test('cannot delete namespace with projects', async ({ request }) => {
    const adminToken = getAdminToken();
    const slug = `ns-del-${Date.now().toString(36)}`;
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

    // Create namespace
    await api.createNamespace(request, adminToken, slug, 'NS With Projects');

    // Create a project in this namespace (via X-Namespace header)
    const projKey = `D${Date.now().toString(36).slice(-3).toUpperCase()}`;
    const res = await request.post(`${BASE_URL}/api/v1/projects`, {
      headers: {
        Authorization: `Bearer ${adminToken}`,
        'X-Namespace': slug,
      },
      data: { key: projKey, name: 'Delete Test Project' },
    });
    expect(res.ok()).toBe(true);

    // Attempt delete — should fail (not empty)
    const delRes = await request.delete(`${BASE_URL}/api/v1/namespaces/${slug}`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(delRes.status()).toBe(409);
    const body = await delRes.json();
    expect(body.error.code).toBe('NAMESPACE_NOT_EMPTY');
  });

  test('cannot create namespace when feature is disabled', async ({ request }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

    // Disable namespaces
    await api.disableNamespaces(request, adminToken);

    const res = await request.post(`${BASE_URL}/api/v1/namespaces`, {
      headers: { Authorization: `Bearer ${adminToken}` },
      data: { slug: 'shouldfail', display_name: 'Should Fail' },
    });
    expect(res.status()).toBe(403);
    const body = await res.json();
    expect(body.error.code).toBe('NAMESPACES_DISABLED');

    // Re-enable for cleanup
    await api.enableNamespaces(request, adminToken);
  });

  test('requesting non-default namespace when feature disabled returns 403', async ({ request }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

    // Create a namespace first (while enabled)
    const slug = `ns-dis-${Date.now().toString(36)}`;
    await api.createNamespace(request, adminToken, slug, 'Disabled Test');

    // Disable namespaces
    await api.disableNamespaces(request, adminToken);

    // Try to list projects with that namespace context — middleware should block
    const res = await request.get(`${BASE_URL}/api/v1/projects`, {
      headers: {
        Authorization: `Bearer ${adminToken}`,
        'X-Namespace': slug,
      },
    });
    expect(res.status()).toBe(403);
    const body = await res.json();
    expect(body.error.code).toBe('NAMESPACES_DISABLED');

    // Re-enable for cleanup
    await api.enableNamespaces(request, adminToken);
    await api.deleteNamespace(request, adminToken, slug);
  });
});

test.describe('Namespace Members', () => {
  test.beforeEach(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.enableNamespaces(request, adminToken);
  });

  test('add, list, update role, and remove namespace member', async ({ request, testUser }) => {
    const adminToken = getAdminToken();
    const slug = `ns-mem-${Date.now().toString(36)}`;

    // Admin creates namespace
    await api.createNamespace(request, adminToken, slug, 'Member Test NS');

    // Add test user as member
    const member = await api.addNamespaceMember(request, adminToken, slug, testUser.id, 'member');
    expect(member.user_id).toBe(testUser.id);
    expect(member.role).toBe('member');

    // List members
    const members = await api.listNamespaceMembers(request, adminToken, slug);
    expect(members.length).toBe(2); // admin (owner) + test user (member)
    const testMember = members.find(m => m.user_id === testUser.id);
    expect(testMember).toBeDefined();
    expect(testMember!.role).toBe('member');

    // Update role to admin
    await api.updateNamespaceMemberRole(request, adminToken, slug, testUser.id, 'admin');

    // Verify updated role
    const membersAfter = await api.listNamespaceMembers(request, adminToken, slug);
    const updatedMember = membersAfter.find(m => m.user_id === testUser.id);
    expect(updatedMember!.role).toBe('admin');

    // Remove member
    await api.removeNamespaceMember(request, adminToken, slug, testUser.id);

    // Verify removed
    const membersAfterRemove = await api.listNamespaceMembers(request, adminToken, slug);
    expect(membersAfterRemove.find(m => m.user_id === testUser.id)).toBeUndefined();

    // Cleanup
    await api.deleteNamespace(request, adminToken, slug);
  });

  test('cannot remove last owner', async ({ request }) => {
    const adminToken = getAdminToken();
    const slug = `ns-own-${Date.now().toString(36)}`;
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

    // Get admin user id
    const loginData = await api.login(request, process.env.ADMIN_EMAIL || 'admin@taskwondo.test', process.env.ADMIN_PASSWORD || 'admin');

    await api.createNamespace(request, adminToken, slug, 'Owner Test NS');

    // Try to remove the sole owner (admin)
    const res = await request.delete(`${BASE_URL}/api/v1/namespaces/${slug}/members/${loginData.user.id}`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    // Should fail — admin is a global admin so they bypass role checks,
    // but the last-owner protection should still apply.
    // Actually, global admins bypass role checks so they can remove themselves.
    // The protection is on CountByRole — if there's only 1 owner, removal is blocked.
    expect(res.status()).toBe(409);

    // Cleanup
    await api.deleteNamespace(request, adminToken, slug);
  });
});

test.describe('Namespace Project Isolation', () => {
  test.beforeEach(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.enableNamespaces(request, adminToken);
  });

  test('same project key in different namespaces', async ({ request }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';
    const slug1 = `ns-iso-a-${Date.now().toString(36)}`;
    const slug2 = `ns-iso-b-${Date.now().toString(36)}`;
    const projKey = `ISO${Date.now().toString(36).slice(-2).toUpperCase()}`;

    // Create two namespaces
    await api.createNamespace(request, adminToken, slug1, 'Isolation A');
    await api.createNamespace(request, adminToken, slug2, 'Isolation B');

    // Create project in namespace 1
    const res1 = await request.post(`${BASE_URL}/api/v1/projects`, {
      headers: {
        Authorization: `Bearer ${adminToken}`,
        'X-Namespace': slug1,
      },
      data: { key: projKey, name: 'Isolation Project A' },
    });
    expect(res1.ok()).toBe(true);

    // Create project with SAME key in namespace 2 — should succeed
    const res2 = await request.post(`${BASE_URL}/api/v1/projects`, {
      headers: {
        Authorization: `Bearer ${adminToken}`,
        'X-Namespace': slug2,
      },
      data: { key: projKey, name: 'Isolation Project B' },
    });
    expect(res2.ok()).toBe(true);

    // Both projects exist with the same key but different namespaces
    const body1 = await res1.json();
    const body2 = await res2.json();
    expect(body1.data.key).toBe(projKey);
    expect(body2.data.key).toBe(projKey);
    expect(body1.data.id).not.toBe(body2.data.id);
  });

  test('migrate project between namespaces', async ({ request }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';
    const srcSlug = `ns-mig-s-${Date.now().toString(36)}`;
    const dstSlug = `ns-mig-d-${Date.now().toString(36)}`;
    const projKey = `MIG${Date.now().toString(36).slice(-2).toUpperCase()}`;

    // Create two namespaces
    await api.createNamespace(request, adminToken, srcSlug, 'Migration Source');
    await api.createNamespace(request, adminToken, dstSlug, 'Migration Dest');

    // Create project in source namespace
    const createRes = await request.post(`${BASE_URL}/api/v1/projects`, {
      headers: {
        Authorization: `Bearer ${adminToken}`,
        'X-Namespace': srcSlug,
      },
      data: { key: projKey, name: 'Migration Test Project' },
    });
    expect(createRes.ok()).toBe(true);

    // Migrate project to destination
    await api.migrateProject(request, adminToken, srcSlug, projKey, dstSlug);

    // Verify project is now accessible in destination namespace
    const listRes = await request.get(`${BASE_URL}/api/v1/projects`, {
      headers: {
        Authorization: `Bearer ${adminToken}`,
        'X-Namespace': dstSlug,
      },
    });
    expect(listRes.ok()).toBe(true);
    const listBody = await listRes.json();
    const migratedProject = listBody.data.find((p: any) => p.key === projKey);
    expect(migratedProject).toBeDefined();

    // Verify project is no longer in source namespace
    const srcListRes = await request.get(`${BASE_URL}/api/v1/projects`, {
      headers: {
        Authorization: `Bearer ${adminToken}`,
        'X-Namespace': srcSlug,
      },
    });
    expect(srcListRes.ok()).toBe(true);
    const srcListBody = await srcListRes.json();
    const srcProject = srcListBody.data.find((p: any) => p.key === projKey);
    expect(srcProject).toBeUndefined();
  });

  test('migrate project blocked when key collides in target', async ({ request }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';
    const srcSlug = `ns-col-s-${Date.now().toString(36)}`;
    const dstSlug = `ns-col-d-${Date.now().toString(36)}`;
    const projKey = `COL${Date.now().toString(36).slice(-2).toUpperCase()}`;

    // Create two namespaces
    await api.createNamespace(request, adminToken, srcSlug, 'Collision Source');
    await api.createNamespace(request, adminToken, dstSlug, 'Collision Dest');

    // Create project with same key in both namespaces
    for (const ns of [srcSlug, dstSlug]) {
      const res = await request.post(`${BASE_URL}/api/v1/projects`, {
        headers: {
          Authorization: `Bearer ${adminToken}`,
          'X-Namespace': ns,
        },
        data: { key: projKey, name: `Collision Project ${ns}` },
      });
      expect(res.ok()).toBe(true);
    }

    // Attempt migration — should fail due to key collision
    const migRes = await request.post(`${BASE_URL}/api/v1/namespaces/${srcSlug}/projects/${projKey}/migrate`, {
      headers: { Authorization: `Bearer ${adminToken}` },
      data: { target_namespace: dstSlug },
    });
    expect(migRes.status()).toBe(409);
  });
});

test.describe('Namespace Backward Compatibility', () => {
  test('projects created without namespace header work normally', async ({
    request,
    testUser,
    testProject,
  }) => {
    // testProject was created without namespace header — should be in default namespace
    expect(testProject.key).toBeTruthy();

    // Work items should work normally
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'No namespace header test',
      type: 'task',
    });
    expect(item.display_id).toContain(testProject.key);
  });

  test('default namespace query param works same as no param', async ({ request }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

    // List projects with explicit "default" namespace
    const res = await request.get(`${BASE_URL}/api/v1/projects?namespace=default`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.ok()).toBe(true);

    // List projects with no namespace
    const res2 = await request.get(`${BASE_URL}/api/v1/projects`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res2.ok()).toBe(true);

    // Both should return the same projects
    const body1 = await res.json();
    const body2 = await res2.json();
    expect(body1.data.length).toBe(body2.data.length);
  });
});
