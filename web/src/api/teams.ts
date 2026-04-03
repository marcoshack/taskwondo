import { api, nsPrefix } from './client'

// --- Types ---

export interface Team {
  id: string
  project_id: string
  name: string
  description: string | null
  created_at: string
  updated_at: string
}

export interface TeamMember {
  id: string
  team_id: string
  user_id: string
  created_at: string
}

export interface TeamMemberWithUser {
  id: string
  team_id: string
  user_id: string
  email: string
  display_name: string
  avatar_url: string | null
  created_at: string
}

export interface CreateTeamInput {
  name: string
  description?: string
}

export interface UpdateTeamInput {
  name?: string
  description?: string | null
}

// --- API Functions ---

interface DataResponse<T> {
  data: T
}

export async function listTeams(projectKey: string) {
  const res = await api.get<DataResponse<Team[]>>(`${nsPrefix()}/projects/${projectKey}/teams`)
  return res.data.data
}

export async function getTeam(projectKey: string, teamId: string) {
  const res = await api.get<DataResponse<Team>>(`${nsPrefix()}/projects/${projectKey}/teams/${teamId}`)
  return res.data.data
}

export async function createTeam(projectKey: string, input: CreateTeamInput) {
  const res = await api.post<DataResponse<Team>>(`${nsPrefix()}/projects/${projectKey}/teams`, input)
  return res.data.data
}

export async function updateTeam(projectKey: string, teamId: string, input: UpdateTeamInput) {
  const res = await api.patch<DataResponse<Team>>(`${nsPrefix()}/projects/${projectKey}/teams/${teamId}`, input)
  return res.data.data
}

export async function deleteTeam(projectKey: string, teamId: string) {
  await api.delete(`${nsPrefix()}/projects/${projectKey}/teams/${teamId}`)
}

export async function listTeamMembers(projectKey: string, teamId: string) {
  const res = await api.get<DataResponse<TeamMemberWithUser[]>>(`${nsPrefix()}/projects/${projectKey}/teams/${teamId}/members`)
  return res.data.data
}

export async function addTeamMember(projectKey: string, teamId: string, userId: string) {
  const res = await api.post<DataResponse<TeamMember>>(`${nsPrefix()}/projects/${projectKey}/teams/${teamId}/members`, { user_id: userId })
  return res.data.data
}

export async function removeTeamMember(projectKey: string, teamId: string, userId: string) {
  await api.delete(`${nsPrefix()}/projects/${projectKey}/teams/${teamId}/members/${userId}`)
}
