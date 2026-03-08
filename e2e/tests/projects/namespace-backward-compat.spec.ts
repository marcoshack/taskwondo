import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

test.describe('Namespace backward compatibility', () => {
  test('create and list projects without namespace context', async ({
    request,
    testUser,
    testProject,
  }) => {
    // testProject was created without any namespace param — should succeed
    expect(testProject.key).toBeTruthy();
    expect(testProject.id).toBeTruthy();

    // Creating another project should also work
    const suffix = Math.random().toString(36).slice(2, 6).toUpperCase();
    const key = `N${suffix}`;
    const project2 = await api.createProject(request, testUser.token, key, 'Namespace Compat 2');
    expect(project2.key).toBe(key);
  });

  test('create and update work items after namespace migration', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Create work item (no namespace context)
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Namespace compat test item',
      type: 'task',
      description: 'Testing backward compatibility',
    });
    expect(item.id).toBeTruthy();
    expect(item.display_id).toContain(testProject.key);

    // Update work item
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      status: 'in_progress',
      priority: 'high',
    });
  });

  test('project detail page loads after namespace migration', async ({
    request,
    testUser,
    testProject,
    page,
  }) => {
    // Create a work item so there's content to show
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'UI backward compat test',
      type: 'task',
    });

    // Navigate to the project page
    await page.goto(`/projects/${testProject.key}/items`);
    await expect(page.getByRole('table').getByText('UI backward compat test')).toBeVisible({ timeout: 15000 });
  });

  test('work item detail page loads after namespace migration', async ({
    request,
    testUser,
    testProject,
    page,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Detail page compat test',
      type: 'task',
      description: 'Verifying detail page works',
    });

    await page.goto(`/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText('Detail page compat test')).toBeVisible({ timeout: 15000 });
  });

  test('comments work after namespace migration', async ({
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Comment compat test',
      type: 'task',
    });

    // Add a comment
    const comment = await api.addComment(
      request,
      testUser.token,
      testProject.key,
      item.item_number,
      'Testing comments after namespace migration',
    );
    expect(comment.id).toBeTruthy();
    expect(comment.body).toBe('Testing comments after namespace migration');
  });

  test('project members work after namespace migration', async ({
    request,
    testUser,
    testProject,
  }) => {
    // Create a second user
    const adminToken = (await import('../../lib/fixtures')).getAdminToken();
    const uniqueId = Math.random().toString(36).slice(2, 10);
    const email = `e2e-ns-${uniqueId}@test.local`;
    const created = await api.createUser(request, adminToken, email, 'NS Compat User');
    const tempLogin = await api.login(request, email, created.temporary_password);
    await api.changePassword(request, tempLogin.token, created.temporary_password, 'TestPass123!');
    const user2Login = await api.login(request, email, 'TestPass123!');

    // Add as member to the project
    await api.addMember(request, testUser.token, testProject.key, user2Login.user.id, 'member');

    // Cleanup
    await api.deactivateUser(request, adminToken, user2Login.user.id).catch(() => {});
  });
});
