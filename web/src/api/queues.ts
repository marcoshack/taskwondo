import { api, nsPrefix } from './client'

// --- Types ---

export interface Queue {
  id: string
  project_id: string
  name: string
  description: string | null
  queue_type: string
  is_public: boolean
  default_priority: string
  default_assignee_id: string | null
  workflow_id: string | null
  created_at: string
  updated_at: string
}

export interface CreateQueueInput {
  name: string
  description?: string
  queue_type: string
  is_public?: boolean
  default_priority?: string
  workflow_id?: string
}

export interface UpdateQueueInput {
  name?: string
  description?: string | null
  queue_type?: string
  is_public?: boolean
  default_priority?: string
  default_assignee_id?: string | null
  workflow_id?: string | null
}

// --- API Functions ---

interface DataResponse<T> {
  data: T
}

export async function listQueues(projectKey: string) {
  const res = await api.get<DataResponse<Queue[]>>(`${nsPrefix()}/projects/${projectKey}/queues`)
  return res.data.data
}

export async function getQueue(projectKey: string, queueId: string) {
  const res = await api.get<DataResponse<Queue>>(`${nsPrefix()}/projects/${projectKey}/queues/${queueId}`)
  return res.data.data
}

export async function createQueue(projectKey: string, input: CreateQueueInput) {
  const res = await api.post<DataResponse<Queue>>(`${nsPrefix()}/projects/${projectKey}/queues`, input)
  return res.data.data
}

export async function updateQueue(projectKey: string, queueId: string, input: UpdateQueueInput) {
  const res = await api.patch<DataResponse<Queue>>(`${nsPrefix()}/projects/${projectKey}/queues/${queueId}`, input)
  return res.data.data
}

export async function deleteQueue(projectKey: string, queueId: string) {
  await api.delete(`${nsPrefix()}/projects/${projectKey}/queues/${queueId}`)
}
