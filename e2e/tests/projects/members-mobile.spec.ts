import { test, expect } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { getAdminToken } from '../../lib/fixtures';
import { randomUUID } from 'crypto';

test.describe('Project members mobile', () => {
  test('member list does not overflow on mobile viewport', async ({ page, request, testUser, testProject }) => {
    // Add a few members so the list has content
    const adminToken = getAdminToken();
    const members = [];
    for (let i = 0; i < 3; i++) {
      const uid = randomUUID().slice(0, 8);
      const created = await api.createUser(request, adminToken, `mob-${uid}@e2e.local`, `Mobile Member ${uid}`);
      const login = await api.login(request, created.user.email, created.temporary_password);
      await api.addMember(request, testUser.token, testProject.key, login.user.id, 'member');
      members.push(login.user);
    }

    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`/d/projects/${testProject.key}/settings`);

    // Wait for member list to render
    await expect(page.getByText('Mobile Member').first()).toBeVisible({ timeout: 10000 });

    // The page container should not have horizontal overflow
    const hasOverflow = await page.evaluate(() => {
      return document.documentElement.scrollWidth > document.documentElement.clientWidth;
    });
    expect(hasOverflow).toBe(false);

    // Each member row should be fully visible (not clipped)
    const memberRows = page.locator('[class*="divide-y"] > div').filter({ hasText: 'Mobile Member' });
    const count = await memberRows.count();
    expect(count).toBeGreaterThanOrEqual(3);

    for (let i = 0; i < count; i++) {
      const box = await memberRows.nth(i).boundingBox();
      expect(box).not.toBeNull();
      // Row should fit within the viewport width
      expect(box!.x + box!.width).toBeLessThanOrEqual(375 + 1); // 1px tolerance
    }
  });
});
