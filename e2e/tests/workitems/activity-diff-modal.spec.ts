import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

test.describe('Activity diff modal', () => {
  test('opens full diff modal when clicking a truncated field change', async ({ request, testUser, testProject, page }) => {
    // Create a work item with a long description
    const longDescription = 'A'.repeat(200);
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Diff modal test',
      type: 'task',
      description: longDescription,
    });

    // Update to a different long description to generate a field change event
    const newDescription = 'B'.repeat(200);
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      description: newDescription,
    });

    // Navigate to the work item detail page
    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to the Activity tab
    await page.getByRole('button', { name: 'Activity', exact: true }).click();

    // Wait for the activity timeline to load — look for the "changed Description" event
    await expect(page.getByText('changed Description')).toBeVisible({ timeout: 10000 });

    // The diff box should be truncated (showing ellipsis) — find and click it
    const diffBox = page.locator('[role="button"]').filter({ hasText: '\u2026' }).first();
    await expect(diffBox).toBeVisible();
    await diffBox.click();

    // The full diff modal should open with the title
    await expect(page.getByText('Description change')).toBeVisible({ timeout: 5000 });

    // The modal should contain the full untruncated values
    const modal = page.locator('.fixed.inset-0.z-50');
    await expect(modal.getByText(longDescription)).toBeVisible();
    await expect(modal.getByText(newDescription)).toBeVisible();

    // Close the modal via Escape
    await page.keyboard.press('Escape');
    await expect(modal).not.toBeVisible();
  });

  test('shows line-level diff with word highlights for multiline description change', async ({ request, testUser, testProject, page }) => {
    // Create a work item with a multiline description
    const originalDesc = [
      'This is the first line of the description.',
      'The second line has some important details.',
      'Third line remains unchanged.',
      'Fourth line with original content.',
    ].join('\n');

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Multiline diff test',
      type: 'task',
      description: originalDesc,
    });

    // Update only specific words in some lines
    const updatedDesc = [
      'This is the first line of the description.',
      'The second line has some updated details.',
      'Third line remains unchanged.',
      'Fourth line with new content.',
    ].join('\n');

    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      description: updatedDesc,
    });

    // Navigate to the work item detail page
    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to the Activity tab
    await page.getByRole('button', { name: 'Activity', exact: true }).click();

    // Wait for the activity timeline to load
    await expect(page.getByText('changed Description')).toBeVisible({ timeout: 10000 });

    // The inline diff should show line-level changes (minus/plus symbols), not the full old/new blocks
    // Only changed lines should appear — "important" → "updated" and "original" → "new"
    const diffContainer = page.locator('.font-mono').filter({ hasText: '\u2212' }).first();
    await expect(diffContainer).toBeVisible();

    // Should show the removed line with the old word
    await expect(diffContainer.getByText('important')).toBeVisible();
    // Should show the added line with the new word
    await expect(diffContainer.getByText('updated')).toBeVisible();

    // Unchanged lines (line 1 and 3) should NOT appear in the collapsed inline diff
    // (they are filtered out as 'same' lines)
    await expect(diffContainer.getByText('first line')).not.toBeVisible();
    await expect(diffContainer.getByText('remains unchanged')).not.toBeVisible();
  });

  test('multiline description diff modal shows full diff with word-level highlights', async ({ request, testUser, testProject, page }) => {
    // Create a work item with a multiline description (> 4 changed lines to trigger collapsed view)
    const originalDesc = Array.from({ length: 8 }, (_, i) => `Line ${i + 1}: original value here`).join('\n');

    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Multiline modal diff test',
      type: 'task',
      description: originalDesc,
    });

    // Update every line to generate a large diff
    const updatedDesc = Array.from({ length: 8 }, (_, i) => `Line ${i + 1}: updated value here`).join('\n');

    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      description: updatedDesc,
    });

    // Navigate to the work item detail page
    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to the Activity tab
    await page.getByRole('button', { name: 'Activity', exact: true }).click();
    await expect(page.getByText('changed Description')).toBeVisible({ timeout: 10000 });

    // Should show "Show X more lines" since there are > 4 changed lines
    const showMore = page.getByText(/Show \d+ more lines/);
    await expect(showMore).toBeVisible();

    // Click to open the full diff modal
    const diffBox = showMore.locator('..');
    await diffBox.click();

    // Modal should open with the title
    const modal = page.locator('.fixed.inset-0.z-50');
    await expect(modal.getByText('Description change')).toBeVisible({ timeout: 5000 });

    // Modal should show all lines with line-level diff
    await expect(modal.getByText('Line 1: original value here')).toBeVisible();
    await expect(modal.getByText('Line 1: updated value here')).toBeVisible();
    await expect(modal.getByText('Line 8: original value here')).toBeVisible();
    await expect(modal.getByText('Line 8: updated value here')).toBeVisible();

    // Close the modal
    await page.keyboard.press('Escape');
    await expect(modal).not.toBeVisible();
  });

  test('comment edit diff shows word-level highlights', async ({ request, testUser, testProject, page }) => {
    // Create a work item
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Comment word diff test',
      type: 'task',
    });

    // Add a multiline comment with specific words
    const originalComment = [
      'This feature works great in production.',
      'We should keep the current approach.',
    ].join('\n');
    const comment = await api.addComment(request, testUser.token, testProject.key, item.item_number, originalComment);

    // Update with specific word changes
    const updatedComment = [
      'This feature works great in staging.',
      'We should change the current approach.',
    ].join('\n');
    await api.updateComment(request, testUser.token, testProject.key, item.item_number, comment.id, updatedComment);

    // Navigate to the work item detail page
    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to the Activity tab
    await page.getByRole('button', { name: 'Activity', exact: true }).click();
    await expect(page.getByText('edited')).toBeVisible({ timeout: 10000 });

    // The inline diff should show the changed lines with word-level highlights
    const diffContainer = page.locator('.font-mono').filter({ hasText: '\u2212' }).first();
    await expect(diffContainer).toBeVisible();

    // The removed line should show "production" and the added line "staging"
    await expect(diffContainer.getByText('production')).toBeVisible();
    await expect(diffContainer.getByText('staging')).toBeVisible();
  });

  test('opens full diff modal for collapsed comment edit diff', async ({ request, testUser, testProject, page }) => {
    // Create a work item
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Comment diff test',
      type: 'task',
    });

    // Add a comment with multiple lines — use enough lines to trigger the collapsed view (> 4 changed lines)
    const originalComment = Array.from({ length: 10 }, (_, i) => `Line ${i + 1}: original content`).join('\n');
    const comment = await api.addComment(request, testUser.token, testProject.key, item.item_number, originalComment);

    // Update every line to generate a large diff
    const updatedComment = Array.from({ length: 10 }, (_, i) => `Line ${i + 1}: updated content`).join('\n');
    await api.updateComment(request, testUser.token, testProject.key, item.item_number, comment.id, updatedComment);

    // Navigate to the work item detail page
    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to the Activity tab
    await page.getByRole('button', { name: 'Activity', exact: true }).click();

    // Wait for the "edited" event to appear
    await expect(page.getByText('edited')).toBeVisible({ timeout: 10000 });

    // The collapsed diff should show "Show X more lines" text
    const showMore = page.getByText(/Show \d+ more lines/);
    await expect(showMore).toBeVisible();

    // Click on the diff box (the parent clickable container)
    const diffBox = showMore.locator('..');
    await diffBox.click();

    // The full diff modal should open with the comment diff title
    await expect(page.getByText('Comment edit diff')).toBeVisible({ timeout: 5000 });

    // The modal should show all diff lines (API stores full comment text, not truncated)
    const modal = page.locator('.fixed.inset-0.z-50');
    await expect(modal.getByText('Line 1: original content')).toBeVisible();
    await expect(modal.getByText('Line 1: updated content')).toBeVisible();
    await expect(modal.getByText('Line 10: original content')).toBeVisible();
    await expect(modal.getByText('Line 10: updated content')).toBeVisible();

    // Close via Escape
    await page.keyboard.press('Escape');
    await expect(modal).not.toBeVisible();
  });
});
