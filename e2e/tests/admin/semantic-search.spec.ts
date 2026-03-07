import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';

// Admin context — feature toggles require admin role
const test = base.extend({
  storageState: async ({}, use) => {
    const adminToken = getAdminToken();
    const state = {
      cookies: [],
      origins: [
        {
          origin: process.env.BASE_URL || 'http://localhost:5173',
          localStorage: [{ name: 'taskwondo_token', value: adminToken }],
        },
      ],
    };
    await use(state as any);
  },
});

test.describe.configure({ mode: 'serial' });

async function attach(page: any, testInfo: any, name: string) {
  const screenshot = await page.screenshot();
  await testInfo.attach(name, { body: screenshot, contentType: 'image/png' });
}

test.describe('Feature Toggle — Semantic Search', () => {
  test('features page displays semantic search toggle', async ({ page }, testInfo) => {
    await page.goto('/admin/features');
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('heading', { name: 'Features' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Semantic Search' })).toBeVisible();

    await attach(page, testInfo, '01-semantic-search-card');
  });

  test('semantic search toggle is disabled when Ollama is unavailable', async ({ page }, testInfo) => {
    await page.goto('/admin/features');
    await page.waitForLoadState('networkidle');

    // Find the semantic search section — it should contain status text about Ollama
    const semanticSearchSection = page.locator('div', { hasText: 'Semantic Search' }).last();
    await expect(semanticSearchSection).toBeVisible();

    // The toggle in the semantic search section should be disabled (Ollama not running in E2E env)
    const toggles = page.getByRole('switch');
    // There should be at least 2 toggles (Activity Graph + Semantic Search)
    const count = await toggles.count();
    expect(count).toBeGreaterThanOrEqual(2);

    // The second toggle (Semantic Search) should be disabled since Ollama is not available
    const semanticToggle = toggles.nth(1);
    await expect(semanticToggle).toBeDisabled();

    await attach(page, testInfo, '02-toggle-disabled');
  });

  test('search API returns 200 with FTS results when semantic is disabled', async ({ request }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

    const res = await request.get(`${BASE_URL}/api/v1/search?q=test`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });

    // FTS always works regardless of semantic search feature flag
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.data.fts).toBeDefined();
    expect(body.data.semantic.available).toBe(false);
  });

  test('search API returns 400 for missing query', async ({ request }) => {
    const adminToken = getAdminToken();
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

    const res = await request.get(`${BASE_URL}/api/v1/search`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });

    expect(res.status()).toBe(400);
  });
});
