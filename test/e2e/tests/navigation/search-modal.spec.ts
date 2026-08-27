import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { openPalette, paletteInput, navRows, entityRows } from '../../lib/palette';
import { randomUUID } from 'crypto';

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}


/** A second project owned by the same user, so scoping has something to exclude. */
async function createSecondProject(
  request: import('@playwright/test').APIRequestContext,
  token: string,
) {
  for (let attempt = 0; attempt < 3; attempt++) {
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    try {
      return await api.createProject(request, token, `S${suffix}`, `E2E Scope ${suffix}`);
    } catch (err: any) {
      if (attempt === 2 || !err.message?.includes('already in use')) throw err;
    }
  }
  throw new Error('unreachable');
}

test.describe('Command palette — entity search', () => {
  test('Cmd/Ctrl+K opens the palette and Escape closes it', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await openPalette(page);
    const searchInput = paletteInput(page);
    await expect(searchInput).toBeVisible({ timeout: 3000 });
    await expect(searchInput).toBeFocused();

    // Close with Escape
    await page.keyboard.press('Escape');
    await expect(searchInput).not.toBeVisible({ timeout: 3000 });
  });

  test('search shows results for matching work items via unified endpoint', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    // Create a work item with a unique title
    const uniqueTitle = `SearchTest-${Date.now()}`;
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: uniqueTitle,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await openPalette(page);
    const searchInput = paletteInput(page);
    await expect(searchInput).toBeVisible({ timeout: 3000 });

    // Type search query
    await searchInput.fill(uniqueTitle.slice(0, 12));

    // Wait for results - FTS results stream via SSE
    const resultItem = page.locator('[data-search-item]').first();
    await expect(resultItem).toBeVisible({ timeout: 10000 });
    await expect(resultItem).toContainText(uniqueTitle);
  });

  test('display ID search returns exact match as top result', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    // Create a work item
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `DisplayIDSearch-${Date.now()}`,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open search and search by display ID
    await openPalette(page);
    const searchInput = paletteInput(page);
    await searchInput.fill(item.display_id);

    // First result should appear
    const resultItem = page.locator('[data-search-item]').first();
    await expect(resultItem).toBeVisible({ timeout: 10000 });

    // Press Enter should navigate to this item
    await searchInput.press('Enter');
    await expect(page).toHaveURL(
      new RegExp(`/d/projects/${testProject.key}/items/${item.item_number}`),
      { timeout: 5000 },
    );
  });

  test('search shows empty state when no results found', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await openPalette(page);
    const searchInput = paletteInput(page);
    await expect(searchInput).toBeVisible({ timeout: 3000 });

    // Search for something that doesn't exist
    await searchInput.fill('zzzznonexistent99999');

    // Wait for empty state
    await expect(page.getByText(/no results found/i)).toBeVisible({ timeout: 10000 });
  });

  test('keyboard navigation works in search results', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    // Create work items with searchable titles
    const prefix = `NavTest-${Date.now()}`;
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `${prefix} First`,
      type: 'task',
    });
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `${prefix} Second`,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await openPalette(page);
    const searchInput = paletteInput(page);
    await searchInput.fill(prefix);

    // Wait for results
    const results = page.locator('[data-search-item]');
    await expect(results.first()).toBeVisible({ timeout: 10000 });

    // First item should be highlighted by default
    const firstItem = results.first();
    await expect(firstItem).toHaveClass(/bg-indigo-50/);

    // Press ArrowDown to move to second item
    await searchInput.press('ArrowDown');
    const secondItem = results.nth(1);
    await expect(secondItem).toHaveClass(/bg-indigo-50/);

    // Press ArrowUp to go back to first
    await searchInput.press('ArrowUp');
    await expect(firstItem).toHaveClass(/bg-indigo-50/);
  });

  test('Enter on a result navigates to the work item', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const uniqueTitle = `EnterNav-${Date.now()}`;
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: uniqueTitle,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open search and find the item
    await openPalette(page);
    const searchInput = paletteInput(page);
    await searchInput.fill(uniqueTitle);

    // Wait for result
    const result = page.locator('[data-search-item]').first();
    await expect(result).toBeVisible({ timeout: 10000 });

    // Press Enter to navigate
    await searchInput.press('Enter');

    // Should navigate to the work item detail page
    await expect(page).toHaveURL(
      new RegExp(`/d/projects/${testProject.key}/items/${item.item_number}`),
      { timeout: 5000 },
    );
  });

  test('clicking a result navigates to the work item', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const uniqueTitle = `ClickNav-${Date.now()}`;
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: uniqueTitle,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open search and find the item
    await openPalette(page);
    const searchInput = paletteInput(page);
    await searchInput.fill(uniqueTitle);

    // Wait for result and click it
    const result = page.locator('[data-search-item]').first();
    await expect(result).toBeVisible({ timeout: 10000 });
    await result.click();

    // Should navigate to the work item detail page
    await expect(page).toHaveURL(
      new RegExp(`/d/projects/${testProject.key}/items/${item.item_number}`),
      { timeout: 5000 },
    );
  });

  // An empty query used to show the two-character hint; the palette shows the
  // navigation catalog instead, and keeps the hint for the one-character case
  // where navigation already answers but entity search has not started.
  test('empty query shows the navigation catalog, one character shows the entity hint', async ({
    page,
    testProject,
  }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    const input = await openPalette(page);

    // Empty query: the catalog, not the hint.
    await expect(navRows(page).first()).toBeVisible({ timeout: 3000 });
    await expect(page.getByText(/type at least 2 characters/i)).toHaveCount(0);

    // One character: navigation still filters live, entity search does not run.
    await input.fill('m');
    await expect(page.getByText(/type at least 2 characters/i)).toBeVisible({ timeout: 3000 });
    await expect(entityRows(page)).toHaveCount(0);
  });

  test('comment deep-link opens comments tab and highlights the comment', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `CommentDeepLink-${Date.now()}`,
      type: 'task',
    });
    const comment = await api.addComment(
      request,
      testUser.token,
      testProject.key,
      item.item_number,
      'Deep-linked comment body',
    );

    // Navigate with tab=comments&highlight=<commentId>
    await page.goto(
      `/d/projects/${testProject.key}/items/${item.item_number}?tab=comments&highlight=${comment.id}`,
    );
    await dismissWelcomeModal(page);

    // Comments tab should be active
    const commentsTab = page.getByRole('button', { name: /comments/i });
    await expect(commentsTab).toHaveClass(/border-indigo|text-indigo/, { timeout: 5000 });

    // The comment should be visible
    await expect(page.getByText('Deep-linked comment body')).toBeVisible({ timeout: 5000 });
  });

  test('attachment deep-link opens attachments tab and highlights the attachment', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `AttachDeepLink-${Date.now()}`,
      type: 'task',
    });
    const attachment = await api.uploadAttachment(
      request,
      testUser.token,
      testProject.key,
      item.item_number,
      'test-deeplink.txt',
      Buffer.from('deep link test content'),
      'text/plain',
    );

    // Navigate with tab=attachments&highlight=<attachmentId>
    await page.goto(
      `/d/projects/${testProject.key}/items/${item.item_number}?tab=attachments&highlight=${attachment.id}`,
    );
    await dismissWelcomeModal(page);

    // Attachments tab should be active
    const attachTab = page.getByRole('button', { name: /attachments/i });
    await expect(attachTab).toHaveClass(/border-indigo|text-indigo/, { timeout: 5000 });

    // The attachment should be visible
    await expect(page.getByText('test-deeplink.txt')).toBeVisible({ timeout: 5000 });
  });

  // TF-432/434: entity hits are scoped to the active project. `project` hits
  // stay global on the backend, but they only exist in the semantic index,
  // which is off in the E2E stack — the palette's route to another project
  // here is the navigation catalog, covered in command-palette.spec.ts.
  test('entity results are scoped to the active project', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const prefix = `ScopeTest${Date.now()}`;
    const other = await createSecondProject(request, testUser.token);

    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `${prefix} here`,
      type: 'task',
    });
    await api.createWorkItem(request, testUser.token, other.key, {
      title: `${prefix} elsewhere`,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    const searchInput = await openPalette(page);
    await searchInput.fill(prefix);

    const results = entityRows(page);
    await expect(results.filter({ hasText: `${prefix} here` })).toHaveCount(1, { timeout: 10000 });
    await expect(results.filter({ hasText: `${prefix} elsewhere` })).toHaveCount(0);
  });

  test('query and results persist after closing with Escape', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const uniqueTitle = `PersistEsc${Date.now()}`;
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: uniqueTitle,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open search, type, wait for results
    await openPalette(page);
    const searchInput = paletteInput(page);
    await searchInput.fill(uniqueTitle);

    const result = page.locator('[data-search-item]').first();
    await expect(result).toBeVisible({ timeout: 10000 });
    await expect(result).toContainText(uniqueTitle);

    // Close with Escape
    await page.keyboard.press('Escape');
    await expect(searchInput).not.toBeVisible({ timeout: 3000 });

    // Reopen — query and results should still be there
    await openPalette(page);
    await expect(searchInput).toBeVisible({ timeout: 3000 });
    await expect(searchInput).toHaveValue(uniqueTitle);
    await expect(searchInput).toBeFocused();
    await expect(result).toBeVisible({ timeout: 5000 });
    await expect(result).toContainText(uniqueTitle);
  });

  test('query and results persist after navigating to a result', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const uniqueTitle = `PersistNav${Date.now()}`;
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: uniqueTitle,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open search, type, wait for results, press Enter to navigate
    await openPalette(page);
    const searchInput = paletteInput(page);
    await searchInput.fill(uniqueTitle);

    const result = page.locator('[data-search-item]').first();
    await expect(result).toBeVisible({ timeout: 10000 });

    await searchInput.press('Enter');
    await expect(page).toHaveURL(
      new RegExp(`/d/projects/${testProject.key}/items/${item.item_number}`),
      { timeout: 5000 },
    );

    // Reopen search — previous query and results should persist
    await openPalette(page);
    await expect(searchInput).toBeVisible({ timeout: 3000 });
    await expect(searchInput).toHaveValue(uniqueTitle);
    await expect(searchInput).toBeFocused();
    await expect(result).toBeVisible({ timeout: 5000 });
    await expect(result).toContainText(uniqueTitle);
  });

  test('input text is selected on reopen so typing replaces it', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const ts = Date.now();
    const firstTitle = `ReplaceFirst${ts}`;
    const secondTitle = `ReplaceSecond${ts}`;
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: firstTitle,
      type: 'task',
    });
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: secondTitle,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // First search
    await openPalette(page);
    const searchInput = paletteInput(page);
    await searchInput.fill(firstTitle);
    await expect(page.locator('[data-search-item]').first()).toBeVisible({ timeout: 10000 });

    // Close and reopen
    await page.keyboard.press('Escape');
    await expect(searchInput).not.toBeVisible({ timeout: 3000 });
    await openPalette(page);
    await expect(searchInput).toBeVisible({ timeout: 3000 });
    // Wait for the previous query to be restored before typing the replacement
    await expect(searchInput).toHaveValue(firstTitle, { timeout: 3000 });

    // Type a new query — since text is selected, it should replace entirely
    await searchInput.pressSequentially(secondTitle);
    await expect(searchInput).toHaveValue(secondTitle);

    // Results should now show second item, not first
    const results = page.locator('[data-search-item]');
    await expect(results.first()).toBeVisible({ timeout: 10000 });
    await expect(results.first()).toContainText(secondTitle);
  });

});
