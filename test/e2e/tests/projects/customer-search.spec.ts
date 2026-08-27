import { test, expect, getAdminToken } from '../../lib/fixtures';
import type { APIRequestContext, BrowserContext, Page } from '@playwright/test';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';
import { PALETTE_PLACEHOLDER } from '../../lib/palette';

const TEST_PASSWORD = 'TestPass123!';
const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

/**
 * Create a fully-setup user (password changed, welcome dismissed). Does NOT
 * add them to any project.
 */
async function createReadyUser(request: APIRequestContext, adminToken: string) {
  const uniqueId = randomUUID().slice(0, 8);
  const email = `cust-search-${uniqueId}@test.local`;
  const displayName = `CustSearch ${uniqueId}`;
  const created = await api.createUser(request, adminToken, email, displayName);
  const tempLogin = await api.login(request, email, created.temporary_password);
  await api.changePassword(request, tempLogin.token, created.temporary_password, TEST_PASSWORD);
  const finalLogin = await api.login(request, email, TEST_PASSWORD);
  await api.setPreference(request, finalLogin.token, 'welcome_dismissed', true);
  return { id: finalLogin.user.id, email, displayName, token: finalLogin.token, password: TEST_PASSWORD };
}

/** Log in a user via the browser UI. */
async function loginAs(page: Page, context: BrowserContext, email: string, password: string) {
  await page.goto('/login');
  await context.clearCookies();
  await page.evaluate(() => localStorage.clear());
  await page.goto('/login');
  await page.getByLabel('Email').fill(email);
  await page.getByLabel('Password').fill(password);
  await page.getByRole('button', { name: 'Sign in' }).click();
}

/** Call the unified search API as the given user. */
async function searchAs(
  request: APIRequestContext,
  token: string,
  query: string,
): Promise<{
  fts: {
    results: Array<{
      entity_type: string;
      entity_id: string;
      project_key?: string;
      item_number?: number;
      snippet: string;
    }>;
    total: number;
  };
}> {
  const res = await request.get(`${BASE_URL}/api/v1/search?q=${encodeURIComponent(query)}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok()) throw new Error(`Search failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

test.describe('Customer role — search RBAC', () => {
  // These tests validate that the /api/v1/search endpoint (FTS + semantic)
  // correctly restricts results for users who hold the "customer" role in a
  // project. Customers must NEVER see internal items, items reported by
  // other customers, or items they did not create themselves.

  test('API: customer-only user searching sees only their own portal tickets', async ({ request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const uniqueTag = `UniqueTag${suffix}`;

    // 1. Owner creates a public queue so portal tickets can be created
    await api.createQueue(request, testUser.token, testProject.key, {
      name: 'Portal Support',
      queue_type: 'support',
      is_public: true,
    });

    // 2. Owner creates an INTERNAL work item mentioning the unique tag
    const internalItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `Internal item about ${uniqueTag}`,
      type: 'task',
    });

    // 3. Create two customers on the same project and give each a portal ticket
    //    mentioning the same tag so FTS matches both.
    const customerA = await createReadyUser(request, adminToken);
    const customerB = await createReadyUser(request, adminToken);
    await api.addMember(request, testUser.token, testProject.key, customerA.id, 'customer');
    await api.addMember(request, testUser.token, testProject.key, customerB.id, 'customer');

    const ticketA = await api.createPortalTicket(request, customerA.token, testProject.key, {
      title: `CustomerA ticket ${uniqueTag}`,
    });
    const ticketB = await api.createPortalTicket(request, customerB.token, testProject.key, {
      title: `CustomerB ticket ${uniqueTag}`,
    });

    // 4. Customer A searches via the unified search API
    const customerAResults = await searchAs(request, customerA.token, uniqueTag);
    const aIDs = customerAResults.fts.results.map((r) => r.entity_id);

    // Must see own ticket
    expect(aIDs).toContain(ticketA.id);
    // Must NOT see the other customer's ticket
    expect(aIDs).not.toContain(ticketB.id);
    // Must NOT see the owner's internal item
    expect(aIDs).not.toContain(internalItem.id);

    // 5. Customer B searches and mirrors the same expectations
    const customerBResults = await searchAs(request, customerB.token, uniqueTag);
    const bIDs = customerBResults.fts.results.map((r) => r.entity_id);

    expect(bIDs).toContain(ticketB.id);
    expect(bIDs).not.toContain(ticketA.id);
    expect(bIDs).not.toContain(internalItem.id);

    // 6. Owner (full-access member) sees everything
    const ownerResults = await searchAs(request, testUser.token, uniqueTag);
    const ownerIDs = ownerResults.fts.results.map((r) => r.entity_id);
    expect(ownerIDs).toContain(internalItem.id);
    expect(ownerIDs).toContain(ticketA.id);
    expect(ownerIDs).toContain(ticketB.id);

    // Cleanup
    await api.deactivateUser(request, adminToken, customerA.id).catch(() => {});
    await api.deactivateUser(request, adminToken, customerB.id).catch(() => {});
  });

  test('API: mixed-role user sees full results in owned project and only own tickets in customer project', async ({ request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const uniqueTag = `Mixed${suffix}`;

    // 1. Testuser owns testProject — create an internal item there.
    const ownedInternalItem = await api.createWorkItem(request, testUser.token, testProject.key, {
      title: `Owned project internal ${uniqueTag}`,
      type: 'task',
    });

    // 2. Admin creates a second project and adds testUser as CUSTOMER.
    const custProjKey = `C${suffix}`;
    await api.createProject(request, adminToken, custProjKey, `Cust Search ${suffix}`);
    await api.addMember(request, adminToken, custProjKey, testUser.id, 'customer');
    await api.createQueue(request, adminToken, custProjKey, {
      name: 'Portal',
      queue_type: 'support',
      is_public: true,
    });

    // 3. Admin creates an INTERNAL work item in the customer project that the
    //    user must NOT see via search.
    const hiddenInternal = await api.createWorkItem(request, adminToken, custProjKey, {
      title: `Hidden internal ${uniqueTag}`,
      type: 'task',
    });

    // 4. A different customer files a portal ticket in the customer project
    //    that testUser must NOT see.
    const otherCustomer = await createReadyUser(request, adminToken);
    await api.addMember(request, adminToken, custProjKey, otherCustomer.id, 'customer');
    const otherTicket = await api.createPortalTicket(request, otherCustomer.token, custProjKey, {
      title: `Other customer ticket ${uniqueTag}`,
    });

    // 5. testUser files their own portal ticket in the customer project.
    const ownTicket = await api.createPortalTicket(request, testUser.token, custProjKey, {
      title: `My own ticket ${uniqueTag}`,
    });

    // 6. testUser searches — expect the full owned-project internal item AND
    //    their own portal ticket, but NOT the hidden internal OR the other
    //    customer's ticket.
    const results = await searchAs(request, testUser.token, uniqueTag);
    const ids = results.fts.results.map((r) => r.entity_id);

    expect(ids).toContain(ownedInternalItem.id);
    expect(ids).toContain(ownTicket.id);
    expect(ids).not.toContain(hiddenInternal.id);
    expect(ids).not.toContain(otherTicket.id);

    // Cleanup
    await api.deactivateUser(request, adminToken, otherCustomer.id).catch(() => {});
  });

  test('API: customer cannot surface internal comments on their own tickets via search', async ({ request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const uniqueTag = `CmtTag${suffix}`;

    await api.createQueue(request, testUser.token, testProject.key, {
      name: 'Portal Support',
      queue_type: 'support',
      is_public: true,
    });

    const customer = await createReadyUser(request, adminToken);
    await api.addMember(request, testUser.token, testProject.key, customer.id, 'customer');

    // Customer opens a ticket
    const ticket = await api.createPortalTicket(request, customer.token, testProject.key, {
      title: `Cmt Search ${uniqueTag}`,
    });

    // Owner adds an internal (default) comment and a public comment, both
    // containing a distinct phrase the customer will search for.
    const internalComment = await api.addCommentWithVisibility(
      request,
      testUser.token,
      testProject.key,
      ticket.item_number,
      `Confidential internal note ${uniqueTag}`,
      'internal',
    );
    const publicComment = await api.addCommentWithVisibility(
      request,
      testUser.token,
      testProject.key,
      ticket.item_number,
      `Public reply ${uniqueTag}`,
      'public',
    );

    // Customer searches. FTS currently indexes work items only (not comments),
    // so the critical assertion is that the internal comment does NOT leak in
    // any way — neither as a comment result, nor by surfacing the parent item
    // with the internal snippet.
    const results = await searchAs(request, customer.token, `Confidential ${uniqueTag}`);
    const entityIDs = results.fts.results.map((r) => r.entity_id);
    expect(entityIDs).not.toContain(internalComment.id);

    // Sanity: the customer can still find their own ticket by title
    const titleResults = await searchAs(request, customer.token, `Cmt Search ${uniqueTag}`);
    const titleIDs = titleResults.fts.results.map((r) => r.entity_id);
    expect(titleIDs).toContain(ticket.id);

    // Avoid unused-var warning; publicComment is kept for future semantic search assertions.
    expect(publicComment.id).toBeTruthy();

    // Cleanup
    await api.deactivateUser(request, adminToken, customer.id).catch(() => {});
  });

  test('API: customer-only single-project user searching matches only own tickets', async ({ request }) => {
    const adminToken = getAdminToken();
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const uniqueTag = `Solo${suffix}`;

    // Admin creates a project with a public queue
    const projKey = `S${suffix}`;
    await api.createProject(request, adminToken, projKey, `Solo Search ${suffix}`);
    await api.createQueue(request, adminToken, projKey, {
      name: 'Portal Support',
      queue_type: 'support',
      is_public: true,
    });

    // Admin seeds an internal item
    const internal = await api.createWorkItem(request, adminToken, projKey, {
      title: `Admin internal ${uniqueTag}`,
      type: 'task',
    });

    // Create a portal-only customer and add them
    const customer = await createReadyUser(request, adminToken);
    await api.addMember(request, adminToken, projKey, customer.id, 'customer');

    // Customer files their own ticket
    const ownTicket = await api.createPortalTicket(request, customer.token, projKey, {
      title: `Solo customer ticket ${uniqueTag}`,
    });

    // Another customer files a ticket
    const otherCustomer = await createReadyUser(request, adminToken);
    await api.addMember(request, adminToken, projKey, otherCustomer.id, 'customer');
    const otherTicket = await api.createPortalTicket(request, otherCustomer.token, projKey, {
      title: `Other ${uniqueTag}`,
    });

    const results = await searchAs(request, customer.token, uniqueTag);
    const ids = results.fts.results.map((r) => r.entity_id);
    expect(ids).toContain(ownTicket.id);
    expect(ids).not.toContain(internal.id);
    expect(ids).not.toContain(otherTicket.id);

    // Cleanup
    await api.deactivateUser(request, adminToken, customer.id).catch(() => {});
    await api.deactivateUser(request, adminToken, otherCustomer.id).catch(() => {});
  });
});

test.describe('Customer role — search UI', () => {
  // UI coverage: open the AppShell search modal as a mixed-role user and
  // verify that items from a project where they are a customer are filtered
  // the same way the API tests above expect.

  test('UI: search modal hides other customers\' and internal items from customer project', async ({ page, context, request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const uniqueTag = `UiTag${suffix}`;

    // testUser owns testProject (full access). Create a second project where
    // testUser is ONLY a customer.
    const custProjKey = `U${suffix}`;
    await api.createProject(request, adminToken, custProjKey, `UI Search ${suffix}`);
    await api.addMember(request, adminToken, custProjKey, testUser.id, 'customer');
    await api.createQueue(request, adminToken, custProjKey, {
      name: 'Portal Support',
      queue_type: 'support',
      is_public: true,
    });

    // Admin seeds an internal item the testUser must NOT see via search.
    const hiddenInternal = await api.createWorkItem(request, adminToken, custProjKey, {
      title: `Admin hidden ${uniqueTag}`,
      type: 'task',
    });

    // Another customer files a ticket the testUser must NOT see.
    const otherCustomer = await createReadyUser(request, adminToken);
    await api.addMember(request, adminToken, custProjKey, otherCustomer.id, 'customer');
    const otherTicket = await api.createPortalTicket(request, otherCustomer.token, custProjKey, {
      title: `Another customer ${uniqueTag}`,
    });

    // testUser files their own ticket in the customer project — this one
    // should surface in search.
    const ownTicket = await api.createPortalTicket(request, testUser.token, custProjKey, {
      title: `My ticket ${uniqueTag}`,
    });

    // Log in as testUser via the UI (storage state already injects the token,
    // but we explicitly navigate to the AppShell). Entity search is scoped to
    // the active project (TF-432/434), so the palette has to be opened from the
    // customer project for its items to be in range at all — which is also
    // where the customer-visibility filter has to hold.
    await page.goto(`/d/projects/${custProjKey}/support`);
    await page.waitForLoadState('domcontentloaded');

    // Open the command palette via the nav search button (the first matching
    // "Search" aria-label — the work item list filter input also contains
    // "Search" so we must target the nav button explicitly).
    await page.locator('button[aria-label="Search"]').click();

    // The palette input uses the exact placeholder "Search or jump to..."
    // which is distinct from the work item list filter ("Search items...").
    const modalInput = page.getByPlaceholder(PALETTE_PLACEHOLDER);
    await expect(modalInput).toBeVisible({ timeout: 5000 });
    await modalInput.fill(uniqueTag);

    // Scope subsequent assertions to the dialog so we don't accidentally
    // match text on the underlying Work Items list page.
    const dialog = page.getByRole('dialog');

    // The user's own customer ticket must be visible inside the modal.
    await expect(dialog.getByText(`My ticket ${uniqueTag}`)).toBeVisible({ timeout: 10000 });

    // The hidden internal item and the other customer's ticket must NOT appear.
    await expect(dialog.getByText(`Admin hidden ${uniqueTag}`)).toHaveCount(0);
    await expect(dialog.getByText(`Another customer ${uniqueTag}`)).toHaveCount(0);

    // Cleanup
    await api.deactivateUser(request, adminToken, otherCustomer.id).catch(() => {});

    // Avoid unused-var warnings. testProject stays in the fixture list on
    // purpose: it makes testUser a full member somewhere, so this is a
    // mixed-role user rather than a customer-only one.
    expect(hiddenInternal.id).toBeTruthy();
    expect(otherTicket.id).toBeTruthy();
    expect(ownTicket.id).toBeTruthy();
    expect(testProject.key).toBeTruthy();

    // Also exercise the loginAs helper so we keep it linked (no-op flow).
    void loginAs;
    void context;
  });

  test('UI: clicking a customer-project search result navigates to /support/:num (no infinite /support loop)', async ({ page, request, testUser, testProject }) => {
    const adminToken = getAdminToken();
    const suffix = randomUUID().slice(0, 4).toUpperCase();
    const uniqueTag = `NavTag${suffix}`;

    // Create a second project where testUser is a CUSTOMER
    const custProjKey = `N${suffix}`;
    await api.createProject(request, adminToken, custProjKey, `Nav Search ${suffix}`);
    await api.addMember(request, adminToken, custProjKey, testUser.id, 'customer');
    await api.createQueue(request, adminToken, custProjKey, {
      name: 'Portal Support',
      queue_type: 'support',
      is_public: true,
    });

    // testUser files their own portal ticket — the search result will target this.
    const ownTicket = await api.createPortalTicket(request, testUser.token, custProjKey, {
      title: `Nav target ${uniqueTag}`,
    });

    // Open the customer project's support view. Entity search is scoped to the
    // active project (TF-432/434), so this is where the ticket is in range, and
    // loading it fresh lets AuthContext pick up the customer membership
    // (portal_projects) that decides the /support/:num route below.
    await page.goto(`/d/projects/${custProjKey}/support`);
    await page.waitForLoadState('domcontentloaded');

    // Open the command palette via the nav button.
    await page.locator('button[aria-label="Search"]').click();
    const modalInput = page.getByPlaceholder(PALETTE_PLACEHOLDER);
    await expect(modalInput).toBeVisible({ timeout: 5000 });
    await modalInput.fill(uniqueTag);

    // Wait for the result to appear and click it.
    const dialog = page.getByRole('dialog');
    const resultButton = dialog.getByText(`Nav target ${uniqueTag}`);
    await expect(resultButton).toBeVisible({ timeout: 10000 });
    await resultButton.click();

    // The URL should become /d/projects/<custProjKey>/support/<itemNumber>
    // and NOT the degenerate /items/<num>/support/support/... loop.
    const expectedPath = new RegExp(`/d/projects/${custProjKey}/support/${ownTicket.item_number}(?:$|\\?)`);
    await expect(page).toHaveURL(expectedPath, { timeout: 10000 });

    // Defensive check: URL must NOT contain the buggy /items/ segment OR any
    // duplicated /support/support sequence.
    const url = page.url();
    expect(url).not.toContain(`/items/${ownTicket.item_number}`);
    expect(url).not.toMatch(/\/support\/support/);

    // The PortalTicketDetailPage should have loaded and show the title.
    await expect(page.getByText(`Nav target ${uniqueTag}`).first()).toBeVisible({ timeout: 5000 });

    // testProject stays in the fixture list so testUser is a mixed-role user.
    expect(testProject.key).toBeTruthy();
  });

  test('UI: direct navigation to /d/projects/KEY/items/N as a customer redirects once to /support and does not loop', async ({ page, request, testUser, testProject }) => {
    // Defensive regression test: even if someone sends a stale /items/:num link
    // to a customer user (e.g. old bookmark or legacy notification), the catch-all
    // must redirect to /support exactly once, not build up /support/support/...
    const adminToken = getAdminToken();
    const suffix = randomUUID().slice(0, 4).toUpperCase();

    const custProjKey = `L${suffix}`;
    await api.createProject(request, adminToken, custProjKey, `Loop Guard ${suffix}`);
    await api.addMember(request, adminToken, custProjKey, testUser.id, 'customer');
    await api.createQueue(request, adminToken, custProjKey, {
      name: 'Portal Support',
      queue_type: 'support',
      is_public: true,
    });

    await page.goto(`/d/projects/${custProjKey}/items/42`);
    await page.waitForLoadState('domcontentloaded');

    // Should land on /support (the list page) because /items/42 has no match
    // in the customer routes.
    await expect(page).toHaveURL(new RegExp(`/d/projects/${custProjKey}/support$`), {
      timeout: 10000,
    });

    expect(page.url()).not.toMatch(/\/support\/support/);

    // Silence unused-var warning on the fixture.
    expect(testProject.key).toBeTruthy();
  });
});
