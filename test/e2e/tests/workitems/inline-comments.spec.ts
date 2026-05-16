import { test, expect } from '../../lib/fixtures';
import type { Page } from '@playwright/test';
import * as api from '../../lib/api';

const DESCRIPTION = [
  'First paragraph stays put.',
  '',
  'Second paragraph is the anchor target.',
  '',
  'Third paragraph that survives edits.',
].join('\n');

/**
 * Selects `substr` inside the rendered description by programmatically placing
 * a DOM Selection over it — the same Selection the component listens for.
 */
async function selectDescriptionText(page: Page, substr: string): Promise<void> {
  const ok = await page.evaluate((needle) => {
    const root = document.querySelector('[data-testid="description-body"]');
    if (!root) return false;
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    let node: Node | null;
    while ((node = walker.nextNode())) {
      const text = node.textContent ?? '';
      const idx = text.indexOf(needle);
      if (idx >= 0) {
        const range = document.createRange();
        range.setStart(node, idx);
        range.setEnd(node, idx + needle.length);
        const sel = window.getSelection();
        sel?.removeAllRanges();
        sel?.addRange(range);
        return true;
      }
    }
    return false;
  }, substr);
  expect(ok, `description should contain "${substr}"`).toBe(true);
}

test.describe('Inline description comments (TF-350)', () => {
  test('selecting text opens the composer; submitting anchors a comment and shows a gutter marker', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `InlineSelect-${Date.now()}`,
      type: 'task',
      description: DESCRIPTION,
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });
    await expect(page.getByTestId('description-body')).toBeVisible();

    await selectDescriptionText(page, 'anchor target');

    // The floating "Comment" button appears below the selection.
    const addButton = page.getByTestId('inline-comment-add-button');
    await expect(addButton).toBeVisible({ timeout: 3000 });
    await addButton.click();

    const composer = page.getByTestId('inline-comment-composer');
    await expect(composer).toBeVisible();
    await composer.locator('textarea').fill('This sentence needs work.');
    await composer.getByRole('button', { name: 'Comment', exact: true }).click();

    // Submitting opens the thread anchored to that text.
    const thread = page.getByTestId('inline-comment-thread');
    await expect(thread).toBeVisible({ timeout: 5000 });
    await expect(thread.getByText('This sentence needs work.')).toBeVisible();

    // A gutter marker now sits beside the anchored line.
    await expect(page.getByTestId('inline-comment-gutter-icon')).toBeVisible();

    // The comment is also listed in the feed with a "View" affordance.
    await expect(page.getByTestId('inline-comment-view')).toBeVisible();
  });

  test('clicking the gutter marker opens the conversation thread', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `InlineGutter-${Date.now()}`,
      type: 'task',
      description: DESCRIPTION,
    });
    await api.addInlineComment(
      request, testUser.token, testProject.key, item.item_number,
      'Opened from the gutter.',
      { start_line: 3, end_line: 3, snippet: 'Second paragraph is the anchor target.' },
    );

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    const gutter = page.getByTestId('inline-comment-gutter-icon');
    await expect(gutter).toBeVisible({ timeout: 5000 });
    await gutter.click();

    const thread = page.getByTestId('inline-comment-thread');
    await expect(thread).toBeVisible();
    await expect(thread.getByText('Opened from the gutter.')).toBeVisible();
  });

  test('two comments on one line: the gutter marker is badged and prev/next cycles them', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `InlineMulti-${Date.now()}`,
      type: 'task',
      description: DESCRIPTION,
    });
    const anchor = { start_line: 3, end_line: 3, snippet: 'Second paragraph is the anchor target.' };
    await api.addInlineComment(request, testUser.token, testProject.key, item.item_number, 'First on the line.', anchor);
    await api.addInlineComment(request, testUser.token, testProject.key, item.item_number, 'Second on the line.', anchor);

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    // A single gutter marker for the line, badged with the comment count.
    const gutter = page.getByTestId('inline-comment-gutter-icon');
    await expect(gutter).toHaveCount(1);
    await expect(page.getByTestId('inline-comment-gutter-count')).toHaveText('2');
    await gutter.click();

    const thread = page.getByTestId('inline-comment-thread');
    await expect(thread).toBeVisible();
    await expect(thread.getByText('First on the line.')).toBeVisible();
    await expect(thread.getByTestId('inline-comment-position')).toHaveText('1 / 2');

    // Next advances to the second comment.
    await thread.getByTestId('inline-comment-next').click();
    await expect(thread.getByText('Second on the line.')).toBeVisible();
    await expect(thread.getByTestId('inline-comment-position')).toHaveText('2 / 2');

    // Next again wraps back to the first.
    await thread.getByTestId('inline-comment-next').click();
    await expect(thread.getByText('First on the line.')).toBeVisible();
    await expect(thread.getByTestId('inline-comment-position')).toHaveText('1 / 2');
  });

  test('the feed "View" link opens the thread and a reply is shown inside it', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `InlineView-${Date.now()}`,
      type: 'task',
      description: DESCRIPTION,
    });
    await api.addInlineComment(
      request, testUser.token, testProject.key, item.item_number,
      'Root comment for the thread.',
      { start_line: 3, end_line: 3, snippet: 'Second paragraph is the anchor target.' },
    );

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    await page.getByTestId('inline-comment-view').click();

    const thread = page.getByTestId('inline-comment-thread');
    await expect(thread).toBeVisible();
    await expect(thread.getByText('Root comment for the thread.')).toBeVisible();
    await expect(thread.getByTestId('inline-comment-thread-item')).toHaveCount(1);

    // Reply within the thread.
    await thread.locator('textarea').fill('A reply to the root.');
    await thread.getByRole('button', { name: 'Reply', exact: true }).click();

    await expect(thread.getByText('A reply to the root.')).toBeVisible({ timeout: 5000 });
    await expect(thread.getByTestId('inline-comment-thread-item')).toHaveCount(2);

    // The reply is also listed in the comments feed (root + reply).
    await expect(page.getByTestId('inline-comment-view')).toHaveCount(2);
  });

  test('replies are stored as child comments sharing the root anchor', async ({
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `InlineReplyApi-${Date.now()}`,
      type: 'task',
      description: DESCRIPTION,
    });
    const root = await api.addInlineComment(
      request, testUser.token, testProject.key, item.item_number,
      'Root', { start_line: 3, end_line: 3, snippet: 'Second paragraph is the anchor target.' },
    );
    const reply = await api.addInlineReply(
      request, testUser.token, testProject.key, item.item_number, 'Reply', root.id,
    );
    expect(reply.parent_comment_id).toBe(root.id);
    expect(reply.anchor?.snippet).toBe(root.anchor?.snippet);
    expect(reply.anchor?.start_line).toBe(root.anchor?.start_line);
  });

  test('editing the description re-anchors the comment when the snippet survives', async ({
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `InlineReanchor-${Date.now()}`,
      type: 'task',
      description: DESCRIPTION,
    });
    const created = await api.addInlineComment(
      request, testUser.token, testProject.key, item.item_number,
      'Comment that should survive an edit',
      { start_line: 3, start_col: 1, end_line: 3, end_col: 7, snippet: 'Second paragraph is the anchor target.' },
    );
    expect(created.anchor?.status).toBe('active');
    expect(created.anchor?.start_line).toBe(3);

    // Push the anchored line down by two.
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      description: ['New intro line', '', ...DESCRIPTION.split('\n')].join('\n'),
    });

    const comments = await api.listComments(request, testUser.token, testProject.key, item.item_number);
    const inline = comments.find((c) => c.id === created.id)!;
    expect(inline.anchor?.status).toBe('active');
    expect(inline.anchor?.start_line).toBe(5);
    expect(inline.anchor?.snippet).toBe('Second paragraph is the anchor target.');
  });

  test('rewriting the description marks the comment outdated; its thread still opens', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `InlineOutdated-${Date.now()}`,
      type: 'task',
      description: DESCRIPTION,
    });
    await api.addInlineComment(
      request, testUser.token, testProject.key, item.item_number,
      'This snippet is about to disappear.',
      { start_line: 3, end_line: 3, snippet: 'Second paragraph is the anchor target.' },
    );

    // Replace the description so the snippet has nowhere to land.
    await api.updateWorkItem(request, testUser.token, testProject.key, item.item_number, {
      description: 'Totally different content with no overlap whatsoever.',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    // The feed shows the comment flagged Outdated, with a "View" link.
    await expect(page.getByTestId('inline-comment-outdated-badge').first()).toBeVisible({ timeout: 5000 });
    await page.getByTestId('inline-comment-view').click();

    const thread = page.getByTestId('inline-comment-thread');
    await expect(thread).toBeVisible();
    await expect(thread.getByText('This snippet is about to disappear.')).toBeVisible();
    await expect(thread.getByTestId('inline-comment-outdated-badge')).toBeVisible();
  });
});
