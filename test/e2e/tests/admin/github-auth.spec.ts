import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';

// Admin-authenticated context for admin settings pages
const adminTest = base.extend({
  storageState: async ({}, use) => {
    const adminToken = getAdminToken();
    const state = {
      cookies: [],
      origins: [
        {
          origin: process.env.BASE_URL || 'http://localhost:5173',
          localStorage: [
            { name: 'taskwondo_token', value: adminToken },
          ],
        },
      ],
    };
    await use(state as any);
  },
});

// Unauthenticated context for login page tests
const unauthTest = base.extend({
  storageState: async ({}, use) => {
    await use({ cookies: [], origins: [] } as any);
  },
});

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

async function gotoAuthSettings(page: any) {
  await page.goto('/admin/authentication');
  await page.waitForLoadState('networkidle');
  await dismissWelcomeModal(page);
}

/** Locate the GitHub card container on the auth settings page */
function githubCard(page: any) {
  return page.locator('div.rounded-lg').filter({ hasText: 'GitHub' }).filter({ hasText: 'Allow users to sign in with their GitHub account.' });
}

adminTest.describe('GitHub Authentication — Admin Settings', () => {
  adminTest.describe.configure({ mode: 'serial' });

  // Start with GitHub disabled and unconfigured
  adminTest.beforeAll(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.setSystemSetting(request, adminToken, 'auth_github_enabled', false);
    // Clear any existing OAuth config so toggle starts disabled
    await api.deleteSystemSetting(request, adminToken, 'oauth_github_config');
  });

  adminTest('displays GitHub provider card on authentication page', async ({ page }) => {
    await gotoAuthSettings(page);

    const card = githubCard(page);
    await expect(card.getByRole('heading', { name: 'GitHub' })).toBeVisible();
    await expect(card.getByText('Allow users to sign in with their GitHub account.')).toBeVisible();
  });

  adminTest('toggle is disabled when provider has no configuration', async ({ page, request }) => {
    // Ensure config is cleared right before this test
    const adminToken = getAdminToken();
    await api.deleteSystemSetting(request, adminToken, 'oauth_github_config');
    await api.setSystemSetting(request, adminToken, 'auth_github_enabled', false);

    await gotoAuthSettings(page);

    const card = githubCard(page);
    const toggle = card.getByRole('switch');

    // Toggle should be off
    await expect(toggle).toHaveAttribute('aria-checked', 'false');

    // Toggle should be disabled (not clickable) since no config exists
    await expect(toggle).toBeDisabled();
  });

  adminTest('expanding GitHub card shows credential fields and redirect URI', async ({ page }) => {
    await gotoAuthSettings(page);

    const card = githubCard(page);

    // Click the heading area to expand
    await card.getByRole('heading', { name: 'GitHub' }).click();

    // Credential fields should be visible inside the card
    await expect(card.getByLabel('Client ID')).toBeVisible();
    await expect(card.getByLabel('Client Secret')).toBeVisible();

    // Redirect URI field with github callback path
    await expect(page.locator('input[value*="/auth/github/callback"]')).toBeVisible();
  });

  adminTest('saving GitHub OAuth credentials via UI persists them', async ({ page }) => {
    await gotoAuthSettings(page);

    const card = githubCard(page);

    // Expand the card
    await card.getByRole('heading', { name: 'GitHub' }).click();

    // Fill in credentials
    await card.getByLabel('Client ID').fill('ui-test-github-id');
    await card.getByLabel('Client Secret').fill('ui-test-github-secret');

    // Save button should be enabled
    const saveButton = card.getByRole('button', { name: 'Save' });
    await expect(saveButton).toBeEnabled();
    await saveButton.click();

    // Wait for save confirmation
    await expect(card.getByText('Saved')).toBeVisible({ timeout: 5000 });

    // Reload and verify credentials persisted
    await page.reload();
    await page.waitForLoadState('networkidle');
    await dismissWelcomeModal(page);

    // Expand card again after reload
    const reloadedCard = githubCard(page);
    await reloadedCard.getByRole('heading', { name: 'GitHub' }).click();

    // Client ID should show the saved value
    await expect(reloadedCard.getByLabel('Client ID')).toHaveValue('ui-test-github-id');

    // Client Secret should be masked (password mask)
    await expect(reloadedCard.getByLabel('Client Secret')).toHaveValue('••••••••');
  });

  adminTest('toggle is enabled after credentials are saved', async ({ page }) => {
    await gotoAuthSettings(page);

    const card = githubCard(page);
    const toggle = card.getByRole('switch');

    // Toggle should now be interactive (not disabled) since credentials are saved
    await expect(toggle).toBeEnabled();
  });

  adminTest('clicking GitHub toggle enables it and persists after reload', async ({ page }) => {
    await gotoAuthSettings(page);

    const card = githubCard(page);
    const toggle = card.getByRole('switch');

    // Toggle should be off
    await expect(toggle).toHaveAttribute('aria-checked', 'false');

    // Click to enable
    await toggle.click();
    await expect(toggle).toHaveAttribute('aria-checked', 'true');

    // Reload page — toggle should still be on (persisted to DB via API)
    await page.reload();
    await page.waitForLoadState('networkidle');
    await dismissWelcomeModal(page);

    const reloadedToggle = githubCard(page).getByRole('switch');
    await expect(reloadedToggle).toHaveAttribute('aria-checked', 'true');
  });

  adminTest('clicking GitHub toggle again disables it and persists after reload', async ({ page }) => {
    await gotoAuthSettings(page);

    const card = githubCard(page);
    const toggle = card.getByRole('switch');

    // Toggle should be on from previous test
    await expect(toggle).toHaveAttribute('aria-checked', 'true');

    // Click to disable
    await toggle.click();
    await expect(toggle).toHaveAttribute('aria-checked', 'false');

    // Reload page — toggle should still be off
    await page.reload();
    await page.waitForLoadState('networkidle');
    await dismissWelcomeModal(page);

    const reloadedToggle = githubCard(page).getByRole('switch');
    await expect(reloadedToggle).toHaveAttribute('aria-checked', 'false');
  });

  // Clean up
  adminTest.afterAll(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.setSystemSetting(request, adminToken, 'auth_github_enabled', false);
  });
});

unauthTest.describe('GitHub Authentication — Login Page', () => {
  unauthTest.describe.configure({ mode: 'serial' });

  unauthTest('login page shows GitHub button when provider is enabled and configured', async ({ page, request }) => {
    const adminToken = getAdminToken();

    // Configure and enable GitHub via API (prerequisite for the UI test)
    const baseUrl = process.env.BASE_URL || 'http://localhost:5173';
    await request.put(`${baseUrl}/api/v1/admin/settings/oauth_config/github`, {
      headers: { Authorization: `Bearer ${adminToken}` },
      data: { client_id: 'login-test-id', client_secret: 'login-test-secret' },
    });
    await api.setSystemSetting(request, adminToken, 'auth_github_enabled', true);

    // Visit login page
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    // GitHub sign-in button should be visible
    await expect(page.getByRole('button', { name: /Sign in with GitHub/ })).toBeVisible();
  });

  unauthTest('login page hides GitHub button when provider is disabled', async ({ page, request }) => {
    const adminToken = getAdminToken();

    // Disable GitHub
    await api.setSystemSetting(request, adminToken, 'auth_github_enabled', false);

    // Visit login page
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    // GitHub sign-in button should NOT be visible
    await expect(page.getByRole('button', { name: /Sign in with GitHub/ })).not.toBeVisible();
  });

  // Clean up
  unauthTest.afterAll(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.setSystemSetting(request, adminToken, 'auth_github_enabled', false);
  });
});
