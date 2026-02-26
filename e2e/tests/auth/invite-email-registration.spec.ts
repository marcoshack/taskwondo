import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

// Unauthenticated context — invite + registration pages are public
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

test.describe('Invite + Email Registration', () => {
  test('user registers via email after clicking invite link and is auto-added to project', async ({ page, request }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const projectKey = `IV${uniqueId.slice(0, 3).toUpperCase()}`;
    const email = `invite-reg-${uniqueId}@e2e.local`;
    const displayName = `Invite Reg ${uniqueId}`;
    const password = 'SecurePass123!';

    // Setup: create a project and an invite link
    await api.createProject(request, adminToken, projectKey, `Invite Test ${uniqueId}`);
    const invite = await api.createInvite(request, adminToken, projectKey, 'member');

    // Clear Mailpit before test
    await api.deleteMailpitMessages(request);

    // Step 1: Navigate to the invite link (unauthenticated)
    await page.goto(`/invite/${invite.code}`);
    await page.waitForLoadState('networkidle');

    // Should see invite info with "Log in to join" button
    await expect(page.getByRole('button', { name: /log in to join/i })).toBeVisible();
    await page.getByRole('button', { name: /log in to join/i }).click();

    // Step 2: Redirected to login — click "Create an account"
    await page.waitForURL(/\/login/);
    await expect(page.getByText('Create an account')).toBeVisible();
    await page.getByText('Create an account').click();
    await page.waitForURL(/\/register/);

    // Step 3: Fill in the registration form
    await page.getByLabel('Display Name').fill(displayName);
    await page.getByLabel('Email').fill(email);
    await page.getByRole('button', { name: 'Create account' }).click();

    // Step 4: Verify "check your email" message
    await expect(page.getByText('Check your email')).toBeVisible({ timeout: 10000 });

    // Step 5: Retrieve the verification email from Mailpit
    let messages: Awaited<ReturnType<typeof api.getMailpitMessages>> = [];
    for (let i = 0; i < 10; i++) {
      messages = await api.getMailpitMessages(request);
      if (messages.length > 0) break;
      await page.waitForTimeout(500);
    }
    expect(messages.length).toBeGreaterThan(0);

    const msg = await api.getMailpitMessage(request, messages[0].ID);
    expect(msg.To[0].Address).toBe(email);

    // Step 6: Extract the verification URL from the email HTML
    const urlMatch = msg.HTML.match(/href="([^"]*verify-email[^"]*)"/);
    expect(urlMatch).toBeTruthy();
    const verifyUrl = urlMatch![1];

    // Step 7: Navigate to the verify URL and set password
    await page.goto(verifyUrl);
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('Set your password')).toBeVisible();
    await page.getByLabel('Password', { exact: true }).fill(password);
    await page.getByLabel('Confirm Password').fill(password);
    await page.getByRole('button', { name: 'Set password & sign in' }).click();

    // Step 8: Should be redirected to the invited project (not /projects)
    await page.waitForURL(new RegExp(`/projects/${projectKey}`, 'i'), { timeout: 10000 });
    await dismissWelcomeModal(page);

    // Verify we're on the project page
    await expect(page.getByText(`Invite Test ${uniqueId}`)).toBeVisible();
  });
});
