import { test, expect } from '../../lib/fixtures';

async function attach(page: any, testInfo: any, name: string) {
  const screenshot = await page.screenshot();
  await testInfo.attach(name, { body: screenshot, contentType: 'image/png' });
}

async function dismissWelcomeModal(page: any) {
  const heading = page.getByRole('heading', { name: 'Welcome' });
  if (await heading.isVisible({ timeout: 2000 }).catch(() => false)) {
    const checkbox = page.getByRole('checkbox', { name: "Don't show this again" });
    if (await checkbox.isVisible({ timeout: 500 }).catch(() => false)) {
      await checkbox.check();
    }
    await page.keyboard.press('Escape');
    await heading.waitFor({ state: 'hidden', timeout: 3000 }).catch(() => {});
  }
}

async function navigateToAPIKeys(page: any) {
  await page.goto('/');
  await dismissWelcomeModal(page);
  await page.goto('/preferences/api-keys');
  await expect(page.getByRole('heading', { name: 'API Keys' })).toBeVisible();
}

async function createKey(page: any, name: string) {
  await page.getByPlaceholder('e.g. CI/CD Pipeline').fill(name);
  await page.getByRole('button', { name: 'Create Key' }).click();
  await expect(page.getByText('API key created')).toBeVisible({ timeout: 10000 });
}

test.describe('API Key Management', () => {
  test('navigate to API keys via sidebar', async ({ page }, testInfo) => {
    await page.goto('/');
    await dismissWelcomeModal(page);

    await page.goto('/preferences');
    // Should redirect to /preferences/appearance
    await expect(page).toHaveURL(/preferences\/appearance/);
    await attach(page, testInfo, '01-appearance-page');

    // Click the API Keys sidebar link
    await page.getByRole('link', { name: 'API Keys' }).click();
    await expect(page).toHaveURL(/preferences\/api-keys/);
    await attach(page, testInfo, '02-api-keys-page');

    // Verify page elements
    await expect(page.getByRole('heading', { name: 'API Keys' })).toBeVisible();
    await expect(page.getByText('No API keys yet')).toBeVisible();
  });

  test('create an API key and verify it appears in the list', async ({ page }, testInfo) => {
    await navigateToAPIKeys(page);
    await attach(page, testInfo, '01-empty-state');

    // Create the key
    await createKey(page, 'E2E Test Key');
    await expect(page.getByText('Copy your key now')).toBeVisible();

    // Verify the key value starts with twk_
    const keyCode = page.locator('.border-amber-300 code, .border-amber-700 code');
    const keyText = await keyCode.textContent();
    expect(keyText).toMatch(/^twk_/);
    await attach(page, testInfo, '02-key-revealed');

    // Dismiss the reveal card
    await page.getByRole('button', { name: 'Done' }).click();
    await expect(page.getByText('API key created')).not.toBeVisible();

    // Verify key appears in the list
    await expect(page.getByText('E2E Test Key')).toBeVisible();
    await attach(page, testInfo, '03-key-in-list');
  });

  test('create and delete an API key', async ({ page }, testInfo) => {
    await navigateToAPIKeys(page);

    // Create a key
    await createKey(page, 'Key To Delete');
    await page.getByRole('button', { name: 'Done' }).click();
    await expect(page.getByText('Key To Delete')).toBeVisible();
    await attach(page, testInfo, '01-key-created');

    // Click the trash icon button (find by the svg inside)
    await page.locator('button:has(.lucide-trash-2)').click();

    // Verify delete confirmation modal
    await expect(page.getByRole('heading', { name: 'Delete API Key' })).toBeVisible();
    await attach(page, testInfo, '02-delete-modal');

    // Confirm deletion
    await page.getByRole('button', { name: 'Delete' }).click();

    // Wait for modal to close, then verify key is gone from the list
    await expect(page.getByRole('heading', { name: 'Delete API Key' })).not.toBeVisible();
    await expect(page.locator('span').filter({ hasText: 'Key To Delete' })).not.toBeVisible();
    await attach(page, testInfo, '03-key-deleted');
  });

  test('create API key with read-only permission and expiration', async ({ page }, testInfo) => {
    await navigateToAPIKeys(page);

    // Fill form with specific options
    await page.getByPlaceholder('e.g. CI/CD Pipeline').fill('Read Only Key');

    // Select Read only permission
    const permSelect = page.locator('select').first();
    await permSelect.selectOption('read');

    // Select 7 days expiration
    const expSelect = page.locator('select').nth(1);
    await expSelect.selectOption('7');

    await attach(page, testInfo, '01-form-configured');

    // Create
    await page.getByRole('button', { name: 'Create Key' }).click();
    await expect(page.getByText('API key created')).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: 'Done' }).click();

    // Verify the key shows Read only badge (use exact locator for the badge span)
    await expect(page.locator('span.rounded-full').filter({ hasText: /^Read only$/ })).toBeVisible();
    await attach(page, testInfo, '02-key-with-permissions');
  });

  test('require name to create API key', async ({ page }, testInfo) => {
    await navigateToAPIKeys(page);

    // Try to create without a name
    await page.getByRole('button', { name: 'Create Key' }).click();

    // Verify validation error
    await expect(page.getByText('Key name is required')).toBeVisible();
    await attach(page, testInfo, '01-validation-error');
  });

  test('navigate between appearance and API keys via sidebar', async ({ page }, testInfo) => {
    await navigateToAPIKeys(page);
    await attach(page, testInfo, '01-api-keys-page');

    // Click Appearance in sidebar
    await page.getByRole('link', { name: 'Appearance' }).click();
    await expect(page).toHaveURL(/preferences\/appearance/);
    await expect(page.getByRole('heading', { name: 'Appearance' })).toBeVisible();
    await attach(page, testInfo, '02-appearance-page');

    // Navigate back to API Keys
    await page.getByRole('link', { name: 'API Keys' }).click();
    await expect(page).toHaveURL(/preferences\/api-keys/);
    await expect(page.getByRole('heading', { name: 'API Keys' })).toBeVisible();
    await attach(page, testInfo, '03-back-to-api-keys');
  });
});
