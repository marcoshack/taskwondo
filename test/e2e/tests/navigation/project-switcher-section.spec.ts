import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';
import { switchProject } from '../../lib/palette';

/**
 * The project switcher keeps the user on the equivalent page of the target
 * project (TF-412): switching from project A's work items list lands on
 * project B's work items list, not on B's overview.
 */
test.describe('Project switcher — section preservation', () => {
  // Creates a second project owned by the same test user, so the switcher has
  // something to switch to.
  async function createSecondProject(request: import('@playwright/test').APIRequestContext, token: string) {
    for (let attempt = 0; attempt < 3; attempt++) {
      const suffix = randomUUID().slice(0, 4).toUpperCase();
      try {
        return await api.createProject(request, token, `F${suffix}`, `E2E Target ${suffix}`);
      } catch (err: any) {
        if (attempt === 2 || !err.message?.includes('already in use')) throw err;
      }
    }
    throw new Error('unreachable');
  }

  // `g p` was retired with the command palette (TF-431); the switcher now opens
  // from the nav project badge.
  async function switchTo(page: import('@playwright/test').Page, projectKey: string) {
    await switchProject(page, projectKey);
  }

  test('switching from the work items list lands on the target items list', async ({ page, request, testUser, testProject }) => {
    const target = await createSecondProject(request, testUser.token);

    await page.goto(`/d/projects/${testProject.key}/items`);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await switchTo(page, target.key);

    await expect(page).toHaveURL(new RegExp(`/d/projects/${target.key}/items$`), { timeout: 5000 });
  });

  test('switching from a work item detail page lands on the target items list', async ({ page, request, testUser, testProject }) => {
    const target = await createSecondProject(request, testUser.token);
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Switcher source item',
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText('Switcher source item')).toBeVisible({ timeout: 5000 });

    await switchTo(page, target.key);

    await expect(page).toHaveURL(new RegExp(`/d/projects/${target.key}/items$`), { timeout: 5000 });
  });

  test('switching from milestones lands on the target milestones page', async ({ page, request, testUser, testProject }) => {
    const target = await createSecondProject(request, testUser.token);

    await page.goto(`/d/projects/${testProject.key}/milestones`);
    await expect(page.getByRole('heading', { name: /milestones/i })).toBeVisible({ timeout: 5000 });

    await switchTo(page, target.key);

    await expect(page).toHaveURL(new RegExp(`/d/projects/${target.key}/milestones$`), { timeout: 5000 });
  });

  test('switching from the overview lands on the target overview', async ({ page, request, testUser, testProject }) => {
    const target = await createSecondProject(request, testUser.token);

    await page.goto(`/d/projects/${testProject.key}`);
    await expect(page.getByRole('heading', { name: 'Open Work Items' })).toBeVisible({ timeout: 5000 });

    await switchTo(page, target.key);

    await expect(page).toHaveURL(new RegExp(`/d/projects/${target.key}$`), { timeout: 5000 });
  });
});
