import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

// Unauthenticated context — password reset pages are public
const test = base.extend({
  storageState: async ({}, use) => {
    await use({ cookies: [], origins: [] } as any);
  },
});

// SMTP is configured by chromium.setup.ts (runs before this project)
test.describe.configure({ mode: 'serial' });

test.describe('Password Reset', () => {
  test('full password reset flow: forgot → email → reset → logged in', async ({ page, request }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const email = `pwreset-${uniqueId}@e2e.local`;
    const displayName = `PwReset User ${uniqueId}`;
    const oldPassword = 'OldPassword123!';
    const newPassword = 'NewPassword456!';

    // Create user via admin API, then set a known password
    const created = await api.createUser(request, adminToken, email, displayName);
    const loginResult = await api.login(request, email, created.temporary_password);
    await api.changePassword(request, loginResult.token, created.temporary_password, oldPassword);

    // Step 1: Verify login page has "Forgot your password?" link, then
    // navigate directly to the forgot-password page (avoids SPA transition
    // race where both pages share an "Email" input).
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('Forgot your password?')).toBeVisible();
    await page.goto('/forgot-password');
    await page.waitForLoadState('networkidle');

    // Step 2: Enter email and submit
    await page.getByLabel('Email').fill(email);
    await page.getByRole('button', { name: 'Send reset link' }).click();

    // Step 3: Verify "check your email" message
    await expect(page.getByText('Check your email')).toBeVisible({ timeout: 10000 });

    // Step 4: Retrieve the password reset email from Mailpit
    const msg = await api.waitForMailpitMessage(request, email);
    expect(msg.Subject).toContain('Reset');

    // Step 5: Extract the reset URL from the email HTML
    const urlMatch = msg.HTML.match(/href="([^"]*reset-password[^"]*)"/);
    expect(urlMatch).toBeTruthy();
    const resetUrl = urlMatch![1];

    // Step 6: Navigate to the reset URL
    await page.goto(resetUrl);
    await page.waitForLoadState('networkidle');

    // Step 7: Set new password
    await expect(page.getByText('Set a new password')).toBeVisible();
    await page.getByLabel('New Password', { exact: true }).fill(newPassword);
    await page.getByLabel('Confirm Password').fill(newPassword);
    await page.getByRole('button', { name: 'Reset password & sign in' }).click();

    // Step 8: Should be redirected to projects (logged in)
    await page.waitForURL(/\/(d|[^/]+)\/projects/, { timeout: 10000 });
    await expect(page.getByRole('heading', { name: 'Projects' }).first()).toBeVisible();

    // Step 9: Verify old password no longer works
    const oldLoginRes = await request.post('/api/v1/auth/login', {
      data: { email, password: oldPassword },
    });
    expect(oldLoginRes.status()).toBe(401);

    // Step 10: Verify new password works
    const newLoginRes = await request.post('/api/v1/auth/login', {
      data: { email, password: newPassword },
    });
    expect(newLoginRes.status()).toBe(200);
  });

  test('forgot password shows same message for unknown email', async ({ page }) => {
    await page.goto('/forgot-password');
    await page.waitForLoadState('networkidle');

    await page.getByLabel('Email').fill('nonexistent@e2e.local');
    await page.getByRole('button', { name: 'Send reset link' }).click();

    // Should still show "check your email" to prevent enumeration
    await expect(page.getByText('Check your email')).toBeVisible({ timeout: 10000 });
  });

  test('reset password rejects mismatched passwords', async ({ page }) => {
    await page.goto('/reset-password?token=fake-token');
    await page.waitForLoadState('networkidle');

    await page.getByLabel('New Password', { exact: true }).fill('Password123!');
    await page.getByLabel('Confirm Password').fill('DifferentPass123!');
    await page.getByRole('button', { name: 'Reset password & sign in' }).click();

    // Should show mismatch error
    await expect(page.getByText('Passwords do not match')).toBeVisible();
  });

  test('reset password rejects short password', async ({ page }) => {
    await page.goto('/reset-password?token=fake-token');
    await page.waitForLoadState('networkidle');

    await page.getByLabel('New Password', { exact: true }).fill('short');
    await page.getByLabel('Confirm Password').fill('short');
    await page.getByRole('button', { name: 'Reset password & sign in' }).click();

    // Should show too-short error
    await expect(page.getByText('at least 8 characters')).toBeVisible();
  });

  test('forgot password API rejects malformed emails', async ({ request }) => {
    const res = await request.post('/api/v1/auth/forgot-password', {
      data: { email: 'not-an-email' },
    });
    expect(res.status()).toBe(400);
  });

  test('reset password API rejects invalid token', async ({ request }) => {
    const res = await request.post('/api/v1/auth/reset-password', {
      data: { token: 'nonexistent-token', password: 'NewPassword123!' },
    });
    expect(res.status()).toBe(404);
  });
});
