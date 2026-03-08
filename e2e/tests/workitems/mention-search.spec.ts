import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

test.describe('@ mention search modal', () => {
  test('typing @ in comment textarea opens mention search modal', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `MentionTest-${Date.now()}`,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    // Focus the comment textarea and type @
    const textarea = page.getByPlaceholder(/add a comment/i);
    await textarea.click();
    await textarea.press('@');

    // Mention search modal should appear with search input
    const searchInput = page.getByPlaceholder(/search to insert link/i);
    await expect(searchInput).toBeVisible({ timeout: 3000 });
    await expect(searchInput).toBeFocused();
  });

  test('searching and selecting a work item inserts markdown link', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const targetTitle = `MentionTarget-${Date.now()}`;
    const target = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: targetTitle,
      type: 'task',
    });

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `MentionSource-${Date.now()}`,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    // Type @ in comment
    const textarea = page.getByPlaceholder(/add a comment/i);
    await textarea.click();
    await textarea.press('@');

    // Search for the target work item
    const searchInput = page.getByPlaceholder(/search to insert link/i);
    await expect(searchInput).toBeVisible({ timeout: 3000 });
    await searchInput.fill(targetTitle.slice(0, 14));

    // Wait for results and click the first one
    const result = page.locator('[data-selected="true"]').first();
    await expect(result).toBeVisible({ timeout: 10000 });
    await expect(result).toContainText(targetTitle);
    await result.click();

    // Modal should close and textarea should contain a markdown link
    await expect(searchInput).not.toBeVisible({ timeout: 3000 });
    const value = await textarea.inputValue();
    expect(value).toContain(`[${target.display_id}]`);
    expect(value).toContain(`/d/projects/${testProject.key}/items/${target.item_number}`);
  });

  test('Enter key selects highlighted result', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const targetTitle = `EnterMention-${Date.now()}`;
    const target = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: targetTitle,
      type: 'task',
    });

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `EnterSource-${Date.now()}`,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    const textarea = page.getByPlaceholder(/add a comment/i);
    await textarea.click();
    await textarea.press('@');

    const searchInput = page.getByPlaceholder(/search to insert link/i);
    await searchInput.fill(targetTitle.slice(0, 14));

    // Wait for results
    await expect(page.locator('[data-selected="true"]')).toBeVisible({ timeout: 10000 });

    // Press Enter to select
    await searchInput.press('Enter');

    // Modal should close and link should be inserted
    await expect(searchInput).not.toBeVisible({ timeout: 3000 });
    const value = await textarea.inputValue();
    expect(value).toContain(`[${target.display_id}]`);
  });

  test('Escape closes mention modal without inserting', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `EscMention-${Date.now()}`,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    const textarea = page.getByPlaceholder(/add a comment/i);
    await textarea.click();
    await textarea.press('@');

    const searchInput = page.getByPlaceholder(/search to insert link/i);
    await expect(searchInput).toBeVisible({ timeout: 3000 });

    // Press Escape
    await page.keyboard.press('Escape');

    // Modal should close
    await expect(searchInput).not.toBeVisible({ timeout: 3000 });

    // Textarea should only contain the @
    const value = await textarea.inputValue();
    expect(value).toBe('@');
  });

  test('@ in work item description textarea opens mention modal', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `DescMention-${Date.now()}`,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    // Double-click the description area to enter edit mode
    const descArea = page.locator('.prose').first();
    if (await descArea.isVisible({ timeout: 2000 }).catch(() => false)) {
      await descArea.dblclick();
    } else {
      // No description yet — click the no-description placeholder
      const noDesc = page.getByText(/no description/i);
      await noDesc.dblclick();
    }

    // Find the description textarea and type @
    const descTextarea = page.locator('textarea').first();
    await expect(descTextarea).toBeVisible({ timeout: 3000 });
    await descTextarea.press('@');

    // Mention search modal should appear
    const searchInput = page.getByPlaceholder(/search to insert link/i);
    await expect(searchInput).toBeVisible({ timeout: 3000 });

    // Close it
    await page.keyboard.press('Escape');
    await expect(searchInput).not.toBeVisible({ timeout: 3000 });
  });

  test('hint shown before typing 2+ characters', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `HintMention-${Date.now()}`,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    const textarea = page.getByPlaceholder(/add a comment/i);
    await textarea.click();
    await textarea.press('@');

    const searchInput = page.getByPlaceholder(/search to insert link/i);
    await expect(searchInput).toBeVisible({ timeout: 3000 });

    // Should show hint before typing
    await expect(page.getByText(/type at least 2 characters/i)).toBeVisible({ timeout: 3000 });
  });
});
