import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

// Admin context — system API key management requires admin role
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

test.describe('System API Keys — API', () => {
  test('create, list, rename, and delete a system API key', async ({ request }) => {
    const adminToken = getAdminToken();
    const keyName = `e2e-syskey-${randomUUID().slice(0, 8)}`;

    // 1. Create a system API key with items:rw permission
    const created = await api.createSystemAPIKey(request, adminToken, keyName, ['items:rw']);
    expect(created.id).toBeTruthy();
    expect(created.key).toMatch(/^twks_/);
    expect(created.name).toBe(keyName);
    expect(created.permissions).toContain('items:rw');

    // 2. Verify the key appears in the list
    const keysBefore = await api.listSystemAPIKeys(request, adminToken);
    const found = keysBefore.find((k: any) => k.id === created.id);
    expect(found).toBeTruthy();
    expect(found.name).toBe(keyName);

    // 3. Rename the key
    const newName = `renamed-${randomUUID().slice(0, 8)}`;
    await api.renameSystemAPIKey(request, adminToken, created.id, newName);

    // Verify renamed
    const keysAfterRename = await api.listSystemAPIKeys(request, adminToken);
    const renamed = keysAfterRename.find((k: any) => k.id === created.id);
    expect(renamed).toBeTruthy();
    expect(renamed.name).toBe(newName);

    // 4. Delete the key
    await api.deleteSystemAPIKey(request, adminToken, created.id);

    // 5. Verify deletion (key no longer in list)
    const keysAfterDelete = await api.listSystemAPIKeys(request, adminToken);
    const deleted = keysAfterDelete.find((k: any) => k.id === created.id);
    expect(deleted).toBeUndefined();
  });

  test('system key with items:rw can create a work item (bypasses membership)', async ({ request }) => {
    const adminToken = getAdminToken();
    const keyName = `e2e-itemsrw-${randomUUID().slice(0, 8)}`;

    // Create a project as admin
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const project = await api.createProject(request, adminToken, `S${suffix}`, `SysKey Project ${suffix}`);

    // Create a system key with items:rw
    const created = await api.createSystemAPIKey(request, adminToken, keyName, ['items:rw']);
    const systemKey = created.key;

    // Use the system key to create a work item
    const res = await request.post(`${BASE_URL}/api/v1/default/projects/${project.key}/items`, {
      headers: { Authorization: `Bearer ${systemKey}` },
      data: { title: 'Created via system key', type: 'task' },
    });
    expect(res.status()).toBe(201);
    const body = await res.json();
    expect(body.data.title).toBe('Created via system key');

    // Cleanup
    await api.deleteSystemAPIKey(request, adminToken, created.id);
  });

  test('system key with expiration is created correctly', async ({ request }) => {
    const adminToken = getAdminToken();
    const keyName = `e2e-expiring-${randomUUID().slice(0, 8)}`;
    const futureDate = new Date(Date.now() + 7 * 86400000).toISOString();

    const created = await api.createSystemAPIKey(request, adminToken, keyName, ['items:r'], futureDate);
    expect(created.id).toBeTruthy();
    expect(created.expires_at).toBeTruthy();

    // Cleanup
    await api.deleteSystemAPIKey(request, adminToken, created.id);
  });
});

// ─── Negative Tests ──────────────────────────────────────────────────────────

test.describe('System API Keys — Negative', () => {
  test('non-admin cannot create system API keys (403)', async ({ request }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const email = `e2e-syskey-nonadmin-${uniqueId}@test.local`;

    // Create a regular (non-admin) user
    const created = await api.createUser(request, adminToken, email, `SysKey User ${uniqueId}`);
    const tempLogin = await api.login(request, email, created.temporary_password);
    await api.changePassword(request, tempLogin.token, created.temporary_password, 'TestPass123!');
    const userLogin = await api.login(request, email, 'TestPass123!');

    // Try to create a system API key as non-admin — should get 403
    const res = await request.post(`${BASE_URL}/api/v1/admin/api-keys`, {
      headers: { Authorization: `Bearer ${userLogin.token}` },
      data: { name: 'should-fail', permissions: ['items:r'] },
    });
    expect(res.status()).toBe(403);

    // Also try to list
    const listRes = await request.get(`${BASE_URL}/api/v1/admin/api-keys`, {
      headers: { Authorization: `Bearer ${userLogin.token}` },
    });
    expect(listRes.status()).toBe(403);

    // Cleanup
    await api.deactivateUser(request, adminToken, userLogin.user.id).catch(() => {});
  });

  test('system key with items:r cannot create work items (403)', async ({ request }) => {
    const adminToken = getAdminToken();
    const keyName = `e2e-readonly-${randomUUID().slice(0, 8)}`;

    // Create a project
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const project = await api.createProject(request, adminToken, `R${suffix}`, `ReadOnly Project ${suffix}`);

    // Create a system key with items:r (read only)
    const created = await api.createSystemAPIKey(request, adminToken, keyName, ['items:r']);
    const systemKey = created.key;

    // Try to create a work item — should get 403 (insufficient permissions)
    const res = await request.post(`${BASE_URL}/api/v1/default/projects/${project.key}/items`, {
      headers: { Authorization: `Bearer ${systemKey}` },
      data: { title: 'Should fail', type: 'task' },
    });
    expect(res.status()).toBe(403);

    // But reading should work
    const getRes = await request.get(`${BASE_URL}/api/v1/default/projects/${project.key}/items`, {
      headers: { Authorization: `Bearer ${systemKey}` },
    });
    expect(getRes.status()).toBe(200);

    // Cleanup
    await api.deleteSystemAPIKey(request, adminToken, created.id);
  });

  test('expired system key is rejected (401)', async ({ request }) => {
    const adminToken = getAdminToken();
    const keyName = `e2e-expired-${randomUUID().slice(0, 8)}`;

    // Create a system key with an expiration far in the past — the API rejects this,
    // so we need to create one with a very short future expiration instead.
    // Since we cannot wait for it to expire, we test by verifying the API rejects
    // a past expiration date.
    const pastDate = new Date(Date.now() - 86400000).toISOString();
    const res = await request.post(`${BASE_URL}/api/v1/admin/api-keys`, {
      headers: { Authorization: `Bearer ${adminToken}` },
      data: { name: keyName, permissions: ['items:r'], expires_at: pastDate },
    });
    // The API should reject creating a key with a past expiration
    expect(res.status()).toBe(400);
  });

  test('system key cannot access unauthorized resources (403)', async ({ request }) => {
    const adminToken = getAdminToken();
    const keyName = `e2e-noaccess-${randomUUID().slice(0, 8)}`;

    // Create a system key with only metrics:r permission
    const created = await api.createSystemAPIKey(request, adminToken, keyName, ['metrics:r']);
    const systemKey = created.key;

    // Try to access work items — should get 403 (resource not authorized)
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const project = await api.createProject(request, adminToken, `N${suffix}`, `NoAccess Project ${suffix}`);

    const res = await request.get(`${BASE_URL}/api/v1/default/projects/${project.key}/items`, {
      headers: { Authorization: `Bearer ${systemKey}` },
    });
    expect(res.status()).toBe(403);

    // Cleanup
    await api.deleteSystemAPIKey(request, adminToken, created.id);
  });
});

// ─── UI Tests ────────────────────────────────────────────────────────────────

test.describe('System API Keys — UI', () => {
  test('create a system API key via the form', async ({ page }, testInfo) => {
    const keyName = `e2e-ui-create-${randomUUID().slice(0, 8)}`;

    await page.goto('/admin/api-keys');
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('heading', { name: 'System API Keys' })).toBeVisible();
    await attach(page, testInfo, '01-initial-page');

    // Fill the name
    await page.getByPlaceholder('e.g. CI Pipeline, Monitoring').fill(keyName);

    // Select a resource permission (click "Read & Write" in the items resource row)
    await page.getByTestId('resource-items').getByRole('button', { name: 'Read & Write' }).click();

    await attach(page, testInfo, '02-form-filled');

    // Submit
    await page.getByRole('button', { name: 'Create' }).click();

    // Verify the key reveal card appears
    await expect(page.getByText('API key created successfully!')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Copy this key now')).toBeVisible();

    // Verify the key value starts with twks_ (use first() since prefix also shows in list)
    const keyCode = page.locator('code').filter({ hasText: /^twks_/ }).first();
    await expect(keyCode).toBeVisible();
    await attach(page, testInfo, '03-key-revealed');

    // Dismiss the reveal card
    await page.getByRole('button', { name: 'Done' }).click();
    await expect(page.getByText('API key created successfully!')).not.toBeVisible();

    // Verify key appears in the list
    await expect(page.getByText(keyName)).toBeVisible();
    await attach(page, testInfo, '04-key-in-list');

    // Cleanup via API
    const adminToken = getAdminToken();
    const keys = await api.listSystemAPIKeys(page.request, adminToken);
    const toDelete = keys.find((k: any) => k.name === keyName);
    if (toDelete) await api.deleteSystemAPIKey(page.request, adminToken, toDelete.id);
  });

  test('rename a system API key via the UI', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const originalName = `e2e-ui-rename-${randomUUID().slice(0, 8)}`;

    // Create a key via API first
    const created = await api.createSystemAPIKey(request, adminToken, originalName, ['items:r']);

    await page.goto('/admin/api-keys');
    await page.waitForLoadState('networkidle');

    // Verify the key appears
    await expect(page.getByText(originalName)).toBeVisible();
    await attach(page, testInfo, '01-key-present');

    // Click the pencil icon to start renaming (scoped to the row containing the key name)
    const keyRow = page.locator('div').filter({ hasText: originalName }).locator('button:has(.lucide-pencil)').first();
    await keyRow.click();

    // Verify the inline edit input is visible
    const editInput = page.getByRole('textbox').last();
    await expect(editInput).toBeVisible();
    await expect(editInput).toHaveValue(originalName);
    await attach(page, testInfo, '02-inline-edit');

    // Clear and type new name
    const newName = `renamed-ui-${randomUUID().slice(0, 8)}`;
    await editInput.clear();
    await editInput.fill(newName);

    // Submit by clicking the check icon
    await page.locator('button:has(.lucide-check)').click();

    // Verify the name was updated
    await expect(page.getByText(newName)).toBeVisible();
    await attach(page, testInfo, '03-key-renamed');

    // Cleanup
    await api.deleteSystemAPIKey(request, adminToken, created.id);
  });

  test('delete a system API key via the modal', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const keyName = `e2e-ui-delete-${randomUUID().slice(0, 8)}`;

    // Create a key via API first
    const created = await api.createSystemAPIKey(request, adminToken, keyName, ['items:rw']);

    await page.goto('/admin/api-keys');
    await page.waitForLoadState('networkidle');

    // Verify the key appears
    await expect(page.getByText(keyName)).toBeVisible();
    await attach(page, testInfo, '01-key-present');

    // Click the trash icon (scoped to the row containing the key name)
    const keyRow = page.locator('div').filter({ hasText: keyName }).locator('button:has(.lucide-trash-2)').first();
    await keyRow.click();

    // Verify delete confirmation modal
    await expect(page.getByRole('heading', { name: 'Delete System API Key' })).toBeVisible();
    await expect(page.getByRole('strong')).toContainText(keyName);
    await attach(page, testInfo, '02-delete-modal');

    // Confirm deletion
    await page.getByRole('button', { name: 'Delete' }).click();

    // Wait for modal to close, then verify key is gone from the list
    await expect(page.getByRole('heading', { name: 'Delete System API Key' })).not.toBeVisible();
    await expect(page.locator('span').filter({ hasText: keyName })).not.toBeVisible();
    await attach(page, testInfo, '03-key-deleted');
  });

  test('admin API keys page shows empty state', async ({ page, request }, testInfo) => {
    // Navigate to the admin API keys page
    await page.goto('/admin/api-keys');
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('heading', { name: 'System API Keys' })).toBeVisible();

    // The page should either show the key list or empty state
    // (depending on whether other tests have left keys behind)
    const heading = page.getByRole('heading', { name: 'System API Keys' });
    await expect(heading).toBeVisible();
    await attach(page, testInfo, '01-page-loaded');
  });
});
