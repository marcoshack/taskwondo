import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

test.describe('Time tracking — mobile layout', () => {
  test('log time form shows date picker fully visible on mobile', async ({ request, testUser, testProject, page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Mobile time layout test',
      type: 'task',
    });

    await page.goto(`/projects/${testProject.key}/items/${item.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to the Time tab
    await page.getByRole('button', { name: 'Time', exact: true }).click();
    await expect(page.getByText('No time entries yet.')).toBeVisible({ timeout: 10000 });

    // Row 1: duration input and date picker should both be visible
    const durationInput = page.getByPlaceholder('e.g. 30m, 2h');
    const dateInput = page.locator('input[type="date"]').first();
    await expect(durationInput).toBeVisible();
    await expect(dateInput).toBeVisible();

    // Date input should not be clipped — its right edge should be within the viewport
    const dateBox = await dateInput.boundingBox();
    expect(dateBox).toBeTruthy();
    expect(dateBox!.x + dateBox!.width).toBeLessThanOrEqual(375);

    // On mobile, the visible description and Log Time button are in the second row (sm:hidden div)
    // The desktop-only ones (hidden sm:block) should not be visible
    const mobileRow = page.locator('.sm\\:hidden').filter({ has: page.getByPlaceholder('What did you work on?') });
    const mobileDescription = mobileRow.getByPlaceholder('What did you work on?');
    const mobileLogTimeButton = mobileRow.getByRole('button', { name: 'Log Time' });
    await expect(mobileDescription).toBeVisible();
    await expect(mobileLogTimeButton).toBeVisible();

    // Both should be in the same row (similar y coordinate)
    const descBox = await mobileDescription.boundingBox();
    const btnBox = await mobileLogTimeButton.boundingBox();
    expect(descBox).toBeTruthy();
    expect(btnBox).toBeTruthy();
    // Same row means their vertical centers are within 20px of each other
    const descCenter = descBox!.y + descBox!.height / 2;
    const btnCenter = btnBox!.y + btnBox!.height / 2;
    expect(Math.abs(descCenter - btnCenter)).toBeLessThan(20);
  });

  test('log time entry via mobile form', async ({ request, testUser, testProject, page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Mobile log time test',
      type: 'task',
    });

    await page.goto(`/projects/${testProject.key}/items/${item.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to the Time tab
    await page.getByRole('button', { name: 'Time', exact: true }).click();
    await expect(page.getByText('No time entries yet.')).toBeVisible({ timeout: 10000 });

    // Fill in duration
    await page.getByPlaceholder('e.g. 30m, 2h').fill('1h 30m');

    // Fill in description — use the mobile-visible row (sm:hidden)
    const mobileRow = page.locator('.sm\\:hidden').filter({ has: page.getByPlaceholder('What did you work on?') });
    await mobileRow.getByPlaceholder('What did you work on?').fill('Mobile entry');

    // Click Log Time (in the mobile row)
    await mobileRow.getByRole('button', { name: 'Log Time' }).click();

    // Verify the entry appears
    const entry = page.locator('.group\\/entry').first();
    await expect(entry.getByText('1h 30m')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Mobile entry')).toBeVisible();

    // Verify via API
    const timeData = await api.listTimeEntries(request, testUser.token, testProject.key, item.item_number);
    expect(timeData.entries).toHaveLength(1);
    expect(timeData.entries[0].duration_seconds).toBe(5400);
    expect(timeData.entries[0].description).toBe('Mobile entry');
  });

  test('log time form on desktop keeps original layout', async ({ request, testUser, testProject, page }) => {
    // Desktop viewport
    await page.setViewportSize({ width: 1280, height: 800 });

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Desktop time layout test',
      type: 'task',
    });

    await page.goto(`/projects/${testProject.key}/items/${item.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to the Time tab
    await page.getByRole('button', { name: 'Time', exact: true }).click();
    await expect(page.getByText('No time entries yet.')).toBeVisible({ timeout: 10000 });

    // On desktop, duration, date, description, and Log Time button should all be in the same row
    const durationInput = page.getByPlaceholder('e.g. 30m, 2h');
    const dateInput = page.locator('input[type="date"]').first();
    // Use first() to target the desktop description input (hidden sm:block)
    const descriptionInput = page.getByPlaceholder('What did you work on?').first();
    // Use first() for the desktop Log Time button (hidden sm:block)
    const logTimeButton = page.getByRole('button', { name: 'Log Time' }).first();

    await expect(durationInput).toBeVisible();
    await expect(dateInput).toBeVisible();
    await expect(descriptionInput).toBeVisible();
    await expect(logTimeButton).toBeVisible();

    // All four should be on the same row (similar y coordinates)
    const durBox = await durationInput.boundingBox();
    const dateBox = await dateInput.boundingBox();
    const descBox = await descriptionInput.boundingBox();
    const btnBox = await logTimeButton.boundingBox();

    expect(durBox).toBeTruthy();
    expect(dateBox).toBeTruthy();
    expect(descBox).toBeTruthy();
    expect(btnBox).toBeTruthy();

    const durCenter = durBox!.y + durBox!.height / 2;
    const dateCenter = dateBox!.y + dateBox!.height / 2;
    const descCenter = descBox!.y + descBox!.height / 2;
    const btnCenter = btnBox!.y + btnBox!.height / 2;

    // All should be within 20px vertically (same row)
    expect(Math.abs(durCenter - dateCenter)).toBeLessThan(20);
    expect(Math.abs(durCenter - descCenter)).toBeLessThan(20);
    expect(Math.abs(durCenter - btnCenter)).toBeLessThan(20);
  });
});
