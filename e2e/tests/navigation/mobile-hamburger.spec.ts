import { test, expect } from '../../lib/fixtures';

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

test.describe('Mobile hamburger menu', () => {
  test('shows hamburger menu on preferences page', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/preferences');
    await dismissWelcomeModal(page);

    // Hamburger button should be visible
    const hamburger = page.getByRole('button', { name: /menu/i });
    await expect(hamburger).toBeVisible();

    // Click hamburger to open AppSidebar mobile dropdown
    await hamburger.click();

    // Mobile dropdown should show main navigation items
    const dropdown = page.locator('.fixed.inset-0.z-40');
    await expect(dropdown).toBeVisible();
    await expect(dropdown.getByText('Inbox')).toBeVisible();
    await expect(dropdown.getByText('Projects')).toBeVisible();
  });

  test('shows hamburger menu on system settings page', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/admin');
    await dismissWelcomeModal(page);

    // Hamburger button should be visible
    const hamburger = page.getByRole('button', { name: /menu/i });
    await expect(hamburger).toBeVisible();

    // Click hamburger to open AppSidebar mobile dropdown
    await hamburger.click();

    // Mobile dropdown should show main navigation items
    const dropdown = page.locator('.fixed.inset-0.z-40');
    await expect(dropdown).toBeVisible();
    await expect(dropdown.getByText('Inbox')).toBeVisible();
    await expect(dropdown.getByText('Projects')).toBeVisible();
  });

  test('hamburger menu navigates from preferences to inbox', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/preferences');
    await dismissWelcomeModal(page);

    // Open hamburger and navigate to inbox
    const hamburger = page.getByRole('button', { name: /menu/i });
    await hamburger.click();

    const dropdown = page.locator('.fixed.inset-0.z-40');
    await dropdown.getByText('Inbox').click();

    await expect(page).toHaveURL(/\/user\/inbox/, { timeout: 5000 });
  });

  test('hamburger menu closes when clicking outside', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/preferences');
    await dismissWelcomeModal(page);

    // Open hamburger
    const hamburger = page.getByRole('button', { name: /menu/i });
    await hamburger.click();

    const dropdown = page.locator('.fixed.inset-0.z-40');
    await expect(dropdown).toBeVisible();

    // Click the backdrop (outside the nav menu)
    await dropdown.click({ position: { x: 10, y: 600 } });
    await expect(dropdown).not.toBeVisible();
  });

  test('shows hamburger menu on project pages', async ({ page, testProject }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Hamburger button should be visible on project pages too
    const hamburger = page.getByRole('button', { name: /menu/i });
    await expect(hamburger).toBeVisible();

    await hamburger.click();

    const dropdown = page.locator('.fixed.inset-0.z-40');
    await expect(dropdown).toBeVisible();
    await expect(dropdown.getByText('Inbox')).toBeVisible();
    await expect(dropdown.getByText('Projects')).toBeVisible();
  });

  test('shows project context navigation items in hamburger menu', async ({ page, testProject }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    const hamburger = page.getByRole('button', { name: /menu/i });
    await hamburger.click();

    const dropdown = page.locator('.fixed.inset-0.z-40');
    await expect(dropdown).toBeVisible();

    // Project context items must appear in mobile menu (TF-215 bug fix)
    await expect(dropdown.getByText('Overview')).toBeVisible();
    await expect(dropdown.getByText('Items')).toBeVisible();
    await expect(dropdown.getByText('Queues')).toBeVisible();
    await expect(dropdown.getByText('Milestones')).toBeVisible();
    await expect(dropdown.getByText('Workflows')).toBeVisible();
    await expect(dropdown.getByText('Settings')).toBeVisible();
  });

  test('mobile hamburger project nav items navigate correctly', async ({ page, testProject }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Navigate to Overview via hamburger
    const hamburger = page.getByRole('button', { name: /menu/i });
    await hamburger.click();

    const dropdown = page.locator('.fixed.inset-0.z-40');
    await dropdown.getByText('Overview').click();

    await expect(page).toHaveURL(new RegExp(`/d/projects/${testProject.key}/?$`), { timeout: 5000 });

    // Menu should close after navigation
    await expect(dropdown).not.toBeVisible();
  });
});
