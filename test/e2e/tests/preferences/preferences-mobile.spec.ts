import { test, expect } from '../../lib/fixtures';

test.describe('Preferences mobile layout', () => {
  test('top bar does not overflow on mobile viewport', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    const sections = ['profile', 'appearance', 'notifications', 'api-keys'];

    for (const section of sections) {
      await page.goto(`/preferences/${section}`);
      await page.waitForLoadState('networkidle');

      // The page should not have horizontal overflow
      const hasOverflow = await page.evaluate(() => {
        return document.documentElement.scrollWidth > document.documentElement.clientWidth;
      });
      expect(hasOverflow).toBe(false);

      // The top bar nav element should fit within the viewport
      const nav = page.locator('nav').first();
      const navBox = await nav.boundingBox();
      expect(navBox).not.toBeNull();
      expect(navBox!.x + navBox!.width).toBeLessThanOrEqual(375 + 1);

      // The user avatar in the top bar should be visible (not clipped)
      const avatar = nav.locator('img, [class*="Avatar"], [class*="avatar"]').first();
      if (await avatar.isVisible()) {
        const avatarBox = await avatar.boundingBox();
        expect(avatarBox).not.toBeNull();
        expect(avatarBox!.x + avatarBox!.width).toBeLessThanOrEqual(375 + 1);
      }
    }
  });

  test('preferences mobile nav tabs are visible and functional', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/preferences/profile');
    await page.waitForLoadState('networkidle');

    // Mobile nav should be visible (sm:hidden means visible on mobile)
    const mobileNav = page.locator('nav.flex').filter({ has: page.getByText('Profile') });
    await expect(mobileNav).toBeVisible();

    // All 4 tabs should be visible
    await expect(mobileNav.getByText('Profile')).toBeVisible();
    await expect(mobileNav.getByText('Appearance')).toBeVisible();
    await expect(mobileNav.getByText('Notifications')).toBeVisible();
    await expect(mobileNav.getByText('API Keys')).toBeVisible();

    // Clicking Appearance tab should navigate
    await mobileNav.getByText('Appearance').click();
    await expect(page).toHaveURL(/\/preferences\/appearance/);
  });
});
