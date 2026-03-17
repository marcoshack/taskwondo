import { test, expect } from '../../lib/fixtures';

test.describe('SLA breach notification preference toggle', () => {
  test('SLA breach toggle exists in per-project notification preferences and defaults to enabled', async ({
    page,
    testProject,
  }) => {
    // Navigate to the notifications preferences page
    await page.goto('/preferences/notifications');
    await page.waitForLoadState('networkidle');

    // Expand the project card
    const projectCard = page.getByText(testProject.name);
    await projectCard.click();

    // The SLA breach checkbox should be visible and checked by default
    const slaLabel = page.getByText('SLA breach');
    await expect(slaLabel).toBeVisible();

    // Verify the description is shown
    await expect(
      page.getByText('Receive email notifications when work items approach or exceed SLA thresholds'),
    ).toBeVisible();

    // Find the checkbox associated with SLA breach
    const slaCheckbox = page
      .locator('label')
      .filter({ hasText: 'SLA breach' })
      .locator('input[type="checkbox"]');
    await expect(slaCheckbox).toBeChecked();
  });

  test('SLA breach toggle can be disabled and persists across page reload', async ({
    page,
    testProject,
  }) => {
    // Navigate to the notifications preferences page
    await page.goto('/preferences/notifications');
    await page.waitForLoadState('networkidle');

    // Expand the project card
    const projectCard = page.getByText(testProject.name);
    await projectCard.click();

    // Uncheck the SLA breach toggle
    const slaCheckbox = page
      .locator('label')
      .filter({ hasText: 'SLA breach' })
      .locator('input[type="checkbox"]');
    await expect(slaCheckbox).toBeChecked();
    await slaCheckbox.click();

    // Wait for save confirmation (green check)
    await expect(page.locator('.text-green-500')).toBeVisible({ timeout: 5000 });

    // Reload and verify it persists as unchecked
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Re-expand the project card
    await page.getByText(testProject.name).click();

    const slaCheckboxAfterReload = page
      .locator('label')
      .filter({ hasText: 'SLA breach' })
      .locator('input[type="checkbox"]');
    await expect(slaCheckboxAfterReload).not.toBeChecked();

    // Toggle it back on
    await slaCheckboxAfterReload.click();
    await expect(page.locator('.text-green-500')).toBeVisible({ timeout: 5000 });
  });

  test('SLA breach toggle persists via API (setProjectUserSetting)', async ({
    request,
    testUser,
    testProject,
    page,
  }) => {
    // Disable SLA breach via API
    await (await import('../../lib/api')).setProjectUserSetting(
      request,
      testUser.token,
      testProject.key,
      'notifications',
      {
        assigned_to_me: true,
        any_update_on_watched: false,
        new_item_created: false,
        comments_on_assigned: false,
        comments_on_watched: false,
        status_changes_intermediate: false,
        status_changes_final: false,
        sla_breach: false,
      },
    );

    // Navigate to the notifications preferences page
    await page.goto('/preferences/notifications');
    await page.waitForLoadState('networkidle');

    // Expand the project card
    await page.getByText(testProject.name).click();

    // Verify SLA breach is unchecked (reflecting API state)
    const slaCheckbox = page
      .locator('label')
      .filter({ hasText: 'SLA breach' })
      .locator('input[type="checkbox"]');
    await expect(slaCheckbox).not.toBeChecked();

    // Re-enable via UI
    await slaCheckbox.click();
    await expect(page.locator('.text-green-500')).toBeVisible({ timeout: 5000 });

    // Reload to verify it saved
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.getByText(testProject.name).click();

    const slaCheckboxAfter = page
      .locator('label')
      .filter({ hasText: 'SLA breach' })
      .locator('input[type="checkbox"]');
    await expect(slaCheckboxAfter).toBeChecked();
  });
});
