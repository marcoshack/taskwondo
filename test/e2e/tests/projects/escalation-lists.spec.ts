import { test, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import { randomUUID } from 'crypto';
import * as api from '../../lib/api';

const TEST_PASSWORD = 'TestPass123!';

/** Create a helper user and add them to the project so user search works. */
async function createHelperUser(
  request: any,
  testUserToken: string,
  projectKey: string,
): Promise<{ id: string; name: string }> {
  const adminToken = getAdminToken();
  const uid = randomUUID().slice(0, 8);
  const name = `EscUser ${uid}`;
  const email = `esc-${uid}@e2e.local`;
  const created = await api.createUser(request, adminToken, email, name);
  const tempLogin = await api.login(request, email, created.temporary_password);
  await api.changePassword(request, tempLogin.token, created.temporary_password, TEST_PASSWORD);
  const finalLogin = await api.login(request, email, TEST_PASSWORD);
  await api.addMember(request, testUserToken, projectKey, finalLogin.user.id, 'member');
  return { id: finalLogin.user.id, name };
}

test.describe('Escalation Lists', () => {

  test('CRUD lifecycle: create via API, edit and delete via UI', async ({ page, request, testUser, testProject }) => {
    const helper = await createHelperUser(request, testUser.token, testProject.key);

    await api.createEscalationList(request, testUser.token, testProject.key, {
      name: 'Critical Escalation',
      levels: [{ threshold_pct: 75, user_ids: [helper.id] }],
    });

    await page.goto(`/d/projects/${testProject.key}/workflows`);
    await page.waitForLoadState('networkidle');

    // The escalation list card should appear
    const listCard = page.getByRole('button', { name: /Critical Escalation.*level/i });
    await expect(listCard).toBeVisible();

    // Expand to see threshold badge
    await listCard.click();
    await expect(page.getByText('75%')).toBeVisible();

    // Edit via pencil button
    const cardContainer = listCard.locator('xpath=ancestor::div[contains(@class,"p-4")]');
    await cardContainer.locator('button').filter({ has: page.locator('svg.h-3\\.5') }).first().click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    const nameInput = dialog.getByRole('textbox', { name: 'Name', exact: true });
    await expect(nameInput).toHaveValue('Critical Escalation', { timeout: 5000 });

    await nameInput.clear();
    await nameInput.fill('Updated Escalation');
    await dialog.getByRole('button', { name: 'Save' }).click();
    await expect(dialog).not.toBeVisible({ timeout: 5000 });

    const updatedCard = page.getByRole('button', { name: /Updated Escalation.*level/i });
    await expect(updatedCard).toBeVisible();

    // Delete via trash button
    const updatedContainer = updatedCard.locator('xpath=ancestor::div[contains(@class,"p-4")]');
    await updatedContainer.locator('button').filter({ has: page.locator('svg.text-red-500') }).click();

    const deleteDialog = page.getByRole('dialog');
    await expect(deleteDialog).toBeVisible();
    await deleteDialog.getByRole('button', { name: 'Delete' }).click();
    await expect(updatedCard).not.toBeVisible({ timeout: 5000 });
    await expect(page.getByText('No escalation lists yet.')).toBeVisible();
  });

  test('create escalation list via UI modal with user search', async ({ page, request, testUser, testProject }) => {
    const helper = await createHelperUser(request, testUser.token, testProject.key);

    await page.goto(`/d/projects/${testProject.key}/workflows`);
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: 'New Escalation List' }).click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    await dialog.getByRole('textbox', { name: 'Name', exact: true }).fill('UI Created List');

    // Add a level
    await dialog.getByRole('button', { name: 'Add Level' }).click();
    await dialog.locator('input[type="number"]').first().fill('50');

    // Search for the helper user
    await dialog.getByPlaceholder('Search by name or email...').fill(helper.name.slice(0, 8));
    await page.waitForTimeout(500);
    await dialog.getByText(helper.name).first().click();

    // Create
    await dialog.getByRole('button', { name: 'Create' }).click();
    await expect(dialog).not.toBeVisible({ timeout: 5000 });

    await expect(page.getByRole('button', { name: /UI Created List.*level/i })).toBeVisible();

    // Cleanup
    const lists = await api.listEscalationLists(request, testUser.token, testProject.key);
    for (const l of lists) {
      await api.deleteEscalationList(request, testUser.token, testProject.key, l.id);
    }
  });

  test('type mapping: assign, verify, remove', async ({ page, request, testUser, testProject }) => {
    const helper = await createHelperUser(request, testUser.token, testProject.key);
    const list = await api.createEscalationList(request, testUser.token, testProject.key, {
      name: 'Mapping Test List',
      levels: [{ threshold_pct: 80, user_ids: [helper.id] }],
    });

    await page.goto(`/d/projects/${testProject.key}/workflows`);
    await page.waitForLoadState('networkidle');

    await page.getByText('Escalation Mapping').scrollIntoViewIfNeeded();

    const escMappingSelects = page.locator('select').filter({ hasText: 'None' });
    const taskSelect = escMappingSelects.first();
    await taskSelect.selectOption(list.id);

    await expect(page.locator('.text-green-500').first()).toBeVisible({ timeout: 5000 });
    await expect(taskSelect).toHaveValue(list.id);

    await taskSelect.selectOption('');
    await page.waitForTimeout(1000);
    await expect(taskSelect).toHaveValue('');

    await api.deleteEscalationList(request, testUser.token, testProject.key, list.id);
  });

  test('validation: name, levels, thresholds', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/workflows`);
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: 'New Escalation List' }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    // Empty name
    await dialog.getByRole('button', { name: 'Create' }).click();
    await expect(dialog.getByText('Escalation list name is required.')).toBeVisible();

    // No levels
    await dialog.getByRole('textbox', { name: 'Name', exact: true }).fill('Test Validation');
    await dialog.getByRole('button', { name: 'Create' }).click();
    await expect(dialog.getByText('At least one escalation level is required.')).toBeVisible();

    // Threshold not set
    await dialog.getByRole('button', { name: 'Add Level' }).click();
    await dialog.getByRole('button', { name: 'Create' }).click();
    await expect(dialog.getByText('All levels must have a threshold greater than 0.')).toBeVisible();

    // No users
    await dialog.locator('input[type="number"]').first().fill('50');
    await dialog.getByRole('button', { name: 'Create' }).click();
    await expect(dialog.getByText('Each level must have at least one user.')).toBeVisible();

    await page.keyboard.press('Escape');
  });

  test('delete warning shows for assigned lists', async ({ page, request, testUser, testProject }) => {
    const helper = await createHelperUser(request, testUser.token, testProject.key);
    const list = await api.createEscalationList(request, testUser.token, testProject.key, {
      name: 'Assigned List',
      levels: [{ threshold_pct: 90, user_ids: [helper.id] }],
    });
    await api.setEscalationMapping(request, testUser.token, testProject.key, 'task', list.id);

    await page.goto(`/d/projects/${testProject.key}/workflows`);
    await page.waitForLoadState('networkidle');

    const listCard = page.getByRole('button', { name: /Assigned List.*level/i });
    await expect(listCard).toBeVisible();

    const cardContainer = listCard.locator('xpath=ancestor::div[contains(@class,"p-4")]');
    await cardContainer.locator('button').filter({ has: page.locator('svg.text-red-500') }).click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await expect(dialog.getByText('Delete Escalation List')).toBeVisible();
    await expect(dialog.getByText('This list is currently assigned to one or more work item types.')).toBeVisible();

    await dialog.getByRole('button', { name: 'Delete' }).click();
    await expect(listCard).not.toBeVisible({ timeout: 5000 });

    await api.deleteEscalationMapping(request, testUser.token, testProject.key, 'task').catch(() => {});
  });
});
