import { test, expect, getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

const TEST_PASSWORD = 'TestPass123!';

test.describe.configure({ mode: 'serial' });

/**
 * Helper: provision a second user, add them to the project, and return their info.
 */
async function createSecondUser(
  request: any,
  projectKey: string,
  ownerToken: string,
): Promise<{ id: string; email: string; displayName: string; token: string }> {
  const adminToken = getAdminToken();
  const uniqueId = randomUUID().slice(0, 8);
  const email = `e2e-i18n-${uniqueId}@test.local`;
  const displayName = `I18n User ${uniqueId}`;

  const created = await api.createUser(request, adminToken, email, displayName);
  const tempLogin = await api.login(request, email, created.temporary_password);
  await api.changePassword(request, tempLogin.token, created.temporary_password, TEST_PASSWORD);
  const finalLogin = await api.login(request, email, TEST_PASSWORD);
  await api.setPreference(request, finalLogin.token, 'welcome_dismissed', true);

  // Add user as a member of the project
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

test.describe('Localized notification emails', () => {
  test('assignment email is sent in the user preferred language (Portuguese)', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Create user B and add to project
    const userB = await createSecondUser(request, testProject.key, testUser.token);

    // Set user B language preference to Portuguese
    await api.setPreference(request, userB.token, 'language', 'pt');

    // Create work item assigned to user B
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Localized assignment test',
      type: 'task',
      assignee_id: userB.id,
    });

    // Wait for notification email
    const msg = await waitForMailTo(request, userB.email);

    // Subject should be in Portuguese
    expect(msg.Subject).toContain('atribuído a você');
    expect(msg.Subject).toContain(testProject.key);
    expect(msg.Subject).toContain(`#${item.item_number}`);

    // Body should contain Portuguese CTA and footer
    const detail = await api.getMailpitMessage(request, msg.ID);
    expect(detail.HTML).toContain('Ver Item de Trabalho');
    expect(detail.HTML).toContain('Localized assignment test');
    expect(detail.HTML).toContain(`/projects/${testProject.key}/items/${item.item_number}`);

    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });

  test('assignment email defaults to English when no language preference is set', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Create user B without setting language preference
    const userB = await createSecondUser(request, testProject.key, testUser.token);

    // Create work item assigned to user B
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Default language test',
      type: 'task',
      assignee_id: userB.id,
    });

    // Wait for notification email
    const msg = await waitForMailTo(request, userB.email);

    // Subject should be in English (default)
    expect(msg.Subject).toContain('assigned to you');
    expect(msg.Subject).toContain(testProject.key);
    expect(msg.Subject).toContain(`#${item.item_number}`);

    // Body should contain English CTA
    const detail = await api.getMailpitMessage(request, msg.ID);
    expect(detail.HTML).toContain('View Work Item');

    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });
});
