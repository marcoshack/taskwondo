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
