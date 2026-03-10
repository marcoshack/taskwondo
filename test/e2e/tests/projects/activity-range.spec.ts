import { test, expect } from '../../lib/fixtures';

async function attach(page: any, testInfo: any, name: string) {
  const screenshot = await page.screenshot();
  await testInfo.attach(name, { body: screenshot, contentType: 'image/png' });
}

test.describe('Activity Graph Range Selector', () => {
  test('preset buttons render and are selectable', async ({ page, testProject }, testInfo) => {
    await page.goto(`/d/projects/${testProject.key}`);

    // Verify preset buttons are visible
    const btn24h = page.getByRole('button', { name: '24h' });
    const btn3d = page.getByRole('button', { name: '3d' });
    const btn7d = page.getByRole('button', { name: '7d' });

    await expect(btn24h).toBeVisible();
    await expect(btn3d).toBeVisible();
    await expect(btn7d).toBeVisible();

    await attach(page, testInfo, '01-presets-visible');

    // 7d should be active by default (has indigo background)
    await expect(btn7d).toHaveClass(/bg-indigo/);

    // Click 24h and verify it becomes active
    await btn24h.click();
    await expect(btn24h).toHaveClass(/bg-indigo/);
    await expect(btn7d).not.toHaveClass(/bg-indigo/);

    await attach(page, testInfo, '02-24h-selected');

    // Click 3d and verify
    await btn3d.click();
    await expect(btn3d).toHaveClass(/bg-indigo/);
    await expect(btn24h).not.toHaveClass(/bg-indigo/);

    await attach(page, testInfo, '03-3d-selected');
  });

  test('custom range input is present and functional', async ({ page, testProject }, testInfo) => {
    await page.goto(`/d/projects/${testProject.key}`);

    // Verify custom range inputs are visible
    const customInput = page.getByTestId('custom-range-value');
    const customUnit = page.getByTestId('custom-range-unit');

    await expect(customInput).toBeVisible();
    await expect(customUnit).toBeVisible();

    await attach(page, testInfo, '01-custom-input-visible');

    // Set custom range to 14 days
    await customInput.fill('14');
    await customUnit.selectOption('d');

    // Wait for the chart to load (no error should appear)
    await page.waitForTimeout(1000);

    // Preset buttons should not have active style when custom is active
    const btn7d = page.getByRole('button', { name: '7d' });
    await expect(btn7d).not.toHaveClass(/bg-indigo/);

    await attach(page, testInfo, '02-custom-14d-active');
  });

  test('preset clears custom range', async ({ page, testProject }, testInfo) => {
    await page.goto(`/d/projects/${testProject.key}`);

    const customInput = page.getByTestId('custom-range-value');
    const customUnit = page.getByTestId('custom-range-unit');

    // Set custom range
    await customInput.fill('30');
    await customUnit.selectOption('d');
    await page.waitForTimeout(500);

    // Click a preset — it should become active
    const btn7d = page.getByRole('button', { name: '7d' });
    await btn7d.click();
    await expect(btn7d).toHaveClass(/bg-indigo/);

    await attach(page, testInfo, '01-preset-clears-custom');
  });

  test('custom range 30d returns data without error', async ({ page, testProject }, testInfo) => {
    // Navigate to project overview
    await page.goto(`/d/projects/${testProject.key}`);

    const customInput = page.getByTestId('custom-range-value');
    const customUnit = page.getByTestId('custom-range-unit');

    // Set to 30 days
    await customInput.fill('30');
    await customUnit.selectOption('d');

    // Verify the Activity heading is still visible (no error replaced the section)
    await expect(page.getByRole('heading', { name: 'Activity' })).toBeVisible();

    // Verify either chart or "no activity data" message is shown (not an error)
    await expect(page.getByText('No activity data available yet.').or(page.locator('.recharts-responsive-container'))).toBeVisible();

    await attach(page, testInfo, '01-30d-no-error');
  });
});
