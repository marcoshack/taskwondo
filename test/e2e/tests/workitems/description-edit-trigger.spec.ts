import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';

test.describe('Description edit trigger', () => {
  test('single-click on description body does NOT enter edit mode', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `DescDblClick-${Date.now()}`,
      type: 'task',
      description: 'Initial description content for the test.',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    const descArea = page.locator('.prose').first();
    await expect(descArea).toBeVisible({ timeout: 3000 });

    await descArea.click();

    // Edit textarea should NOT appear after a single click
    const descTextarea = page.locator('textarea').filter({ hasNot: page.locator(':scope[placeholder*="comment" i]') }).first();
    await expect(descTextarea).not.toBeVisible({ timeout: 1000 });
    await expect(descArea).toBeVisible();
  });

  test('double-click on description body enters edit mode', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `DescDblClick-${Date.now()}`,
      type: 'task',
      description: 'Initial description content for the test.',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    const descArea = page.locator('.prose').first();
    await expect(descArea).toBeVisible({ timeout: 3000 });

    await descArea.dblclick();

    const descTextarea = page.locator('textarea').first();
    await expect(descTextarea).toBeVisible({ timeout: 3000 });
    await expect(descTextarea).toHaveValue('Initial description content for the test.');
  });

  test('pencil edit button still enters edit mode on single click', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `DescPencil-${Date.now()}`,
      type: 'task',
      description: 'Pencil-button click test description.',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    // Hover the description heading area to reveal the pencil button
    const descHeading = page.getByRole('heading', { name: /description/i }).first();
    await descHeading.hover();

    // The pencil button is the sibling button with the common.edit tooltip
    const pencilButton = page.locator('button[class*="group/edit"]').first();
    await expect(pencilButton).toBeVisible({ timeout: 3000 });

    await pencilButton.click();

    const descTextarea = page.locator('textarea').first();
    await expect(descTextarea).toBeVisible({ timeout: 3000 });
    await expect(descTextarea).toHaveValue('Pencil-button click test description.');
  });

  test('single-click on empty-description placeholder does NOT enter edit mode', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `DescEmpty-${Date.now()}`,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    const noDesc = page.getByText(/no description/i);
    await expect(noDesc).toBeVisible({ timeout: 3000 });

    await noDesc.click();

    // Should still see the placeholder, not the textarea
    await expect(noDesc).toBeVisible({ timeout: 1000 });
    const descTextarea = page.locator('textarea').filter({ hasNot: page.locator(':scope[placeholder*="comment" i]') }).first();
    await expect(descTextarea).not.toBeVisible({ timeout: 1000 });
  });

  test('double-click on empty-description placeholder enters edit mode', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `DescEmptyDbl-${Date.now()}`,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items/${item.item_number}`);
    await expect(page.getByText(item.title)).toBeVisible({ timeout: 5000 });

    const noDesc = page.getByText(/no description/i);
    await expect(noDesc).toBeVisible({ timeout: 3000 });

    await noDesc.dblclick();

    const descTextarea = page.locator('textarea').first();
    await expect(descTextarea).toBeVisible({ timeout: 3000 });
  });
});
