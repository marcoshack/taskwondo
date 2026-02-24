import { test, expect } from '../../lib/fixtures';

test.describe('Login', () => {
  // Login tests need a fresh browser with no stored auth
  test.use({ storageState: { cookies: [], origins: [] } });

  test('successful login with email and password', async ({ testUser, page }) => {
    await page.goto('/');

    await page.getByLabel('Email').fill(testUser.email);
    await page.getByLabel('Password').fill(testUser.password);
    await page.getByRole('button', { name: 'Sign in', exact: true }).click();
    await expect(page).not.toHaveURL(/login/, { timeout: 10000 });
  });

  test('shows error with invalid credentials', async ({ testUser, page }) => {
    await page.goto('/');

    await page.getByLabel('Email').fill(testUser.email);
    await page.getByLabel('Password').fill('wrongpassword');
    await page.getByRole('button', { name: 'Sign in', exact: true }).click();

    await expect(page.getByText('invalid email or password')).toBeVisible({ timeout: 5000 });
    await expect(page).toHaveURL(/login/);
  });
});
