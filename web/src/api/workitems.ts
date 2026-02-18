import { api } from './client'

// --- Types ---

export interface WorkItem {
  id: string
  project_key: string
  item_number: number
  display_id: string
  type: string
  title: string
  description: string | null
  status: string
  priority: string
  assignee_id: string | null
  reporter_id: string
  queue_id: string | null
  milestone_id: string | null
  visibility: string
  labels: string[]
  custom_fields: Record<string, unknown>
  due_date: string | null
  sla_deadline: string | null
  resolved_at: string | null
  created_at: string
  updated_at: string
}

export interface WorkItemListMeta {
  cursor: string
  has_more: boolean
  total: number
}

export interface WorkItemFilter {
  q?: string
  type?: string[]
  status?: string[]
  priority?: string[]
  assignee?: string[]
  queue?: string
  milestone?: string
  label?: string
  cursor?: string
  limit?: number
  sort?: string
  order?: 'asc' | 'desc'
}

export interface CreateWorkItemInput {
  type: string
  title: string
  description?: string
  priority?: string
  assignee_id?: string
  labels?: string[]
  parent_id?: string
  queue_id?: string
  milestone_id?: string
  visibility?: string
  due_date?: string
}

export interface UpdateWorkItemInput {
  title?: string
  description?: string | null
  status?: string
  priority?: string
  type?: string
  assignee_id?: string | null
  labels?: string[]
  visibility?: string
  due_date?: string | null
  parent_id?: string | null
  queue_id?: string | null
  milestone_id?: string | null
}

export interface Comment {
  id: string
  author_id: string | null
  body: string
  visibility: string
  created_at: string
  updated_at: string
}

export interface Relation {
  id: string
  source_display_id: string
  source_title: string
  target_display_id: string
  target_title: string
  relation_type: string
  created_by: string
  created_at: string
}

export interface EventActor {
  id: string
  display_name: string
}

export interface WorkItemEvent {
  id: string
  event_type: string
  actor?: EventActor
  field_name?: string
  old_value?: string
  new_value?: string
  metadata: Record<string, unknown>
  visibility: string
  created_at: string
}

// --- API Functions ---

interface WorkItemListResponse {
  data: WorkItem[]
  meta: WorkItemListMeta
}

interface DataResponse<T> {
  data: T
}

export async function listWorkItems(projectKey: string, filter: WorkItemFilter = {}) {
  const params = new URLSearchParams()
  if (filter.q) params.set('q', filter.q)
  if (filter.type?.length) params.set('type', filter.type.join(','))
  if (filter.status?.length) params.set('status', filter.status.join(','))
  if (filter.priority?.length) params.set('priority', filter.priority.join(','))
  if (filter.assignee?.length) params.set('assignee', filter.assignee.join(','))
  if (filter.queue) params.set('queue', filter.queue)
  if (filter.milestone) params.set('milestone', filter.milestone)
  if (filter.label) params.set('label', filter.label)
  if (filter.cursor) params.set('cursor', filter.cursor)
  if (filter.limit) params.set('limit', String(filter.limit))
  if (filter.sort) params.set('sort', filter.sort)
  if (filter.order) params.set('order', filter.order)
  const res = await api.get<WorkItemListResponse>(`/projects/${projectKey}/items`, { params })
  return res.data
}

export async function getWorkItem(projectKey: string, itemNumber: number) {
  const res = await api.get<DataResponse<WorkItem>>(`/projects/${projectKey}/items/${itemNumber}`)
  return res.data.data
}

export async function createWorkItem(projectKey: string, input: CreateWorkItemInput) {
  const res = await api.post<DataResponse<WorkItem>>(`/projects/${projectKey}/items`, input)
  return res.data.data
}

export async function updateWorkItem(projectKey: string, itemNumber: number, input: UpdateWorkItemInput) {
  const res = await api.patch<DataResponse<WorkItem>>(`/projects/${projectKey}/items/${itemNumber}`, input)
  return res.data.data
}

export async function deleteWorkItem(projectKey: string, itemNumber: number) {
  await api.delete(`/projects/${projectKey}/items/${itemNumber}`)
}

export async function listComments(projectKey: string, itemNumber: number) {
  const res = await api.get<DataResponse<Comment[]>>(`/projects/${projectKey}/items/${itemNumber}/comments`)
  return res.data.data
}

export async function createComment(projectKey: string, itemNumber: number, body: string, visibility?: string) {
  const res = await api.post<DataResponse<Comment>>(`/projects/${projectKey}/items/${itemNumber}/comments`, { body, visibility })
  return res.data.data
}

export async function updateComment(projectKey: string, itemNumber: number, commentId: string, body: string) {
  const res = await api.patch<DataResponse<Comment>>(`/projects/${projectKey}/items/${itemNumber}/comments/${commentId}`, { body })
  return res.data.data
}

export async function deleteComment(projectKey: string, itemNumber: number, commentId: string) {
  await api.delete(`/projects/${projectKey}/items/${itemNumber}/comments/${commentId}`)
}

export async function listRelations(projectKey: string, itemNumber: number) {
  const res = await api.get<DataResponse<Relation[]>>(`/projects/${projectKey}/items/${itemNumber}/relations`)
  return res.data.data
}

export async function createRelation(projectKey: string, itemNumber: number, targetDisplayId: string, relationType: string) {
  const res = await api.post<DataResponse<Relation>>(`/projects/${projectKey}/items/${itemNumber}/relations`, {
    target_display_id: targetDisplayId,
    relation_type: relationType,
  })
  return res.data.data
}

export async function deleteRelation(projectKey: string, itemNumber: number, relationId: string) {
  await api.delete(`/projects/${projectKey}/items/${itemNumber}/relations/${relationId}`)
}

export async function listEvents(projectKey: string, itemNumber: number) {
  const res = await api.get<DataResponse<WorkItemEvent[]>>(`/projects/${projectKey}/items/${itemNumber}/events`)
  return res.data.data
}

// --- Attachment Types ---

export interface Attachment {
  id: string
  uploader_id: string
  filename: string
  content_type: string
  size_bytes: number
  comment: string
  download_url: string
  created_at: string
}

// --- Attachment API Functions ---

export async function listAttachments(projectKey: string, itemNumber: number) {
  const res = await api.get<DataResponse<Attachment[]>>(
    `/projects/${projectKey}/items/${itemNumber}/attachments`
  )
  return res.data.data
}

export async function uploadAttachment(
  projectKey: string,
  itemNumber: number,
  file: File,
  comment?: string
) {
  const formData = new FormData()
  formData.append('file', file)
  if (comment) {
    formData.append('comment', comment)
  }
  const res = await api.post<DataResponse<Attachment>>(
    `/projects/${projectKey}/items/${itemNumber}/attachments`,
    formData,
    { headers: { 'Content-Type': undefined as unknown as string } }
  )
  return res.data.data
}

export async function deleteAttachment(
  projectKey: string,
  itemNumber: number,
  attachmentId: string
) {
  await api.delete(
    `/projects/${projectKey}/items/${itemNumber}/attachments/${attachmentId}`
  )
}

export function getAttachmentDownloadURL(
  projectKey: string,
  itemNumber: number,
  attachmentId: string
): string {
  return `/api/v1/projects/${projectKey}/items/${itemNumber}/attachments/${attachmentId}`
}
