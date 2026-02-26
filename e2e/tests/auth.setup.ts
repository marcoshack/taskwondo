import { test as setup } from '@playwright/test';
import path from 'path';
import fs from 'fs';
import { login } from '../lib/api';

const authDir = path.join(__dirname, '../.auth');
const adminAuthFile = path.join(authDir, 'admin.json');

setup('authenticate admin', async ({ request }) => {
  const email = process.env.ADMIN_EMAIL;
  const password = process.env.ADMIN_PASSWORD;
  if (!email || !password) {
    throw new Error('ADMIN_EMAIL and ADMIN_PASSWORD must be set in .env');
  }

  const data = await login(request, email, password);

  // Dismiss the Welcome modal for the admin user
  const { setPreference } = await import('../lib/api');
  await setPreference(request, data.token, 'welcome_dismissed', true);

  fs.mkdirSync(authDir, { recursive: true });
  fs.writeFileSync(adminAuthFile, JSON.stringify({ token: data.token }, null, 2));
});
