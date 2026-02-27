import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';

// Override storageState to inject admin token instead of the regular test user
const test = base.extend({
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

/** Navigate to integrations page and wait for SMTP config to load */
async function gotoIntegrations(page: any) {
  await page.goto('/admin/integrations');
  await page.waitForLoadState('networkidle');
  await dismissWelcomeModal(page);
  // Expand the SMTP card so fields are visible
  await page.getByRole('heading', { name: 'SMTP' }).click();
}

// SMTP config is a shared global resource — tests must run serially
test.describe.configure({ mode: 'serial' });

test.describe('SMTP Configuration', () => {
  // Reset SMTP config before each test to start fresh
  test.beforeEach(async ({ request }) => {
    const adminToken = getAdminToken();
    await api.resetSMTPConfig(request, adminToken);
  });

  test('displays SMTP configuration page with disabled state', async ({ page }, testInfo) => {
    await gotoIntegrations(page);

    // Heading is visible
    await expect(page.getByRole('heading', { name: 'SMTP' })).toBeVisible();

    // Toggle is off (config was reset to disabled)
    const toggle = page.getByRole('switch');
    await expect(toggle).toHaveAttribute('aria-checked', 'false');

    // Fields should be visible
    await expect(page.getByLabel('SMTP Host')).toBeVisible();
    await expect(page.getByLabel('SMTP Port')).toBeVisible();
    await expect(page.getByLabel('Username')).toBeVisible();
    await expect(page.getByLabel('Password')).toBeVisible();

    await attach(page, testInfo, '01-default-state');
  });

  test('enables SMTP and saves configuration', async ({ page }, testInfo) => {
    await gotoIntegrations(page);

    // Enable SMTP
    await page.getByRole('switch').click();

    // Fill in SMTP fields
    await page.getByLabel('SMTP Host').fill('smtp.example.com');
    await page.getByLabel('SMTP Port').fill('587');
    await page.getByLabel('Username').fill('testuser@example.com');
    await page.getByLabel('Password').fill('secret-password');
    await page.getByLabel('From Address').fill('noreply@example.com');
    await page.getByLabel('From Name').fill('Taskwondo Test');

    // Select encryption via the select element
    await page.locator('select').selectOption('tls');

    // Fill IMAP fields
    await page.getByLabel('IMAP Host').fill('imap.example.com');
    await page.getByLabel('IMAP Port').fill('993');

    await attach(page, testInfo, '02-filled-form');

    // Save
    await page.getByRole('button', { name: 'Save' }).click();

    // Wait for success message
    await expect(page.getByText('SMTP settings saved.')).toBeVisible();
    await attach(page, testInfo, '03-saved');
  });

  test('masks password and loads fields after save', async ({ request, page }, testInfo) => {
    // Save config via API first
    const adminToken = getAdminToken();
    await api.setSMTPConfig(request, adminToken, {
      enabled: true,
      smtp_host: 'smtp.example.com',
      smtp_port: 587,
      imap_host: '',
      imap_port: 993,
      username: 'user@example.com',
      password: 'my-secret-pass',
      encryption: 'starttls',
      from_address: 'noreply@example.com',
      from_name: 'Taskwondo',
    });

    await gotoIntegrations(page);

    // Password should be masked
    await expect(page.getByLabel('Password')).toHaveValue('••••••••');

    // Verify other fields loaded correctly
    await expect(page.getByLabel('SMTP Host')).toHaveValue('smtp.example.com');
    await expect(page.getByLabel('Username')).toHaveValue('user@example.com');
    await expect(page.getByLabel('From Address')).toHaveValue('noreply@example.com');
    await expect(page.getByLabel('From Name')).toHaveValue('Taskwondo');

    // Toggle should be on
    await expect(page.getByRole('switch')).toHaveAttribute('aria-checked', 'true');
    await attach(page, testInfo, '01-loaded-config');
  });

  test('preserves password when saving without changing it', async ({ request, page }, testInfo) => {
    // Save config via API
    const adminToken = getAdminToken();
    await api.setSMTPConfig(request, adminToken, {
      enabled: true,
      smtp_host: 'smtp.example.com',
      smtp_port: 587,
      imap_host: '',
      imap_port: 993,
      username: 'user@example.com',
      password: 'original-password',
      encryption: 'starttls',
      from_address: 'noreply@example.com',
      from_name: '',
    });

    await gotoIntegrations(page);

    // Change the from name without touching password
    await page.getByLabel('From Name').fill('Updated Name');
    await page.getByRole('button', { name: 'Save' }).click();
    await expect(page.getByText('SMTP settings saved.')).toBeVisible();

    // Reload and wait for data to load
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.getByRole('heading', { name: 'SMTP' }).click();

    // Verify password is still masked and from name persisted
    await expect(page.getByLabel('Password')).toHaveValue('••••••••');
    await expect(page.getByLabel('From Name')).toHaveValue('Updated Name');
    await attach(page, testInfo, '01-password-preserved');
  });

  test('clears password field on focus for new input', async ({ request, page }, testInfo) => {
    // Save config via API
    const adminToken = getAdminToken();
    await api.setSMTPConfig(request, adminToken, {
      enabled: true,
      smtp_host: 'smtp.example.com',
      smtp_port: 587,
      imap_host: '',
      imap_port: 993,
      username: 'user@example.com',
      password: 'old-password',
      encryption: 'starttls',
      from_address: 'noreply@example.com',
      from_name: '',
    });

    await gotoIntegrations(page);

    // Focus the password field — should clear the mask
    const passwordInput = page.getByLabel('Password');
    await expect(passwordInput).toHaveValue('••••••••');
    await passwordInput.focus();
    await expect(passwordInput).toHaveValue('');
    await attach(page, testInfo, '01-password-cleared-on-focus');
  });

  test('toggles form field interactivity', async ({ page }, testInfo) => {
    await gotoIntegrations(page);

    // SMTP is off by default — verify fields are not interactive
    const toggle = page.getByRole('switch');
    await expect(toggle).toHaveAttribute('aria-checked', 'false');

    // Enable SMTP
    await toggle.click();
    await expect(toggle).toHaveAttribute('aria-checked', 'true');

    // Fields should be interactive when enabled
    await page.getByLabel('SMTP Host').fill('test-host');
    await expect(page.getByLabel('SMTP Host')).toHaveValue('test-host');

    // Disable SMTP
    await toggle.click();
    await expect(toggle).toHaveAttribute('aria-checked', 'false');

    await attach(page, testInfo, '01-toggle-interactivity');
  });

  test('persists encryption selection', async ({ page }, testInfo) => {
    await gotoIntegrations(page);

    // Enable SMTP and fill required fields
    await page.getByRole('switch').click();
    await page.getByLabel('SMTP Host').fill('smtp.example.com');
    await page.getByLabel('SMTP Port').fill('465');
    await page.getByLabel('Username').fill('user@example.com');
    await page.getByLabel('Password').fill('pass');
    await page.getByLabel('From Address').fill('noreply@example.com');

    // Select TLS encryption
    await page.locator('select').selectOption('tls');

    // Save
    await page.getByRole('button', { name: 'Save' }).click();
    await expect(page.getByText('SMTP settings saved.')).toBeVisible();

    // Reload and verify encryption persisted
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.getByRole('heading', { name: 'SMTP' }).click();
    await expect(page.locator('select')).toHaveValue('tls');
    await attach(page, testInfo, '01-encryption-persisted');
  });

  test('non-admin user cannot access integrations page', async ({ page, testUser }, testInfo) => {
    await page.goto('/');
    await dismissWelcomeModal(page);

    // Inject regular user token
    await page.evaluate((token: string) => {
      localStorage.setItem('taskwondo_token', token);
    }, testUser.token);

    // Try to access admin page
    await page.goto('/admin/integrations');
    await page.waitForLoadState('networkidle');

    // Should not see the SMTP heading
    await expect(page.getByRole('heading', { name: 'SMTP' })).not.toBeVisible();
    await attach(page, testInfo, '01-non-admin-blocked');
  });
});
