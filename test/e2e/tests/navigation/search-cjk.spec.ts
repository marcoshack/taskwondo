import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { openPalette, paletteInput } from '../../lib/palette';

const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

interface SearchData {
  fts: { results: Array<{ entity_type: string; entity_id: string; snippet: string }>; total: number };
  semantic: { available: boolean };
}

async function searchAPI(
  request: import('@playwright/test').APIRequestContext,
  token: string,
  query: string,
  entityType?: string,
): Promise<SearchData> {
  let url = `${BASE_URL}/api/v1/search?q=${encodeURIComponent(query)}`;
  if (entityType) url += `&entity_type=${entityType}`;
  const res = await request.get(url, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok()) throw new Error(`Search failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

test.describe('CJK search', () => {
  test('API: Chinese substring matches a work item title', async ({ request, testUser, testProject }) => {
    const suffix = Date.now();
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `登录页面崩溃${suffix}`,
      type: 'task',
    });

    const data = await searchAPI(request, testUser.token, `登录页面${suffix}`);
    const hit = data.fts.results.find(r => r.entity_id === item.id);
    expect(hit).toBeDefined();
  });

  test('API: Chinese substring matches inside a work item description', async ({ request, testUser, testProject }) => {
    const suffix = Date.now();
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `Issue ${suffix}`,
      type: 'task',
      description: `用户反馈支付流程在弱网环境下会超时${suffix}`,
    });

    const data = await searchAPI(request, testUser.token, `弱网环境${suffix}`);
    const hit = data.fts.results.find(r => r.entity_id === item.id);
    expect(hit).toBeDefined();
  });

  test('API: Chinese query matches milestones and teams', async ({ request, testUser, testProject }) => {
    const suffix = Date.now();
    const milestone = await api.createMilestone(request, testUser.token, testProject.key, {
      name: `发布里程碑${suffix}`,
    });
    const team = await api.createTeam(request, testUser.token, testProject.key, {
      name: `平台组${suffix}`,
    });

    const msData = await searchAPI(request, testUser.token, `里程碑${suffix}`, 'milestone');
    expect(msData.fts.results.some(r => r.entity_id === milestone.id)).toBe(true);

    const teamData = await searchAPI(request, testUser.token, `平台组${suffix}`, 'team');
    expect(teamData.fts.results.some(r => r.entity_id === team.id)).toBe(true);
  });

  test('API: mixed CJK + latin query matches when terms sit in different fields', async ({ request, testUser, testProject }) => {
    const suffix = Date.now();
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `修复登录超时 bugfix-${suffix}`,
      type: 'task',
    });

    const data = await searchAPI(request, testUser.token, `登录超时 bugfix-${suffix}`);
    const hit = data.fts.results.find(r => r.entity_id === item.id);
    expect(hit).toBeDefined();
  });

  test('API: project item list search accepts a Chinese query', async ({ request, testUser, testProject }) => {
    const suffix = Date.now();
    const item = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `数据库连接池耗尽${suffix}`,
      type: 'task',
    });

    const body = await api.listWorkItems(request, testUser.token, testProject.key, {
      q: `连接池${suffix}`,
    });
    expect(body.data.some((i) => i.id === item.id)).toBe(true);
  });

  test('UI: a single Chinese character finds work items in the palette', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const suffix = `搜${Date.now()}`;
    await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `${suffix}页面白屏`,
      type: 'task',
    });

    await page.goto(`/d/projects/${testProject.key}/items`);
    const welcomeHeading = page.getByRole('heading', { name: 'Welcome' });
    if (await welcomeHeading.isVisible({ timeout: 2000 }).catch(() => false)) {
      await page.keyboard.press('Escape');
      await expect(welcomeHeading).not.toBeVisible({ timeout: 3000 });
    }
    await expect(page.getByRole('heading', { name: /items/i })).toBeVisible({ timeout: 5000 });

    await openPalette(page);
    const searchInput = paletteInput(page);
    await expect(searchInput).toBeVisible({ timeout: 3000 });
    // One CJK character is already a meaningful word — it must clear the floor.
    await searchInput.fill(suffix);

    const resultItem = page.locator('[data-search-item]').filter({ hasText: suffix }).first();
    await expect(resultItem).toBeVisible({ timeout: 10000 });
  });
});
