import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

async function waitForTable(page: import('@playwright/test').Page) {
  await expect(page.getByRole('table')).toBeVisible({ timeout: 10000 });
}

test.describe('Work Items Refresh Button', () => {
  test('refresh button manual refresh', async ({ request, testUser, testProject, page }) => {
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'WI refresh manual',
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await waitForTable(page);
    await expect(page.getByRole('table').getByText('WI refresh manual')).toBeVisible({ timeout: 10000 });

    const refreshBtn = page.getByRole('button', { name: 'Refresh', exact: true });
    await expect(refreshBtn).toBeVisible();
    await refreshBtn.click();

    // Item still visible after refresh
    await expect(page.getByRole('table').getByText('WI refresh manual')).toBeVisible();
  });

  test('refresh dropdown lets the user pick an interval and persists it', async ({ request, testUser, testProject, page }) => {
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'WI refresh persisted',
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await waitForTable(page);
    await expect(page.getByRole('table').getByText('WI refresh persisted')).toBeVisible({ timeout: 10000 });

    // Open dropdown and verify all options
    await page.getByRole('button', { name: 'Auto-refresh' }).click();
    await expect(page.getByRole('button', { name: 'Off' })).toBeVisible({ timeout: 3000 });
    await expect(page.getByRole('button', { name: '5s' })).toBeVisible();
    await expect(page.getByRole('button', { name: '10s' })).toBeVisible();
    await expect(page.getByRole('button', { name: '30s' })).toBeVisible();
    await expect(page.getByRole('button', { name: '1m', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: '5m', exact: true })).toBeVisible();

    await page.getByRole('button', { name: '30s' }).click();
    await expect(page.getByRole('button', { name: 'Refresh', exact: true }).getByText('30s')).toBeVisible();

    // Reload — interval should persist via workitems_refresh_interval preference
    await page.reload();
    await dismissWelcomeModal(page);
    await waitForTable(page);
    await expect(page.getByRole('button', { name: 'Refresh', exact: true }).getByText('30s')).toBeVisible({ timeout: 5000 });
  });

  test('selecting Off disables auto-refresh and reverts label', async ({ request, testUser, testProject, page }) => {
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: 'WI refresh off',
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await waitForTable(page);
    await expect(page.getByRole('table').getByText('WI refresh off')).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: 'Auto-refresh' }).click();
    await page.getByRole('button', { name: '10s' }).click();
    await expect(page.getByRole('button', { name: 'Refresh', exact: true }).getByText('10s')).toBeVisible();

    await page.getByRole('button', { name: 'Auto-refresh' }).click();
    await page.getByRole('button', { name: 'Off' }).click();
    await expect(page.getByRole('button', { name: 'Refresh', exact: true }).getByText('Refresh')).toBeVisible();
  });
});
