import { test, expect, getAdminToken } from '../../lib/fixtures';
import { randomUUID } from 'crypto';
import * as api from '../../lib/api';

const TEST_PASSWORD = 'TestPass123!';

/** Create a second user, add to project, and return their details. */
async function createSecondUser(
  request: any,
  ownerToken: string,
  projectKey: string,
): Promise<{ id: string; email: string; displayName: string; token: string }> {
  const adminToken = getAdminToken();
  const uid = randomUUID().slice(0, 8);
  const email = `sla-retro-${uid}@e2e.local`;
  const displayName = `SLA Retro ${uid}`;
  const created = await api.createUser(request, adminToken, email, displayName);
  const tempLogin = await api.login(request, email, created.temporary_password);
  await api.changePassword(request, tempLogin.token, created.temporary_password, TEST_PASSWORD);
  const finalLogin = await api.login(request, email, TEST_PASSWORD);
  await api.addMember(request, ownerToken, projectKey, finalLogin.user.id, 'member');
  return { id: finalLogin.user.id, email, displayName, token: finalLogin.token };
}

test.describe('SLA retroactive notification prevention', () => {

  test('no SLA breach email for items breached before escalation list was created', async ({
    request,
    testUser,
    testProject,
  }) => {
    test.setTimeout(60000);

    const userB = await createSecondUser(request, testUser.token, testProject.key);

    // Get the default workflow for SLA target configuration
    const workflows = await api.listProjectWorkflows(request, testUser.token, testProject.key);
    const defaultWf = workflows.find((w) => w.is_default) ?? workflows[0];

    // Configure SLA target: 5 seconds for initial status, ticket type, medium priority
    await api.setSLATargets(request, testUser.token, testProject.key, 'ticket', defaultWf.id, [
      { status_name: 'backlog', priority: 'medium', target_seconds: 5, calendar_mode: '24x7' },
    ]);

    // Create a work item BEFORE the escalation list exists — it will breach before config
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Pre-existing breached item',
      type: 'ticket',
    });

    // Wait for the SLA to breach (5s target + buffer for monitor cycle)
    await new Promise((r) => setTimeout(r, 10000));

    // Clear mailpit to ensure a clean slate
    await api.deleteMailpitMessages(request);

    // NOW create the escalation list and mapping (after the item is already breached)
    const escList = await api.createEscalationList(request, testUser.token, testProject.key, {
      name: 'Retro Test Escalation',
      levels: [{ threshold_pct: 100, user_ids: [userB.id] }],
    });
    await api.setEscalationMapping(request, testUser.token, testProject.key, 'ticket', escList.id);

    // Wait for several SLA monitor cycles (monitor runs every 3s in E2E)
    await new Promise((r) => setTimeout(r, 12000));

    // Verify NO SLA breach email was sent (retroactive breach should be skipped)
    const messages = await api.searchMailpitMessages(request, `to:${userB.email}`);
    const slaMessages = messages.filter((m) => m.Subject.includes('SLA'));
    expect(slaMessages).toHaveLength(0);

    // Cleanup
    const adminToken = getAdminToken();
    await api.deactivateUser(request, adminToken, userB.id).catch(() => {});
  });
});
