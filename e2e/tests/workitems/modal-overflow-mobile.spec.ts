import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

test.describe('Modal header overflow on mobile', () => {
  test('file preview modal header text does not overlap action icons', async ({ request, testUser, testProject, page }) => {
    // Create a work item and upload an attachment with a very long filename and comment
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Modal overflow test',
      type: 'task',
    });

    const longFilename = 'this-is-a-very-long-filename-that-should-be-truncated-on-mobile-devices.png';
    const longComment = 'Detailed specification document for the milestone dashboard feature with advanced analytics';

    // Create a small valid PNG (1x1 pixel)
    const pngBuffer = Buffer.from(
      'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
      'base64',
    );

    await api.uploadAttachment(
      request, testUser.token, testProject.key, item.item_number,
      longFilename, pngBuffer, 'image/png', longComment,
    );

    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    // Navigate to the work item detail page
    await page.goto(`/projects/${testProject.key}/items/${item.item_number}`);

    // Switch to the Attachments tab
    await page.getByRole('button', { name: 'Attachments', exact: false }).click();

    // Wait for the attachment to appear and click to preview
    const attachmentRow = page.getByText(longFilename);
    await expect(attachmentRow).toBeVisible({ timeout: 10000 });
    await attachmentRow.click();

    // Wait for the preview modal to open
    const modal = page.locator('.fixed.inset-0.z-50');
    await expect(modal).toBeVisible({ timeout: 5000 });

    // Get the header area within the modal
    const header = modal.locator('div').filter({ has: page.locator('button') }).first();

    // The close button (last button in header) should be fully visible and not overlapped
    const closeButton = modal.locator('button').last();
    await expect(closeButton).toBeVisible();
    const closeBox = await closeButton.boundingBox();
    expect(closeBox).not.toBeNull();

    // The close button should be within the viewport
    expect(closeBox!.x).toBeGreaterThanOrEqual(0);
    expect(closeBox!.x + closeBox!.width).toBeLessThanOrEqual(375 + 1);

    // The text container (filename + comment) should be truncated and not
    // extend past the action buttons
    const downloadButton = modal.locator('button').first();
    const downloadBox = await downloadButton.boundingBox();
    expect(downloadBox).not.toBeNull();

    // The truncated text container should end before the buttons start
    const textContainer = modal.locator('.truncate').first();
    const textBox = await textContainer.boundingBox();
    expect(textBox).not.toBeNull();
    expect(textBox!.x + textBox!.width).toBeLessThanOrEqual(downloadBox!.x + 1); // 1px tolerance

    // Page should have no horizontal overflow
    const hasOverflow = await page.evaluate(() => {
      return document.documentElement.scrollWidth > document.documentElement.clientWidth;
    });
    expect(hasOverflow).toBe(false);

    // Close the modal
    await page.keyboard.press('Escape');
    await expect(modal).not.toBeVisible();
  });

  test('diff preview modal title does not overflow on mobile', async ({ request, testUser, testProject, page }) => {
    // Create a work item with a long description to generate a field change event
    const longDescription = 'A'.repeat(200);
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Diff modal overflow test',
      type: 'task',
      description: longDescription,
    });

    // Update to a different description
    const newDescription = 'B'.repeat(200);
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      description: newDescription,
    });

    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    await page.goto(`/projects/${testProject.key}/items/${item.item_number}`);

    // Switch to the Activity tab
    await page.getByRole('button', { name: 'Activity', exact: true }).click();
    await expect(page.getByText('changed Description')).toBeVisible({ timeout: 10000 });

    // Click the diff box to open the full diff modal
    const diffBox = page.locator('[role="button"]').filter({ hasText: '\u2026' }).first();
    await expect(diffBox).toBeVisible();
    await diffBox.click();

    // Modal should open
    const modal = page.locator('.fixed.inset-0.z-50');
    await expect(modal.getByText('Description change')).toBeVisible({ timeout: 5000 });

    // No horizontal overflow
    const hasOverflow = await page.evaluate(() => {
      return document.documentElement.scrollWidth > document.documentElement.clientWidth;
    });
    expect(hasOverflow).toBe(false);

    // Close button should be visible and within viewport
    const closeButton = modal.locator('button').first();
    await expect(closeButton).toBeVisible();
    const closeBox = await closeButton.boundingBox();
    expect(closeBox).not.toBeNull();
    expect(closeBox!.x + closeBox!.width).toBeLessThanOrEqual(375 + 1);

    await page.keyboard.press('Escape');
    await expect(modal).not.toBeVisible();
  });
});
