import { api } from './client'

export interface Project {
  id: string
  name: string
  key: string
  description?: string
  default_workflow_id?: string
  item_counter: number
  created_at: string
  updated_at: string
}

interface ProjectListResponse {
  data: Project[]
}

interface ProjectResponse {
  data: Project
}

export async function listProjects() {
  const res = await api.get<ProjectListResponse>('/projects')
  return res.data.data
}

export async function getProject(projectKey: string) {
  const res = await api.get<ProjectResponse>(`/projects/${projectKey}`)
  return res.data.data
}
