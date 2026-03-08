import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

test.describe('Attachment list on mobile', () => {
  test('long filename does not overflow into action icons', async ({ request, testUser, testProject, page }) => {
    // Create a work item and upload an attachment with a very long filename
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Attachment list overflow test',
      type: 'task',
    });

    const longFilename = 'TF-220-milestone-advanced-analytics-specification-document-with-extras.md';

    // Create a small text file
    const content = Buffer.from('# Spec\nPlaceholder content for testing.');

    await api.uploadAttachment(
      request, testUser.token, testProject.key, item.item_number,
      longFilename, content, 'text/markdown', 'Detailed specification for milestone advanced analytics features',
    );

    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    // Navigate to the work item detail page
    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);

    // Switch to the Attachments tab
    await page.getByRole('button', { name: 'Attachments', exact: false }).click();

    // Wait for the attachment to appear
    await expect(page.getByText(longFilename)).toBeVisible({ timeout: 10000 });

    // The action icons (download, edit, delete) should be fully within the viewport
    const actionButtons = page.locator('.shrink-0').filter({ has: page.locator('button') });
    const actionBox = await actionButtons.first().boundingBox();
    expect(actionBox).not.toBeNull();
    expect(actionBox!.x).toBeGreaterThanOrEqual(0);
    expect(actionBox!.x + actionBox!.width).toBeLessThanOrEqual(375 + 1);

    // The page should have no horizontal overflow
    const hasOverflow = await page.evaluate(() => {
      return document.documentElement.scrollWidth > document.documentElement.clientWidth;
    });
    expect(hasOverflow).toBe(false);

    // The filename is inside a ScrollableRow (overflow-x-auto container) that clips
    // the long text. Verify the filename button itself is wider than the visible
    // container — proving the ScrollableRow is active and clipping.
    const isClipped = await page.getByText(longFilename).evaluate((el) => {
      const container = el.closest('.overflow-x-auto');
      if (!container) return false;
      return container.scrollWidth > container.clientWidth;
    });
    expect(isClipped).toBe(true);
  });
});
