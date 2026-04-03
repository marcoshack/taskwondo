import { api, nsPrefix } from './client'
import type { Team } from './teams'

// --- Types ---

export interface QueueCategory {
  id: string
  queue_id: string
  name: string
  description: string | null
  position: number
  created_at: string
  updated_at: string
}

export interface CreateCategoryInput {
  name: string
  description?: string
  position?: number
}

export interface UpdateCategoryInput {
  name?: string
  description?: string | null
  position?: number
}

// --- API Functions ---

interface DataResponse<T> {
  data: T
}

// Categories
export async function listCategories(projectKey: string, queueId: string) {
  const res = await api.get<DataResponse<QueueCategory[]>>(`${nsPrefix()}/projects/${projectKey}/queues/${queueId}/categories`)
  return res.data.data
}

export async function createCategory(projectKey: string, queueId: string, input: CreateCategoryInput) {
  const res = await api.post<DataResponse<QueueCategory>>(`${nsPrefix()}/projects/${projectKey}/queues/${queueId}/categories`, input)
  return res.data.data
}

export async function updateCategory(projectKey: string, queueId: string, categoryId: string, input: UpdateCategoryInput) {
  const res = await api.patch<DataResponse<QueueCategory>>(`${nsPrefix()}/projects/${projectKey}/queues/${queueId}/categories/${categoryId}`, input)
  return res.data.data
}

export async function deleteCategory(projectKey: string, queueId: string, categoryId: string) {
  await api.delete(`${nsPrefix()}/projects/${projectKey}/queues/${queueId}/categories/${categoryId}`)
}

// Queue-Team assignment
export async function listQueueTeams(projectKey: string, queueId: string) {
  const res = await api.get<DataResponse<Team[]>>(`${nsPrefix()}/projects/${projectKey}/queues/${queueId}/teams`)
  return res.data.data
}

export async function assignQueueTeam(projectKey: string, queueId: string, teamId: string) {
  await api.post(`${nsPrefix()}/projects/${projectKey}/queues/${queueId}/teams`, { team_id: teamId })
}

export async function unassignQueueTeam(projectKey: string, queueId: string, teamId: string) {
  await api.delete(`${nsPrefix()}/projects/${projectKey}/queues/${queueId}/teams/${teamId}`)
}
