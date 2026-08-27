import { test, expect } from '../../lib/fixtures';
import { openPalette, openProjectSwitcher, paletteInput } from '../../lib/palette';

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

test.describe('Keyboard shortcuts', () => {
  test('g then i navigates to inbox from project items', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    // Wait for page content to be ready
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('g');
    await page.keyboard.press('i');

    await expect(page).toHaveURL(/\/user\/inbox/, { timeout: 5000 });
  });

  test('g then i navigates to inbox from project list', async ({ page }) => {
    await page.goto('/d/projects');
    await dismissWelcomeModal(page);

    // Wait for page content to be ready
    await expect(page.getByRole('heading', { name: /projects/i })).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('g');
    await page.keyboard.press('i');

    await expect(page).toHaveURL(/\/user\/inbox/, { timeout: 5000 });
  });

  // TF-431 retired `g p` (project switcher) and `g k` (search): both now live
  // behind the command palette / the nav project badge. The sequence keys must
  // do nothing at all — not fall through to the bare single-key shortcuts
  // either, which is what `p` and `k` would otherwise hit on a list page.
  test('g then p no longer opens the project switcher', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('g');
    await page.keyboard.press('p');
    // The sequence window is 800 ms; let it lapse before asserting.
    await page.waitForTimeout(1000);

    await expect(page.getByRole('dialog')).toHaveCount(0);
    await expect(page.getByPlaceholder(/search projects/i)).toHaveCount(0);

    // Positive control: the switcher itself still opens, from the nav badge.
    await openProjectSwitcher(page);
  });

  test('g then k no longer opens the command palette', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('g');
    await page.keyboard.press('k');
    await page.waitForTimeout(1000);

    await expect(paletteInput(page)).toHaveCount(0);

    // Positive control: Cmd/Ctrl+K still opens it.
    await openPalette(page);
  });

  test('? opens keyboard shortcuts modal listing Cmd/Ctrl+K and not g p / g k', async ({
    page,
    testProject,
  }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);

    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('?');

    // Shortcuts modal should open
    await expect(page.getByRole('heading', { name: 'Keyboard Shortcuts' })).toBeVisible({ timeout: 5000 });

    const modal = page.getByRole('dialog');

    // The palette row: label, both keycaps, and the chord separator. The keycap
    // renders the literal "Ctrl/⌘" on every OS — it is deliberately not
    // platform-sniffed, so one fixed string is correct everywhere.
    const paletteRow = modal.locator('div', { hasText: /^Open command palette/ }).last();
    await expect(paletteRow).toBeVisible();
    await expect(paletteRow.locator('kbd', { hasText: 'Ctrl/\u2318' })).toHaveCount(1);
    await expect(paletteRow.locator('kbd', { hasText: 'K' })).toHaveCount(1);
    await expect(paletteRow.getByText('+', { exact: true })).toHaveCount(1);

    // The retired rows are gone.
    await expect(modal.getByText('Switch project', { exact: true })).toHaveCount(0);
    await expect(modal.getByText('Global search', { exact: true })).toHaveCount(0);

    // The sequences that survived are still listed.
    await expect(modal.getByText('Go to items', { exact: true })).toBeVisible();
    await expect(modal.getByText('Go to inbox', { exact: true })).toBeVisible();
  });
});
