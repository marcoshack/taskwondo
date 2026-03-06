import { APIRequestContext } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://localhost:5173';

interface LoginResponse {
  data: {
    token: string;
    user: { id: string; email: string; display_name: string; global_role: string };
    force_password_change: boolean;
  };
}

interface CreateUserResponse {
  data: {
    user: { id: string; email: string; display_name: string };
    temporary_password: string;
  };
}

export async function login(request: APIRequestContext, email: string, password: string): Promise<LoginResponse['data']> {
  const res = await request.post(`${BASE_URL}/api/v1/auth/login`, {
    data: { email, password },
  });
  if (!res.ok()) throw new Error(`Login failed (${res.status()}): ${await res.text()}`);
  const body: LoginResponse = await res.json();
  return body.data;
}

export async function createUser(
  request: APIRequestContext,
  adminToken: string,
  email: string,
  displayName: string,
): Promise<CreateUserResponse['data']> {
  const res = await request.post(`${BASE_URL}/api/v1/admin/users`, {
    headers: { Authorization: `Bearer ${adminToken}` },
    data: { email, display_name: displayName },
  });
  if (!res.ok()) throw new Error(`Create user failed (${res.status()}): ${await res.text()}`);
  const body: CreateUserResponse = await res.json();
  return body.data;
}

export async function changePassword(
  request: APIRequestContext,
  token: string,
  oldPassword: string,
  newPassword: string,
): Promise<void> {
  const res = await request.post(`${BASE_URL}/api/v1/auth/change-password`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { old_password: oldPassword, new_password: newPassword },
  });
  if (!res.ok()) throw new Error(`Change password failed (${res.status()}): ${await res.text()}`);
}

export async function createProject(
  request: APIRequestContext,
  token: string,
  key: string,
  name: string,
): Promise<{ id: string; key: string; name: string }> {
  const res = await request.post(`${BASE_URL}/api/v1/projects`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { key, name },
  });
  if (!res.ok()) throw new Error(`Create project failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

export async function listUsers(
  request: APIRequestContext,
  adminToken: string,
): Promise<{ id: string; email: string; is_active: boolean }[]> {
  const res = await request.get(`${BASE_URL}/api/v1/admin/users`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  });
  if (!res.ok()) throw new Error(`List users failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

export async function setMaxProjects(
  request: APIRequestContext,
  adminToken: string,
  userId: string,
  maxProjects: number,
): Promise<void> {
  const res = await request.patch(`${BASE_URL}/api/v1/admin/users/${userId}`, {
    headers: { Authorization: `Bearer ${adminToken}` },
    data: { max_projects: maxProjects },
  });
  if (!res.ok()) throw new Error(`Set max projects failed (${res.status()}): ${await res.text()}`);
}

export async function deactivateUser(
  request: APIRequestContext,
  adminToken: string,
  userId: string,
): Promise<void> {
  const res = await request.patch(`${BASE_URL}/api/v1/admin/users/${userId}`, {
    headers: { Authorization: `Bearer ${adminToken}` },
    data: { is_active: false },
  });
  if (!res.ok()) throw new Error(`Deactivate user failed (${res.status()}): ${await res.text()}`);
}

export async function addMember(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  userId: string,
  role: string,
): Promise<void> {
  const res = await request.post(`${BASE_URL}/api/v1/projects/${projectKey}/members`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { user_id: userId, role },
  });
  if (!res.ok()) throw new Error(`Add member failed (${res.status()}): ${await res.text()}`);
}

export async function removeMember(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  userId: string,
): Promise<void> {
  const res = await request.delete(`${BASE_URL}/api/v1/projects/${projectKey}/members/${userId}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok()) throw new Error(`Remove member failed (${res.status()}): ${await res.text()}`);
}

export async function setProjectUserSetting(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  key: string,
  value: unknown,
): Promise<void> {
  const res = await request.put(`${BASE_URL}/api/v1/projects/${projectKey}/user-settings/${key}`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { value },
  });
  if (!res.ok()) throw new Error(`Set project user setting failed (${res.status()}): ${await res.text()}`);
}

export async function createWorkItem(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  data: { title: string; type: string; description?: string; assignee_id?: string; watcher_ids?: string[] },
): Promise<{ id: string; item_number: number; display_id: string }> {
  const res = await request.post(`${BASE_URL}/api/v1/projects/${projectKey}/items`, {
    headers: { Authorization: `Bearer ${token}` },
    data,
  });
  if (!res.ok()) throw new Error(`Create work item failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

export async function updateWorkItem(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  itemNumber: number,
  data: Record<string, unknown>,
): Promise<void> {
  const res = await request.patch(`${BASE_URL}/api/v1/projects/${projectKey}/items/${itemNumber}`, {
    headers: { Authorization: `Bearer ${token}` },
    data,
  });
  if (!res.ok()) throw new Error(`Update work item failed (${res.status()}): ${await res.text()}`);
}

export async function addComment(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  itemNumber: number,
  body: string,
): Promise<{ id: string }> {
  const res = await request.post(`${BASE_URL}/api/v1/projects/${projectKey}/items/${itemNumber}/comments`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { body },
  });
  if (!res.ok()) throw new Error(`Add comment failed (${res.status()}): ${await res.text()}`);
  const json = await res.json();
  return json.data;
}

export async function updateComment(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  itemNumber: number,
  commentId: string,
  body: string,
): Promise<void> {
  const res = await request.patch(`${BASE_URL}/api/v1/projects/${projectKey}/items/${itemNumber}/comments/${commentId}`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { body },
  });
  if (!res.ok()) throw new Error(`Update comment failed (${res.status()}): ${await res.text()}`);
}

export async function createTimeEntry(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  itemNumber: number,
  data: { started_at: string; duration_seconds: number; description?: string },
): Promise<{ id: string; duration_seconds: number }> {
  const res = await request.post(`${BASE_URL}/api/v1/projects/${projectKey}/items/${itemNumber}/time-entries`, {
    headers: { Authorization: `Bearer ${token}` },
    data,
  });
  if (!res.ok()) throw new Error(`Create time entry failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

export async function listTimeEntries(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  itemNumber: number,
): Promise<{ entries: { id: string; duration_seconds: number; description?: string }[]; total_logged_seconds: number }> {
  const res = await request.get(`${BASE_URL}/api/v1/projects/${projectKey}/items/${itemNumber}/time-entries`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok()) throw new Error(`List time entries failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

export async function deleteTimeEntry(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  itemNumber: number,
  entryId: string,
): Promise<void> {
  const res = await request.delete(`${BASE_URL}/api/v1/projects/${projectKey}/items/${itemNumber}/time-entries/${entryId}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok()) throw new Error(`Delete time entry failed (${res.status()}): ${await res.text()}`);
}

// --- Milestones ---

export async function createMilestone(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  data: { name: string; due_date?: string },
): Promise<{ id: string; name: string }> {
  const res = await request.post(`${BASE_URL}/api/v1/projects/${projectKey}/milestones`, {
    headers: { Authorization: `Bearer ${token}` },
    data,
  });
  if (!res.ok()) throw new Error(`Create milestone failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

// --- Relations ---

export async function createRelation(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  itemNumber: number,
  data: { target_display_id: string; relation_type: string },
): Promise<{ id: string }> {
  const res = await request.post(`${BASE_URL}/api/v1/projects/${projectKey}/items/${itemNumber}/relations`, {
    headers: { Authorization: `Bearer ${token}` },
    data,
  });
  if (!res.ok()) throw new Error(`Create relation failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

// --- Inbox ---

export async function addToInbox(
  request: APIRequestContext,
  token: string,
  workItemId: string,
): Promise<void> {
  const res = await request.post(`${BASE_URL}/api/v1/user/inbox`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { work_item_id: workItemId },
  });
  if (!res.ok()) throw new Error(`Add to inbox failed (${res.status()}): ${await res.text()}`);
}

export async function listInboxItems(
  request: APIRequestContext,
  token: string,
  params?: { search?: string; include_completed?: boolean; project?: string[] },
): Promise<{ items: { id: string; work_item_id: string; position: number; display_id: string; title: string; status: string; status_category: string; project_key: string }[]; total: number }> {
  const query = new URLSearchParams();
  if (params?.search) query.set('search', params.search);
  if (params?.include_completed) query.set('include_completed', 'true');
  if (params?.project?.length) query.set('project', params.project.join(','));
  const qs = query.toString();
  const url = `${BASE_URL}/api/v1/user/inbox${qs ? `?${qs}` : ''}`;
  const res = await request.get(url, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok()) throw new Error(`List inbox items failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

export async function removeFromInbox(
  request: APIRequestContext,
  token: string,
  inboxItemId: string,
): Promise<void> {
  const res = await request.delete(`${BASE_URL}/api/v1/user/inbox/${inboxItemId}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok()) throw new Error(`Remove from inbox failed (${res.status()}): ${await res.text()}`);
}

export async function reorderInboxItem(
  request: APIRequestContext,
  token: string,
  inboxItemId: string,
  position: number,
): Promise<void> {
  const res = await request.patch(`${BASE_URL}/api/v1/user/inbox/${inboxItemId}`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { position },
  });
  if (!res.ok()) throw new Error(`Reorder inbox item failed (${res.status()}): ${await res.text()}`);
}

export async function getInboxCount(
  request: APIRequestContext,
  token: string,
): Promise<number> {
  const res = await request.get(`${BASE_URL}/api/v1/user/inbox/count`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok()) throw new Error(`Get inbox count failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data.count;
}

export async function clearCompletedInbox(
  request: APIRequestContext,
  token: string,
): Promise<number> {
  const res = await request.delete(`${BASE_URL}/api/v1/user/inbox/completed`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok()) throw new Error(`Clear completed inbox failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data.removed;
}

// --- Saved Searches ---

export async function createSavedSearch(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  data: { name: string; filters: Record<string, unknown>; view_mode: string; shared: boolean },
): Promise<{ id: string; name: string; scope: string }> {
  const res = await request.post(`${BASE_URL}/api/v1/projects/${projectKey}/saved-searches`, {
    headers: { Authorization: `Bearer ${token}` },
    data,
  });
  if (!res.ok()) throw new Error(`Create saved search failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

export async function listSavedSearches(
  request: APIRequestContext,
  token: string,
  projectKey: string,
): Promise<{ id: string; name: string; scope: string; filters: Record<string, unknown>; view_mode: string }[]> {
  const res = await request.get(`${BASE_URL}/api/v1/projects/${projectKey}/saved-searches`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok()) throw new Error(`List saved searches failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

export async function deleteSavedSearch(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  searchId: string,
): Promise<void> {
  const res = await request.delete(`${BASE_URL}/api/v1/projects/${projectKey}/saved-searches/${searchId}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok()) throw new Error(`Delete saved search failed (${res.status()}): ${await res.text()}`);
}

export async function updateSavedSearch(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  searchId: string,
  data: { name?: string; filters?: Record<string, unknown>; view_mode?: string; position?: number },
): Promise<{ id: string; name: string; scope: string; position: number }> {
  const res = await request.patch(`${BASE_URL}/api/v1/projects/${projectKey}/saved-searches/${searchId}`, {
    headers: { Authorization: `Bearer ${token}` },
    data,
  });
  if (!res.ok()) throw new Error(`Update saved search failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

// --- Preferences ---

export async function setPreference(
  request: APIRequestContext,
  token: string,
  key: string,
  value: unknown,
): Promise<void> {
  const res = await request.put(`${BASE_URL}/api/v1/user/preferences/${key}`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { value },
  });
  if (!res.ok()) throw new Error(`Set preference failed (${res.status()}): ${await res.text()}`);
}

// --- SMTP Config ---

export interface SMTPConfig {
  enabled: boolean;
  smtp_host: string;
  smtp_port: number;
  imap_host: string;
  imap_port: number;
  username: string;
  password: string;
  encryption: 'starttls' | 'tls' | 'none';
  from_address: string;
  from_name: string;
}

export async function setSMTPConfig(
  request: APIRequestContext,
  adminToken: string,
  config: SMTPConfig,
): Promise<SMTPConfig> {
  const res = await request.put(`${BASE_URL}/api/v1/admin/settings/smtp_config`, {
    headers: { Authorization: `Bearer ${adminToken}` },
    data: config,
  });
  if (!res.ok()) throw new Error(`Set SMTP config failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

export async function getSMTPConfig(
  request: APIRequestContext,
  adminToken: string,
): Promise<SMTPConfig> {
  const res = await request.get(`${BASE_URL}/api/v1/admin/settings/smtp_config`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  });
  if (!res.ok()) throw new Error(`Get SMTP config failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

export async function deleteSMTPConfig(
  request: APIRequestContext,
  adminToken: string,
): Promise<void> {
  const res = await request.delete(`${BASE_URL}/api/v1/admin/settings/smtp_config`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  });
  // 404 is OK — means it was already gone
  if (!res.ok() && res.status() !== 404) {
    throw new Error(`Delete SMTP config failed (${res.status()}): ${await res.text()}`);
  }
}

export async function resetSMTPConfig(
  request: APIRequestContext,
  adminToken: string,
): Promise<void> {
  // Reset to a clean disabled state
  await setSMTPConfig(request, adminToken, {
    enabled: false,
    smtp_host: '',
    smtp_port: 587,
    imap_host: '',
    imap_port: 993,
    username: '',
    password: '',
    encryption: 'starttls',
    from_address: '',
    from_name: '',
  });
}

/** Configure SMTP to use Mailpit (for E2E email tests). */
export async function configureMailpitSMTP(
  request: APIRequestContext,
  adminToken: string,
): Promise<void> {
  await setSMTPConfig(request, adminToken, {
    enabled: true,
    smtp_host: 'mailpit',
    smtp_port: 1025,
    imap_host: '',
    imap_port: 993,
    username: 'test@e2e.local',
    password: 'unused',
    encryption: 'none',
    from_address: 'noreply@e2e.local',
    from_name: 'Taskwondo E2E',
  });
}

// --- System Settings ---

export async function setSystemSetting(
  request: APIRequestContext,
  adminToken: string,
  key: string,
  value: unknown,
): Promise<void> {
  const res = await request.put(`${BASE_URL}/api/v1/admin/settings/${key}`, {
    headers: { Authorization: `Bearer ${adminToken}` },
    data: { value },
  });
  if (!res.ok()) throw new Error(`Set system setting failed (${res.status()}): ${await res.text()}`);
}

export async function deleteSystemSetting(
  request: APIRequestContext,
  adminToken: string,
  key: string,
): Promise<void> {
  const res = await request.delete(`${BASE_URL}/api/v1/admin/settings/${key}`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  });
  // 404 is fine — setting may not exist yet
  if (!res.ok() && res.status() !== 404) throw new Error(`Delete system setting failed (${res.status()}): ${await res.text()}`);
}

/** Enable email auth (login + registration). Requires SMTP to be configured first. */
export async function enableEmailAuth(
  request: APIRequestContext,
  adminToken: string,
): Promise<void> {
  await setSystemSetting(request, adminToken, 'auth_email_login_enabled', true);
  await setSystemSetting(request, adminToken, 'auth_email_registration_enabled', true);
}

/** Disable email registration. */
export async function disableEmailRegistration(
  request: APIRequestContext,
  adminToken: string,
): Promise<void> {
  await setSystemSetting(request, adminToken, 'auth_email_registration_enabled', false);
}

// --- Mailpit ---

const MAILPIT_URL = process.env.MAILPIT_URL || 'http://localhost:8025';

interface MailpitMessage {
  ID: string;
  From: { Address: string; Name: string };
  To: { Address: string; Name: string }[];
  Subject: string;
  Snippet: string;
}

interface MailpitMessageDetail extends MailpitMessage {
  HTML: string;
  Text: string;
}

export async function getMailpitMessages(
  request: APIRequestContext,
): Promise<MailpitMessage[]> {
  const res = await request.get(`${MAILPIT_URL}/api/v1/messages`);
  if (!res.ok()) throw new Error(`Mailpit list failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.messages ?? [];
}

export async function getMailpitMessage(
  request: APIRequestContext,
  id: string,
): Promise<MailpitMessageDetail> {
  const res = await request.get(`${MAILPIT_URL}/api/v1/message/${id}`);
  if (!res.ok()) throw new Error(`Mailpit get failed (${res.status()}): ${await res.text()}`);
  return await res.json();
}

export async function deleteMailpitMessages(
  request: APIRequestContext,
): Promise<void> {
  await request.delete(`${MAILPIT_URL}/api/v1/messages`);
}

export async function searchMailpitMessages(
  request: APIRequestContext,
  query: string,
): Promise<MailpitMessage[]> {
  const res = await request.get(`${MAILPIT_URL}/api/v1/search`, {
    params: { query },
  });
  if (!res.ok()) throw new Error(`Mailpit search failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.messages ?? [];
}

/**
 * Poll Mailpit for a message sent to a specific email address.
 * Returns the full message detail once found, or throws after timeout.
 */
export async function waitForMailpitMessage(
  request: APIRequestContext,
  recipientEmail: string,
  { timeoutMs = 5000, intervalMs = 500 } = {},
): Promise<MailpitMessageDetail> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const messages = await searchMailpitMessages(request, `to:${recipientEmail}`);
    if (messages.length > 0) {
      return getMailpitMessage(request, messages[0].ID);
    }
    await new Promise((r) => setTimeout(r, intervalMs));
  }
  throw new Error(`No Mailpit message found for ${recipientEmail} within ${timeoutMs}ms`);
}

// --- Invites ---

export async function createInvite(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  role: string,
): Promise<{ code: string; url: string }> {
  const res = await request.post(`${BASE_URL}/api/v1/projects/${projectKey}/invites`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { role },
  });
  if (!res.ok()) throw new Error(`Create invite failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

// --- Watchers ---

export async function toggleWatch(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  itemNumber: number,
): Promise<{ is_watching: boolean }> {
  const res = await request.post(`${BASE_URL}/api/v1/projects/${projectKey}/items/${itemNumber}/watch`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok()) throw new Error(`Toggle watch failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

export async function addWatcher(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  itemNumber: number,
  userId: string,
): Promise<void> {
  const res = await request.post(`${BASE_URL}/api/v1/projects/${projectKey}/items/${itemNumber}/watchers`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { user_id: userId },
  });
  if (!res.ok()) throw new Error(`Add watcher failed (${res.status()}): ${await res.text()}`);
}

export async function removeWatcher(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  itemNumber: number,
  userId: string,
): Promise<void> {
  const res = await request.delete(`${BASE_URL}/api/v1/projects/${projectKey}/items/${itemNumber}/watchers/${userId}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok()) throw new Error(`Remove watcher failed (${res.status()}): ${await res.text()}`);
}

export async function listWatchers(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  itemNumber: number,
): Promise<unknown> {
  const res = await request.get(`${BASE_URL}/api/v1/projects/${projectKey}/items/${itemNumber}/watchers`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok()) throw new Error(`List watchers failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}

export async function listWatchedItems(
  request: APIRequestContext,
  token: string,
  projectKeys?: string | string[],
): Promise<{ data: Array<{ id: string; title: string; item_number: number; display_id: string; project_key: string }>, meta: { total: number; has_more: boolean } }> {
  const params = new URLSearchParams({ mode: 'list' });
  if (projectKeys) {
    const keys = Array.isArray(projectKeys) ? projectKeys.join(',') : projectKeys;
    if (keys) params.set('project', keys);
  }
  const res = await request.get(`${BASE_URL}/api/v1/user/watchlist?${params}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok()) throw new Error(`List watched items failed (${res.status()}): ${await res.text()}`);
  return await res.json();
}

// --- Attachments ---

export async function uploadAttachment(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  itemNumber: number,
  filename: string,
  content: Buffer,
  contentType: string,
  comment?: string,
): Promise<{ id: string; filename: string }> {
  const res = await request.post(`${BASE_URL}/api/v1/projects/${projectKey}/items/${itemNumber}/attachments`, {
    headers: { Authorization: `Bearer ${token}` },
    multipart: {
      file: { name: filename, mimeType: contentType, buffer: content },
      ...(comment ? { comment } : {}),
    },
  });
  if (!res.ok()) throw new Error(`Upload attachment failed (${res.status()}): ${await res.text()}`);
  const body = await res.json();
  return body.data;
}
