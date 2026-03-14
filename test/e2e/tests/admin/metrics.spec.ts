import { test as base, expect } from '../../lib/fixtures';
import { getAdminToken } from '../../lib/fixtures';
import * as api from '../../lib/api';
import { randomUUID } from 'crypto';

const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

// Admin context for system API key management
const test = base.extend({
  storageState: async ({}, use) => {
    const adminToken = getAdminToken();
    const state = {
      cookies: [],
      origins: [
        {
          origin: BASE_URL,
          localStorage: [{ name: 'taskwondo_token', value: adminToken }],
        },
      ],
    };
    await use(state as any);
  },
});

// ─── API Tests ───────────────────────────────────────────────────────────────

test.describe('Prometheus Metrics — API', () => {
  test('metrics endpoint returns Prometheus format with system API key', async ({ request }) => {
    const adminToken = getAdminToken();
    const keyName = `e2e-metrics-${randomUUID().slice(0, 8)}`;

    // Create a system API key with metrics:r permission
    const created = await api.createSystemAPIKey(request, adminToken, keyName, ['metrics:r']);
    const systemKey = created.key;

    // Fetch metrics endpoint using the system API key
    const res = await request.get(`${BASE_URL}/metrics`, {
      headers: { Authorization: `Bearer ${systemKey}` },
    });
    expect(res.status()).toBe(200);

    const body = await res.text();

    // Should contain Go runtime metrics
    expect(body).toContain('go_goroutines');

    // Should contain process metrics
    expect(body).toContain('process_open_fds');

    // Should contain HTTP request metrics (at least the HELP/TYPE for registered counters)
    expect(body).toContain('taskwondo_http_request_duration_seconds');

    // Should contain DB connection pool metrics
    expect(body).toContain('taskwondo_db_connections_open');
    expect(body).toContain('taskwondo_db_connections_idle');
    expect(body).toContain('taskwondo_db_connections_in_use');
    expect(body).toContain('taskwondo_db_connections_wait_total');
    expect(body).toContain('taskwondo_db_connections_wait_duration_seconds_total');

    // Content type should be Prometheus-compatible
    const contentType = res.headers()['content-type'] || '';
    expect(contentType).toContain('text/plain');

    // Cleanup
    await api.deleteSystemAPIKey(request, adminToken, created.id);
  });

  test('metrics endpoint records HTTP request counts', async ({ request }) => {
    const adminToken = getAdminToken();
    const keyName = `e2e-metrics-count-${randomUUID().slice(0, 8)}`;

    // Create a system API key with metrics:r permission
    const created = await api.createSystemAPIKey(request, adminToken, keyName, ['metrics:r']);
    const systemKey = created.key;

    // Make a few requests to generate traffic
    await request.get(`${BASE_URL}/healthz`);
    await request.get(`${BASE_URL}/readyz`);

    // Fetch metrics
    const res = await request.get(`${BASE_URL}/metrics`, {
      headers: { Authorization: `Bearer ${systemKey}` },
    });
    expect(res.status()).toBe(200);

    const body = await res.text();

    // Should contain taskwondo_http_requests_total with status labels
    expect(body).toContain('taskwondo_http_requests_total');

    // Cleanup
    await api.deleteSystemAPIKey(request, adminToken, created.id);
  });
});

// ─── Auth Tests ──────────────────────────────────────────────────────────────

test.describe('Prometheus Metrics — Auth', () => {
  test('metrics endpoint rejects unauthenticated requests (401)', async ({ request }) => {
    const res = await request.get(`${BASE_URL}/metrics`);
    expect(res.status()).toBe(401);
  });

  test('metrics endpoint rejects system key without metrics permission (403)', async ({ request }) => {
    const adminToken = getAdminToken();
    const keyName = `e2e-metrics-noperm-${randomUUID().slice(0, 8)}`;

    // Create a system API key with only items:r permission (no metrics)
    const created = await api.createSystemAPIKey(request, adminToken, keyName, ['items:r']);
    const systemKey = created.key;

    // Try to access metrics — should get 403
    const res = await request.get(`${BASE_URL}/metrics`, {
      headers: { Authorization: `Bearer ${systemKey}` },
    });
    expect(res.status()).toBe(403);

    // Cleanup
    await api.deleteSystemAPIKey(request, adminToken, created.id);
  });

  test('metrics endpoint works with admin JWT token', async ({ request }) => {
    const adminToken = getAdminToken();

    // JWT tokens don't go through resource-based permission checks,
    // so admin JWT should be able to access /metrics
    const res = await request.get(`${BASE_URL}/metrics`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(res.status()).toBe(200);

    const body = await res.text();
    expect(body).toContain('go_goroutines');
  });
});
