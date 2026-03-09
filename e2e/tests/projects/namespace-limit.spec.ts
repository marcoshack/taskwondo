import { test, expect, getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

async function attach(page: any, testInfo: any, name: string) {
  const screenshot = await page.screenshot();
  await testInfo.attach(name, { body: screenshot, contentType: 'image/png' });
}

async function dismissWelcomeModal(page: any) {
  const heading = page.getByRole('heading', { name: 'Welcome' });
  if (await heading.isVisible({ timeout: 1000 }).catch(() => false)) {
    const checkbox = page.getByRole('checkbox', { name: "Don't show this again" });
    if (await checkbox.isVisible({ timeout: 500 }).catch(() => false)) {
      await checkbox.check();
    }
    await page.keyboard.press('Escape');
    await heading.waitFor({ state: 'hidden', timeout: 2000 }).catch(() => {});
  }
}

test.describe('Namespace Limit', () => {
  test.beforeEach(async ({ request }) => {
    const adminToken = getAdminToken();
    // Ensure namespaces feature is enabled
    await api.enableNamespaces(request, adminToken);
  });

  test('prevents creating namespaces beyond the user limit', async ({ request, testUser, page }, testInfo) => {
    const adminToken = getAdminToken();
    const NAMESPACE_LIMIT = 1;

    // Set per-user namespace limit to 1 (fresh user owns 0 namespaces)
    await api.setMaxNamespaces(request, adminToken, testUser.id, NAMESPACE_LIMIT);

    // Go to projects page
    await page.goto('/d/projects');
    await dismissWelcomeModal(page);

    // Open namespace switcher and click "Create namespace"
    await page.getByTestId('namespace-switcher').click();
    await page.getByText('Create namespace').click();

    // Modal should show the counter 0/1
    const dialog = page.getByRole('dialog');
    await expect(dialog.getByText(`0/${NAMESPACE_LIMIT}`)).toBeVisible();
    await attach(page, testInfo, '01-modal-below-limit');

    // Close the modal
    await page.keyboard.press('Escape');

    // Create one namespace via API to hit the limit
    const slug = `ns-limit-${randomUUID().slice(0, 6)}`;
    await api.createNamespace(request, testUser.token, slug, 'Limit Test');

    // Reload and open the create modal again
    await page.reload();
    await dismissWelcomeModal(page);
    await page.getByTestId('namespace-switcher').click();
    await page.getByText('Create namespace').click();

    // Verify counter shows 1/1 (at limit)
    await expect(page.getByRole('dialog').getByText(`${NAMESPACE_LIMIT}/${NAMESPACE_LIMIT}`)).toBeVisible();

    // Verify warning message is visible
    await expect(page.getByText(/limit reached/i)).toBeVisible();

    // Verify form is disabled
    await expect(dialog.getByLabel(/display name/i)).toBeDisabled();
    await expect(dialog.getByLabel(/slug/i)).toBeDisabled();
    await expect(dialog.getByRole('button', { name: /^create$/i })).toBeDisabled();

    await attach(page, testInfo, '02-modal-limit-reached');

    // Cleanup
    await api.deleteNamespace(request, adminToken, slug);
  });

  test('API rejects namespace creation beyond the limit', async ({ request, testUser }) => {
    const adminToken = getAdminToken();

    // Set limit to 1 (fresh user owns 0)
    await api.setMaxNamespaces(request, adminToken, testUser.id, 1);

    // Create one namespace to hit the limit
    const slug = `ns-api-limit-${randomUUID().slice(0, 6)}`;
    const ns = await api.createNamespace(request, testUser.token, slug, 'First NS');
    expect(ns.slug).toBe(slug); // Verify creation succeeded

    // Try to create a second namespace — should be rejected (403 FORBIDDEN)
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';
    const res = await request.post(`${BASE_URL}/api/v1/namespaces`, {
      headers: { Authorization: `Bearer ${testUser.token}` },
      data: { slug: 'should-fail', display_name: 'Should Fail' },
    });

    expect(res.ok()).toBe(false);
    expect(res.status()).toBe(403);

    // Cleanup
    await api.deleteNamespace(request, adminToken, slug);
  });

  test('admin is exempt from the namespace limit', async ({ request }) => {
    const adminToken = getAdminToken();

    // Set global limit to 1
    await api.setSystemSetting(request, adminToken, 'max_namespaces_per_user', 1);

    // Admin should still be able to create namespaces beyond the limit
    const slug = `admin-exempt-${randomUUID().slice(0, 6)}`;
    const ns = await api.createNamespace(request, adminToken, slug, 'Admin Exempt');
    expect(ns.slug).toBe(slug);

    // Cleanup
    await api.deleteNamespace(request, adminToken, slug);
    await api.deleteSystemSetting(request, adminToken, 'max_namespaces_per_user');
  });

  test('API rejects adding owner beyond the limit', async ({ request, testUser }) => {
    const adminToken = getAdminToken();

    // Set test user's limit to 1
    await api.setMaxNamespaces(request, adminToken, testUser.id, 1);

    // Create one namespace as testUser to hit their limit
    const ownedSlug = `ns-owned-${randomUUID().slice(0, 6)}`;
    const ns = await api.createNamespace(request, testUser.token, ownedSlug, 'Owned NS');
    expect(ns.slug).toBe(ownedSlug); // Verify creation succeeded

    // Create another namespace as admin
    const adminSlug = `ns-admin-${randomUUID().slice(0, 6)}`;
    await api.createNamespace(request, adminToken, adminSlug, 'Admin NS');

    // Try to add testUser as owner of the admin's namespace — should be rejected
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';
    const res = await request.post(`${BASE_URL}/api/v1/namespaces/${adminSlug}/members`, {
      headers: { Authorization: `Bearer ${adminToken}` },
      data: { user_id: testUser.id, role: 'owner' },
    });

    expect(res.ok()).toBe(false);
    expect(res.status()).toBe(403);

    // Cleanup
    await api.deleteNamespace(request, adminToken, adminSlug);
    await api.deleteNamespace(request, adminToken, ownedSlug);
  });
});
