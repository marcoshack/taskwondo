import { api, nsPrefix } from './client'

// --- Types ---

export interface OncallRotation {
  id: string
  team_id: string
  period_days: number
  rotation_time: string  // "HH:MM:SS"
  timezone: string
  start_date: string     // "YYYY-MM-DD"
  current_user_id: string | null
  current_position: number
  next_rotation_at: string | null
  created_at: string
  updated_at: string
}

export interface OncallRotationMember {
  id: string
  rotation_id: string
  user_id: string
  position: number
  email: string
  display_name: string
  avatar_url?: string
}

export interface OncallRotationHistory {
  id: string
  rotation_id: string
  user_id: string
  started_at: string
  ended_at: string | null
  display_name: string
  avatar_url?: string
}

export interface OncallRotationWithMembers extends OncallRotation {
  members: OncallRotationMember[]
}

export interface CreateOncallRotationInput {
  period_days: number
  rotation_time: string
  timezone: string
  start_date: string
  member_ids: string[]    // ordered by position
}

export interface UpdateOncallRotationInput {
  period_days?: number
  rotation_time?: string
  timezone?: string
  start_date?: string
  member_ids?: string[]
}

// --- API Functions ---

interface DataResponse<T> {
  data: T
}

export async function getOncallRotation(projectKey: string, teamId: string) {
  const res = await api.get<DataResponse<OncallRotationWithMembers>>(
    `${nsPrefix()}/projects/${projectKey}/teams/${teamId}/oncall`,
  )
  return res.data.data
}

export async function createOncallRotation(projectKey: string, teamId: string, input: CreateOncallRotationInput) {
  const res = await api.post<DataResponse<OncallRotationWithMembers>>(
    `${nsPrefix()}/projects/${projectKey}/teams/${teamId}/oncall`,
    input,
  )
  return res.data.data
}

export async function updateOncallRotation(projectKey: string, teamId: string, input: UpdateOncallRotationInput) {
  const res = await api.patch<DataResponse<OncallRotationWithMembers>>(
    `${nsPrefix()}/projects/${projectKey}/teams/${teamId}/oncall`,
    input,
  )
  return res.data.data
}

export async function deleteOncallRotation(projectKey: string, teamId: string) {
  await api.delete(`${nsPrefix()}/projects/${projectKey}/teams/${teamId}/oncall`)
}

export async function getOncallHistory(projectKey: string, teamId: string, limit?: number, offset?: number) {
  const params: Record<string, string> = {}
  if (limit !== undefined) params.limit = String(limit)
  if (offset !== undefined) params.offset = String(offset)
  const res = await api.get<DataResponse<OncallRotationHistory[]>>(
    `${nsPrefix()}/projects/${projectKey}/teams/${teamId}/oncall/history`,
    { params },
  )
  return res.data.data
}
