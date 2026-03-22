import { api } from './client'

export interface AdminUser {
  id: string
  email: string
  display_name: string
  global_role: string
  avatar_url?: string
  is_active: boolean
  max_projects?: number | null
  max_namespaces?: number | null
  force_password_change: boolean
  last_login_at?: string
  created_at: string
}

export interface UserProject {
  project_id: string
  project_name: string
  project_key: string
  role: string
  owner_count: number
  created_at: string
}

export async function listUsers() {
  const res = await api.get<{ data: AdminUser[] }>('/admin/users')
  return res.data.data
}

export async function updateUser(userId: string, input: { global_role?: string; is_active?: boolean; max_projects?: number; max_namespaces?: number }) {
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

export async function updateUserProjectRole(userId: string, projectId: string, role: string) {
  await api.patch(`/admin/users/${userId}/projects/${projectId}`, { role })
}

export async function removeUserFromProject(userId: string, projectId: string) {
  await api.delete(`/admin/users/${userId}/projects/${projectId}`)
}

// User creation and password management

export interface CreateUserInput {
  email: string
  display_name: string
}

export interface CreateUserResponse {
  user: AdminUser
  temporary_password: string
}

export interface ResetPasswordResponse {
  temporary_password: string
}

export async function createUser(input: CreateUserInput): Promise<CreateUserResponse> {
  const res = await api.post<{ data: CreateUserResponse }>('/admin/users', input)
  return res.data.data
}

export async function resetUserPassword(userId: string): Promise<ResetPasswordResponse> {
  const res = await api.post<{ data: ResetPasswordResponse }>(`/admin/users/${userId}/reset-password`)
  return res.data.data
}

// Admin Projects & Namespaces

export interface AdminProject {
  id: string
  key: string
  name: string
  namespace_slug: string
  namespace_display_name: string
  owner_display_name: string
  owner_email: string
  member_count: number
  item_count: number
  storage_bytes: number
  created_at: string
}

export interface AdminNamespace {
  id: string
  slug: string
  display_name: string
  is_default: boolean
  project_count: number
  member_count: number
  storage_bytes: number
  created_at: string
}

export interface PaginatedResponse<T> {
  data: T[]
  meta: {
    cursor: string
    has_more: boolean
  }
}

export async function listAdminProjects(params: { search?: string; cursor?: string; limit?: number }): Promise<PaginatedResponse<AdminProject>> {
  const res = await api.get<PaginatedResponse<AdminProject>>('/admin/projects', { params })
  return res.data
}

export async function listAdminNamespaces(params: { search?: string; cursor?: string; limit?: number }): Promise<PaginatedResponse<AdminNamespace>> {
  const res = await api.get<PaginatedResponse<AdminNamespace>>('/admin/namespaces', { params })
  return res.data
}

// Admin Stats

export interface AdminStats {
  projects: number
  namespaces: number
  users: number
  storage_bytes: number
}

export async function fetchAdminStats(): Promise<AdminStats> {
  const res = await api.get<{ data: AdminStats }>('/admin/stats')
  return res.data.data
}
