import { api } from './client'

export interface User {
  id: string
  email: string
  display_name: string
  global_role: string
}

interface LoginResponse {
  data: {
    token: string
    user: User
    force_password_change: boolean
  }
}

interface MeResponse {
  data: User
}

interface RefreshResponse {
  data: {
    token: string
  }
}

export async function login(email: string, password: string) {
  const res = await api.post<LoginResponse>('/auth/login', { email, password })
  return res.data.data
}

export async function getMe() {
  const res = await api.get<MeResponse>('/auth/me')
  return res.data.data
}

export async function refresh() {
  const res = await api.post<RefreshResponse>('/auth/refresh')
  return res.data.data
}

export interface UserSearchResult {
  id: string
  email: string
  display_name: string
  global_role: string
}

export async function searchUsers(query: string) {
  const res = await api.get<{ data: UserSearchResult[] }>('/users/search', { params: { q: query } })
  return res.data.data
}

// Auth providers

export type AuthProviders = Record<string, boolean>

export async function getAuthProviders() {
  const res = await api.get<{ data: AuthProviders }>('/auth/providers')
  return res.data.data
}

// Generic OAuth

interface OAuthAuthResponse {
  data: {
    url: string
  }
}

export async function getOAuthURL(provider: string) {
  const res = await api.get<OAuthAuthResponse>(`/auth/${provider}`)
  return res.data.data
}

interface OAuthCallbackResponse {
  data: {
    token: string
    user: User
  }
}

export async function oauthCallback(provider: string, code: string, state: string) {
  const res = await api.post<OAuthCallbackResponse>(`/auth/${provider}/callback`, { code, state })
  return res.data.data
}

// API keys

export interface APIKey {
  id: string
  name: string
  key_prefix: string
  permissions: string[]
  last_used_at?: string
  expires_at?: string
  created_at: string
}

export interface CreatedAPIKey extends APIKey {
  key: string
}

export async function createAPIKey(name: string, permissions: string[] = [], expiresAt?: string) {
  const res = await api.post<{ data: CreatedAPIKey }>('/user/api-keys', {
    name,
    permissions,
    expires_at: expiresAt,
  })
  return res.data.data
}

export async function listAPIKeys() {
  const res = await api.get<{ data: APIKey[] }>('/user/api-keys')
  return res.data.data
}

export async function deleteAPIKey(id: string) {
  await api.delete(`/user/api-keys/${id}`)
}

// Registration

export async function register(email: string, displayName: string) {
  const res = await api.post<{ data: { message: string } }>('/auth/register', {
    email,
    display_name: displayName,
  })
  return res.data.data
}

export async function verifyEmail(token: string, password: string) {
  const res = await api.post<{ data: { token: string; user: User } }>('/auth/verify-email', {
    token,
    password,
  })
  return res.data.data
}

// Password management

export async function changePassword(oldPassword: string, newPassword: string) {
  const res = await api.post<{ data: { token: string } }>('/auth/change-password', {
    old_password: oldPassword,
    new_password: newPassword,
  })
  return res.data.data
}
