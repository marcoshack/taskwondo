import { api } from './client'

export interface Namespace {
  id: string
  slug: string
  display_name: string
  icon: string
  color: string
  is_default: boolean
  created_at: string
  updated_at: string
}

export interface NamespaceMember {
  user_id: string
  email: string
  display_name: string
  avatar_url?: string
  role: string
  created_at: string
}

interface DataResponse<T> {
  data: T
}

export interface NamespaceListResult {
  namespaces: Namespace[]
  ownedNamespaceCount: number
  maxNamespaces: number
}

// --- Namespace CRUD ---

export async function listNamespaces(): Promise<NamespaceListResult> {
  const res = await api.get<{ data: Namespace[]; meta?: { owned_namespace_count?: number; max_namespaces?: number } }>('/namespaces')
  return {
    namespaces: res.data.data,
    ownedNamespaceCount: res.data.meta?.owned_namespace_count ?? 0,
    maxNamespaces: res.data.meta?.max_namespaces ?? 0,
  }
}

export async function getNamespace(slug: string): Promise<Namespace> {
  const res = await api.get<DataResponse<Namespace>>(`/namespaces/${slug}`)
  return res.data.data
}

export interface CreateNamespaceInput {
  slug: string
  display_name: string
}

export async function createNamespace(input: CreateNamespaceInput): Promise<Namespace> {
  const res = await api.post<DataResponse<Namespace>>('/namespaces', input)
  return res.data.data
}

export interface UpdateNamespaceInput {
  slug?: string
  display_name?: string
  icon?: string
  color?: string
}

export async function updateNamespace(slug: string, input: UpdateNamespaceInput): Promise<Namespace> {
  const res = await api.patch<DataResponse<Namespace>>(`/namespaces/${slug}`, input)
  return res.data.data
}

export async function deleteNamespace(slug: string): Promise<void> {
  await api.delete(`/namespaces/${slug}`)
}

// --- Namespace Members ---

export async function listNamespaceMembers(slug: string): Promise<NamespaceMember[]> {
  const res = await api.get<DataResponse<NamespaceMember[]>>(`/namespaces/${slug}/members`)
  return res.data.data
}

export interface AddNamespaceMemberInput {
  user_id: string
  role: string
}

export async function addNamespaceMember(slug: string, input: AddNamespaceMemberInput): Promise<void> {
  await api.post(`/namespaces/${slug}/members`, input)
}

export async function updateNamespaceMemberRole(slug: string, userId: string, role: string): Promise<void> {
  await api.put(`/namespaces/${slug}/members/${userId}`, { role })
}

export async function removeNamespaceMember(slug: string, userId: string): Promise<void> {
  await api.delete(`/namespaces/${slug}/members/${userId}`)
}

// --- Project Migration ---

export async function migrateProject(fromSlug: string, projectKey: string, targetSlug: string): Promise<void> {
  await api.post(`/namespaces/${fromSlug}/projects/${projectKey}/migrate`, { target_namespace: targetSlug })
}
