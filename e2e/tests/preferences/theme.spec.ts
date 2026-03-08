import { test, expect } from '../../lib/fixtures';

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

test.describe('Theme Preferences', () => {
  test('switch between light and dark themes across pages', async ({ page, testProject }, testInfo) => {
    await page.goto('/');
    await dismissWelcomeModal(page);

    // --- Light theme ---
    await page.goto('/preferences/appearance');
    await page.getByRole('button', { name: /Light Always light/i }).click();
    const isLight = await page.evaluate(() => !document.documentElement.classList.contains('dark'));
    expect(isLight).toBe(true);
    await attach(page, testInfo, '01-preferences-light');

    // Projects page in light mode
    await page.getByRole('button', { name: 'Taskwondo' }).click();
    await expect(page).toHaveURL(/projects/);
    await dismissWelcomeModal(page);
    await attach(page, testInfo, '02-projects-light');

    // Items page in light mode
    await page.goto(`/d/projects/${testProject.key}/items`);
    await page.waitForLoadState('networkidle');
    await attach(page, testInfo, '03-items-light');

    // --- Dark theme ---
    await page.goto('/preferences/appearance');
    await page.getByRole('button', { name: /Dark Always dark/i }).click();
    const isDark = await page.evaluate(() => document.documentElement.classList.contains('dark'));
    expect(isDark).toBe(true);
    await attach(page, testInfo, '04-preferences-dark');

    // Projects page in dark mode
    await page.getByRole('button', { name: 'Taskwondo' }).click();
    await expect(page).toHaveURL(/projects/);
    await dismissWelcomeModal(page);
    await attach(page, testInfo, '05-projects-dark');

    // Items page in dark mode
    await page.goto(`/d/projects/${testProject.key}/items`);
    await page.waitForLoadState('networkidle');
    await attach(page, testInfo, '06-items-dark');
  });
});
