import { test as base, expect, getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';
import { openProjectSwitcher } from '../../lib/palette';

// Run serially inside this file — tests share the namespaces_enabled setting
// and the shared project fixture set up in beforeAll.
base.describe.configure({ mode: 'serial' });

// Use admin context since namespace creation requires admin role.
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

function uniqueSlug() {
  return `ns-go-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 5)}`;
}

function uniqueKey() {
  return `G${randomUUID().slice(0, 4).toUpperCase()}`;
}

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

test.describe('g-o shortcut after namespace switch (TF-345)', () => {
  let nsSlug: string;
  let sharedKey: string;

  test.beforeAll(async ({ request }) => {
    const adminToken = getAdminToken();

    await api.enableNamespaces(request, adminToken);

    nsSlug = uniqueSlug();
    await api.createNamespace(request, adminToken, nsSlug, `Test NS ${nsSlug}`);

    // Use the same project key in both namespaces: this pins `activeProjectKey`
    // across the namespace switch so the only dep that changes is `p` — which
    // is exactly the stale-closure that TF-345 is about.
    sharedKey = uniqueKey();
    await api.createProject(request, adminToken, sharedKey, `Default Proj ${sharedKey}`);
    await api.createProject(request, adminToken, sharedKey, `NS Proj ${sharedKey}`, nsSlug);

    await api.setPreference(request, adminToken, 'welcome_dismissed', true);
    // Make sure the project switcher shows both namespaces' projects.
    await api.setPreference(request, adminToken, 'project_switcher_all_namespaces', true);
  });

  test.afterAll(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.deleteNamespace(request, adminToken, nsSlug).catch(() => {});
  });

  test('g o re-binds to the active namespace after an SPA namespace switch', async ({ page }) => {
    // The bug only reproduces when AppShell stays mounted across the namespace
    // change — a fresh page.goto() remounts React and re-registers `g o` with
    // the current `p`. This whole flow therefore stays within one SPA session.

    // 1. Land on the project in the secondary namespace. `g o` works here.
    await page.goto(`/${nsSlug}/projects/${sharedKey}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 10000 });

    await page.keyboard.press('g');
    await page.keyboard.press('o');
    await expect(page).toHaveURL(new RegExp(`/${nsSlug}/projects/${sharedKey}/items`), { timeout: 5000 });
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // 2. Open the same project key in the *default* namespace via the project
    //    switcher (SPA nav). `activeProjectKey` stays equal to `sharedKey`, but
    //    the namespace segment flips from `<nsSlug>` to `d`, so `p` changes.
    // `g p` is retired (TF-431) — the switcher opens from the nav project badge.
    const modal = await openProjectSwitcher(page);
    const searchInput = modal.getByRole('textbox', { name: /search projects/i });
    // Search the unique project name so only the default-namespace row matches.
    await searchInput.fill(`Default Proj ${sharedKey}`);
    const defaultRow = modal.getByRole('button', { name: new RegExp(`Default Proj ${sharedKey}`, 'i') });
    await expect(defaultRow).toBeVisible({ timeout: 3000 });
    await defaultRow.click();
    await expect(page).toHaveURL(new RegExp(`/d/projects/${sharedKey}`), { timeout: 5000 });

    // Wait for AppShell's namespace context to finish syncing. The sidebar
    // "Items" link is built from `p(...)`, so once its href points at /d/...,
    // `p` has updated — and with the TF-345 fix, the `g o` combo has been
    // re-registered against the new `p`. Without this wait the key press can
    // race the syncFromUrl effect.
    const itemsLink = page.locator('a[href="/d/projects/' + sharedKey + '/items"]').first();
    await expect(itemsLink).toBeVisible({ timeout: 5000 });

    // 3. Press g o. Without the TF-345 fix, `p` is stale and the URL flips
    //    back to the secondary namespace. With the fix, it must stay under /d/.
    await page.keyboard.press('g');
    await page.keyboard.press('o');
    await expect(page).toHaveURL(new RegExp(`/d/projects/${sharedKey}/items`), { timeout: 5000 });
    expect(page.url()).not.toMatch(new RegExp(`/${nsSlug}/`));
    // Confirm the page actually renders, not just a URL flip.
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });
  });
});
