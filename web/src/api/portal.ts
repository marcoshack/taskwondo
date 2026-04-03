import { api } from './client'

// Portal routes use a different path prefix than regular routes
function portalPrefix(namespace: string) {
  return `/portal/${namespace}`
}

// --- Types ---

export interface PortalQueue {
  id: string
  name: string
  description: string | null
  queue_type: string
  categories: PortalCategory[]
}

export interface PortalCategory {
  id: string
  name: string
  description: string | null
}

export interface PortalTicket {
  id: string
  item_number: number
  display_id: string
  type: string
  title: string
  description: string | null
  status: string
  priority: string
  visibility: string
  queue_id: string | null
  category_id: string | null
  reporter_id: string
  resolved_at: string | null
  created_at: string
  updated_at: string
}

export interface PortalTicketList {
  items: PortalTicket[]
  cursor: string
  has_more: boolean
  total: number
}

export interface PortalComment {
  id: string
  body: string
  visibility: string
  author_id: string
  author_name: string
  created_at: string
  updated_at: string
}

export interface PortalEvent {
  id: string
  event_type: string
  field_name: string | null
  old_value: string | null
  new_value: string | null
  actor_display_name: string | null
  visibility: string
  created_at: string
}

export interface CreatePortalTicketInput {
  title: string
  description?: string
  priority?: string
  category_id?: string
}

// --- API Functions ---

interface DataResponse<T> {
  data: T
}

export async function listPortalQueues(namespace: string, projectKey: string) {
  const res = await api.get<DataResponse<PortalQueue[]>>(
    `${portalPrefix(namespace)}/projects/${projectKey}/queues`
  )
  return res.data.data
}

export async function createPortalTicket(namespace: string, projectKey: string, input: CreatePortalTicketInput) {
  const res = await api.post<DataResponse<PortalTicket>>(
    `${portalPrefix(namespace)}/projects/${projectKey}/tickets`, input
  )
  return res.data.data
}

export async function listPortalTickets(
  namespace: string,
  projectKey: string,
  params?: { status?: string; search?: string; cursor?: string; limit?: number; hide_completed?: boolean }
) {
  const res = await api.get<{ data: PortalTicket[]; cursor: string; has_more: boolean; total: number }>(
    `${portalPrefix(namespace)}/projects/${projectKey}/tickets`, { params }
  )
  return {
    items: res.data.data,
    cursor: res.data.cursor,
    has_more: res.data.has_more,
    total: res.data.total,
  } as PortalTicketList
}

export async function getPortalTicket(namespace: string, projectKey: string, itemNumber: number) {
  const res = await api.get<DataResponse<PortalTicket>>(
    `${portalPrefix(namespace)}/projects/${projectKey}/tickets/${itemNumber}`
  )
  return res.data.data
}

export async function updatePortalTicket(namespace: string, projectKey: string, itemNumber: number, input: { title?: string; description?: string | null }) {
  const res = await api.patch<DataResponse<PortalTicket>>(
    `${portalPrefix(namespace)}/projects/${projectKey}/tickets/${itemNumber}`, input
  )
  return res.data.data
}

export async function listPortalComments(namespace: string, projectKey: string, itemNumber: number) {
  const res = await api.get<DataResponse<PortalComment[]>>(
    `${portalPrefix(namespace)}/projects/${projectKey}/tickets/${itemNumber}/comments`
  )
  return res.data.data
}

export async function addPortalComment(namespace: string, projectKey: string, itemNumber: number, body: string) {
  const res = await api.post<DataResponse<PortalComment>>(
    `${portalPrefix(namespace)}/projects/${projectKey}/tickets/${itemNumber}/comments`, { body }
  )
  return res.data.data
}

export async function listPortalEvents(namespace: string, projectKey: string, itemNumber: number) {
  const res = await api.get<DataResponse<PortalEvent[]>>(
    `${portalPrefix(namespace)}/projects/${projectKey}/tickets/${itemNumber}/events`
  )
  return res.data.data
}

// --- Attachments ---

export interface PortalAttachment {
  id: string
  uploader_id: string
  filename: string
  content_type: string
  size_bytes: number
  comment: string
  download_url: string
  created_at: string
}

export async function listPortalAttachments(namespace: string, projectKey: string, itemNumber: number) {
  const res = await api.get<DataResponse<PortalAttachment[]>>(
    `${portalPrefix(namespace)}/projects/${projectKey}/tickets/${itemNumber}/attachments`
  )
  return res.data.data
}

export async function uploadPortalAttachment(namespace: string, projectKey: string, itemNumber: number, file: File, comment?: string) {
  const formData = new FormData()
  formData.append('file', file)
  if (comment) formData.append('comment', comment)
  const res = await api.post<DataResponse<PortalAttachment>>(
    `${portalPrefix(namespace)}/projects/${projectKey}/tickets/${itemNumber}/attachments`,
    formData,
    { headers: { 'Content-Type': 'multipart/form-data' } }
  )
  return res.data.data
}

export async function updatePortalAttachmentComment(namespace: string, projectKey: string, itemNumber: number, attachmentId: string, comment: string) {
  const res = await api.patch<DataResponse<PortalAttachment>>(
    `${portalPrefix(namespace)}/projects/${projectKey}/tickets/${itemNumber}/attachments/${attachmentId}`,
    { comment }
  )
  return res.data.data
}

export async function deletePortalAttachment(namespace: string, projectKey: string, itemNumber: number, attachmentId: string) {
  await api.delete(`${portalPrefix(namespace)}/projects/${projectKey}/tickets/${itemNumber}/attachments/${attachmentId}`)
}
