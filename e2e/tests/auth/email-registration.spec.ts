import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
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

// SMTP and auth settings are configured by chromium.setup.ts (runs before this project)
test.describe.configure({ mode: 'serial' });

test.describe('Email Registration', () => {
  test('full registration flow: register → verify email → set password → logged in', async ({ page, request }) => {
    const uniqueId = randomUUID().slice(0, 8);
    const email = `reg-${uniqueId}@e2e.local`;
    const displayName = `Reg User ${uniqueId}`;
    const password = 'SecurePass123!';

    // Step 1: Navigate to login page and click "Create an account"
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('Create an account')).toBeVisible();
    await page.getByText('Create an account').click();
    await page.waitForURL(/\/register/);

    // Step 2: Fill in the registration form
    await page.getByLabel('Display Name').fill(displayName);
    await page.getByLabel('Email').fill(email);
    await page.getByRole('button', { name: 'Create account' }).click();

    // Step 3: Verify "check your email" message
    await expect(page.getByText('Check your email')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(email)).toBeVisible();

    // Step 4: Retrieve the verification email from Mailpit (search by recipient)
    const msg = await api.waitForMailpitMessage(request, email);
    expect(msg.Subject).toContain('Verify');

    // Step 5: Extract the verification URL from the email HTML
    const urlMatch = msg.HTML.match(/href="([^"]*verify-email[^"]*)"/);
    expect(urlMatch).toBeTruthy();
    const verifyUrl = urlMatch![1];

    // Step 6: Navigate to the verify URL
    await page.goto(verifyUrl);
    await page.waitForLoadState('networkidle');

    // Step 7: Set password
    await expect(page.getByText('Set your password')).toBeVisible();
    await page.getByLabel('Password', { exact: true }).fill(password);
    await page.getByLabel('Confirm Password').fill(password);
    await page.getByRole('button', { name: 'Set password & sign in' }).click();

    // Step 8: Should be redirected to projects (logged in)
    await page.waitForURL(/\/projects/, { timeout: 10000 });
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: 'Projects' }).first()).toBeVisible();
  });

  test('registration shows error for duplicate email', async ({ page, request }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const email = `dup-${uniqueId}@e2e.local`;

    // Create user via admin API first
    await api.createUser(request, adminToken, email, `Dup User ${uniqueId}`);

    // Try to register with same email
    await page.goto('/register');
    await page.waitForLoadState('networkidle');
    await page.getByLabel('Display Name').fill('Duplicate User');
    await page.getByLabel('Email').fill(email);
    await page.getByRole('button', { name: 'Create account' }).click();

    // Should show an error
    await expect(page.locator('.text-red-600, .text-red-400')).toBeVisible({ timeout: 5000 });
  });

  test('registration API rejects malformed emails', async ({ request }) => {
    const malformedEmails = ['@invalid', 'user@', 'no-at-sign', 'User <user@example.com>'];
    for (const email of malformedEmails) {
      const res = await request.post('/api/v1/auth/register', {
        data: { email, display_name: 'Bad Email User' },
      });
      expect(res.status(), `expected 400 for email "${email}"`).toBe(400);
    }
  });
});
