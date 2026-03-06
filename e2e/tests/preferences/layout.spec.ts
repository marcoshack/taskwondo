import { test, expect } from '../../lib/fixtures';

async function attach(page: any, testInfo: any, name: string) {
  const screenshot = await page.screenshot();
  await testInfo.attach(name, { body: screenshot, contentType: 'image/png' });
}

async function dismissWelcomeModal(page: any) {
  const heading = page.getByRole('heading', { name: 'Welcome' });
  if (await heading.isVisible({ timeout: 1000 }).catch(() => false)) {
    const checkbox = page.getByRole('checkbox', { name: "Don't show this again" });
    if (await checkbox.isVisible({ timeout: 500 }).catch(() => false)) {
      await checkbox.check();
    }
    await page.keyboard.press('Escape');
    await heading.waitFor({ state: 'hidden', timeout: 2000 }).catch(() => {});
  }
}

test.describe('Layout Preferences', () => {
  test('switch between centered and expanded layout modes', async ({ page, testProject }, testInfo) => {
    await page.goto('/');
    await dismissWelcomeModal(page);

    // --- Default: Centered layout ---
    await page.goto('/preferences/appearance');
    await page.waitForLoadState('networkidle');

    // The Centered button should be selected by default
    const centeredBtn = page.getByRole('button', { name: /Centered/i });
    const expandedBtn = page.getByRole('button', { name: /Expanded/i });
    await expect(centeredBtn).toBeVisible();
    await expect(expandedBtn).toBeVisible();
    await attach(page, testInfo, '01-appearance-centered-default');

    // Nav bar should have max-w-7xl when centered
    const navContent = page.locator('nav > div').first();
    const navClassCentered = await navContent.getAttribute('class');
    expect(navClassCentered).toContain('max-w-7xl');

    // --- Switch to Expanded ---
    await expandedBtn.click();
    await attach(page, testInfo, '02-appearance-expanded-selected');

    // Nav bar should NOT have max-w-7xl when expanded
    const navClassExpanded = await navContent.getAttribute('class');
    expect(navClassExpanded).not.toContain('max-w-7xl');

    // Check items page in expanded mode
    await page.goto(`/projects/${testProject.key}/items`);
    await page.waitForLoadState('networkidle');
    await attach(page, testInfo, '03-items-expanded');

    // The project detail container should not have max-w-7xl
    const projectContainer = page.locator('main > div').first();
    const projectClassExpanded = await projectContainer.getAttribute('class');
    expect(projectClassExpanded).not.toContain('max-w-7xl');

    // Check inbox page in expanded mode
    await page.goto('/user/inbox');
    await page.waitForLoadState('networkidle');
    await attach(page, testInfo, '04-inbox-expanded');

    const inboxContainer = page.locator('main > div').first();
    const inboxClassExpanded = await inboxContainer.getAttribute('class');
    expect(inboxClassExpanded).not.toContain('max-w-7xl');

    // --- Switch back to Centered ---
    await page.goto('/preferences/appearance');
    await page.waitForLoadState('networkidle');
    await centeredBtn.click();
    await attach(page, testInfo, '05-appearance-centered-again');

    // Nav should be back to centered
    const navClassBack = await navContent.getAttribute('class');
    expect(navClassBack).toContain('max-w-7xl');

    // Items page should be centered again
    await page.goto(`/projects/${testProject.key}/items`);
    await page.waitForLoadState('networkidle');
    const projectClassCentered = await projectContainer.getAttribute('class');
    expect(projectClassCentered).toContain('max-w-7xl');
    await attach(page, testInfo, '06-items-centered');
  });

  test('layout persists across page navigation', async ({ page, testProject }, testInfo) => {
    await page.goto('/');
    await dismissWelcomeModal(page);

    // Set expanded layout
    await page.goto('/preferences/appearance');
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: /Expanded/i }).click();

    // Navigate to items page
    await page.goto(`/projects/${testProject.key}/items`);
    await page.waitForLoadState('networkidle');

    // Should still be expanded
    const projectContainer = page.locator('main > div').first();
    const cls = await projectContainer.getAttribute('class');
    expect(cls).not.toContain('max-w-7xl');
    await attach(page, testInfo, '01-expanded-persists');

    // Reset back to centered for other tests
    await page.goto('/preferences/appearance');
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: /Centered/i }).click();
  });
});
