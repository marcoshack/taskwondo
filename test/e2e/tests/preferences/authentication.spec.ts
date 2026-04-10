import { test, expect } from '../../lib/fixtures';

const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

async function attach(page: any, testInfo: any, name: string) {
  const screenshot = await page.screenshot();
  await testInfo.attach(name, { body: screenshot, contentType: 'image/png' });
}

async function dismissWelcomeModal(page: any) {
  const heading = page.getByRole('heading', { name: 'Welcome' });
  if (await heading.isVisible({ timeout: 2000 }).catch(() => false)) {
    const checkbox = page.getByRole('checkbox', { name: "Don't show this again" });
    if (await checkbox.isVisible({ timeout: 500 }).catch(() => false)) {
      await checkbox.check();
    }
    await page.keyboard.press('Escape');
    await heading.waitFor({ state: 'hidden', timeout: 3000 }).catch(() => {});
  }
}

async function navigateToAuthentication(page: any) {
  await page.goto('/');
  await dismissWelcomeModal(page);
  await page.goto('/preferences/authentication');
  await expect(page.getByRole('heading', { name: 'Authentication' })).toBeVisible();
}

test.describe('Authentication Page', () => {
  test('renders authentication page with three tabs', async ({ page }, testInfo) => {
    await navigateToAuthentication(page);

    // Verify all three tabs are visible (use exact: true to avoid matching submit buttons)
    await expect(page.getByRole('button', { name: 'Password', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Connected Accounts' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'API Keys' })).toBeVisible();
    await attach(page, testInfo, '01-authentication-page');
  });

  test('password tab is active by default', async ({ page }, testInfo) => {
    await navigateToAuthentication(page);

    // Password tab should be active (has indigo styling)
    const passwordTab = page.getByRole('button', { name: 'Password', exact: true });
    await expect(passwordTab).toHaveClass(/text-indigo/);

    // Password form fields should be visible
    await expect(page.getByLabel('Current Password')).toBeVisible();
    await expect(page.getByLabel('New Password', { exact: true })).toBeVisible();
    await expect(page.getByLabel('Confirm New Password')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Change Password' })).toBeVisible();
    await attach(page, testInfo, '01-password-tab');
  });

  test('switch between tabs', async ({ page }, testInfo) => {
    await navigateToAuthentication(page);

    // Start on password tab - verify form is visible
    await expect(page.getByLabel('Current Password')).toBeVisible();
    await attach(page, testInfo, '01-password-tab');

    // Switch to Connected Accounts tab
    await page.getByRole('button', { name: 'Connected Accounts' }).click();
    await expect(page.getByLabel('Current Password')).not.toBeVisible();
    // Connected accounts tab should show empty state
    await expect(page.getByText('No connected accounts')).toBeVisible();
    await attach(page, testInfo, '02-connected-accounts-tab');

    // Switch to API Keys tab
    await page.getByRole('button', { name: 'API Keys' }).click();
    await expect(page.getByText(/Create New Key|No API keys yet/)).toBeVisible();
    await attach(page, testInfo, '03-api-keys-tab');

    // Switch back to Password tab
    await page.getByRole('button', { name: 'Password', exact: true }).click();
    await expect(page.getByLabel('Current Password')).toBeVisible();
    await attach(page, testInfo, '04-back-to-password');
  });

  test('password change validates minimum length', async ({ page }, testInfo) => {
    await navigateToAuthentication(page);

    // Fill in passwords with short new password
    await page.getByLabel('Current Password').fill('currentpass');
    await page.getByLabel('New Password', { exact: true }).fill('short');
    await page.getByLabel('Confirm New Password').fill('short');

    // Submit
    await page.getByRole('button', { name: 'Change Password' }).click();

    // Should show validation error
    await expect(page.locator('p.text-red-600, p.text-red-400').filter({ hasText: '8 characters' })).toBeVisible();
    await attach(page, testInfo, '01-too-short-error');
  });

  test('password change disables submit and shows inline error when passwords do not match', async ({ page }, testInfo) => {
    await navigateToAuthentication(page);

    const submitButton = page.getByRole('button', { name: 'Change Password' });

    // Fill in passwords that don't match
    await page.getByLabel('Current Password').fill('currentpass');
    await page.getByLabel('New Password', { exact: true }).fill('newpassword123');
    await page.getByLabel('Confirm New Password').fill('differentpassword');

    // Submit button should be disabled while the passwords mismatch
    await expect(submitButton).toBeDisabled();
    // Inline error should be visible under the Confirm New Password field
    await expect(page.getByText('Passwords do not match.')).toBeVisible();
    await attach(page, testInfo, '01-mismatch-disabled');

    // Fix the confirmation so the passwords match — button becomes enabled, error goes away
    await page.getByLabel('Confirm New Password').fill('newpassword123');
    await expect(submitButton).toBeEnabled();
    await expect(page.getByText('Passwords do not match.')).not.toBeVisible();
    await attach(page, testInfo, '02-match-enabled');
  });

  test('connected accounts tab shows empty state', async ({ page }, testInfo) => {
    await navigateToAuthentication(page);

    // Switch to Connected Accounts tab
    await page.getByRole('button', { name: 'Connected Accounts' }).click();

    // Should show empty state message
    await expect(page.getByText('No connected accounts')).toBeVisible();
    await attach(page, testInfo, '01-empty-connected-accounts');
  });

  test('sidebar shows lock icon for authentication', async ({ page }, testInfo) => {
    await page.goto('/');
    await dismissWelcomeModal(page);
    await page.goto('/preferences/profile');

    // The Authentication sidebar link should be visible
    const authLink = page.getByRole('link', { name: 'Authentication' });
    await expect(authLink).toBeVisible();

    // Click it to navigate
    await authLink.click();
    await expect(page).toHaveURL(/preferences\/authentication/);
    await expect(page.getByRole('heading', { name: 'Authentication' })).toBeVisible();
    await attach(page, testInfo, '01-auth-via-sidebar');
  });
});

test.describe('Authentication Page — API', () => {
  test('/auth/me returns has_password: true for users with a password', async ({ request, testUser }) => {
    const res = await request.get(`${BASE_URL}/api/v1/auth/me`, {
      headers: { Authorization: `Bearer ${testUser.token}` },
    });
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.data.has_password).toBe(true);
  });

  test('change-password rejects request without new_password', async ({ request, testUser }) => {
    const res = await request.post(`${BASE_URL}/api/v1/auth/change-password`, {
      headers: { Authorization: `Bearer ${testUser.token}` },
      data: { old_password: testUser.password },
    });
    expect(res.status()).toBe(400);
    const body = await res.json();
    expect(body.error.message).toContain('new_password');
  });

  test('change-password rejects empty old_password for user with password', async ({ request, testUser }) => {
    // Regular users have a password, so empty old_password must be rejected (401 invalid current password)
    const res = await request.post(`${BASE_URL}/api/v1/auth/change-password`, {
      headers: { Authorization: `Bearer ${testUser.token}` },
      data: { old_password: '', new_password: 'newpassword123' },
    });
    expect(res.status()).toBe(401);
  });

  test('change-password succeeds with correct old_password and updates the user', async ({ request, testUser }) => {
    const newPassword = 'NewPass456!';
    const res = await request.post(`${BASE_URL}/api/v1/auth/change-password`, {
      headers: { Authorization: `Bearer ${testUser.token}` },
      data: { old_password: testUser.password, new_password: newPassword },
    });
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.data.token).toBeTruthy();

    // Verify we can login with the new password
    const loginRes = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: { email: testUser.email, password: newPassword },
    });
    expect(loginRes.status()).toBe(200);
  });
});
