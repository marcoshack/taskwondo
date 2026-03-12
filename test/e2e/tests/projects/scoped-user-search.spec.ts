import { test, expect } from '@playwright/test';
import { getAdminToken } from '../../lib/fixtures';
import { randomUUID } from 'crypto';
import * as api from '../../lib/api';

/**
 * TF-279: Scoped user search — /api/v1/users only returns co-project members.
 */
test.describe('Scoped user search', () => {
  const TEST_PASSWORD = 'TestPass123!';
  let adminToken: string;
  let userAToken: string;
  let userAId: string;
  let userBId: string;
  let userCId: string;
  let searchUid: string;

  test.beforeAll(async ({ request }) => {
    adminToken = getAdminToken();
    const uid = randomUUID().slice(0, 8);
    searchUid = uid;

    // Create three users
    const userAData = await api.createUser(request, adminToken, `usera-${uid}@e2e.local`, `User A ${uid}`);
    userAId = userAData.user.id;
    const tempLoginA = await api.login(request, `usera-${uid}@e2e.local`, userAData.temporary_password);
    await api.changePassword(request, tempLoginA.token, userAData.temporary_password, TEST_PASSWORD);
    userAToken = (await api.login(request, `usera-${uid}@e2e.local`, TEST_PASSWORD)).token;

    const userBData = await api.createUser(request, adminToken, `userb-${uid}@e2e.local`, `User B ${uid}`);
    userBId = userBData.user.id;
    const tempLoginB = await api.login(request, `userb-${uid}@e2e.local`, userBData.temporary_password);
    await api.changePassword(request, tempLoginB.token, userBData.temporary_password, TEST_PASSWORD);

    const userCData = await api.createUser(request, adminToken, `userc-${uid}@e2e.local`, `User C ${uid}`);
    userCId = userCData.user.id;
    const tempLoginC = await api.login(request, `userc-${uid}@e2e.local`, userCData.temporary_password);
    await api.changePassword(request, tempLoginC.token, userCData.temporary_password, TEST_PASSWORD);

    // Create a project owned by User A and add User B as member
    const projKey = `S${uid.slice(0, 4).toUpperCase()}`;
    await api.createProject(request, userAToken, projKey, `Scoped Search ${uid}`);
    await api.addMember(request, userAToken, projKey, userBId, 'member');
    // User C is NOT added to any shared project
  });

  test('returns co-project members when searching', async ({ request }) => {
    const results = await api.searchUsers(request, userAToken, `userb-${searchUid}`);
    expect(results.some((u) => u.id === userBId)).toBe(true);
  });

  test('does not return users with no shared projects', async ({ request }) => {
    const results = await api.searchUsers(request, userAToken, `userc-${searchUid}`);
    expect(results.some((u) => u.id === userCId)).toBe(false);
  });

  test('does not return the caller themselves', async ({ request }) => {
    const results = await api.searchUsers(request, userAToken, `usera-${searchUid}`);
    expect(results.some((u) => u.id === userAId)).toBe(false);
  });

  test('returns results without query param (all visible users)', async ({ request }) => {
    const results = await api.searchUsers(request, userAToken);
    expect(results.some((u) => u.id === userBId)).toBe(true);
    expect(results.some((u) => u.id === userCId)).toBe(false);
  });

  test('deprecated /users/search path still works', async ({ request }) => {
    const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';
    const res = await request.get(`${BASE_URL}/api/v1/users/search?q=userb-${searchUid}`, {
      headers: { Authorization: `Bearer ${userAToken}` },
    });
    expect(res.ok()).toBe(true);
    const body = await res.json();
    expect(body.data.some((u: { id: string }) => u.id === userBId)).toBe(true);
  });
});
