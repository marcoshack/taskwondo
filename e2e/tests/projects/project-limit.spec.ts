import { test, expect, getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

async function attach(page: any, testInfo: any, name: string) {
  const screenshot = await page.screenshot();
  await testInfo.attach(name, { body: screenshot, contentType: 'image/png' });
}

async function dismissWelcomeModal(page: any) {
  const heading = page.getByRole('heading', { name: 'Welcome' });
  if (await heading.isVisible({ timeout: 1000 }).catch(() => false)) {
    const checkbox = page.getByRole('checkbox', { name: "Don't show this again" });
    if (await checkbox.isVisible({ timeout: 500 }).catch(() => false)) {
      await checkbox.check();
    }
    await page.keyboard.press('Escape');
    await heading.waitFor({ state: 'hidden', timeout: 2000 }).catch(() => {});
  }
}

const PROJECT_LIMIT = 3;

test.describe('Project Limit', () => {
  test('prevents creating projects beyond the user limit', async ({ request, testUser, page }, testInfo) => {
    const adminToken = getAdminToken();

    // Set the user's project limit
    await api.setMaxProjects(request, adminToken, testUser.id, PROJECT_LIMIT);

    // Go to projects page and open the New Project modal
    await page.goto('/projects');
    await dismissWelcomeModal(page);
    await page.getByRole('button', { name: 'New Project' }).click();

    // Verify the counter shows 0/3
    await expect(page.getByText(`0/${PROJECT_LIMIT}`)).toBeVisible();
    await attach(page, testInfo, '01-modal-empty');

    // Close the modal
    await page.keyboard.press('Escape');

    // Create projects via API up to the limit
    for (let i = 0; i < PROJECT_LIMIT; i++) {
      const suffix = randomUUID().slice(0, 2).toUpperCase();
      await api.createProject(request, testUser.token, `T${suffix}`, `Test Project ${i + 1}`);
    }

    // Reload and open the modal again
    await page.reload();
    await dismissWelcomeModal(page);
    await page.getByRole('button', { name: 'New Project' }).click();

    // Verify counter shows 3/3
    await expect(page.getByText(`${PROJECT_LIMIT}/${PROJECT_LIMIT}`)).toBeVisible();

    // Verify warning message is visible
    await expect(page.getByText(/limit reached/i)).toBeVisible();

    // Verify form is disabled
    await expect(page.getByLabel(/name/i).first()).toBeDisabled();
    await expect(page.getByRole('button', { name: /create/i })).toBeDisabled();

    await attach(page, testInfo, '02-modal-limit-reached');
  });
});
