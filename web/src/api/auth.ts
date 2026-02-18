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
