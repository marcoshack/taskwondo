import { test, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

test.describe('Email-based Project Invites', () => {

  test('inviting an existing user by email adds them directly to the project', async ({ request, testProject }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const email = `exist-${uniqueId}@test.local`;
    const displayName = `Existing User ${uniqueId}`;

    // Create a user that already exists
    const created = await api.createUser(request, adminToken, email, displayName);
    const tempLogin = await api.login(request, email, created.temporary_password);
    await api.changePassword(request, tempLogin.token, created.temporary_password, 'TestPass123!');

    // Invite by email — should add directly since user exists
    const result = await api.createEmailInvite(request, adminToken, testProject.key, email, 'member');
    expect(result.direct_add).toBe(true);

    // Verify the user is now a member
    const members = await api.listMembers(request, adminToken, testProject.key);
    const found = members.find((m: any) => m.email === email);
    expect(found).toBeTruthy();
    expect(found!.role).toBe('member');
  });

  test('inviting a non-existing user by email creates an invite with invitee_email', async ({ request, testProject }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const nonExistentEmail = `nouser-${uniqueId}@test.local`;

    // Invite by email — should create an invite since user doesn't exist
    const result = await api.createEmailInvite(request, adminToken, testProject.key, nonExistentEmail, 'member');
    expect(result.direct_add).toBeFalsy();
    expect(result.code).toBeTruthy();

    // Verify the invite appears in the invite list with the invitee_email
    const invites = await api.listInvites(request, adminToken, testProject.key);
    const found = invites.find((inv: any) => inv.code === result.code);
    expect(found).toBeTruthy();
    expect(found!.invitee_email).toBe(nonExistentEmail);
    expect(found!.max_uses).toBe(1);
  });

  test('email invite sends a notification email to the invitee', async ({ request, testProject }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const inviteeEmail = `invited-${uniqueId}@e2e.local`;

    // Clear any existing messages
    await api.deleteMailpitMessages(request);

    // Create email invite
    const result = await api.createEmailInvite(request, adminToken, testProject.key, inviteeEmail, 'member');
    expect(result.code).toBeTruthy();

    // Wait for the invite email to arrive in Mailpit
    const msg = await api.waitForMailpitMessage(request, inviteeEmail, { timeoutMs: 10000 });
    expect(msg.Subject).toContain(testProject.name);
    expect(msg.HTML).toContain(result.code!);
    expect(msg.HTML).toContain('Accept Invite');
  });

  test('inviting an already-member user by email returns an error', async ({ request, testProject, testUser }) => {
    const adminToken = getAdminToken();

    // testUser is already a member (owner) of testProject
    try {
      await api.createEmailInvite(request, adminToken, testProject.key, testUser.email, 'member');
      expect(true).toBe(false); // Should not reach here
    } catch (err: any) {
      expect(err.message).toContain('already a member');
    }
  });

  test('email invite shows invitee email in the settings page invite list', async ({ page, request, testProject }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const inviteeEmail = `ui-${uniqueId}@test.local`;

    // Create email invite via API
    await api.createEmailInvite(request, adminToken, testProject.key, inviteeEmail, 'member');

    // Navigate to project settings
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await page.waitForLoadState('networkidle');

    // The invitee email should appear in the invite list
    await expect(page.getByText(inviteeEmail)).toBeVisible({ timeout: 5000 });
  });

  test('invite by email form works in the UI', async ({ page, testProject }) => {
    // Navigate to project settings
    await page.goto(`/d/projects/${testProject.key}/settings`);
    await page.waitForLoadState('networkidle');

    // Find the email invite input
    const emailInput = page.getByPlaceholder('Invite by email address');
    await expect(emailInput).toBeVisible();

    // Type an email address
    const uniqueId = randomUUID().slice(0, 8);
    const email = `ui-invite-${uniqueId}@test.local`;
    await emailInput.fill(email);

    // Click Invite button
    await page.getByRole('button', { name: /^Invite$/i }).click();

    // Confirmation modal should appear
    await expect(page.getByText('Send Email Invite')).toBeVisible();
    await expect(page.getByText(email)).toBeVisible();

    // Click Send Invite
    await page.getByRole('button', { name: 'Send Invite' }).click();

    // Should see success checkmark
    await expect(page.locator('.text-green-500').first()).toBeVisible({ timeout: 5000 });
  });
});
