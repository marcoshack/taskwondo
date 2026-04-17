import { test, expect, getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

test.describe.configure({ mode: 'serial' });

function uniqueNamespaceSlug(): string {
  return `ns-assign-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 5)}`;
}

function uniqueProjectKey(): string {
  return `P${randomUUID().slice(0, 4).toUpperCase()}`;
}

async function waitForMailTo(
  request: any,
  recipientEmail: string,
  timeoutMs = 15000,
): Promise<{ ID: string }> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const messages = await api.searchMailpitMessages(request, `to:${recipientEmail}`);
    if (messages.length > 0) return messages[0];
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`No email received by ${recipientEmail} within ${timeoutMs}ms`);
}

// TF-346 regression: notification email links must carry the project's actual
// namespace segment, not the hardcoded default ("d"). This file lives in the
// `namespace` Playwright project so it shares the serialised namespaces_enabled
// setting with the rest of the namespace suite.
test.describe('Assignment email notifications — non-default namespace (TF-346)', () => {
  test('assignment email for a project in a custom namespace links to /{namespace}/projects/...', async ({
    request,
  }) => {
    const adminToken = getAdminToken();
    await api.enableNamespaces(request, adminToken);

    const nsSlug = uniqueNamespaceSlug();
    const projectKey = uniqueProjectKey();

    await api.createNamespace(request, adminToken, nsSlug, `Notif NS ${nsSlug}`);

    try {
      await api.createProject(
        request,
        adminToken,
        projectKey,
        `Notif NS Project ${projectKey}`,
        nsSlug,
      );

      // Create the assignee and add them as a project member scoped to the namespace.
      const uniqueId = randomUUID().slice(0, 8);
      const assigneeEmail = `e2e-notif-ns-${uniqueId}@test.local`;
      const assigneeDisplay = `NS Notif User ${uniqueId}`;
      const created = await api.createUser(request, adminToken, assigneeEmail, assigneeDisplay);
      const assigneeId = created.user.id;

      try {
        await api.addMemberInNamespace(request, adminToken, projectKey, assigneeId, 'member', nsSlug);

        const item = await api.createWorkItem(
          request,
          adminToken,
          projectKey,
          {
            title: 'Assignment in custom namespace',
            type: 'task',
            assignee_id: assigneeId,
          },
          nsSlug,
        );

        const msg = await waitForMailTo(request, assigneeEmail);
        const detail = await api.getMailpitMessage(request, msg.ID);

        const expectedPath = `/${nsSlug}/projects/${projectKey}/items/${item.item_number}`;
        const defaultPath = `/d/projects/${projectKey}/items/${item.item_number}`;

        expect(detail.HTML).toContain(expectedPath);
        expect(detail.HTML).not.toContain(defaultPath);
      } finally {
        await api.deactivateUser(request, adminToken, assigneeId).catch(() => {});
      }
    } finally {
      await api.deleteNamespace(request, adminToken, nsSlug).catch(() => {});
    }
  });
});
