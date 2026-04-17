import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

// Serial within this file — we toggle namespaces_enabled once in beforeAll.
base.describe.configure({ mode: 'serial' });

// Admin-authenticated page context — namespace management requires admin or
// namespace owner role. Admin has both.
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

test.describe('Namespace email invites', () => {
  test.beforeAll(async ({ request }) => {
    await api.enableNamespaces(request, getAdminToken());
  });

  test('inviting a non-existing email creates a pending invite', async ({ request }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const ns = await api.createNamespace(request, adminToken, `ns-${uniqueId}`, `NS ${uniqueId}`);
    const email = `noone-${uniqueId}@test.local`;

    const result = await api.createNamespaceEmailInvite(request, adminToken, ns.slug, email, 'member');
    expect(result.code).toBeTruthy();
    expect(result.invitee_email).toBe(email);
    expect(result.max_uses).toBe(1);

    const invites = await api.listNamespaceInvites(request, adminToken, ns.slug);
    expect(invites.some((inv) => inv.code === result.code)).toBe(true);

    await api.deleteNamespace(request, adminToken, ns.slug);
  });

  test('inviting a registered user creates a pending invite without auto-adding them', async ({ request }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const ns = await api.createNamespace(request, adminToken, `ns-${uniqueId}`, `NS ${uniqueId}`);

    const email = `reg-${uniqueId}@test.local`;
    const created = await api.createUser(request, adminToken, email, `Reg ${uniqueId}`);
    const tempLogin = await api.login(request, email, created.temporary_password);
    await api.changePassword(request, tempLogin.token, created.temporary_password, 'TestPass123!');

    const result = await api.createNamespaceEmailInvite(request, adminToken, ns.slug, email, 'member');
    expect(result.code).toBeTruthy();

    const members = await api.listNamespaceMembers(request, adminToken, ns.slug);
    expect(members.find((m) => m.email === email)).toBeUndefined();

    const invites = await api.listNamespaceInvites(request, adminToken, ns.slug);
    expect(invites.some((inv) => inv.code === result.code && inv.invitee_email === email)).toBe(true);

    await api.deleteNamespace(request, adminToken, ns.slug);
  });

  test('registered user can accept the namespace invite and becomes a member', async ({ request }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const ns = await api.createNamespace(request, adminToken, `ns-${uniqueId}`, `NS ${uniqueId}`);

    const email = `accept-${uniqueId}@test.local`;
    const created = await api.createUser(request, adminToken, email, `Accept ${uniqueId}`);
    const tempLogin = await api.login(request, email, created.temporary_password);
    await api.changePassword(request, tempLogin.token, created.temporary_password, 'TestPass123!');
    const userLogin = await api.login(request, email, 'TestPass123!');

    const invite = await api.createNamespaceEmailInvite(request, adminToken, ns.slug, email, 'admin');

    const accepted = await api.acceptInvite(request, userLogin.token, invite.code);
    expect(accepted.type).toBe('namespace');
    expect(accepted.namespace_slug).toBe(ns.slug);

    const members = await api.listNamespaceMembers(request, adminToken, ns.slug);
    const me = members.find((m) => m.email === email);
    expect(me).toBeTruthy();
    expect(me!.role).toBe('admin');

    await api.deleteNamespace(request, adminToken, ns.slug);
  });

  test('invite info endpoint returns namespace details for a namespace invite', async ({ request }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const ns = await api.createNamespace(request, adminToken, `ns-${uniqueId}`, `Acme ${uniqueId}`);

    const invite = await api.createNamespaceEmailInvite(request, adminToken, ns.slug, `info-${uniqueId}@test.local`, 'member');

    const info = await api.getInviteInfo(request, invite.code);
    expect(info.type).toBe('namespace');
    expect(info.namespace_slug).toBe(ns.slug);
    expect(info.namespace_display_name).toBe(`Acme ${uniqueId}`);
    expect(info.role).toBe('member');
    expect(info.expired).toBe(false);
    expect(info.full).toBe(false);

    await api.deleteNamespace(request, adminToken, ns.slug);
  });

  test('email invite sends a notification email with the invite link', async ({ request }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const ns = await api.createNamespace(request, adminToken, `ns-${uniqueId}`, `NS ${uniqueId}`);
    const email = `email-${uniqueId}@e2e.local`;

    await api.deleteMailpitMessages(request);

    const invite = await api.createNamespaceEmailInvite(request, adminToken, ns.slug, email, 'member');
    expect(invite.code).toBeTruthy();

    const msg = await api.waitForMailpitMessage(request, email, { timeoutMs: 10000 });
    expect(msg.HTML).toContain(invite.code);
    expect(msg.HTML).toContain('Accept Invite');

    await api.deleteNamespace(request, adminToken, ns.slug);
  });

  test('inviting an already-member user returns an error', async ({ request }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const ns = await api.createNamespace(request, adminToken, `ns-${uniqueId}`, `NS ${uniqueId}`);

    const email = `member-${uniqueId}@test.local`;
    const created = await api.createUser(request, adminToken, email, `Mem ${uniqueId}`);
    const tempLogin = await api.login(request, email, created.temporary_password);
    await api.changePassword(request, tempLogin.token, created.temporary_password, 'TestPass123!');
    const userLogin = await api.login(request, email, 'TestPass123!');

    const invite = await api.createNamespaceEmailInvite(request, adminToken, ns.slug, email, 'member');
    await api.acceptInvite(request, userLogin.token, invite.code);

    await expect(
      api.createNamespaceEmailInvite(request, adminToken, ns.slug, email, 'admin'),
    ).rejects.toThrow(/already a member/i);

    await api.deleteNamespace(request, adminToken, ns.slug);
  });

  test('pending namespace invite shows in the settings UI', async ({ page, request }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const ns = await api.createNamespace(request, adminToken, `ns-${uniqueId}`, `NS ${uniqueId}`);
    const email = `ui-${uniqueId}@test.local`;

    await api.createNamespaceEmailInvite(request, adminToken, ns.slug, email, 'member');

    await page.goto(`/${ns.slug}/settings?tab=users`);
    await page.waitForLoadState('networkidle');

    await expect(page.getByText(email)).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Invited')).toBeVisible({ timeout: 5000 });

    await api.deleteNamespace(request, adminToken, ns.slug);
  });

  test('accepting a project invite redirects to the project in its own namespace', async ({ page, request }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const suffix = uniqueId.slice(0, 3).toUpperCase();
    const nsSlug = `ns-${uniqueId}`;
    const projectKey = `N${suffix}`;

    // Create a non-default namespace with a project in it
    await api.createNamespace(request, adminToken, nsSlug, `NS ${uniqueId}`);
    await api.createProject(request, adminToken, projectKey, `Project ${uniqueId}`, nsSlug);

    // Create a second user who will be invited
    const inviteeEmail = `accept-redir-${uniqueId}@test.local`;
    const created = await api.createUser(request, adminToken, inviteeEmail, `Invitee ${uniqueId}`);
    const tempLogin = await api.login(request, inviteeEmail, created.temporary_password);
    await api.changePassword(request, tempLogin.token, created.temporary_password, 'TestPass123!');
    const userLogin = await api.login(request, inviteeEmail, 'TestPass123!');
    await api.setPreference(request, userLogin.token, 'welcome_dismissed', true);

    // Admin creates an email invite for the project in the custom namespace
    const invite = await request.post(
      `${process.env.BASE_URL || 'http://localhost:5173'}/api/v1/${nsSlug}/projects/${projectKey}/invites`,
      {
        headers: { Authorization: `Bearer ${adminToken}` },
        data: { role: 'member', email: inviteeEmail },
      },
    );
    if (!invite.ok()) throw new Error(`Create project email invite failed: ${await invite.text()}`);
    const inviteBody = await invite.json();
    const inviteCode = inviteBody.data.code as string;

    // Open a browser context authenticated as the invitee
    const context = await page.context().browser()!.newContext({
      storageState: {
        cookies: [],
        origins: [
          {
            origin: process.env.BASE_URL || 'http://localhost:5173',
            localStorage: [{ name: 'taskwondo_token', value: userLogin.token }],
          },
        ],
      },
    });
    const userPage = await context.newPage();

    // Visit the invite link and click Accept
    await userPage.goto(`/invite/${inviteCode}`);
    await userPage.waitForLoadState('networkidle');
    await userPage.getByRole('button', { name: /^join/i }).click();

    // Must land on the project URL under the custom namespace, not under /d/
    await userPage.waitForURL(new RegExp(`/${nsSlug}/projects/${projectKey}`), { timeout: 10000 });
    expect(userPage.url()).not.toContain('/d/projects/');

    // Cleanup
    await context.close();
    await request.delete(
      `${process.env.BASE_URL || 'http://localhost:5173'}/api/v1/${nsSlug}/projects/${projectKey}`,
      { headers: { Authorization: `Bearer ${adminToken}` } },
    );
    await api.deleteNamespace(request, adminToken, nsSlug);
  });

  test('namespace settings UI accepts email input and creates an invite', async ({ page, request }) => {
    const adminToken = getAdminToken();
    const uniqueId = randomUUID().slice(0, 8);
    const ns = await api.createNamespace(request, adminToken, `ns-${uniqueId}`, `NS ${uniqueId}`);

    await page.goto(`/${ns.slug}/settings?tab=users`);
    await page.waitForLoadState('networkidle');

    // The email input should be the one with the email placeholder (no user autocomplete)
    const emailInput = page.getByPlaceholder(/@example\.com/i);
    await expect(emailInput).toBeVisible();

    const email = `ui-invite-${uniqueId}@test.local`;
    await emailInput.fill(email);

    // Click "Send invite" in the invite form (first button)
    await page.getByRole('button', { name: /send invite/i }).first().click();

    // Confirmation modal appears
    const modal = page.getByRole('dialog');
    await expect(modal).toBeVisible();
    await expect(modal.getByText(email)).toBeVisible();

    // Confirm in the modal
    await modal.getByRole('button', { name: /send invite/i }).click();

    // After success, the invite should appear in the member list (pending invite row)
    await expect(page.getByText(email)).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Invited')).toBeVisible({ timeout: 5000 });

    // Verify via API that invite exists
    const invites = await api.listNamespaceInvites(request, adminToken, ns.slug);
    expect(invites.some((inv) => inv.invitee_email === email)).toBe(true);

    await api.deleteNamespace(request, adminToken, ns.slug);
  });
});
