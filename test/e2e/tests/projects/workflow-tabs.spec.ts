import { test, expect } from '../../lib/fixtures';

test.describe('Workflow page tabs', () => {

  test('defaults to Workflow tab showing definitions and mapping', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/workflows`);
    await page.waitForLoadState('networkidle');

    // Page title is visible
    await expect(page.getByRole('heading', { name: 'Workflows', exact: true })).toBeVisible();

    // Workflow tab is active by default
    const workflowTab = page.getByRole('button', { name: 'Workflow', exact: true });
    await expect(workflowTab).toBeVisible();
    await expect(workflowTab).toHaveClass(/border-indigo/);

    // Workflow content is visible
    await expect(page.getByRole('heading', { name: 'Workflow Definitions' })).toBeVisible();
    await expect(page.getByText('Workflow Mapping', { exact: true })).toBeVisible();

    // Escalation content is not visible
    await expect(page.getByRole('heading', { name: 'Escalation Lists' })).not.toBeVisible();
    await expect(page.getByText('Escalation Mapping and SLA', { exact: true })).not.toBeVisible();
  });

  test('switching to Escalation tab shows escalation content', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/workflows`);
    await page.waitForLoadState('networkidle');

    // Click Escalation tab
    await page.getByRole('button', { name: 'Escalation' }).click();

    // Escalation content is visible
    await expect(page.getByRole('heading', { name: 'Escalation Lists' })).toBeVisible();
    await expect(page.getByText('Escalation Mapping and SLA', { exact: true })).toBeVisible();

    // Workflow content is not visible
    await expect(page.getByRole('heading', { name: 'Workflow Definitions' })).not.toBeVisible();
    await expect(page.getByText('Workflow Mapping', { exact: true })).not.toBeVisible();
  });

  test('switching back to Workflow tab restores workflow content', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/workflows`);
    await page.waitForLoadState('networkidle');

    // Go to Escalation
    await page.getByRole('button', { name: 'Escalation' }).click();
    await expect(page.getByRole('heading', { name: 'Escalation Lists' })).toBeVisible();

    // Switch back to Workflow
    await page.getByRole('button', { name: 'Workflow', exact: true }).click();
    await expect(page.getByRole('heading', { name: 'Workflow Definitions' })).toBeVisible();
    await expect(page.getByText('Workflow Mapping', { exact: true })).toBeVisible();

    // Escalation content hidden again
    await expect(page.getByRole('heading', { name: 'Escalation Lists' })).not.toBeVisible();
  });
});
