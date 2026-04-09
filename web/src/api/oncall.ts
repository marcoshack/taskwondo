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
  is_override: boolean
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
  overrides: OncallOverride[]
  shifts?: OncallScheduleShift[]
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

export interface OncallOverride {
  id: string
  rotation_id: string
  override_user_id: string
  override_user_name: string
  override_avatar_url?: string
  start_at: string
  end_at: string
  reason?: string
  created_by: string
  created_by_name: string
  created_at: string
}

export interface CreateOncallOverrideInput {
  override_user_id: string
  start_at: string   // RFC3339
  end_at: string     // RFC3339
  reason?: string
}

export interface UpdateOncallOverrideInput {
  override_user_id?: string
  start_at?: string
  end_at?: string
  reason?: string | null
}

// --- Schedule types ---

export interface OncallScheduleShift {
  user_id: string
  start_at: string   // RFC3339
  end_at: string     // RFC3339
  is_override: boolean
  override_id?: string
}

// --- API Functions ---

interface DataResponse<T> {
  data: T
}

export async function getOncallRotation(projectKey: string, teamId: string, start?: string, end?: string) {
  const params: Record<string, string> = {}
  if (start) params.start = start
  if (end) params.end = end
  const res = await api.get<DataResponse<OncallRotationWithMembers>>(
    `${nsPrefix()}/projects/${projectKey}/teams/${teamId}/oncall`,
    { params },
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

export async function createOncallOverride(projectKey: string, teamId: string, input: CreateOncallOverrideInput) {
  const res = await api.post<DataResponse<OncallOverride>>(
    `${nsPrefix()}/projects/${projectKey}/teams/${teamId}/oncall/overrides`,
    input,
  )
  return res.data.data
}

export async function updateOncallOverride(projectKey: string, teamId: string, overrideId: string, input: UpdateOncallOverrideInput) {
  const res = await api.patch<DataResponse<OncallOverride>>(
    `${nsPrefix()}/projects/${projectKey}/teams/${teamId}/oncall/overrides/${overrideId}`,
    input,
  )
  return res.data.data
}

export async function deleteOncallOverride(projectKey: string, teamId: string, overrideId: string) {
  await api.delete(`${nsPrefix()}/projects/${projectKey}/teams/${teamId}/oncall/overrides/${overrideId}`)
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
