import { test, expect } from '../../lib/fixtures';

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

test.describe('Keyboard shortcuts', () => {
  test('g then i navigates to inbox from project items', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Wait for page content to be ready
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('g');
    await page.keyboard.press('i');

    await expect(page).toHaveURL(/\/user\/inbox/, { timeout: 5000 });
  });

  test('g then i navigates to inbox from project list', async ({ page }) => {
    await page.goto('/d/projects');
    await dismissWelcomeModal(page);

    // Wait for page content to be ready
    await expect(page.getByRole('heading', { name: /projects/i })).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('g');
    await page.keyboard.press('i');

    await expect(page).toHaveURL(/\/user\/inbox/, { timeout: 5000 });
  });

  test('g then p opens project switcher', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('g');
    await page.keyboard.press('p');

    // Project switcher modal should open
    await expect(page.getByPlaceholder(/search.*project/i)).toBeVisible({ timeout: 5000 });
  });

  test('? opens keyboard shortcuts modal', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('?');

    // Shortcuts modal should open
    await expect(page.getByRole('heading', { name: 'Keyboard Shortcuts' })).toBeVisible({ timeout: 5000 });
  });
});
