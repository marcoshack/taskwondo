import { api } from './client'

// --- Types ---

export interface WorkflowStatus {
  id: string
  name: string
  display_name: string
  category: 'todo' | 'in_progress' | 'done' | 'cancelled'
  position: number
  color: string | null
}

export interface WorkflowTransition {
  id: string
  from_status: string
  to_status: string
  name: string | null
}

export interface Workflow {
  id: string
  project_id?: string | null
  name: string
  description: string | null
  is_default: boolean
  statuses: WorkflowStatus[]
  created_at: string
  updated_at: string
}

export interface WorkflowDetail extends Workflow {
  transitions: WorkflowTransition[]
}

export interface CreateWorkflowInput {
  name: string
  description?: string | null
  statuses: Omit<WorkflowStatus, 'id'>[]
  transitions: Omit<WorkflowTransition, 'id'>[]
}

export interface UpdateWorkflowInput {
  name?: string
  description?: string | null
  statuses?: Omit<WorkflowStatus, 'id'>[]
  transitions?: Omit<WorkflowTransition, 'id'>[]
}

// --- API Functions ---

interface DataResponse<T> {
  data: T
}

export async function listWorkflows() {
  const res = await api.get<DataResponse<Workflow[]>>('/workflows')
  return res.data.data
}

export async function getWorkflow(workflowId: string) {
  const res = await api.get<DataResponse<WorkflowDetail>>(`/workflows/${workflowId}`)
  return res.data.data
}

export async function getTransitionsMap(workflowId: string) {
  const res = await api.get<DataResponse<Record<string, WorkflowTransition[]>>>(`/workflows/${workflowId}/transitions`)
  return res.data.data
}

// --- System Workflow API Functions ---

export async function createSystemWorkflow(input: CreateWorkflowInput) {
  const res = await api.post<DataResponse<WorkflowDetail>>('/workflows', input)
  return res.data.data
}

export async function updateSystemWorkflow(workflowId: string, input: UpdateWorkflowInput) {
  const res = await api.patch<DataResponse<WorkflowDetail>>(`/workflows/${workflowId}`, input)
  return res.data.data
}

export async function deleteSystemWorkflow(workflowId: string) {
  await api.delete(`/workflows/${workflowId}`)
}

export async function listSystemStatuses() {
  const res = await api.get<DataResponse<WorkflowStatus[]>>('/workflows/statuses')
  return res.data.data
}

// --- Project Workflow API Functions ---

export async function listProjectWorkflows(projectKey: string) {
  const res = await api.get<DataResponse<Workflow[]>>(`/projects/${projectKey}/workflows`)
  return res.data.data
}

export async function getProjectWorkflow(projectKey: string, workflowId: string) {
  const res = await api.get<DataResponse<WorkflowDetail>>(`/projects/${projectKey}/workflows/${workflowId}`)
  return res.data.data
}

export async function createProjectWorkflow(projectKey: string, input: CreateWorkflowInput) {
  const res = await api.post<DataResponse<WorkflowDetail>>(`/projects/${projectKey}/workflows`, input)
  return res.data.data
}

export async function updateProjectWorkflow(projectKey: string, workflowId: string, input: UpdateWorkflowInput) {
  const res = await api.patch<DataResponse<WorkflowDetail>>(`/projects/${projectKey}/workflows/${workflowId}`, input)
  return res.data.data
}

export async function deleteProjectWorkflow(projectKey: string, workflowId: string) {
  await api.delete(`/projects/${projectKey}/workflows/${workflowId}`)
}

export async function listAvailableStatuses(projectKey: string) {
  const res = await api.get<DataResponse<WorkflowStatus[]>>(`/projects/${projectKey}/workflows/statuses`)
  return res.data.data
}
