import { api, nsPrefix } from './client'

// --- Types ---

export interface SLATarget {
  id: string
  work_item_type: string
  workflow_id: string
  status_name: string
  target_seconds: number
  calendar_mode: '24x7' | 'business_hours'
  created_at: string
  updated_at: string
}

export interface SLATargetInput {
  status_name: string
  target_seconds: number
  calendar_mode: string
}

export interface BulkUpsertSLAInput {
  work_item_type: string
  workflow_id: string
  targets: SLATargetInput[]
}

// --- API Functions ---

interface DataResponse<T> {
  data: T
}

export async function listSLATargets(projectKey: string) {
  const res = await api.get<DataResponse<SLATarget[]>>(`${nsPrefix()}/projects/${projectKey}/sla-targets`)
  return res.data.data
}

export async function bulkUpsertSLATargets(projectKey: string, input: BulkUpsertSLAInput) {
  const res = await api.put<DataResponse<SLATarget[]>>(`${nsPrefix()}/projects/${projectKey}/sla-targets`, input)
  return res.data.data
}

export async function deleteSLATarget(projectKey: string, targetId: string) {
  await api.delete(`${nsPrefix()}/projects/${projectKey}/sla-targets/${targetId}`)
}
