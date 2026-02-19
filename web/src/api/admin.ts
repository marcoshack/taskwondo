import { api } from './client'

export interface AdminUser {
  id: string
  email: string
  display_name: string
  global_role: string
  avatar_url?: string
  is_active: boolean
  last_login_at?: string
  created_at: string
}

export interface UserProject {
  project_id: string
  project_name: string
  project_key: string
  role: string
  created_at: string
}

export async function listUsers() {
  const res = await api.get<{ data: AdminUser[] }>('/admin/users')
  return res.data.data
}

export async function updateUser(userId: string, input: { global_role?: string; is_active?: boolean }) {
  const res = await api.patch<{ data: AdminUser }>(`/admin/users/${userId}`, input)
  return res.data.data
}

export async function listUserProjects(userId: string) {
  const res = await api.get<{ data: UserProject[] }>(`/admin/users/${userId}/projects`)
  return res.data.data
}

export async function addUserToProject(userId: string, input: { project_id: string; role: string }) {
  await api.post(`/admin/users/${userId}/projects`, input)
}

export async function removeUserFromProject(userId: string, projectId: string) {
  await api.delete(`/admin/users/${userId}/projects/${projectId}`)
}
