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
