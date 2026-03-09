import { test, expect } from '../../lib/fixtures';

async function waitForPageReady(page: import('@playwright/test').Page) {
  await page.waitForLoadState('domcontentloaded');
  await expect(page.locator('nav.hidden.sm\\:block')).toBeVisible({ timeout: 15000 });
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 1000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

test.describe('Namespace URL routing', () => {
  test('root redirects to /d/projects', async ({ page }) => {
    await page.goto('/');
    await waitForPageReady(page);
    await expect(page).toHaveURL(/\/d\/projects$/, { timeout: 5000 });
  });

  test('default namespace uses /d/ prefix for project list', async ({ page }) => {
    await page.goto('/d/projects');
    await waitForPageReady(page);
    await expect(page).toHaveURL(/\/d\/projects$/);

    // Should render the project list page
    await expect(page.getByRole('heading', { name: /projects/i })).toBeVisible({ timeout: 5000 });
  });

  test('unknown namespace sub-path stays under namespace', async ({ page }) => {
    // Paths like /d/something that match /:namespace but have no valid sub-route
    await page.goto('/d/some-nonexistent-path');
    await page.waitForLoadState('domcontentloaded');
    // Should still be under /d/ namespace (NamespaceGuard renders, but no child matches)
    expect(page.url()).toMatch(/\/d\//);
  });

  test('project pages use namespace prefix in URL', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForPageReady(page);
    await expect(page).toHaveURL(new RegExp(`/d/projects/${testProject.key}/items`));

    // Project name should appear in the top bar
    const topBar = page.locator('nav').first();
    await expect(topBar.getByText(testProject.name)).toBeVisible({ timeout: 5000 });
  });

  test('sidebar Inbox link uses /user/ path', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await waitForPageReady(page);

    const sidebar = page.locator('nav.hidden.sm\\:block');
    await sidebar.getByText('Inbox').click();
    await expect(page).toHaveURL(/\/user\/inbox/, { timeout: 5000 });
  });

  test('preferences page does not use namespace prefix', async ({ page }) => {
    await page.goto('/preferences');
    await waitForPageReady(page);
    await expect(page).toHaveURL(/\/preferences/);
    // URL should NOT contain /d/preferences
    expect(page.url()).not.toMatch(/\/d\/preferences/);
  });

  test('admin page does not use namespace prefix', async ({ page }) => {
    // Navigate to admin; non-admin users will be redirected, so we just
    // check the URL pattern stays without namespace prefix.
    await page.goto('/admin');
    // Either loads admin or redirects — but the path should not gain a namespace segment
    await page.waitForLoadState('domcontentloaded');
    expect(page.url()).not.toMatch(/\/d\/admin/);
  });
});
