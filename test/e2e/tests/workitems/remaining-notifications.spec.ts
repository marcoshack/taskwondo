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
  const email = `e2e-remaining-${uniqueId}@test.local`;
  const displayName = `Remaining Notif ${uniqueId}`;

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
 * Helper: create a user WITHOUT adding them to any project.
 */
async function createStandaloneUser(
  request: any,
): Promise<{ id: string; email: string; displayName: string; token: string }> {
  const adminToken = getAdminToken();
  const uniqueId = randomUUID().slice(0, 8);
  const email = `e2e-standalone-${uniqueId}@test.local`;
  const displayName = `Standalone ${uniqueId}`;

  const created = await api.createUser(request, adminToken, email, displayName);
  const tempLogin = await api.login(request, email, created.temporary_password);
  await api.changePassword(request, tempLogin.token, created.temporary_password, TEST_PASSWORD);
  const finalLogin = await api.login(request, email, TEST_PASSWORD);
  await api.setPreference(request, finalLogin.token, 'welcome_dismissed', true);

  return { id: finalLogin.user.id, email, displayName, token: finalLogin.token };
}

/**
 * Helper: poll Mailpit for a message matching the recipient, with timeout.
 */
async function waitForMailTo(
  request: any,
  recipientEmail: string,
  timeoutMs = 15000,
  subjectContains?: string,
): Promise<{ ID: string; Subject: string; To: { Address: string }[] }> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const messages = await api.searchMailpitMessages(request, `to:${recipientEmail}`);
    const match = subjectContains
      ? messages.find((m: any) => m.Subject.includes(subjectContains))
      : messages[0];
    if (match) return match;
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`No email received by ${recipientEmail}${subjectContains ? ` with subject containing "${subjectContains}"` : ''} within ${timeoutMs}ms`);
}

/**
 * Helper: enable specific notification preferences for a user in a project.
 */
async function setNotificationPrefs(
  request: any,
  token: string,
  projectKey: string,
  overrides: Partial<Record<string, boolean>> = {},
): Promise<void> {
  const defaults = {
    assigned_to_me: true,
    any_update_on_watched: false,
    new_item_created: false,
    comments_on_assigned: false,
    comments_on_watched: false,
    status_changes_intermediate: false,
    status_changes_final: false,
    added_to_project: false,
  };
  await api.setProjectUserSetting(request, token, projectKey, 'notifications', {
    ...defaults,
    ...overrides,
  });
}

// ===== 1. New item created notification =====

test.describe('New item created notifications', () => {
  test('project member receives email when another member creates a work item', async ({
    request,
    testUser,
    testProject,
  }) => {
    const userB = await createSecondUser(request, testProject.key, testUser.token);

    // Enable new_item_created for user B
    await setNotificationPrefs(request, userB.token, testProject.key, { new_item_created: true });

    // User A creates a work item
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'New item notification test',
      type: 'task',
    });

    const msg = await waitForMailTo(request, userB.email);

    expect(msg.Subject).toContain(testProject.key);
    expect(msg.Subject).toContain('New task created');
    expect(msg.Subject).toContain('New item notification test');

    const detail = await api.getMailpitMessage(request, msg.ID);
    expect(detail.HTML).toContain('New item notification test');
    expect(detail.HTML).toContain(`${testProject.key}-${item.item_number}`);

    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });

  test('creator does not receive their own new-item notification', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Enable new_item_created for the creator
    await setNotificationPrefs(request, testUser.token, testProject.key, { new_item_created: true });

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Self-create notification test',
      type: 'task',
    });

    await new Promise((r) => setTimeout(r, 3000));
    const messages = await api.searchMailpitMessages(request, `to:${testUser.email}`);
    const selfNotification = messages.find((m: any) =>
      m.Subject.includes(`#${item.item_number}`) && m.Subject.includes('New task created'),
    );
    expect(selfNotification).toBeUndefined();
  });

  test('member with disabled preference does not receive new-item email', async ({
    request,
    testUser,
    testProject,
  }) => {
    const userB = await createSecondUser(request, testProject.key, testUser.token);

    // Do NOT enable new_item_created (default is false)

    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Disabled pref new item test',
      type: 'task',
    });

    await new Promise((r) => setTimeout(r, 3000));
    const messages = await api.searchMailpitMessages(request, `to:${userB.email}`);
    expect(messages).toHaveLength(0);

    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });
});

// ===== 2. Comments on assigned notification =====

test.describe('Comments on assigned notifications', () => {
  test('assignee receives email when someone comments on their item', async ({
    request,
    testUser,
    testProject,
  }) => {
    const userB = await createSecondUser(request, testProject.key, testUser.token);

    // Enable comments_on_assigned for user B
    await setNotificationPrefs(request, userB.token, testProject.key, { comments_on_assigned: true });

    // Create work item assigned to user B
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Comment on assigned test',
      type: 'task',
      assignee_id: userB.id,
    });

    // User A comments on the item
    await api.addComment(request, testUser.token, testProject.key, item.item_number, 'Please review this');

    // Wait for comment notification (search by subject to skip assignment email)
    const msg = await waitForMailTo(request, userB.email, 15000, 'New comment');

    expect(msg.Subject).toContain(testProject.key);
    expect(msg.Subject).toContain(`#${item.item_number}`);
    expect(msg.Subject).toContain('New comment');

    const detail = await api.getMailpitMessage(request, msg.ID);
    expect(detail.HTML).toContain('Comment on assigned test');
    expect(detail.HTML).toContain('Please review this');

    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });

  test('assignee does not receive email for self-comment', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Create work item assigned to testUser (self)
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Self-comment test',
      type: 'task',
      assignee_id: testUser.id,
    });

    // Enable comments_on_assigned for testUser
    await setNotificationPrefs(request, testUser.token, testProject.key, { comments_on_assigned: true });

    // testUser comments on their own item
    await api.addComment(request, testUser.token, testProject.key, item.item_number, 'My own comment');

    await new Promise((r) => setTimeout(r, 3000));
    const messages = await api.searchMailpitMessages(request, `to:${testUser.email}`);
    const selfEmail = messages.find((m: any) =>
      m.Subject.includes(`#${item.item_number}`) && m.Subject.includes('New comment'),
    );
    expect(selfEmail).toBeUndefined();
  });

  test('assignee with disabled preference does not receive comment email', async ({
    request,
    testUser,
    testProject,
  }) => {
    const userB = await createSecondUser(request, testProject.key, testUser.token);

    // Do NOT enable comments_on_assigned (default is false)

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Comment pref disabled test',
      type: 'task',
      assignee_id: userB.id,
    });

    await api.addComment(request, testUser.token, testProject.key, item.item_number, 'Should not notify');

    await new Promise((r) => setTimeout(r, 3000));
    const messages = await api.searchMailpitMessages(request, `to:${userB.email}`);
    const commentEmail = messages.find((m: any) => m.Subject.includes('New comment'));
    expect(commentEmail).toBeUndefined();

    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });
});

// ===== 3. Status changes (intermediate) notification =====

test.describe('Status change (intermediate) notifications', () => {
  test('assignee receives email when status changes to in_progress', async ({
    request,
    testUser,
    testProject,
  }) => {
    const userB = await createSecondUser(request, testProject.key, testUser.token);

    // Enable status_changes_intermediate for user B
    await setNotificationPrefs(request, userB.token, testProject.key, { status_changes_intermediate: true });

    // Create work item assigned to user B
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Status intermediate test',
      type: 'task',
      assignee_id: userB.id,
    });

    // User A changes status to in_progress
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      status: 'in_progress',
    });

    // Wait for status change email specifically
    const msg = await waitForMailTo(request, userB.email, 15000, 'status changed');

    expect(msg.Subject).toContain(testProject.key);
    expect(msg.Subject).toContain(`#${item.item_number}`);
    expect(msg.Subject).toContain('status changed');

    const detail = await api.getMailpitMessage(request, msg.ID);
    expect(detail.HTML).toContain('Status intermediate test');
    expect(detail.HTML).toContain('in_progress');

    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });

  test('self-status-change does not trigger notification', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Create work item assigned to testUser
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Self-status change test',
      type: 'task',
      assignee_id: testUser.id,
    });

    await setNotificationPrefs(request, testUser.token, testProject.key, { status_changes_intermediate: true });

    // testUser changes their own item's status
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      status: 'in_progress',
    });

    await new Promise((r) => setTimeout(r, 3000));
    const messages = await api.searchMailpitMessages(request, `to:${testUser.email}`);
    const selfEmail = messages.find((m: any) =>
      m.Subject.includes(`#${item.item_number}`) && m.Subject.includes('status changed'),
    );
    expect(selfEmail).toBeUndefined();
  });
});

// ===== 4. Status changes (final) notification =====

test.describe('Status change (final) notifications', () => {
  test('assignee receives email when status changes to done', async ({
    request,
    testUser,
    testProject,
  }) => {
    const userB = await createSecondUser(request, testProject.key, testUser.token);

    // Enable status_changes_final for user B
    await setNotificationPrefs(request, userB.token, testProject.key, { status_changes_final: true });

    // Create work item assigned to user B
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Status final test',
      type: 'task',
      assignee_id: userB.id,
    });

    // Move through workflow: open → in_progress → in_review → done
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      status: 'in_progress',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      status: 'in_review',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      status: 'done',
    });

    // Wait for "done" status change email specifically
    const msg = await waitForMailTo(request, userB.email, 15000, 'done');

    expect(msg.Subject).toContain(testProject.key);
    expect(msg.Subject).toContain(`#${item.item_number}`);
    expect(msg.Subject).toContain('status changed');
    expect(msg.Subject).toContain('done');

    const detail = await api.getMailpitMessage(request, msg.ID);
    expect(detail.HTML).toContain('Status final test');

    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });

  test('assignee with only intermediate enabled does not receive done notification', async ({
    request,
    testUser,
    testProject,
  }) => {
    const userB = await createSecondUser(request, testProject.key, testUser.token);

    // Only enable intermediate, NOT final
    await setNotificationPrefs(request, userB.token, testProject.key, { status_changes_intermediate: true, status_changes_final: false });

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Final disabled test',
      type: 'task',
      assignee_id: userB.id,
    });

    // Move to in_progress (should send) then in_review → done (should NOT send final)
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      status: 'in_progress',
    });

    // Wait for intermediate email
    await waitForMailTo(request, userB.email, 15000, 'status changed');

    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      status: 'in_review',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      status: 'done',
    });

    await new Promise((r) => setTimeout(r, 3000));
    const messages = await api.searchMailpitMessages(request, `to:${userB.email}`);
    const doneEmail = messages.find((m: any) => m.Subject.includes('done'));
    expect(doneEmail).toBeUndefined();

    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });
});

// ===== 5. Added to project notification =====

test.describe('Added to project notifications', () => {
  test('user with enabled preference receives email when re-added to a project', async ({
    request,
    testUser,
    testProject,
  }) => {
    const adminToken = getAdminToken();

    // Create user and add to project so they can set prefs
    const userB = await createStandaloneUser(request);
    await api.addMember(request, testUser.token, testProject.key, userB.id, 'member');
    await new Promise((r) => setTimeout(r, 1000));

    // Enable added_to_project preference
    await setNotificationPrefs(request, userB.token, testProject.key, { added_to_project: true });

    // Remove user B
    await api.removeMember(request, testUser.token, testProject.key, userB.id);

    // Re-add user B — should get email
    await api.addMember(request, testUser.token, testProject.key, userB.id, 'member');

    const msg = await waitForMailTo(request, userB.email, 15000, 'added');

    expect(msg.Subject).toContain(testProject.name);
    expect(msg.Subject).toContain('added to project');

    const detail = await api.getMailpitMessage(request, msg.ID);
    expect(detail.HTML).toContain(testProject.name);
    expect(detail.HTML).toContain(testProject.key);
    expect(detail.HTML).toContain('member');

    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });

  test('user without preference enabled does not receive added-to-project email', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Create standalone user — default is false (opt-in), so no email expected
    const userB = await createStandaloneUser(request);

    // Add user B to the project
    await api.addMember(request, testUser.token, testProject.key, userB.id, 'member');

    await new Promise((r) => setTimeout(r, 3000));
    const messages = await api.searchMailpitMessages(request, `to:${userB.email}`);
    expect(messages).toHaveLength(0);

    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });
});
