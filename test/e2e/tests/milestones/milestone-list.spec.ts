import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

test.describe('Milestones list', () => {
  test('clicking anywhere on the card opens the milestone', async ({ request, testUser, testProject, page }) => {
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'Whole Card Test',
      description: 'Everything ships here',
    });

    await page.goto(`/d/projects/${testProject.key}/milestones`);
    await expect(page.getByText('Whole Card Test')).toBeVisible({ timeout: 10000 });

    // Click the description — not the name — to prove the whole card is the target
    await page.getByText('Everything ships here').click();

    await page.waitForURL(new RegExp(`/milestones/${milestone.id}`));
    await expect(page.getByRole('heading', { name: 'Whole Card Test' })).toBeVisible({ timeout: 10000 });
  });

  test('card has no edit or delete buttons', async ({ request, testUser, testProject, page }) => {
    await api.createMilestone(request, testUser.token, testProject.key, {
      name: 'No Actions Test',
    });

    await page.goto(`/d/projects/${testProject.key}/milestones`);

    const card = page.getByRole('link').filter({ hasText: 'No Actions Test' });
    await expect(card).toBeVisible({ timeout: 10000 });

    await expect(card.locator('svg.lucide-pencil')).toHaveCount(0);
    await expect(card.locator('svg.lucide-trash-2')).toHaveCount(0);
    await expect(card.getByRole('button')).toHaveCount(0);
  });
});
