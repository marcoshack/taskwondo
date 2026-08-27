import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

/**
 * Covers the New Work Item modal: the two-column desktop layout, the Status
 * field that lets an item start in any open status, and attachments that are
 * held in the form until Create and uploaded to the new item afterwards.
 */

async function openModal(page: import('@playwright/test').Page) {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('/user/inbox');
  await page.waitForLoadState('networkidle');

  await page.getByRole('button', { name: 'New Item', exact: true }).click();
  // .first() because the discard prompt renders a second dialog on top.
  const dialog = page.getByRole('dialog').first();
  await expect(dialog).toBeVisible();
  return dialog;
}

async function openCreateModal(page: import('@playwright/test').Page, projectKey: string) {
  const dialog = await openModal(page);
  await dialog.getByRole('button', { name: /Select project/ }).click();
  await page.getByRole('button', { name: new RegExp(projectKey) }).first().click();
  await dialog.locator('select#type').selectOption('task');
  return dialog;
}

test.describe('New Work Item modal', () => {
  test('offers only open statuses, scrolls its fields, and submits with Cmd+Enter in the chosen status', async ({ page, request, testUser, testProject }) => {
    const dialog = await openCreateModal(page, testProject.key);

    // Every field is present from the start, and the ones past the fold are
    // reached by scrolling the fields column rather than growing the modal.
    const box = await dialog.boundingBox();
    expect(box!.height).toBeLessThanOrEqual(page.viewportSize()!.height);
    const column = dialog.locator('select#visibility').locator('xpath=ancestor::div[contains(@class,"overflow-y-auto")][1]');
    const overflows = await column.evaluate((el) => el.scrollHeight > el.clientHeight);
    expect(overflows).toBe(true);

    // Done and cancelled statuses must not be offered — items reach those
    // through transitions, which is what records resolved_at.
    const statuses = await dialog.locator('select#status option').allTextContents();
    expect(statuses).toContain('Open');
    expect(statuses).toContain('In Progress');
    expect(statuses).not.toContain('Done');
    expect(statuses).not.toContain('Cancelled');

    await dialog.locator('input#title').fill('Started before it was filed');
    await dialog.locator('select#status').selectOption('in_progress');
    // Cmd/Ctrl+Enter submits, same as clicking Create.
    await page.keyboard.press('ControlOrMeta+Enter');
    await expect(dialog).not.toBeVisible();

    const items = await api.listWorkItems(request, testUser.token, testProject.key);
    const created = items.data.find((i) => i.title === 'Started before it was filed');
    expect(created).toBeTruthy();

    const item = await api.getWorkItem(request, testUser.token, testProject.key, created!.item_number);
    expect(item.status).toBe('in_progress');
  });


  test('never blocks typing, marks what is missing, and guards discarding', async ({ page }) => {
    const dialog = await openModal(page);

    // Title and description accept input before a project or type is chosen.
    await expect(dialog.locator('input#title')).toBeEnabled();
    await expect(dialog.locator('textarea')).toBeEnabled();
    await dialog.locator('input#title').fill('Typed before choosing a type');

    // Create stays clickable; a click marks the missing required fields.
    const create = dialog.getByRole('button', { name: 'Create' });
    await expect(create).toBeEnabled();
    await create.click();
    await expect(dialog).toBeVisible();
    await expect(dialog.locator('select#type')).toHaveClass(/border-red-300/);

    // Escape on a form with content asks before throwing it away.
    await page.keyboard.press('Escape');
    const confirm = page.getByRole('heading', { name: 'Discard this work item?' });
    await expect(confirm).toBeVisible();

    await page.getByRole('button', { name: 'Keep editing' }).click();
    await expect(confirm).not.toBeVisible();
    await expect(dialog.locator('input#title')).toHaveValue('Typed before choosing a type');

    // Cancel routes through the same prompt, and discarding closes everything.
    await dialog.getByRole('button', { name: 'Cancel' }).click();
    await expect(confirm).toBeVisible();
    await page.getByRole('button', { name: 'Discard' }).click();
    await expect(page.getByRole('dialog')).toHaveCount(0);
  });

  test('holds attachments until Create, then uploads them to the new item', async ({ page, request, testUser, testProject }) => {
    const dialog = await openCreateModal(page, testProject.key);
    await dialog.locator('input#title').fill('Item with attachments');

    await dialog.locator('input[type=file]').setInputFiles([
      { name: 'crash-log.txt', mimeType: 'text/plain', buffer: Buffer.from('boom') },
      { name: 'api-trace.har', mimeType: 'application/json', buffer: Buffer.from('{}') },
    ]);
    await expect(dialog.getByText('crash-log.txt')).toBeVisible();
    await expect(dialog.getByText('api-trace.har')).toBeVisible();

    // Nothing is uploaded while the form is still open.
    const before = await api.listWorkItems(request, testUser.token, testProject.key);
    expect(before.data.find((i) => i.title === 'Item with attachments')).toBeUndefined();

    await dialog.getByRole('button', { name: 'Create' }).click();
    await expect(dialog).not.toBeVisible();

    const items = await api.listWorkItems(request, testUser.token, testProject.key);
    const created = items.data.find((i) => i.title === 'Item with attachments');
    expect(created).toBeTruthy();

    const attachments = await api.listAttachments(request, testUser.token, testProject.key, created!.item_number);
    expect(attachments.map((a) => a.filename).sort()).toEqual(['api-trace.har', 'crash-log.txt']);
  });
});
