import { test, expect, getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';

/**
 * Finds the workflow mapped to a given item type for the project.
 * Falls back to the project's default workflow if no type-specific mapping exists.
 */
async function getTypeWorkflow(
  request: any,
  token: string,
  adminToken: string,
  projectKey: string,
  itemType: string,
): Promise<{ workflowId: string; initialStatus: string }> {
  const workflows = await api.listProjectWorkflows(request, token, projectKey);
  // Try to find type-specific workflow by matching name conventions
  // Ticket Workflow for ticket/feedback, Task Workflow for task/bug/epic
  const typeWorkflowName = itemType === 'ticket' || itemType === 'feedback' ? 'Ticket Workflow' : 'Task Workflow';
  const wf = workflows.find((w) => w.name === typeWorkflowName) ?? workflows.find((w) => w.is_default) ?? workflows[0];
  const detail = await api.getWorkflow(request, adminToken, wf.id);
  const sorted = detail.statuses.sort((a: any, b: any) => a.position - b.position);
  return { workflowId: wf.id, initialStatus: sorted[0].name };
}

test.describe('SLA sorting', () => {

  test('items with SLA sort before items without SLA when sorting by sla_target_at', async ({
    request,
    testUser,
    testProject,
  }) => {
    const adminToken = getAdminToken();

    // Get the ticket workflow (auto-mapped to Ticket Workflow with initial status "new")
    const { workflowId, initialStatus } = await getTypeWorkflow(
      request, testUser.token, adminToken, testProject.key, 'ticket',
    );
    console.log(`Ticket workflow initial status: ${initialStatus}`);

    // Create task items (no SLA will be configured for tasks)
    const noSla1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'No SLA item 1',
      type: 'task',
    });
    const noSla2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'No SLA item 2',
      type: 'task',
    });

    // Configure SLA targets for ticket type using the actual initial status
    await api.setSLATargets(request, testUser.token, testProject.key, 'ticket', workflowId, [
      { status_name: initialStatus, priority: 'medium', target_seconds: 3600, calendar_mode: '24x7' },
      { status_name: initialStatus, priority: 'high', target_seconds: 1800, calendar_mode: '24x7' },
    ]);

    // Create ticket items AFTER SLA is configured
    const withSla1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'With SLA high priority',
      type: 'ticket',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, withSla1.item_number, {
      priority: 'high',
    });

    const withSla2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'With SLA medium priority',
      type: 'ticket',
    });

    // List items sorted by sla_target_at ascending
    const result = await api.listWorkItems(request, testUser.token, testProject.key, {
      sort: 'sla_target_at',
      order: 'asc',
    });

    const ids = result.data.map((item) => item.display_id);
    const slaIdx1 = ids.indexOf(withSla1.display_id);
    const slaIdx2 = ids.indexOf(withSla2.display_id);
    const noSlaIdx1 = ids.indexOf(noSla1.display_id);
    const noSlaIdx2 = ids.indexOf(noSla2.display_id);

    // SLA items must appear before non-SLA items
    expect(slaIdx1).toBeLessThan(noSlaIdx1);
    expect(slaIdx1).toBeLessThan(noSlaIdx2);
    expect(slaIdx2).toBeLessThan(noSlaIdx1);
    expect(slaIdx2).toBeLessThan(noSlaIdx2);

    // High priority (shorter deadline) should come before medium priority
    expect(slaIdx1).toBeLessThan(slaIdx2);
  });

  test('BulkUpsertTargets backfills sla_target_at for existing items', async ({
    request,
    testUser,
    testProject,
  }) => {
    const adminToken = getAdminToken();

    // Get the ticket workflow and its initial status
    const { workflowId, initialStatus } = await getTypeWorkflow(
      request, testUser.token, adminToken, testProject.key, 'ticket',
    );

    // Create ticket items BEFORE SLA is configured
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Pre-existing ticket 1',
      type: 'ticket',
    });
    const item2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Pre-existing ticket 2',
      type: 'ticket',
    });
    // Create a task item (different type, should NOT be backfilled)
    const taskItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Pre-existing task',
      type: 'task',
    });

    // Verify sla_target_at is not set before SLA configuration
    const before = await api.listWorkItems(request, testUser.token, testProject.key, {
      sort: 'created_at',
      order: 'asc',
    });
    const ticket1Before = before.data.find((i) => i.display_id === item1.display_id);
    expect(ticket1Before!.sla_target_at).toBeFalsy();

    // NOW configure SLA targets — this should backfill existing ticket items
    await api.setSLATargets(request, testUser.token, testProject.key, 'ticket', workflowId, [
      { status_name: initialStatus, priority: 'medium', target_seconds: 7200, calendar_mode: '24x7' },
    ]);

    // Verify sla_target_at is now set for ticket items
    const after = await api.listWorkItems(request, testUser.token, testProject.key, {
      sort: 'sla_target_at',
      order: 'asc',
    });

    const ticket1After = after.data.find((i) => i.display_id === item1.display_id);
    const ticket2After = after.data.find((i) => i.display_id === item2.display_id);
    const taskAfter = after.data.find((i) => i.display_id === taskItem.display_id);

    // Ticket items should have sla_target_at backfilled
    expect(ticket1After!.sla_target_at).toBeTruthy();
    expect(ticket2After!.sla_target_at).toBeTruthy();

    // Task item should still have NULL sla_target_at (different type)
    expect(taskAfter!.sla_target_at).toBeFalsy();

    // Ticket items should sort before the task item
    const ids = after.data.map((i) => i.display_id);
    expect(ids.indexOf(ticket1After!.display_id)).toBeLessThan(ids.indexOf(taskAfter!.display_id));
  });
});
