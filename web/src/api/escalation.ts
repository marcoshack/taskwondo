import { api, nsPrefix } from './client'

// --- Types ---

export interface EscalationLevelUser {
  id: string
  display_name: string
  email: string
}

export interface EscalationLevel {
  id: string
  escalation_list_id: string
  threshold_pct: number
  position: number
  users: EscalationLevelUser[]
}

export interface EscalationList {
  id: string
  name: string
  levels: EscalationLevel[]
  created_at: string
  updated_at: string
}

export interface TypeEscalationMapping {
  work_item_type: string
  escalation_list_id: string
}

export interface EscalationListInput {
  name: string
  levels: { threshold_pct: number; user_ids: string[] }[]
}

// --- API Functions ---

interface DataResponse<T> {
  data: T
}

const base = (projectKey: string) => `${nsPrefix()}/projects/${projectKey}/escalation-lists`

export async function listEscalationLists(projectKey: string): Promise<EscalationList[]> {
  const res = await api.get<DataResponse<EscalationList[]>>(base(projectKey))
  return res.data.data
}

export async function getEscalationList(projectKey: string, id: string): Promise<EscalationList> {
  const res = await api.get<DataResponse<EscalationList>>(`${base(projectKey)}/${id}`)
  return res.data.data
}

export async function createEscalationList(projectKey: string, input: EscalationListInput): Promise<EscalationList> {
  const res = await api.post<DataResponse<EscalationList>>(base(projectKey), input)
  return res.data.data
}

export async function updateEscalationList(projectKey: string, id: string, input: EscalationListInput): Promise<EscalationList> {
  const res = await api.put<DataResponse<EscalationList>>(`${base(projectKey)}/${id}`, input)
  return res.data.data
}

export async function deleteEscalationList(projectKey: string, id: string): Promise<void> {
  await api.delete(`${base(projectKey)}/${id}`)
}

export async function getEscalationMappings(projectKey: string): Promise<TypeEscalationMapping[]> {
  const res = await api.get<DataResponse<TypeEscalationMapping[]>>(`${base(projectKey)}/mappings`)
  return res.data.data
}

export async function updateEscalationMapping(projectKey: string, workItemType: string, escalationListId: string): Promise<void> {
  await api.put(`${base(projectKey)}/mappings/${workItemType}`, { escalation_list_id: escalationListId })
}

export async function deleteEscalationMapping(projectKey: string, workItemType: string): Promise<void> {
  await api.delete(`${base(projectKey)}/mappings/${workItemType}`)
}
