import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';

// These tests toggle the global namespaces_enabled setting (disable then
// re-enable). They run in the namespace-toggle project which depends on the
// namespace project, so they execute AFTER all other namespace tests finish.
// This prevents races where one test disables namespaces while another needs
// them enabled.

// Admin context — namespace management requires admin role
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

async function attach(page: any, testInfo: any, name: string) {
  const screenshot = await page.screenshot();
  await testInfo.attach(name, { body: screenshot, contentType: 'image/png' });
}

function uniqueSlug() {
  return `ns-fe-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 5)}`;
}

test.describe('Namespace disable/enable toggle', () => {
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

  test('requesting non-default namespace when feature disabled returns 404', async ({ request }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

    // Create a namespace first (while enabled)
    const slug = `ns-dis-${Date.now().toString(36)}`;
    await api.createNamespace(request, adminToken, slug, 'Disabled Test');

    // Disable namespaces
    await api.disableNamespaces(request, adminToken);

    // Try to list projects with that namespace context — middleware should block
    const res = await request.get(`${BASE_URL}/api/v1/${slug}/projects`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.status()).toBe(404);
    const body = await res.json();
    expect(body.error.code).toBe('NOT_FOUND');

    // Re-enable for cleanup
    await api.enableNamespaces(request, adminToken);
    await api.deleteNamespace(request, adminToken, slug);
  });

  test('enabling namespaces via admin features toggle', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();

    // Ensure namespaces are disabled via API, then navigate to pick up the new state
    await api.disableNamespaces(request, adminToken);

    // Navigate to the admin features page (fresh load picks up API state)
    await page.goto('/admin/features');
    await page.waitForLoadState('networkidle');

    // Find the Namespaces section and its toggle
    const namespacesSection = page.locator('div').filter({ hasText: /^Namespaces/ }).first();
    const toggle = namespacesSection.locator('button[role="switch"]');

    await expect(toggle).toHaveAttribute('aria-checked', 'false');
    await attach(page, testInfo, '00-toggle-off');

    // Click to enable
    await toggle.click();

    // Toggle should now be on
    await expect(toggle).toHaveAttribute('aria-checked', 'true', { timeout: 10000 });
    await attach(page, testInfo, '00-toggle-on');

    // Verify the feature actually works: create a namespace via API.
    await api.enableNamespaces(request, adminToken);
    const slug = uniqueSlug();
    await api.createNamespace(request, adminToken, slug, 'Toggle Test NS');

    // Navigate to projects — namespace switcher should appear
    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');
    const switcher = page.getByTestId('namespace-switcher');
    await expect(switcher).toBeVisible();
    await attach(page, testInfo, '00-feature-active');

    // Cleanup
    await api.deleteNamespace(request, adminToken, slug);
  });
});
