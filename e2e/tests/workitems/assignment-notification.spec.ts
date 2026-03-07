import { test, expect, getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

const TEST_PASSWORD = 'TestPass123!';

test.describe.configure({ mode: 'serial' });

/**
 * Helper: provision a second user (user B), add them to the project, and return their info.
 */
async function createSecondUser(
  request: any,
  projectKey: string,
  ownerToken: string,
): Promise<{ id: string; email: string; displayName: string; token: string }> {
  const adminToken = getAdminToken();
  const uniqueId = randomUUID().slice(0, 8);
  const email = `e2e-notif-${uniqueId}@test.local`;
  const displayName = `Notif User ${uniqueId}`;

  const created = await api.createUser(request, adminToken, email, displayName);
  const tempLogin = await api.login(request, email, created.temporary_password);
  await api.changePassword(request, tempLogin.token, created.temporary_password, TEST_PASSWORD);
  const finalLogin = await api.login(request, email, TEST_PASSWORD);
  await api.setPreference(request, finalLogin.token, 'welcome_dismissed', true);

  // Add user B as a member of the project
  await api.addMember(request, ownerToken, projectKey, finalLogin.user.id, 'member');

  return { id: finalLogin.user.id, email, displayName, token: finalLogin.token };
}

/**
 * Helper: poll Mailpit for a message matching the recipient, with timeout.
 */
async function waitForMailTo(
  request: any,
  recipientEmail: string,
  timeoutMs = 15000,
): Promise<{ ID: string; Subject: string; To: { Address: string }[] }> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const messages = await api.searchMailpitMessages(request, `to:${recipientEmail}`);
    if (messages.length > 0) return messages[0];
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`No email received by ${recipientEmail} within ${timeoutMs}ms`);
}

test.describe('Assignment email notifications', () => {
  test('user A assigns work item to user B on creation — user B receives email', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Create user B and add to project
    const userB = await createSecondUser(request, testProject.key, testUser.token);

    // Create work item assigned to user B
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Assignment on creation test',
      type: 'task',
      assignee_id: userB.id,
    });

    // Wait for notification email
    const msg = await waitForMailTo(request, userB.email);

    // Verify email subject and content
    expect(msg.Subject).toContain(testProject.key);
    expect(msg.Subject).toContain(`#${item.item_number}`);
    expect(msg.Subject).toContain('assigned to you');

    // Verify full email body
    const detail = await api.getMailpitMessage(request, msg.ID);
    expect(detail.HTML).toContain('Assignment on creation test');
    expect(detail.HTML).toContain(testProject.key);
    expect(detail.HTML).toContain(`/projects/${testProject.key}/items/${item.item_number}`);

    // Cleanup
    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });

  test('user A edits work item to assign to user B — user B receives email', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Create user B and add to project
    const userB = await createSecondUser(request, testProject.key, testUser.token);

    // Create work item with no assignee
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Assignment on edit test',
      type: 'task',
    });

    // Verify no email was sent yet
    const earlyMessages = await api.searchMailpitMessages(request, `to:${userB.email}`);
    expect(earlyMessages).toHaveLength(0);

    // Now assign user B via update
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      assignee_id: userB.id,
    });

    // Wait for notification email
    const msg = await waitForMailTo(request, userB.email);

    // Verify email
    expect(msg.Subject).toContain(testProject.key);
    expect(msg.Subject).toContain(`#${item.item_number}`);
    expect(msg.Subject).toContain('assigned to you');

    const detail = await api.getMailpitMessage(request, msg.ID);
    expect(detail.HTML).toContain('Assignment on edit test');
    expect(detail.HTML).toContain(`/projects/${testProject.key}/items/${item.item_number}`);

    // Cleanup
    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });
});
