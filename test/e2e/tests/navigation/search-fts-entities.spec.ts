import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { openPalette, paletteInput } from '../../lib/palette';

const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

/** Helper to perform an API search and return the parsed data. */
async function searchAPI(
  request: import('@playwright/test').APIRequestContext,
  token: string,
  query: string,
  entityType?: string,
): Promise<{ fts: { results: Array<{ entity_type: string; snippet: string; entity_id: string }>; total: number }; semantic: { available: boolean } }> {
  let url = `${BASE_URL}/api/v1/search?q=${encodeURIComponent(query)}`;
  if (entityType) url += `&entity_type=${entityType}`;
  const res = await request.get(url, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok()) throw new Error(`Search failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

test.describe('FTS search for teams, queues, and milestones', () => {
  test('API: teams appear in FTS search results', async ({ request, testUser, testProject }) => {
    const unique = `FTSTeam-${Date.now()}`;
    await api.createTeam(request, testUser.token, testProject.key, {
      name: unique,
      description: 'Search indexing test team',
    });

    const data = await searchAPI(request, testUser.token, unique);
    expect(data.fts.total).toBeGreaterThanOrEqual(1);
    const teamResult = data.fts.results.find(r => r.entity_type === 'team');
    expect(teamResult).toBeDefined();
    expect(teamResult!.snippet).toContain(unique);
  });

  test('API: queues appear in FTS search results', async ({ request, testUser, testProject }) => {
    const unique = `FTSQueue-${Date.now()}`;
    await api.createQueue(request, testUser.token, testProject.key, {
      name: unique,
      queue_type: 'support',
    });

    const data = await searchAPI(request, testUser.token, unique);
    expect(data.fts.total).toBeGreaterThanOrEqual(1);
    const queueResult = data.fts.results.find(r => r.entity_type === 'queue');
    expect(queueResult).toBeDefined();
    expect(queueResult!.snippet).toContain(unique);
  });

  test('API: milestones appear in FTS search results', async ({ request, testUser, testProject }) => {
    const unique = `FTSMilestone-${Date.now()}`;
    await api.createMilestone(request, testUser.token, testProject.key, {
      name: unique,
    });

    const data = await searchAPI(request, testUser.token, unique);
    expect(data.fts.total).toBeGreaterThanOrEqual(1);
    const milestoneResult = data.fts.results.find(r => r.entity_type === 'milestone');
    expect(milestoneResult).toBeDefined();
    expect(milestoneResult!.snippet).toContain(unique);
  });

  test('API: entity_type filter restricts FTS results', async ({ request, testUser, testProject }) => {
    // Create a team and a work item with the same unique keyword
    const unique = `FTSFilter-${Date.now()}`;
    await api.createTeam(request, testUser.token, testProject.key, {
      name: unique,
    });
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: unique,
      type: 'task',
    });

    // Search with entity_type=team — should only return the team
    const teamData = await searchAPI(request, testUser.token, unique, 'team');
    expect(teamData.fts.total).toBeGreaterThanOrEqual(1);
    for (const r of teamData.fts.results) {
      expect(r.entity_type).toBe('team');
    }

    // Search with entity_type=work_item — should only return the work item
    const wiData = await searchAPI(request, testUser.token, unique, 'work_item');
    expect(wiData.fts.total).toBeGreaterThanOrEqual(1);
    for (const r of wiData.fts.results) {
      expect(r.entity_type).toBe('work_item');
    }
  });

  test('API: all entity types returned when no filter', async ({ request, testUser, testProject }) => {
    const unique = `FTSAll-${Date.now()}`;
    await api.createTeam(request, testUser.token, testProject.key, { name: unique });
    await api.createQueue(request, testUser.token, testProject.key, { name: unique, queue_type: 'general' });
    await api.createMilestone(request, testUser.token, testProject.key, { name: unique });
    await api.createWorkItem(request, testUser.token, testProject.key, { title: unique, type: 'task' });

    const data = await searchAPI(request, testUser.token, unique);
    expect(data.fts.total).toBeGreaterThanOrEqual(4);

    const types = new Set(data.fts.results.map(r => r.entity_type));
    expect(types.has('work_item')).toBe(true);
    expect(types.has('team')).toBe(true);
    expect(types.has('queue')).toBe(true);
    expect(types.has('milestone')).toBe(true);
  });

  test('UI: team search results appear in search modal', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const unique = `UITeamSearch-${Date.now()}`;
    await api.createTeam(request, testUser.token, testProject.key, {
      name: unique,
      description: 'Visible in search modal',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    // Dismiss welcome modal if visible
    const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
    if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
      await page.keyboard.press('Escape');
      await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
    }

    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open the command palette
    await openPalette(page);
    const searchInput = paletteInput(page);
    await expect(searchInput).toBeVisible({ timeout: 3000 });

    // Search for the team
    await searchInput.fill(unique);

    // Wait for FTS results to stream in
    const resultItem = page.locator('[data-search-item]').first();
    await expect(resultItem).toBeVisible({ timeout: 10000 });
    await expect(resultItem).toContainText(unique);
  });

  test('UI: clicking a team search result navigates to the team detail page', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const unique = `TeamNav-${Date.now()}`;
    const team = await api.createTeam(request, testUser.token, testProject.key, {
      name: unique,
      description: 'Navigate to team detail',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
    if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
      await page.keyboard.press('Escape');
      await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
    }
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open the command palette and find the team
    await openPalette(page);
    const searchInput = paletteInput(page);
    await expect(searchInput).toBeVisible({ timeout: 3000 });
    await searchInput.fill(unique);

    const resultItem = page.locator('[data-search-item]').first();
    await expect(resultItem).toBeVisible({ timeout: 10000 });
    await resultItem.click();

    // Should navigate to the specific team page, not the settings/teams list
    await expect(page).toHaveURL(
      new RegExp(`/d/projects/${testProject.key}/teams/${team.id}`),
      { timeout: 5000 },
    );
  });

  test('UI: duplicate results in FTS and semantic are shown only in FTS section', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const unique = `Dedup-${Date.now()}`;
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: unique,
      type: 'task',
    });

    // Intercept the search response to inject a duplicate into the semantic section
    await page.route('**/api/v1/search**', async (route) => {
      const response = await route.fetch();
      const body = await response.json();

      // Ensure the item is in FTS results
      const ftsHasItem = body.data.fts.results?.some(
        (r: { entity_id: string }) => r.entity_id === item.id,
      );
      if (ftsHasItem) {
        // Add the same item to semantic results to simulate a duplicate
        const ftsItem = body.data.fts.results.find(
          (r: { entity_id: string }) => r.entity_id === item.id,
        );
        body.data.semantic = {
          results: [{ ...ftsItem, score: 0.85 }],
          total: 1,
          available: true,
          status: 'complete',
        };
      }

      await route.fulfill({
        status: response.status(),
        headers: response.headers(),
        body: JSON.stringify(body),
      });
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
    if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
      await page.keyboard.press('Escape');
      await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
    }
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    // Open search and type the unique title
    await openPalette(page);
    const searchInput = paletteInput(page);
    await expect(searchInput).toBeVisible({ timeout: 3000 });
    await searchInput.fill(unique);

    // Wait for FTS results
    const resultItems = page.locator('[data-search-item]');
    await expect(resultItems.first()).toBeVisible({ timeout: 10000 });

    // The item should appear exactly once — in FTS only, not duplicated in semantic
    const matchingResults = resultItems.filter({ hasText: unique });
    await expect(matchingResults).toHaveCount(1);
  });
});
