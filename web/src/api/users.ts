import { api } from './client'

export interface UserSearchResult {
  id: string
  email: string
  display_name: string
  global_role: string
  avatar_url?: string
}

export async function searchUsers(query: string) {
  const res = await api.get<{ data: UserSearchResult[] }>('/users', { params: { q: query } })
  return res.data.data
}
