import { test as base, expect, getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';
import { openPaletteAndType } from '../../lib/palette';

/**
 * The namespace half of the command palette's navigation catalog (TF-431).
 *
 * These rows only exist on an instance with namespaces turned on — the same
 * condition AppSidebar hides its switcher under — so they live in the
 * `namespace` Playwright project. A namespace row is not a route: activating it
 * calls `setActiveNamespace`, which switches the context and lands on that
 * namespace's project list. That is how a project in another namespace stays
 * reachable now that `g p` is retired and entity search is scoped to the active
 * project (`project` hits stay global on the backend, but they live only in the
 * semantic index, which the E2E stack runs without).
 */

// These tests share the global namespaces_enabled setting, so run serially.
base.describe.configure({ mode: 'serial' });

// Namespace creation requires the admin role.
const test = base.extend({
  storageState: async ({}, use) => {
    const state = {
      cookies: [],
      origins: [
        {
          origin: process.env.BASE_URL || 'http://localhost:5173',
          localStorage: [{ name: 'taskwondo_token', value: getAdminToken() }],
        },
      ],
    };
    await use(state as any);
  },
});

/** A navigation row whose whole label is exactly `label`. */
function navRow(page: import('@playwright/test').Page, label: string) {
  return page.locator('[data-nav-item]', { hasText: new RegExp(`^${label}$`) });
}

test.describe('Command palette — namespace rows', () => {
  let nsSlug: string;
  let nsName: string;
  let nsProjectKey: string;
  let nsProjectName: string;
  let uid: string;

  test.beforeAll(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.enableNamespaces(request, adminToken);

    uid = `CP${randomUUID().slice(0, 4).toUpperCase()}`;
    nsSlug = `ns-palette-${uid.toLowerCase()}`;
    nsName = `Palette NS ${uid}`;
    await api.createNamespace(request, adminToken, nsSlug, nsName);

    nsProjectKey = `Q${randomUUID().slice(0, 4).toUpperCase()}`;
    nsProjectName = `Palette NS Project ${uid}`;
    await api.createProject(request, adminToken, nsProjectKey, nsProjectName, nsSlug);

    await api.setPreference(request, adminToken, 'welcome_dismissed', true);
  });

  test.afterAll(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.deleteNamespace(request, adminToken, nsSlug).catch(() => {});
  });

  test('a namespace row switches namespace and reaches its project', async ({ page }) => {
    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');

    // The namespace's unique id matches both of its rows: "Switch to <name>"
    // and "<name> settings".
    await openPaletteAndType(page, uid);

    const switchRow = navRow(page, `Switch to ${nsName}`);
    await expect(switchRow).toHaveCount(1);
    await switchRow.click();

    // Switching a namespace is context state, not a route the palette owns —
    // the context navigates to that namespace's project list itself.
    await expect(page).toHaveURL(new RegExp(`/${nsSlug}/projects/?$`), { timeout: 10000 });

    // ...and the project that only exists in that namespace is right there.
    // Match the desktop table row, not the `sm:hidden` card list that shares
    // the DOM with it.
    const projectRow = page.getByRole('row', { name: new RegExp(nsProjectName) });
    await expect(projectRow).toBeVisible({ timeout: 10000 });
    await projectRow.click();
    await expect(page).toHaveURL(new RegExp(`/${nsSlug}/projects/${nsProjectKey}`), {
      timeout: 10000,
    });
    await expect(page.getByText('Project not found.')).toHaveCount(0);
  });

  test('a non-default namespace also offers its settings page', async ({ page }) => {
    await page.goto('/d/projects');
    await page.waitForLoadState('networkidle');

    await openPaletteAndType(page, uid);

    // The catalog only offers Settings for a non-default namespace, because
    // the default namespace has no settings page to offer.
    const settingsRow = navRow(page, `${nsName} settings`);
    await expect(settingsRow).toHaveCount(1);
    await settingsRow.click();

    await expect(page).toHaveURL(new RegExp(`/${nsSlug}/settings$`), { timeout: 10000 });
  });
});
