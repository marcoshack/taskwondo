import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

test.describe('Login redirect', () => {
  // Need fresh browser with no stored auth for login tests
  test.use({ storageState: { cookies: [], origins: [] } });

  test('redirects to original URL after login', async ({ testUser, testProject, page, request }) => {
    // Create a work item so we have a deep link to test
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Redirect Test Item',
      type: 'task',
    });

    const targetPath = `/d/projects/${testProject.key}/items/${item.item_number}`;

    // Navigate directly to the work item page (while not logged in)
    await page.goto(targetPath);

    // Should be redirected to login with next param
    await expect(page).toHaveURL(/\/login\?next=/, { timeout: 10000 });

    // Log in
    await page.getByLabel('Email').fill(testUser.email);
    await page.getByLabel('Password').fill(testUser.password);
    await page.getByRole('button', { name: 'Sign in', exact: true }).click();

    // Should be redirected to the original work item page, not /d/projects
    await expect(page).toHaveURL(new RegExp(targetPath.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')), { timeout: 10000 });

    // Verify the work item content is visible
    await expect(page.getByText('Redirect Test Item')).toBeVisible({ timeout: 10000 });
  });

  test('redirects to /d/projects when no next param', async ({ testUser, page }) => {
    await page.goto('/login');

    await page.getByLabel('Email').fill(testUser.email);
    await page.getByLabel('Password').fill(testUser.password);
    await page.getByRole('button', { name: 'Sign in', exact: true }).click();

    // Should land on /d/projects (default)
    await expect(page).toHaveURL(/\/d\/projects/, { timeout: 10000 });
  });
});
