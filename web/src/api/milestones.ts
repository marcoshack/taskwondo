import { api, nsPrefix } from './client'

// --- Types ---

export interface Milestone {
  id: string
  project_id: string
  name: string
  description: string | null
  due_date: string | null
  status: 'open' | 'closed'
  open_count: number
  closed_count: number
  total_count: number
  total_estimated_seconds: number
  total_spent_seconds: number
  created_at: string
  updated_at: string
}

export interface CreateMilestoneInput {
  name: string
  description?: string
  due_date?: string
}

export interface UpdateMilestoneInput {
  name?: string
  description?: string | null
  due_date?: string | null
  status?: 'open' | 'closed'
}

// --- API Functions ---

interface DataResponse<T> {
  data: T
}

export async function listMilestones(projectKey: string) {
  const res = await api.get<DataResponse<Milestone[]>>(`${nsPrefix()}/projects/${projectKey}/milestones`)
  return res.data.data
}

export async function getMilestone(projectKey: string, milestoneId: string) {
  const res = await api.get<DataResponse<Milestone>>(`${nsPrefix()}/projects/${projectKey}/milestones/${milestoneId}`)
  return res.data.data
}

export async function createMilestone(projectKey: string, input: CreateMilestoneInput) {
  const res = await api.post<DataResponse<Milestone>>(`${nsPrefix()}/projects/${projectKey}/milestones`, input)
  return res.data.data
}

export async function updateMilestone(projectKey: string, milestoneId: string, input: UpdateMilestoneInput) {
  const res = await api.patch<DataResponse<Milestone>>(`${nsPrefix()}/projects/${projectKey}/milestones/${milestoneId}`, input)
  return res.data.data
}

export async function deleteMilestone(projectKey: string, milestoneId: string) {
  await api.delete(`${nsPrefix()}/projects/${projectKey}/milestones/${milestoneId}`)
}

// --- Stats Types ---

export interface StatusCount {
  open: number
  closed: number
}

export interface MilestoneStats {
  by_type: Record<string, StatusCount>
  by_priority: Record<string, StatusCount>
  by_label: Record<string, number>
}

export async function getMilestoneStats(projectKey: string, milestoneId: string) {
  const res = await api.get<DataResponse<MilestoneStats>>(`${nsPrefix()}/projects/${projectKey}/milestones/${milestoneId}/stats`)
  return res.data.data
}
