import { api } from './client'
import type { SLAInfo } from './workitems'

// --- Types ---

export interface InboxItem {
  id: string
  work_item_id: string
  position: number
  created_at: string
  display_id: string
  title: string
  type: string
  status: string
  status_category: string
  priority: string
  project_key: string
  project_name: string
  namespace_slug: string
  namespace_name: string
  assignee_id: string | null
  assignee_display_name: string
  description: string
  due_date: string | null
  sla: SLAInfo | null
  sla_target_at: string | null
  updated_at: string
}

export interface InboxListResponse {
  items: InboxItem[]
  cursor: string
  has_more: boolean
  total: number
}

export interface InboxFilter {
  search?: string
  include_completed?: boolean
  work_item_id?: string
  project?: string[]
  cursor?: string
  limit?: number
}

// --- API Functions ---

export async function listInboxItems(filter: InboxFilter = {}): Promise<InboxListResponse> {
  const params: Record<string, string> = {}
  if (filter.search) params.search = filter.search
  if (filter.include_completed) params.include_completed = 'true'
  if (filter.work_item_id) params.work_item_id = filter.work_item_id
  if (filter.project?.length) params.project = filter.project.join(',')
  if (filter.cursor) params.cursor = filter.cursor
  if (filter.limit) params.limit = String(filter.limit)

  const { data } = await api.get('/user/inbox', { params })
  return data.data
}

export async function addToInbox(workItemId: string): Promise<void> {
  await api.post('/user/inbox', { work_item_id: workItemId })
}

export async function removeFromInbox(inboxItemId: string): Promise<void> {
  await api.delete(`/user/inbox/${inboxItemId}`)
}

export async function reorderInboxItem(inboxItemId: string, position: number): Promise<void> {
  await api.patch(`/user/inbox/${inboxItemId}`, { position })
}

export async function clearCompletedInboxItems(): Promise<{ removed: number }> {
  const { data } = await api.delete('/user/inbox/completed')
  return data.data
}

export async function getInboxCount(): Promise<number> {
  const { data } = await api.get('/user/inbox/count')
  return data.data.count
}
