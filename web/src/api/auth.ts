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

export interface AuthProviders {
  discord: boolean
}

export async function getAuthProviders() {
  const res = await api.get<{ data: AuthProviders }>('/auth/providers')
  return res.data.data
}

// Discord OAuth

interface DiscordAuthResponse {
  data: {
    url: string
  }
}

export async function getDiscordAuthURL() {
  const res = await api.get<DiscordAuthResponse>('/auth/discord')
  return res.data.data
}

interface DiscordCallbackResponse {
  data: {
    token: string
    user: User
  }
}

export async function discordCallback(code: string, state: string) {
  const res = await api.post<DiscordCallbackResponse>('/auth/discord/callback', { code, state })
  return res.data.data
}
