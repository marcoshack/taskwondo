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

export async function createWorkItem(
  request: APIRequestContext,
  token: string,
  projectKey: string,
  data: { title: string; type: string; description?: string },
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
  params?: { search?: string; include_completed?: boolean },
): Promise<{ items: { id: string; work_item_id: string; position: number; display_id: string; title: string; status: string; status_category: string }[]; total: number }> {
  const query = new URLSearchParams();
  if (params?.search) query.set('search', params.search);
  if (params?.include_completed) query.set('include_completed', 'true');
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
