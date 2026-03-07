import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

/**
 * Validates that the mention search modal appears within the viewport
 * and near the textarea that triggered it, even when the page is scrolled.
 */
test.describe('@ mention modal positioning', () => {
  const longDescription = Array.from({ length: 40 }, (_, i) =>
    `Paragraph ${i + 1}: Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.`,
  ).join('\n\n');

  const longComment = Array.from({ length: 10 }, (_, i) =>
    `Line ${i + 1}: This is a lengthy comment to push content further down the page and force scrolling.`,
  ).join('\n\n');

  async function assertModalNearTextarea(
    page: import('@playwright/test').Page,
    textarea: import('@playwright/test').Locator,
  ) {
    const searchInput = page.getByPlaceholder(/search to insert link/i);
    await expect(searchInput).toBeVisible({ timeout: 3000 });

    const modal = page.locator('.fixed.z-50').first();
    const modalBox = await modal.boundingBox();
    const textareaBox = await textarea.boundingBox();
    const viewport = page.viewportSize()!;

    expect(modalBox).not.toBeNull();
    expect(textareaBox).not.toBeNull();

    // Modal must be mostly inside the viewport (allow small scrollbar-width overflow)
    expect(modalBox!.x).toBeGreaterThanOrEqual(-1);
    expect(modalBox!.y).toBeGreaterThanOrEqual(-1);
    expect(modalBox!.x + modalBox!.width).toBeLessThanOrEqual(viewport.width + 50);
    expect(modalBox!.y + modalBox!.height).toBeLessThanOrEqual(viewport.height + 50);

    // Modal should be within 400px vertically of the textarea — the critical
    // check for the scroll-offset bug (where it would be 1000+ px away)
    const textareaBottom = textareaBox!.y + textareaBox!.height;
    const distance = Math.min(
      Math.abs(modalBox!.y - textareaBottom),
      Math.abs(modalBox!.y + modalBox!.height - textareaBox!.y),
    );
    expect(distance).toBeLessThan(400);

    return searchInput;
  }

  test('modal appears near comment textarea on a scrolled page', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    // Create item with long description to make the page scrollable
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `ScrollMention-${Date.now()}`,
      type: 'task',
      description: longDescription,
    });

    // Add several comments to push the comment textarea further down
    for (let i = 0; i < 5; i++) {
      await api.addComment(request, testUser.token, testProject.key, item.item_number, longComment);
    }

    await page.goto(`/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    // Scroll the comment textarea into view and type @
    const textarea = page.getByPlaceholder(/add a comment/i);
    await textarea.scrollIntoViewIfNeeded();
    await textarea.click();
    await textarea.press('@');

    const searchInput = await assertModalNearTextarea(page, textarea);

    // Clean up
    await page.keyboard.press('Escape');
    await expect(searchInput).not.toBeVisible({ timeout: 3000 });
  });

  test('modal appears near description textarea on a scrolled page', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `ScrollDescMention-${Date.now()}`,
      type: 'task',
      description: longDescription,
    });

    // Add comments so the page is tall
    for (let i = 0; i < 3; i++) {
      await api.addComment(request, testUser.token, testProject.key, item.item_number, longComment);
    }

    await page.goto(`/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    // Scroll back to the top to find the description
    await page.evaluate(() => window.scrollTo(0, 0));

    // Double-click description to enter edit mode
    const descArea = page.locator('.prose').first();
    await expect(descArea).toBeVisible({ timeout: 3000 });
    await descArea.dblclick();

    const descTextarea = page.locator('textarea').first();
    await expect(descTextarea).toBeVisible({ timeout: 3000 });

    // Move cursor to end and type @
    await descTextarea.press('End');
    await descTextarea.press('@');

    const searchInput = await assertModalNearTextarea(page, descTextarea);

    await page.keyboard.press('Escape');
    await expect(searchInput).not.toBeVisible({ timeout: 3000 });
  });

  test('modal stays in viewport after scrolling midway down the page', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `MidScrollMention-${Date.now()}`,
      type: 'task',
      description: longDescription,
    });

    for (let i = 0; i < 8; i++) {
      await api.addComment(request, testUser.token, testProject.key, item.item_number, longComment);
    }

    await page.goto(`/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    // Scroll to a middle position
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight / 2));
    await page.waitForTimeout(300);

    // The comment textarea should be at the top of the comments section
    const textarea = page.getByPlaceholder(/add a comment/i);
    await textarea.scrollIntoViewIfNeeded();
    await textarea.click();
    await textarea.press('@');

    const searchInput = await assertModalNearTextarea(page, textarea);

    await page.keyboard.press('Escape');
    await expect(searchInput).not.toBeVisible({ timeout: 3000 });
  });

  test('selecting a result after scrolling inserts link correctly', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const targetTitle = `ScrollTarget-${Date.now()}`;
    const target = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: targetTitle,
      type: 'task',
    });

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `ScrollSource-${Date.now()}`,
      type: 'task',
      description: longDescription,
    });

    for (let i = 0; i < 5; i++) {
      await api.addComment(request, testUser.token, testProject.key, item.item_number, longComment);
    }

    await page.goto(`/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    const textarea = page.getByPlaceholder(/add a comment/i);
    await textarea.scrollIntoViewIfNeeded();
    await textarea.click();
    await textarea.press('@');

    const searchInput = page.getByPlaceholder(/search to insert link/i);
    await expect(searchInput).toBeVisible({ timeout: 3000 });
    await searchInput.fill(targetTitle.slice(0, 14));

    const result = page.locator('[data-selected="true"]').first();
    await expect(result).toBeVisible({ timeout: 10000 });
    await expect(result).toContainText(targetTitle);
    await result.click();

    await expect(searchInput).not.toBeVisible({ timeout: 3000 });
    const value = await textarea.inputValue();
    expect(value).toContain(`[${target.display_id}]`);
    expect(value).toContain(`/projects/${testProject.key}/items/${target.item_number}`);
  });
});
