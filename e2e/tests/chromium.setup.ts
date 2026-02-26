import { test as setup } from '@playwright/test';
import { getAdminToken } from '../lib/fixtures';
import * as api from '../lib/api';

/**
 * Runs after admin tests finish. Configures SMTP (Mailpit) and enables all
 * auth mechanisms so chromium-project tests can use email registration, etc.
 */
setup('configure SMTP and auth providers', async ({ request }) => {
  const adminToken = getAdminToken();

  // Point SMTP at Mailpit
  await api.configureMailpitSMTP(request, adminToken);

  // Enable email login + registration
  await api.enableEmailAuth(request, adminToken);

  // Clear any leftover emails from admin tests
  await api.deleteMailpitMessages(request);
});
