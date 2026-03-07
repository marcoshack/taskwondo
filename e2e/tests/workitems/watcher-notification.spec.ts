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
  const email = `e2e-watcher-${uniqueId}@test.local`;
  const displayName = `Watcher User ${uniqueId}`;

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
    const messages = await api.getMailpitMessages(request);
    const match = messages.find((m: any) =>
      m.To.some((t: any) => t.Address === recipientEmail),
    );
    if (match) return match;
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`No email received by ${recipientEmail} within ${timeoutMs}ms`);
}

/**
 * Helper: enable watcher notification preference for a user in a project.
 */
async function enableWatcherNotifications(
  request: any,
  token: string,
  projectKey: string,
): Promise<void> {
  await api.setProjectUserSetting(request, token, projectKey, 'notifications', {
    assigned_to_me: true,
    any_update_on_watched: true,
    new_item_created: false,
    comments_on_assigned: false,
    comments_on_watched: false,
    status_changes_intermediate: false,
    status_changes_final: false,
    added_to_project: false,
  });
}

test.describe('Watcher email notifications', () => {
  test('watcher receives email when a watched item status is changed by another user', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Clear mailpit
    await api.deleteMailpitMessages(request);

    // Create user B and add to project
    const userB = await createSecondUser(request, testProject.key, testUser.token);

    // Create work item
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watcher notification test - status change',
      type: 'task',
    });

    // User B watches the item
    await api.toggleWatch(request, userB.token, testProject.key, item.item_number);

    // User B enables watcher notifications for the project
    await enableWatcherNotifications(request, userB.token, testProject.key);

    // User A (testUser) changes the status
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      status: 'in_progress',
    });

    // Wait for notification email to user B
    const msg = await waitForMailTo(request, userB.email);

    // Verify email subject
    expect(msg.Subject).toContain(testProject.key);
    expect(msg.Subject).toContain(`#${item.item_number}`);
    expect(msg.Subject).toContain('updated');

    // Verify email body
    const detail = await api.getMailpitMessage(request, msg.ID);
    expect(detail.HTML).toContain('Watcher notification test - status change');
    expect(detail.HTML).toContain(testProject.key);
    expect(detail.HTML).toContain(`/projects/${testProject.key}/items/${item.item_number}`);
    expect(detail.HTML).toContain('status');

    // Cleanup
    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });

  test('actor does not receive self-notification when they change a watched item', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Clear mailpit
    await api.deleteMailpitMessages(request);

    // User A (testUser) creates a work item
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watcher self-notification test',
      type: 'task',
    });

    // User A watches their own item
    await api.toggleWatch(request, testUser.token, testProject.key, item.item_number);

    // User A enables watcher notifications
    await enableWatcherNotifications(request, testUser.token, testProject.key);

    // User A changes the status themselves
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      priority: 'high',
    });

    // Wait briefly and verify no email was sent to user A
    await new Promise((r) => setTimeout(r, 3000));
    const messages = await api.getMailpitMessages(request);
    const selfNotification = messages.find((m: any) =>
      m.To.some((t: any) => t.Address === testUser.email) &&
      m.Subject.includes(`#${item.item_number}`),
    );
    expect(selfNotification).toBeUndefined();
  });

  test('watcher with disabled preference does not receive email', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Clear mailpit
    await api.deleteMailpitMessages(request);

    // Create user B and add to project
    const userB = await createSecondUser(request, testProject.key, testUser.token);

    // Create work item
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watcher preference disabled test',
      type: 'task',
    });

    // User B watches the item but does NOT enable watcher notifications (default is false)
    await api.toggleWatch(request, userB.token, testProject.key, item.item_number);

    // User A changes the priority
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      priority: 'critical',
    });

    // Wait briefly and verify no email was sent
    await new Promise((r) => setTimeout(r, 3000));
    const messages = await api.getMailpitMessages(request);
    const watcherEmail = messages.find((m: any) =>
      m.To.some((t: any) => t.Address === userB.email) &&
      m.Subject.includes(`#${item.item_number}`),
    );
    expect(watcherEmail).toBeUndefined();

    // Cleanup
    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });

  test('watcher with enabled preference receives email after opt-in', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Clear mailpit
    await api.deleteMailpitMessages(request);

    // Create user B and add to project
    const userB = await createSecondUser(request, testProject.key, testUser.token);

    // Create work item
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Watcher preference opt-in test',
      type: 'task',
    });

    // User B watches the item and enables watcher notifications
    await api.toggleWatch(request, userB.token, testProject.key, item.item_number);
    await enableWatcherNotifications(request, userB.token, testProject.key);

    // User A changes the priority
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      priority: 'high',
    });

    // Wait for notification email to user B
    const msg = await waitForMailTo(request, userB.email);

    expect(msg.Subject).toContain(testProject.key);
    expect(msg.Subject).toContain(`#${item.item_number}`);
    expect(msg.Subject).toContain('updated');

    // Cleanup
    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });

  test('multiple watchers receive notifications when a third user changes the item', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Clear mailpit
    await api.deleteMailpitMessages(request);

    // Create user B and user C
    const userB = await createSecondUser(request, testProject.key, testUser.token);
    const userC = await createSecondUser(request, testProject.key, testUser.token);

    // Create work item
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Multi-watcher notification test',
      type: 'task',
    });

    // Both users watch the item and enable notifications
    await api.toggleWatch(request, userB.token, testProject.key, item.item_number);
    await enableWatcherNotifications(request, userB.token, testProject.key);
    await api.toggleWatch(request, userC.token, testProject.key, item.item_number);
    await enableWatcherNotifications(request, userC.token, testProject.key);

    // User A changes the status
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      status: 'in_progress',
    });

    // Wait for notification emails to both users
    const msgB = await waitForMailTo(request, userB.email);
    const msgC = await waitForMailTo(request, userC.email);

    expect(msgB.Subject).toContain(`#${item.item_number}`);
    expect(msgC.Subject).toContain(`#${item.item_number}`);

    // Cleanup
    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
    await api.deactivateUser(request, adminToken, userC.id).catch(() => {});
  });

  test('watcher receives email when a comment is added by another user', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Clear mailpit
    await api.deleteMailpitMessages(request);

    // Create user B and add to project
    const userB = await createSecondUser(request, testProject.key, testUser.token);

    // Create work item
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Comment notification test',
      type: 'task',
    });

    // User B watches the item and enables comment watcher notifications
    await api.toggleWatch(request, userB.token, testProject.key, item.item_number);
    await api.setProjectUserSetting(request, userB.token, testProject.key, 'notifications', {
      assigned_to_me: true,
      any_update_on_watched: false,
      new_item_created: false,
      comments_on_assigned: false,
      comments_on_watched: true,
      status_changes_intermediate: false,
      status_changes_final: false,
      added_to_project: false,
    });

    // User A adds a comment
    await api.addComment(request, testUser.token, testProject.key, item.item_number, 'This is a test comment');

    // Wait for notification email to user B
    const msg = await waitForMailTo(request, userB.email);

    expect(msg.Subject).toContain(testProject.key);
    expect(msg.Subject).toContain(`#${item.item_number}`);
    expect(msg.Subject).toContain('updated');

    // Verify email body contains the comment preview
    const detail = await api.getMailpitMessage(request, msg.ID);
    expect(detail.HTML).toContain('Comment notification test');
    expect(detail.HTML).toContain(`/projects/${testProject.key}/items/${item.item_number}`);

    // Cleanup
    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });

  test('watcher notification shows assignee display name instead of UUID', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Clear mailpit
    await api.deleteMailpitMessages(request);

    // Create user B (will be assigned) and user C (watcher)
    const userB = await createSecondUser(request, testProject.key, testUser.token);
    const userC = await createSecondUser(request, testProject.key, testUser.token);

    // Create work item
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Assignee display name test',
      type: 'task',
    });

    // User C watches the item and enables watcher notifications
    await api.toggleWatch(request, userC.token, testProject.key, item.item_number);
    await enableWatcherNotifications(request, userC.token, testProject.key);

    // User A assigns the item to user B
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      assignee_id: userB.id,
    });

    // Wait for watcher notification email to user C
    const msg = await waitForMailTo(request, userC.email);

    // Verify email body contains the display name, NOT the UUID
    const detail = await api.getMailpitMessage(request, msg.ID);
    expect(detail.HTML).toContain(userB.displayName);
    expect(detail.HTML).not.toContain(userB.id);

    // Cleanup
    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
    await api.deactivateUser(request, adminToken, userC.id).catch(() => {});
  });

  test('comment notification not sent when CommentsOnWatched is disabled', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Clear mailpit
    await api.deleteMailpitMessages(request);

    // Create user B and add to project
    const userB = await createSecondUser(request, testProject.key, testUser.token);

    // Create work item
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Comment notification disabled test',
      type: 'task',
    });

    // User B watches the item but only enables field-change notifications (not comments)
    await api.toggleWatch(request, userB.token, testProject.key, item.item_number);
    await api.setProjectUserSetting(request, userB.token, testProject.key, 'notifications', {
      assigned_to_me: true,
      any_update_on_watched: true,
      new_item_created: false,
      comments_on_assigned: false,
      comments_on_watched: false,
      status_changes_intermediate: false,
      status_changes_final: false,
      added_to_project: false,
    });

    // User A adds a comment
    await api.addComment(request, testUser.token, testProject.key, item.item_number, 'This comment should not trigger an email');

    // Wait briefly and verify no email was sent
    await new Promise((r) => setTimeout(r, 3000));
    const messages = await api.getMailpitMessages(request);
    const commentEmail = messages.find((m: any) =>
      m.To.some((t: any) => t.Address === userB.email) &&
      m.Subject.includes(`#${item.item_number}`),
    );
    expect(commentEmail).toBeUndefined();

    // Cleanup
    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });
});
