import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';

base.describe.configure({ mode: 'serial' });

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

async function attach(page: any, testInfo: any, name: string) {
  const screenshot = await page.screenshot();
  await testInfo.attach(name, { body: screenshot, contentType: 'image/png' });
}

function uniqueSlug() {
  return `ns-tabs-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 5)}`;
}

test.describe('Namespace Settings Tabs', () => {
  test.beforeEach(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.enableNamespaces(request, adminToken);
  });

  test('default tab is General with display name, slug, icon, color, and Danger Zone', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const slug = uniqueSlug();
    await api.createNamespace(request, adminToken, slug, 'Tabs Default NS');

    await page.goto(`/${slug}/settings`);
    await page.waitForLoadState('networkidle');

    // Header
    await expect(page.getByRole('heading', { name: 'Namespace Settings' })).toBeVisible();

    // No ?tab= in URL — defaults to general
    expect(new URL(page.url()).searchParams.get('tab')).toBeNull();

    // General tab is active (visible content: display name, slug, color, danger zone)
    await expect(page.getByLabel('Display name', { exact: false })).toBeVisible();
    await expect(page.getByLabel('Slug', { exact: false })).toBeVisible();
    await expect(page.getByText('Color', { exact: true })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Danger Zone' })).toBeVisible();

    // Members section is NOT visible on General tab
    await expect(page.getByText('Invite by email', { exact: false })).not.toBeVisible();

    await attach(page, testInfo, 'general-default');

    await api.deleteNamespace(request, adminToken, slug);
  });

  test('clicking Users tab shows members and invite form, hides general fields', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const slug = uniqueSlug();
    await api.createNamespace(request, adminToken, slug, 'Tabs Click NS');

    await page.goto(`/${slug}/settings`);
    await page.waitForLoadState('networkidle');

    // Click on the Users tab
    await page.getByRole('button', { name: 'Users', exact: true }).click();

    // URL updates to ?tab=users
    await expect.poll(() => new URL(page.url()).searchParams.get('tab')).toBe('users');

    // Users tab content visible
    await expect(page.getByText('Invite by email', { exact: false })).toBeVisible();

    // General tab content is hidden (display name input)
    await expect(page.getByLabel('Display name', { exact: false })).not.toBeVisible();
    await expect(page.getByRole('heading', { name: 'Danger Zone' })).not.toBeVisible();

    await attach(page, testInfo, 'users-clicked');

    await api.deleteNamespace(request, adminToken, slug);
  });

  test('deep-link to ?tab=users opens the Users tab directly', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const slug = uniqueSlug();
    await api.createNamespace(request, adminToken, slug, 'Tabs Deep NS');

    await page.goto(`/${slug}/settings?tab=users`);
    await page.waitForLoadState('networkidle');

    // Users tab content should render right away
    await expect(page.getByText('Invite by email', { exact: false })).toBeVisible();
    await expect(page.getByLabel('Display name', { exact: false })).not.toBeVisible();

    await attach(page, testInfo, 'users-deeplink');

    await api.deleteNamespace(request, adminToken, slug);
  });

  test('unknown tab value falls back to General', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const slug = uniqueSlug();
    await api.createNamespace(request, adminToken, slug, 'Tabs Fallback NS');

    await page.goto(`/${slug}/settings?tab=bogus`);
    await page.waitForLoadState('networkidle');

    // General tab content should be visible
    await expect(page.getByLabel('Display name', { exact: false })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Danger Zone' })).toBeVisible();

    await attach(page, testInfo, 'fallback-general');

    await api.deleteNamespace(request, adminToken, slug);
  });

  test('switching back to General clears ?tab from URL', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const slug = uniqueSlug();
    await api.createNamespace(request, adminToken, slug, 'Tabs Switch NS');

    await page.goto(`/${slug}/settings?tab=users`);
    await page.waitForLoadState('networkidle');

    // Click General tab
    await page.getByRole('button', { name: 'General', exact: true }).click();

    // URL no longer carries ?tab=users
    await expect.poll(() => new URL(page.url()).searchParams.get('tab')).toBeNull();

    // General content visible
    await expect(page.getByLabel('Display name', { exact: false })).toBeVisible();

    await attach(page, testInfo, 'switch-back-general');

    await api.deleteNamespace(request, adminToken, slug);
  });

  test('editing display name in General tab still works', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const slug = uniqueSlug();
    await api.createNamespace(request, adminToken, slug, 'Tabs Edit NS');

    await page.goto(`/${slug}/settings`);
    await page.waitForLoadState('networkidle');

    const nameInput = page.getByLabel('Display name', { exact: false });
    await nameInput.fill('Tabs Edit NS Renamed');

    await page.getByRole('button', { name: /^save$/i }).click();
    await page.waitForTimeout(1000);

    const ns = await api.getNamespace(request, adminToken, slug);
    expect(ns.display_name).toBe('Tabs Edit NS Renamed');

    await attach(page, testInfo, 'edit-display-name');

    await api.deleteNamespace(request, adminToken, slug);
  });

  test('email invite form on Users tab opens confirmation modal', async ({ page, request }, testInfo) => {
    const adminToken = getAdminToken();
    const slug = uniqueSlug();
    await api.createNamespace(request, adminToken, slug, 'Tabs Invite NS');

    await page.goto(`/${slug}/settings?tab=users`);
    await page.waitForLoadState('networkidle');

    const emailInput = page.getByPlaceholder('user@example.com');
    await expect(emailInput).toBeVisible();
    await emailInput.fill('invitee@example.com');

    await page.getByRole('button', { name: /send invite/i }).click();

    // Confirmation modal appears
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await expect(dialog.getByText(/invitee@example\.com/)).toBeVisible();

    await attach(page, testInfo, 'invite-modal');

    // Dismiss modal
    await dialog.getByRole('button', { name: /cancel/i }).click();

    await api.deleteNamespace(request, adminToken, slug);
  });
});
