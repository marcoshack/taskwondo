import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

test.describe('Work item relations', () => {
  test('add relation via UI and verify in list', async ({ request, testUser, testProject, page }) => {
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Parent item',
      type: 'task',
    });
    const item2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Child item',
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item1.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to Relations tab
    await page.getByRole('button', { name: 'Relations', exact: true }).click();

    // Fill in the work item picker with the target display ID
    const picker = page.getByPlaceholder(/search/i);
    await picker.fill(item2.display_id);

    // Wait for and select the autocomplete result
    await page.getByRole('option').filter({ hasText: item2.display_id }).first().click();

    // Click the Add button
    await page.getByRole('button', { name: 'Add', exact: true }).click();

    // Verify the relation appears in the list
    await expect(page.getByText(item2.display_id)).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Child item')).toBeVisible();
  });

  test('relation form is usable on mobile viewport', async ({ request, testUser, testProject, page }) => {
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Mobile relation source',
      type: 'task',
    });
    const item2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Mobile relation target',
      type: 'task',
    });

    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`/d/projects/${testProject.key}/items/${item1.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to Relations tab
    await page.getByRole('button', { name: 'Relations', exact: true }).click();

    // The form should be visible and not cropped
    const picker = page.getByPlaceholder(/search/i);
    await expect(picker).toBeVisible();

    // The relation type select and Add button should be visible
    const addButton = page.getByRole('button', { name: 'Add', exact: true });
    await expect(addButton).toBeVisible();

    // Fill in a relation and verify the dropdown is not cropped
    await picker.fill(item2.display_id);

    // The dropdown should be visible with results
    const dropdown = page.getByRole('option').first();
    await expect(dropdown).toBeVisible({ timeout: 10000 });

    // Select the result and add
    await dropdown.click();
    await addButton.click();

    // Verify the relation appears
    await expect(page.getByText(item2.display_id)).toBeVisible({ timeout: 10000 });
  });

  test('create relation via API and verify in Relations tab', async ({ request, testUser, testProject, page }) => {
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Source item',
      type: 'task',
    });
    const item2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Target item',
      type: 'task',
    });

    // Create relation via API
    await api.createRelation(request, testUser.token, testProject.key, item1.item_number, {
      target_display_id: item2.display_id,
      relation_type: 'blocks',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item1.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to Relations tab
    await page.getByRole('button', { name: 'Relations', exact: true }).click();

    // Verify the relation is displayed
    await expect(page.getByText(item2.display_id)).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Target item')).toBeVisible();
  });

  test('relation rows show equal-width priority badges', async ({ request, testUser, testProject, page }) => {
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Priority badge source',
      type: 'task',
    });
    const criticalItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Critical related item',
      type: 'task',
      priority: 'critical',
    });
    const lowItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Low related item',
      type: 'task',
      priority: 'low',
    });

    await api.createRelation(request, testUser.token, testProject.key, item1.item_number, {
      target_display_id: criticalItem.display_id,
      relation_type: 'relates_to',
    });
    await api.createRelation(request, testUser.token, testProject.key, item1.item_number, {
      target_display_id: lowItem.display_id,
      relation_type: 'relates_to',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item1.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to Relations tab
    await page.getByRole('button', { name: 'Relations', exact: true }).click();

    // Each related item shows a priority badge with the full label
    const criticalBadge = page.getByTestId('priority-badge').and(page.locator('[data-priority="critical"]'));
    const lowBadge = page.getByTestId('priority-badge').and(page.locator('[data-priority="low"]'));
    await expect(criticalBadge).toBeVisible({ timeout: 10000 });
    await expect(lowBadge).toBeVisible();
    await expect(criticalBadge).toHaveText('Critical', { useInnerText: true });
    await expect(lowBadge).toHaveText('Low', { useInnerText: true });

    // All badges have the same width so titles align
    const criticalBox = await criticalBadge.boundingBox();
    const lowBox = await lowBadge.boundingBox();
    expect(criticalBox).not.toBeNull();
    expect(lowBox).not.toBeNull();
    expect(Math.abs(criticalBox!.width - lowBox!.width)).toBeLessThanOrEqual(1);
  });

  test('priority badges collapse to first letter on mobile viewport', async ({ request, testUser, testProject, page }) => {
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Mobile priority source',
      type: 'task',
    });
    const highItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'High related item',
      type: 'task',
      priority: 'high',
    });

    await api.createRelation(request, testUser.token, testProject.key, item1.item_number, {
      target_display_id: highItem.display_id,
      relation_type: 'relates_to',
    });

    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`/d/projects/${testProject.key}/items/${item1.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to Relations tab
    await page.getByRole('button', { name: 'Relations', exact: true }).click();

    // On mobile the badge shows only the first letter of the priority
    const highBadge = page.getByTestId('priority-badge').and(page.locator('[data-priority="high"]'));
    await expect(highBadge).toBeVisible({ timeout: 10000 });
    await expect(highBadge).toHaveText('H', { useInnerText: true });
  });

  test('remove relation via UI', async ({ request, testUser, testProject, page }) => {
    const item1 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Item with relation',
      type: 'task',
    });
    const item2 = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'Related item',
      type: 'task',
    });

    // Create relation via API
    await api.createRelation(request, testUser.token, testProject.key, item1.item_number, {
      target_display_id: item2.display_id,
      relation_type: 'relates_to',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item1.item_number}`);
    await dismissWelcomeModal(page);

    // Switch to Relations tab
    await page.getByRole('button', { name: 'Relations', exact: true }).click();

    // Verify the relation is there
    await expect(page.getByText(item2.display_id)).toBeVisible({ timeout: 10000 });

    // Click remove
    await page.getByRole('button', { name: /remove/i }).click();

    // Verify the relation is gone
    await expect(page.getByText(item2.display_id)).not.toBeVisible({ timeout: 5000 });
  });
});
