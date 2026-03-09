import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';

// Admin context — workflow management requires admin role
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

const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

test.describe.configure({ mode: 'serial' });

test.describe('Admin Workflows API', () => {
  let createdWorkflowId: string;

  test('list system workflows via /admin/workflows', async ({ request }) => {
    const adminToken = getAdminToken();
    const workflows = await api.listWorkflows(request, adminToken);

    // Default seeded workflows should exist
    expect(workflows.length).toBeGreaterThanOrEqual(2);

    const names = workflows.map((w) => w.name);
    expect(names).toContain('Task Workflow');
    expect(names).toContain('Ticket Workflow');
  });

  test('get workflow detail via /admin/workflows/:id', async ({ request }) => {
    const adminToken = getAdminToken();
    const workflows = await api.listWorkflows(request, adminToken);
    const taskWorkflow = workflows.find((w) => w.name === 'Task Workflow');
    expect(taskWorkflow).toBeTruthy();

    const detail = await api.getWorkflow(request, adminToken, taskWorkflow!.id);
    expect(detail.name).toBe('Task Workflow');
    expect(detail.statuses.length).toBeGreaterThan(0);
    expect(detail.transitions.length).toBeGreaterThan(0);
  });

  test('list system statuses via /admin/workflows/statuses', async ({ request }) => {
    const adminToken = getAdminToken();
    const statuses = await api.listSystemStatuses(request, adminToken);

    expect(statuses.length).toBeGreaterThan(0);
    // All statuses should have a category
    for (const s of statuses) {
      expect(['todo', 'in_progress', 'done', 'cancelled']).toContain(s.category);
    }
  });

  test('get transitions map via /admin/workflows/:id/transitions', async ({ request }) => {
    const adminToken = getAdminToken();
    const workflows = await api.listWorkflows(request, adminToken);
    const taskWorkflow = workflows.find((w) => w.name === 'Task Workflow');

    const transitions = await api.getTransitionsMap(request, adminToken, taskWorkflow!.id);
    // Should be an object with status names as keys
    expect(typeof transitions).toBe('object');
    expect(Object.keys(transitions).length).toBeGreaterThan(0);
  });

  test('non-admin user gets 403 on /admin/workflows', async ({ request }) => {
    const adminToken = getAdminToken();
    const uniqueId = Date.now().toString(36).slice(-6);
    const email = `e2e-wf-${uniqueId}@test.local`;

    // Create a regular (non-admin) user
    const created = await api.createUser(request, adminToken, email, `WF User ${uniqueId}`);
    const tempLogin = await api.login(request, email, created.temporary_password);
    await api.changePassword(request, tempLogin.token, created.temporary_password, 'TestPass123!');
    const userLogin = await api.login(request, email, 'TestPass123!');

    // Try to list workflows as non-admin — should get 403
    const res = await request.get(`${BASE_URL}/api/v1/admin/workflows`, {
      headers: { Authorization: `Bearer ${userLogin.token}` },
    });
    expect(res.status()).toBe(403);

    // Cleanup
    await api.deactivateUser(request, adminToken, userLogin.user.id).catch(() => {});
  });

  test('old /workflows path returns 404', async ({ request }) => {
    const adminToken = getAdminToken();
    const res = await request.get(`${BASE_URL}/api/v1/workflows`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    // The old path should no longer work
    expect([404, 405]).toContain(res.status());
  });
});
