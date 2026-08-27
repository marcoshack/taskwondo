/**
 * Helpers for driving the Cmd/Ctrl+K command palette (TF-431).
 *
 * The palette replaced the old search modal, so the two shortcuts that used to
 * reach it are gone: `g k` (search) and `g p` (project switcher). Everything
 * that used to press those keys goes through these helpers instead — the
 * palette opens on Cmd/Ctrl+K, and the project switcher opens from the nav
 * project badge.
 */
import { expect } from '@playwright/test';
import type { Locator, Page } from '@playwright/test';

/** English `palette.placeholder`, replacing `"Search across all projects..."`. */
export const PALETTE_PLACEHOLDER = 'Search or jump to...';

/** The palette's query input. Absent from the DOM while the palette is closed. */
export function paletteInput(page: Page): Locator {
  return page.getByPlaceholder(PALETTE_PLACEHOLDER);
}

/** Every selectable palette row, navigation and entity alike. */
export function paletteRows(page: Page): Locator {
  return page.locator('[data-palette-item]');
}

/** Navigation-catalog rows only. */
export function navRows(page: Page): Locator {
  return page.locator('[data-nav-item]');
}

/** Entity-search rows only — the same attribute the old search modal used. */
export function entityRows(page: Page): Locator {
  return page.locator('[data-search-item]');
}

/**
 * Press Cmd/Ctrl+K and wait until the palette is ready for typing.
 * `ControlOrMeta` resolves to Meta on macOS and Control everywhere else, which
 * is exactly the pair `useKeyboardShortcut({ key: 'k', ctrlKey: true })` accepts.
 */
export async function openPalette(page: Page): Promise<Locator> {
  await page.keyboard.press('ControlOrMeta+k');
  const input = paletteInput(page);
  await expect(input).toBeVisible({ timeout: 5000 });
  await expect(input).toBeFocused();
  return input;
}

/** Open the palette and type a query into it. */
export async function openPaletteAndType(page: Page, query: string): Promise<Locator> {
  const input = await openPalette(page);
  await input.fill(query);
  return input;
}

/**
 * Open the project switcher from the nav project badge. Requires an active
 * project (the badge only renders when one is resolved) and a desktop viewport.
 */
export async function openProjectSwitcher(page: Page): Promise<Locator> {
  await page.getByTestId('project-switcher-badge').click();
  const modal = page.getByRole('dialog');
  await expect(modal.getByRole('textbox', { name: /search projects/i })).toBeVisible({
    timeout: 5000,
  });
  return modal;
}

/** Open the project switcher and pick the row matching `projectKey`. */
export async function switchProject(page: Page, projectKey: string): Promise<void> {
  const modal = await openProjectSwitcher(page);
  await modal.getByRole('textbox', { name: /search projects/i }).fill(projectKey);
  const row = modal.getByRole('button', { name: new RegExp(projectKey, 'i') });
  await expect(row).toBeVisible({ timeout: 3000 });
  await row.click();
}
