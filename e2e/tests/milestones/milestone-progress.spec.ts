import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

test.describe('Milestone progress counter', () => {
  test('counts all done items as closed in milestone progress', async ({ request, testUser, testProject, page }) => {
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Progress Test',
    });

    // Create three work items and assign to milestone
    const items = [];
    for (let i = 1; i <= 3; i++) {
      const item = await api.createWorkItem(request, testUser.token, testProject.key, {
        title: `Progress item ${i}`,
        type: 'task',
      });
      await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
        milestone_id: milestone.id,
      });
      items.push(item);
    }

    // Close all three items via status transitions: open → in_progress → in_review → done
    for (const item of items) {
      await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
        status: 'in_progress',
      });
      await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
        status: 'in_review',
      });
      await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
        status: 'done',
      });
    }

    // Navigate to milestones page
    await page.goto(`/projects/${testProject.key}/milestones`);

    // Verify milestone shows 3/3 items and 100%
    await expect(page.getByText('Progress Test')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('3/3 items')).toBeVisible();
    await expect(page.getByText('100%')).toBeVisible();
  });

  test('shows correct open vs closed split in milestone progress', async ({ request, testUser, testProject, page }) => {
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Split Progress',
    });

    // Create three work items assigned to milestone
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Done item',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, item1.item_number, {
      milestone_id: milestone.id,
    });

    const item2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Open item',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, item2.item_number, {
      milestone_id: milestone.id,
    });

    const item3 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Cancelled item',
      type: 'task',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, item3.item_number, {
      milestone_id: milestone.id,
    });

    // Close item1 (done: open → in_progress → in_review → done) and item3 (cancelled: open → cancelled), leave item2 open
    await api.updateWorkItem(request, testUser.token, testProject.key, item1.item_number, {
      status: 'in_progress',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, item1.item_number, {
      status: 'in_review',
    });
    await api.updateWorkItem(request, testUser.token, testProject.key, item1.item_number, {
      status: 'done',
    });

    await api.updateWorkItem(request, testUser.token, testProject.key, item3.item_number, {
      status: 'cancelled',
    });

    // Navigate to milestones page
    await page.goto(`/projects/${testProject.key}/milestones`);

    // Verify milestone shows 2/3 items (done + cancelled count as closed)
    await expect(page.getByText('Split Progress')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('2/3 items')).toBeVisible();
    await expect(page.getByText('67%')).toBeVisible();
  });
});
