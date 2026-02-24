import { test as teardown } from '@playwright/test';
import path from 'path';
import fs from 'fs';
import * as api from '../lib/api';

const ADMIN_AUTH_FILE = path.join(__dirname, '../.auth/admin.json');

teardown('cleanup e2e test users', async ({ request }) => {
  const raw = fs.readFileSync(ADMIN_AUTH_FILE, 'utf-8');
  const adminToken: string = JSON.parse(raw).token;

  const users = await api.listUsers(request, adminToken);
  const e2eUsers = users.filter((u) => u.email.startsWith('e2e-') && u.is_active);

  for (const user of e2eUsers) {
    await api.deactivateUser(request, adminToken, user.id).catch(() => {});
  }

  if (e2eUsers.length > 0) {
    console.log(`Cleaned up ${e2eUsers.length} e2e test user(s)`);
  }
});
