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

  test('redirects to original URL after OAuth login', async ({ testUser, testProject, page, request }) => {
    // Deep link target (the URL the user was originally trying to reach)
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'OAuth Redirect Test Item',
      type: 'task',
    });
    const targetPath = `/d/projects/${testProject.key}/items/${item.item_number}`;

    // Simulate the OAuth round-trip: LoginPage stores `next` in sessionStorage
    // before redirecting to the provider. We pre-populate it here so the
    // OAuth callback page can consume it.
    await page.addInitScript((path) => {
      window.sessionStorage.setItem('taskwondo_oauth_next', path);
    }, targetPath);

    // Stub the provider callback to issue a real test-user session, so we
    // don't need a real OAuth provider in the e2e environment.
    await page.route('**/api/v1/auth/discord/callback', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            token: testUser.token,
            user: {
              id: testUser.id,
              email: testUser.email,
              display_name: testUser.displayName,
              global_role: 'user',
              has_password: true,
            },
          },
        }),
      });
    });

    // Land on the OAuth callback page as the provider would redirect us
    await page.goto('/auth/discord/callback?code=test&state=test');

    // Should land on the original deep link, not /d/projects
    await expect(page).toHaveURL(new RegExp(targetPath.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')), { timeout: 10000 });
    await expect(page.getByText('OAuth Redirect Test Item')).toBeVisible({ timeout: 10000 });
  });
});
