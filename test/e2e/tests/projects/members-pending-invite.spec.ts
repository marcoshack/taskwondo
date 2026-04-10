import { test, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

test.describe('Pending email invites in members list', () => {

  test('non-registered email invite shows in members tab with Invited badge', async ({ page, request, testProject }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const inviteeEmail = `pending-${uniqueId}@test.local`;

    // Create email invite to a non-registered user via API
    const result = await api.createEmailInvite(request, adminToken, testProject.key, inviteeEmail, 'member');
    expect(result.direct_add).toBeFalsy();

    // Navigate to project settings members tab
    await page.goto(`/d/projects/${testProject.key}/settings?tab=users`);
    await page.waitForLoadState('networkidle');

    // The invitee email should appear in the member list
    await expect(page.getByText(inviteeEmail)).toBeVisible({ timeout: 5000 });

    // The "Invited" badge should be visible in the same row
    const inviteRow = page.locator('[class*="divide-y"] > div').filter({ hasText: inviteeEmail });
    await expect(inviteRow.getByText('Invited')).toBeVisible();

    // The role badge should also be shown
    await expect(inviteRow.getByText('Member')).toBeVisible();
  });

  test('pending email invite can be cancelled from members tab', async ({ page, request, testProject }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const inviteeEmail = `cancel-${uniqueId}@test.local`;

    // Create email invite to a non-registered user via API
    await api.createEmailInvite(request, adminToken, testProject.key, inviteeEmail, 'member');

    // Navigate to project settings members tab
    await page.goto(`/d/projects/${testProject.key}/settings?tab=users`);
    await page.waitForLoadState('networkidle');

    // Verify the invite row is present
    const inviteRow = page.locator('[class*="divide-y"] > div').filter({ hasText: inviteeEmail });
    await expect(inviteRow).toBeVisible({ timeout: 5000 });

    // Click the trash/cancel button in the invite row
    await inviteRow.locator('button').click();

    // The revoke confirmation modal should appear
    await expect(page.getByRole('heading', { name: 'Revoke Invite Link' })).toBeVisible();

    // Confirm deletion
    await page.getByRole('button', { name: 'Delete' }).click();

    // The invite row should disappear
    await expect(page.getByText(inviteeEmail)).not.toBeVisible({ timeout: 5000 });
  });

  test('invite sent via UI form shows in members tab immediately', async ({ page, testProject }) => {
    const uniqueId = randomUUID().slice(0, 8);
    const inviteeEmail = `ui-pending-${uniqueId}@test.local`;

    // Navigate to project settings members tab
    await page.goto(`/d/projects/${testProject.key}/settings?tab=users`);
    await page.waitForLoadState('networkidle');

    // Use the email invite form to send an invite
    const emailInput = page.getByPlaceholder('Invite by email address');
    await emailInput.fill(inviteeEmail);
    await page.getByRole('button', { name: /^Invite$/i }).click();

    // Confirm the invite modal
    await expect(page.getByText('Send Email Invite')).toBeVisible();
    await page.getByRole('button', { name: 'Send Invite' }).click();

    // The invited email should appear in the member list with the Invited badge
    await expect(page.getByText(inviteeEmail)).toBeVisible({ timeout: 5000 });
    const inviteRow = page.locator('[class*="divide-y"] > div').filter({ hasText: inviteeEmail });
    await expect(inviteRow.getByText('Invited')).toBeVisible();
  });
});
