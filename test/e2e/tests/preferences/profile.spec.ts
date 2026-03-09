import { test, expect } from '../../lib/fixtures';
import path from 'path';

async function attach(page: any, testInfo: any, name: string) {
  const screenshot = await page.screenshot();
  await testInfo.attach(name, { body: screenshot, contentType: 'image/png' });
}

test.describe('Profile Preferences', () => {
  test('default redirect goes to profile page', async ({ page }) => {
    await page.goto('/preferences');
    await expect(page).toHaveURL(/\/preferences\/profile/);
  });

  test('update display name', async ({ page, testUser }, testInfo) => {
    await page.goto('/preferences/profile');
    await attach(page, testInfo, '01-profile-page');

    // Clear and type new name
    const nameInput = page.getByPlaceholder('Enter your display name');
    await expect(nameInput).toBeVisible();
    await expect(nameInput).toHaveValue(testUser.displayName);

    const newName = `Updated ${testUser.displayName}`;
    await nameInput.clear();
    await nameInput.fill(newName);

    // Click save
    const saveButton = page.getByRole('button', { name: 'Save' });
    await saveButton.click();

    // Wait for the green checkmark to appear (success feedback)
    await expect(page.locator('.text-green-500')).toBeVisible({ timeout: 5000 });
    await attach(page, testInfo, '02-name-saved');

    // Reload and verify persistence
    await page.reload();
    await expect(page.getByPlaceholder('Enter your display name')).toHaveValue(newName);
    await attach(page, testInfo, '03-name-persisted');
  });

  test('upload and remove avatar', async ({ page }, testInfo) => {
    await page.goto('/preferences/profile');

    // Upload an avatar image
    const fileInput = page.locator('input[type="file"]');
    const testImage = path.join(__dirname, '../../fixtures/test-avatar.png');
    await fileInput.setInputFiles(testImage);

    // Crop modal should appear
    await expect(page.getByText('Crop Profile Picture')).toBeVisible({ timeout: 5000 });
    await attach(page, testInfo, '01-crop-modal');

    // Click save in crop modal
    const cropSaveButton = page.getByRole('button', { name: 'Save' }).last();
    await cropSaveButton.click();

    // Wait for modal to close and avatar to update
    await expect(page.getByText('Crop Profile Picture')).not.toBeVisible({ timeout: 10000 });
    await attach(page, testInfo, '02-avatar-uploaded');

    // Remove picture button should appear once avatar is uploaded
    const removeButton = page.getByRole('button', { name: 'Remove picture' });
    await expect(removeButton).toBeVisible({ timeout: 5000 });

    // Remove avatar
    await removeButton.click();

    // Confirmation modal should appear
    await expect(page.getByText('Are you sure you want to remove your profile picture?')).toBeVisible();
    await attach(page, testInfo, '03-remove-confirm');

    // Confirm removal
    await page.getByRole('button', { name: 'Remove picture' }).last().click();

    // Wait for modal to close
    await expect(page.getByText('Are you sure you want to remove your profile picture?')).not.toBeVisible({ timeout: 5000 });
    await attach(page, testInfo, '04-avatar-removed');
  });

  test('sidebar navigation shows profile link', async ({ page }) => {
    await page.goto('/preferences/profile');

    // Desktop sidebar should show Profile nav item
    const profileLink = page.locator('nav').getByRole('link', { name: 'Profile' }).first();
    await expect(profileLink).toBeVisible();
  });
});
