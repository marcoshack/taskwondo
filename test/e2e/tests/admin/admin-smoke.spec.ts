import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import type { Page } from '@playwright/test';

// Admin context — every admin sub-page requires an admin-role token
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

/**
 * Attach a page-error and console-error listener to the page and return a getter
 * for the collected errors. Used by the smoke tests below to fail if navigating
 * to an admin page surfaces any uncaught JS exceptions or React render errors.
 *
 * The classic example this catches: backend returning `{"data": null}` for an
 * empty list, which causes `flatMap(p => p.data)` → `[null]` → a render-time
 * TypeError on `row.key`.
 *
 * "Failed to load resource" messages are ignored — those are Chrome's browser-
 * level logging of HTTP error responses (e.g. optional endpoints returning 404),
 * not uncaught app errors. Real failures either throw (captured by `pageerror`)
 * or are logged by React itself (captured by `console.error`).
 */
function collectPageErrors(page: Page): () => string[] {
  const errors: string[] = [];
  page.on('pageerror', (err) => {
    errors.push(`pageerror: ${err.message}`);
  });
  page.on('console', (msg) => {
    if (msg.type() !== 'error') return;
    const text = msg.text();
    if (text.includes('Failed to load resource')) return;
    errors.push(`console.error: ${text}`);
  });
  return () => errors;
}

// All admin sub-pages registered in SystemSettingsPage.tsx. Each entry defines
// the route and a heading role-locator that confirms the page body rendered.
const ADMIN_PAGES: { route: string; heading: string }[] = [
  { route: '/admin/general', heading: 'General' },
  { route: '/admin/users', heading: 'Users' },
  { route: '/admin/project-overview', heading: 'Projects & Namespaces' },
  { route: '/admin/workflows', heading: 'Workflows' },
  { route: '/admin/integrations', heading: 'Integrations' },
  { route: '/admin/authentication', heading: 'Authentication' },
  { route: '/admin/api-keys', heading: 'System API Keys' },
  { route: '/admin/features', heading: 'Features' },
];

test.describe('Admin pages — smoke tests', () => {
  for (const { route, heading } of ADMIN_PAGES) {
    test(`${route} renders without errors`, async ({ page }) => {
      const getErrors = collectPageErrors(page);

      await page.goto(route);
      await page.waitForLoadState('networkidle');

      await expect(page.getByRole('heading', { name: heading, exact: true })).toBeVisible({
        timeout: 10000,
      });

      const errors = getErrors();
      expect(errors, `unexpected JS/console errors on ${route}:\n${errors.join('\n')}`).toEqual([]);
    });
  }
});

// Regression test for the nil-slice bug on the admin project-overview page.
// Previously AdminRepository.ListProjects / ListNamespaces used `var items []T`,
// which marshals to JSON `null` when the DB has no rows. The frontend then did
// `pages.flatMap(p => p.data)` producing `[null]` and the DataTable crashed
// reading `row.key` on null.
test.describe('Admin project-overview — empty state regression', () => {
  test('renders without errors when the API returns empty lists', async ({ page }) => {
    const getErrors = collectPageErrors(page);

    // Stub the three endpoints the page relies on with empty — but well-formed
    // — responses. If the frontend ever stops handling `{data: []}` gracefully
    // this test will fail with a visible render error.
    await page.route('**/api/v1/admin/projects**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], meta: { cursor: '', has_more: false } }),
      });
    });
    await page.route('**/api/v1/admin/namespaces**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], meta: { cursor: '', has_more: false } }),
      });
    });
    await page.route('**/api/v1/admin/stats**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { projects: 0, namespaces: 0, users: 0, storage_bytes: 0 } }),
      });
    });

    await page.goto('/admin/project-overview');
    await page.waitForLoadState('networkidle');

    // Page heading must still render
    await expect(
      page.getByRole('heading', { name: 'Projects & Namespaces', exact: true }),
    ).toBeVisible({ timeout: 10000 });

    // Projects tab (default) should show its empty-state message
    await expect(page.getByText('No projects found', { exact: false })).toBeVisible({
      timeout: 10000,
    });

    // Switch to Namespaces tab and verify the same
    await page.getByRole('button', { name: 'Namespaces' }).click();
    await expect(page.getByText('No namespaces found', { exact: false })).toBeVisible({
      timeout: 10000,
    });

    const errors = getErrors();
    expect(
      errors,
      `empty-state regression — unexpected errors on /admin/project-overview:\n${errors.join('\n')}`,
    ).toEqual([]);
  });
});
