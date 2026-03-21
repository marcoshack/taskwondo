import { test as base, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

// Unauthenticated context — registration pages are public
const test = base.extend({
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

test.describe.configure({ mode: 'serial' });

test.describe('Email Verification Token Lifecycle', () => {
  test('re-registration invalidates old token and new token works', async ({ page, request }) => {
    const uniqueId = randomUUID().slice(0, 8);
    const email = `token-lc-${uniqueId}@e2e.local`;
    const displayName = `Token LC ${uniqueId}`;
    const password = 'SecurePass123!';

    // Step 1: Register for the first time (creates token 1)
    const res1 = await request.post('/api/v1/auth/register', {
      data: { email, display_name: displayName },
    });
    expect(res1.status()).toBe(200);
    const msg1 = await api.waitForMailpitMessage(request, email);
    const urlMatch1 = msg1.HTML.match(/href="([^"]*verify-email[^"]*)"/);
    expect(urlMatch1).toBeTruthy();
    const oldVerifyUrl = urlMatch1![1];

    // Clear mailbox before re-registration
    await api.deleteMailpitMessages(request);

    // Step 2: Re-register with the same email (old token deleted, new token created)
    const res2 = await request.post('/api/v1/auth/register', {
      data: { email, display_name: displayName },
    });
    expect(res2.status()).toBe(200);
    const msg2 = await api.waitForMailpitMessage(request, email);
    const urlMatch2 = msg2.HTML.match(/href="([^"]*verify-email[^"]*)"/);
    expect(urlMatch2).toBeTruthy();
    const newVerifyUrl = urlMatch2![1];

    // Step 3: Old token should be invalid — submitting the form should show an error
    // (The verify-email page renders the password form before validating the token)
    await page.goto(oldVerifyUrl);
    await page.waitForLoadState('networkidle');
    await page.getByLabel('Password', { exact: true }).fill(password);
    await page.getByLabel('Confirm Password').fill(password);
    await page.getByRole('button', { name: 'Set password & sign in' }).click();
    // Should show an error (token was deleted by re-registration)
    await expect(page.locator('.text-red-600, .text-red-400')).toBeVisible({ timeout: 5000 });

    // Step 4: New token should work — verify and set password
    await page.goto(newVerifyUrl);
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('Set your password')).toBeVisible();
    await page.getByLabel('Password', { exact: true }).fill(password);
    await page.getByLabel('Confirm Password').fill(password);
    await page.getByRole('button', { name: 'Set password & sign in' }).click();

    // Step 5: Should be logged in
    await page.waitForURL(/\/d\/projects/, { timeout: 10000 });
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: 'Projects' }).first()).toBeVisible();
  });

  test('submitting with a fake token shows error', async ({ page }) => {
    // Use a fabricated token that doesn't exist — simulates an expired/cleaned-up token
    const fakeToken = randomUUID();

    await page.goto(`/verify-email?token=${fakeToken}`);
    await page.waitForLoadState('networkidle');

    // Fill and submit the form
    await page.getByLabel('Password', { exact: true }).fill('SecurePass123!');
    await page.getByLabel('Confirm Password').fill('SecurePass123!');
    await page.getByRole('button', { name: 'Set password & sign in' }).click();

    // Should display an error since the token doesn't exist
    await expect(page.locator('.text-red-600, .text-red-400')).toBeVisible({ timeout: 5000 });
  });
});
