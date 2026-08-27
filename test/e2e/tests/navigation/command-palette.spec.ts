import { test as base, expect, getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';
import {
  navRows,
  openPalette,
  openPaletteAndType,
  openProjectSwitcher,
  paletteInput,
} from '../../lib/palette';

/**
 * The Cmd/Ctrl+K command palette (TF-431): how it opens and closes, and the
 * navigation catalog half that replaced the old search modal's "type at least
 * 2 characters" hint.
 *
 * Entity-search behaviour lives in `search-modal.spec.ts`, which the palette
 * inherited; the namespace half of the catalog lives in
 * `command-palette-namespace.spec.ts` (the `namespace` Playwright project).
 */

const test = base;

async function dismissWelcomeModal(page: import('@playwright/test').Page) {
  const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
  if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
    await page.keyboard.press('Escape');
    await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
  }
}

/** A navigation row whose whole label is exactly `label`. */
function navRow(page: import('@playwright/test').Page, label: string) {
  return page.locator('[data-nav-item]', { hasText: new RegExp(`^${label}$`) });
}

test.describe('Command palette — opening and closing', () => {
  test('Cmd/Ctrl+K opens the palette from a project page and a second press closes it', async ({
    page,
    testProject,
  }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await openPalette(page);

    // The global handler never sees this press — KeyboardShortcutContext is
    // muted while a modal is open and again for INPUT targets — so the palette
    // has to answer it from its own input.
    await page.keyboard.press('ControlOrMeta+k');
    await expect(paletteInput(page)).toHaveCount(0);
  });

  test('Cmd/Ctrl+K opens the palette from a non-project page', async ({ page }) => {
    await page.goto('/preferences/profile');
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /profile/i }).first()).toBeVisible({
      timeout: 10000,
    });

    await openPalette(page);
    await expect(navRows(page).first()).toBeVisible({ timeout: 3000 });
  });

  test('Escape closes the palette', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await openPalette(page);
    await page.keyboard.press('Escape');
    await expect(paletteInput(page)).toHaveCount(0);
  });
});

test.describe('Command palette — navigation catalog', () => {
  test('an empty query lists the catalog, including the active project sections', async ({
    page,
    testProject,
  }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await openPalette(page);

    // Active project sections, in the order AppSidebar renders them.
    for (const label of ['Overview', 'Items', 'Milestones', 'Queues', 'Workflows', 'Settings']) {
      await expect(navRow(page, label)).toHaveCount(1);
    }
    // Personal pages and preferences, which do not depend on a project.
    for (const label of ['Projects', 'Inbox', 'Watchlist', 'Feed', 'Profile', 'Appearance']) {
      await expect(navRow(page, label)).toHaveCount(1);
    }

    // The project group is headed by the active project's key.
    await expect(
      page.getByRole('dialog').getByText(`Project ${testProject.key}`, { exact: true }),
    ).toBeVisible();
  });

  test('filtering on "milestone" reaches the active project milestones page', async ({
    page,
    testProject,
  }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await openPaletteAndType(page, 'milestone');

    const row = navRow(page, 'Milestones');
    await expect(row).toHaveCount(1);
    await row.click();

    await expect(page).toHaveURL(new RegExp(`/d/projects/${testProject.key}/milestones$`), {
      timeout: 5000,
    });
    await expect(page.getByRole('heading', { name: /milestones/i }).first()).toBeVisible({
      timeout: 5000,
    });
    // The palette closes on activation.
    await expect(paletteInput(page)).toHaveCount(0);
  });

  test('a preferences page is reachable from the palette', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await openPaletteAndType(page, 'appearance');

    const row = navRow(page, 'Appearance');
    await expect(row).toHaveCount(1);
    await row.click();

    await expect(page).toHaveURL(/\/preferences\/appearance$/, { timeout: 5000 });
  });

  test('a plain member sees no System Settings rows', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await openPaletteAndType(page, 'Directory');

    // "Directory" is the one System Settings label that no other sidebar
    // shares, so a member matching it at all would be an admin-gate leak.
    await expect(navRow(page, 'Directory')).toHaveCount(0);
    await expect(
      page.getByRole('dialog').getByText('System Settings', { exact: true }),
    ).toHaveCount(0);
  });

  test('another project is still reachable from the palette', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    // Entity search is scoped to the active project (TF-432/434). `project`
    // hits stay global on the backend — there are Go tests for that carve-out —
    // but they only exist in the semantic index, which the E2E stack runs
    // without, so the palette's route to another project here is the catalog's
    // "Projects" row. That row is deliberately outside the active-project gate
    // precisely because `g p` is gone.
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const other = await api.createProject(
      request,
      testUser.token,
      `O${suffix}`,
      `E2E Other ${suffix}`,
    );

    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await openPaletteAndType(page, 'Projects');

    const row = navRow(page, 'Projects');
    await expect(row).toHaveCount(1);
    await row.click();

    await expect(page).toHaveURL(/\/d\/projects\/?$/, { timeout: 5000 });
    // The desktop table, not the `sm:hidden` card list that is also in the DOM.
    await expect(page.getByRole('row', { name: new RegExp(other.name) })).toBeVisible({
      timeout: 10000,
    });
  });

  test('the project badge still opens the project switcher', async ({ page, testProject }) => {
    await page.goto(`/d/projects/${testProject.key}/items`);
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    const modal = await openProjectSwitcher(page);
    await expect(modal.getByRole('heading', { name: 'Switch Project' })).toBeVisible();
  });
});

// --- The admin gate, from the admin's side ---

const adminTest = base.extend({
  storageState: async ({}, use) => {
    const state = {
      cookies: [],
      origins: [
        {
          origin: process.env.BASE_URL || 'http://localhost:5173',
          localStorage: [{ name: 'taskwondo_token', value: getAdminToken() }],
        },
      ],
    };
    await use(state as any);
  },
});

adminTest.describe('Command palette — system admin gate', () => {
  adminTest.beforeAll(async ({ request }) => {
    await api.setPreference(request, getAdminToken(), 'welcome_dismissed', true);
  });

  adminTest('a system admin sees System Settings rows and can open one', async ({ page }) => {
    await page.goto('/d/projects');
    await dismissWelcomeModal(page);
    await expect(page.getByRole('heading', { name: /projects/i }).first()).toBeVisible({
      timeout: 10000,
    });

    await openPaletteAndType(page, 'Directory');

    const row = navRow(page, 'Directory');
    await expect(row).toHaveCount(1);
    await expect(
      page.getByRole('dialog').getByText('System Settings', { exact: true }),
    ).toBeVisible();

    await row.click();
    await expect(page).toHaveURL(/\/admin\/directory$/, { timeout: 5000 });
  });
});
