import { test, expect, getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

const TEST_PASSWORD = 'TestPass123!';

/**
 * TF-347: Unified Add-user UI for project Settings/Users tab.
 *
 * The Users tab exposes a single search input. Selecting a namespace member
 * stages a chip; clicking Add commits all staged chips with the chosen role.
 * Typing an unknown email surfaces an "Invite by email" row that opens the
 * existing confirmation modal and creates a pending project invite.
 */
test.describe('Unified Add-user — TF-347', () => {
  test('staging two chips and clicking Add adds both as project members', async ({
    page,
    request,
    testUser,
    testProject,
  }) => {
    const adminToken = getAdminToken();
    const uid = randomUUID().slice(0, 4);

    // Create two ready users that we'll stage as chips.
    const cand1Email = `cand1-${uid}@e2e.local`;
    const cand1Name = `Candidate One ${uid}`;
    const cand1 = await api.createUser(request, adminToken, cand1Email, cand1Name);
    const cand1TempLogin = await api.login(request, cand1Email, cand1.temporary_password);
    await api.changePassword(request, cand1TempLogin.token, cand1.temporary_password, TEST_PASSWORD);

    const cand2Email = `cand2-${uid}@e2e.local`;
    const cand2Name = `Candidate Two ${uid}`;
    const cand2 = await api.createUser(request, adminToken, cand2Email, cand2Name);
    const cand2TempLogin = await api.login(request, cand2Email, cand2.temporary_password);
    await api.changePassword(request, cand2TempLogin.token, cand2.temporary_password, TEST_PASSWORD);

    // Add both candidates as namespace members of the default namespace so the
    // unified UI classifies them as "addable" (chip), not "invite by email".
    await api.addNamespaceMember(request, adminToken, 'default', cand1.user.id, 'member');
    await api.addNamespaceMember(request, adminToken, 'default', cand2.user.id, 'member');

    // The owner of testProject must share at least one project with each
    // candidate so they show up in the search (search is scoped to co-project
    // members). Create a seed project owned by admin and add everyone to it.
    const seedKey = `S${uid.toUpperCase()}`;
    await api.createProject(request, adminToken, seedKey, `Seed ${uid}`);
    await api.addMember(request, adminToken, seedKey, testUser.id, 'member');
    await api.addMember(request, adminToken, seedKey, cand1.user.id, 'member');
    await api.addMember(request, adminToken, seedKey, cand2.user.id, 'member');

    // Open the project settings — Users tab.
    await page.goto(`/d/projects/${testProject.key}/settings?tab=users`);
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByPlaceholder('Search by name or email...');
    await expect(searchInput).toBeVisible();

    // Stage candidate one
    await searchInput.fill(cand1Name);
    const cand1Row = page.getByRole('button', { name: new RegExp(cand1Name) });
    await expect(cand1Row).toBeVisible({ timeout: 5000 });
    await cand1Row.click();

    // Chip for candidate one should appear
    await expect(page.getByText(cand1Name, { exact: false })).toBeVisible();

    // The input should be cleared and ready for the next search
    await expect(searchInput).toHaveValue('');

    // Stage candidate two
    await searchInput.fill(cand2Name);
    const cand2Row = page.getByRole('button', { name: new RegExp(cand2Name) });
    await expect(cand2Row).toBeVisible({ timeout: 5000 });
    await cand2Row.click();

    await expect(page.getByText(cand2Name, { exact: false })).toBeVisible();

    // Click Add to commit both chips
    await page.getByRole('button', { name: /^Add$/ }).click();

    // The success checkmark should appear
    await expect(page.locator('.text-green-500').first()).toBeVisible({ timeout: 5000 });

    // Both candidates should now appear in the member list
    await expect(page.getByText(cand1Email)).toBeVisible({ timeout: 5000 });
    await expect(page.getByText(cand2Email)).toBeVisible({ timeout: 5000 });

    // Verify membership via API
    const members = await api.listMembers(request, testUser.token, testProject.key);
    const memberEmails = members.map((m: any) => m.email);
    expect(memberEmails).toContain(cand1Email);
    expect(memberEmails).toContain(cand2Email);

    // Cleanup
    await api.deactivateUser(request, adminToken, cand1.user.id).catch(() => {});
    await api.deactivateUser(request, adminToken, cand2.user.id).catch(() => {});
  });

  test('search returns namespace_slugs field with non-empty array for namespace members', async ({
    request,
    testUser,
  }) => {
    const adminToken = getAdminToken();
    const uid = randomUUID().slice(0, 4);

    // Create a candidate, add them to the default namespace, and to a shared project
    const candEmail = `nsmem-${uid}@e2e.local`;
    const candName = `NsMem ${uid}`;
    const cand = await api.createUser(request, adminToken, candEmail, candName);
    const tempLogin = await api.login(request, candEmail, cand.temporary_password);
    await api.changePassword(request, tempLogin.token, cand.temporary_password, TEST_PASSWORD);
    await api.addNamespaceMember(request, adminToken, 'default', cand.user.id, 'member');

    const seedKey = `N${uid.toUpperCase()}`;
    await api.createProject(request, adminToken, seedKey, `NsSeed ${uid}`);
    await api.addMember(request, adminToken, seedKey, testUser.id, 'member');
    await api.addMember(request, adminToken, seedKey, cand.user.id, 'member');

    // Search via the public users API as testUser
    const results = await api.searchUsers(request, testUser.token, `nsmem-${uid}`);
    const found = results.find((u) => u.id === cand.user.id);
    expect(found).toBeTruthy();
    expect(found!.namespace_slugs).toContain('default');

    // Cleanup
    await api.deactivateUser(request, adminToken, cand.user.id).catch(() => {});
  });
});
