import { api } from './client'

export interface BusinessHoursConfig {
  days: number[]       // 0=Sun, 1=Mon, ..., 6=Sat
  start_hour: number   // 0-23
  end_hour: number     // 0-23
  timezone: string     // IANA timezone name
}

export interface Project {
  id: string
  name: string
  key: string
  description?: string
  default_workflow_id?: string
  allowed_complexity_values: number[]
  business_hours?: BusinessHoursConfig | null
  item_counter: number
  member_count: number
  open_count: number
  in_progress_count: number
  created_at: string
  updated_at: string
}

interface ProjectListResponse {
  data: Project[]
  meta: {
    owned_project_count: number
    max_projects: number
  }
}

interface ProjectResponse {
  data: Project
}

export interface ProjectListResult {
  projects: Project[]
  ownedProjectCount: number
  maxProjects: number
}

export async function listProjects(): Promise<ProjectListResult> {
  const res = await api.get<ProjectListResponse>('/projects')
  return {
    projects: res.data.data,
    ownedProjectCount: res.data.meta.owned_project_count,
    maxProjects: res.data.meta.max_projects,
  }
}

export async function getProject(projectKey: string) {
  const res = await api.get<ProjectResponse>(`/projects/${projectKey}`)
  return res.data.data
}

export interface CreateProjectInput {
  name: string
  key: string
  description?: string
}

export async function createProject(input: CreateProjectInput) {
  const res = await api.post<ProjectResponse>('/projects', input)
  return res.data.data
}

export interface ProjectMember {
  user_id: string
  email: string
  display_name: string
  avatar_url?: string
  role: string
  created_at: string
}

export async function listMembers(projectKey: string) {
  const res = await api.get<{ data: ProjectMember[] }>(`/projects/${projectKey}/members`)
  return res.data.data
}

export interface AddMemberInput {
  user_id: string
  role: string
}

export async function addMember(projectKey: string, input: AddMemberInput) {
  const res = await api.post<{ data: ProjectMember }>(`/projects/${projectKey}/members`, input)
  return res.data.data
}

export async function updateMemberRole(projectKey: string, userId: string, role: string) {
  await api.patch(`/projects/${projectKey}/members/${userId}`, { role })
}

export async function removeMember(projectKey: string, userId: string) {
  await api.delete(`/projects/${projectKey}/members/${userId}`)
}

export interface UpdateProjectInput {
  name?: string
  description?: string | null
  allowed_complexity_values?: number[]
  business_hours?: BusinessHoursConfig | null
}

export async function updateProject(projectKey: string, input: UpdateProjectInput) {
  const res = await api.patch<ProjectResponse>(`/projects/${projectKey}`, input)
  return res.data.data
}

export async function deleteProject(projectKey: string) {
  await api.delete(`/projects/${projectKey}`)
}

// --- Type Workflow Mappings ---

export interface ProjectTypeWorkflow {
  work_item_type: string
  workflow_id: string
}

export async function getTypeWorkflows(projectKey: string) {
  const res = await api.get<{ data: ProjectTypeWorkflow[] }>(`/projects/${projectKey}/type-workflows`)
  return res.data.data
}

export async function updateTypeWorkflow(projectKey: string, workItemType: string, workflowId: string) {
  const res = await api.put<{ data: ProjectTypeWorkflow }>(`/projects/${projectKey}/type-workflows/${workItemType}`, { workflow_id: workflowId })
  return res.data.data
}
