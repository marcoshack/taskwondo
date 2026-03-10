import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

test.describe('Board View Scroll Chevrons', () => {
  test('shows chevrons when columns overflow and scrolls on click', async ({
    request,
    testUser,
    testProject,
    page,
  }, testInfo) => {
    // Create a work item so the board has content
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Chevron test item',
      type: 'task',
    });

    // Use a narrow viewport so not all columns fit
    await page.setViewportSize({ width: 800, height: 600 });

    // Navigate to board view
    await page.goto(`/d/projects/${testProject.key}/items?view=board`);
    await page.waitForLoadState('networkidle');

    // Click the Board toggle if not already active
    const boardBtn = page.getByRole('button', { name: /Board/i });
    await boardBtn.click();

    // Wait for board columns to render (statuses load via chained API calls)
    await page.waitForFunction(() => {
      const el = document.querySelector('[class*="overflow-x-auto"]');
      return el && el.scrollWidth > el.clientWidth;
    }, null, { timeout: 15000 });

    // The board scroll container should exist
    const scrollContainer = page.locator('[class*="overflow-x-auto"]').first();
    await expect(scrollContainer).toBeVisible();

    // Right chevron should be visible (columns overflow to the right)
    const rightChevron = page.getByRole('button', { name: /Scroll to next status column/i });
    await expect(rightChevron).toBeVisible({ timeout: 10000 });
    await testInfo.attach('01-right-chevron-visible', {
      body: await page.screenshot(),
      contentType: 'image/png',
    });

    // Left chevron should NOT be visible (we're at the start)
    const leftChevron = page.getByRole('button', { name: /Scroll to previous status column/i });
    await expect(leftChevron).not.toBeVisible();

    // Click right chevron to scroll
    await rightChevron.click();
    await page.waitForTimeout(600); // wait for smooth scroll

    // After scrolling right, left chevron should now appear
    await expect(leftChevron).toBeVisible({ timeout: 3000 });
    await testInfo.attach('02-after-scroll-right', {
      body: await page.screenshot(),
      contentType: 'image/png',
    });

    // Click left chevron to scroll back
    await leftChevron.click();
    await page.waitForTimeout(600);

    // Left chevron should be hidden again (back at start)
    await expect(leftChevron).not.toBeVisible({ timeout: 3000 });
    await testInfo.attach('03-after-scroll-left', {
      body: await page.screenshot(),
      contentType: 'image/png',
    });
  });
});
