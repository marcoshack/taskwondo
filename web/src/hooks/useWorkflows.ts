import { useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listWorkflows,
  getWorkflow,
  getTransitionsMap,
  createSystemWorkflow,
  updateSystemWorkflow,
  deleteSystemWorkflow,
  listSystemStatuses,
  listProjectWorkflows,
  getProjectWorkflow,
  createProjectWorkflow,
  updateProjectWorkflow,
  deleteProjectWorkflow,
  listAvailableStatuses,
  type CreateWorkflowInput,
  type UpdateWorkflowInput,
  type WorkflowTransition,
} from '@/api/workflows'
import { useProject, useTypeWorkflows } from './useProjects'

export function useWorkflows() {
  return useQuery({
    queryKey: ['workflows'],
    queryFn: listWorkflows,
  })
}

export function useWorkflow(workflowId: string) {
  return useQuery({
    queryKey: ['workflows', workflowId],
    queryFn: () => getWorkflow(workflowId),
    enabled: !!workflowId,
  })
}

export function useTransitionsMap(workflowId: string) {
  return useQuery({
    queryKey: ['workflows', workflowId, 'transitions'],
    queryFn: () => getTransitionsMap(workflowId),
    enabled: !!workflowId,
  })
}

/**
 * Resolves the effective workflow for a project and optional work item type.
 * Uses project-level endpoints accessible to all project members.
 * Priority: type-specific mapping > project default > first available default.
 */
export function useProjectWorkflow(projectKey: string, workItemType?: string) {
  const { data: project } = useProject(projectKey)
  const { data: projectWorkflows } = useProjectWorkflows(projectKey)
  const { data: typeWorkflows } = useTypeWorkflows(projectKey)

  // Resolve workflow ID: type-specific > project default > first available default
  let workflowId = ''
  if (workItemType && typeWorkflows) {
    const mapping = typeWorkflows.find((tw) => tw.work_item_type === workItemType)
    if (mapping) workflowId = mapping.workflow_id
  }
  if (!workflowId) {
    workflowId = project?.default_workflow_id
      ?? projectWorkflows?.find((w) => w.is_default)?.id
      ?? ''
  }

  // Fetch workflow detail via project-level endpoint (includes statuses + transitions)
  const workflowQuery = useProjectWorkflowDetail(projectKey, workflowId)

  // Build transitions map from workflow detail
  const transitionsMap = useMemo(() => {
    if (!workflowQuery.data?.transitions) return undefined
    const map: Record<string, WorkflowTransition[]> = {}
    for (const t of workflowQuery.data.transitions) {
      if (!map[t.from_status]) map[t.from_status] = []
      map[t.from_status].push(t)
    }
    return map
  }, [workflowQuery.data?.transitions])

  return {
    workflowId,
    workflow: workflowQuery.data,
    statuses: workflowQuery.data?.statuses ?? [],
    transitionsMap,
  }
}

// --- System workflow hooks ---

export function useSystemStatuses() {
  return useQuery({
    queryKey: ['workflows', 'statuses'],
    queryFn: listSystemStatuses,
  })
}

export function useCreateSystemWorkflow() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateWorkflowInput) => createSystemWorkflow(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['workflows'] })
    },
  })
}

export function useUpdateSystemWorkflow() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ workflowId, input }: { workflowId: string; input: UpdateWorkflowInput }) =>
      updateSystemWorkflow(workflowId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['workflows'] })
    },
  })
}

export function useDeleteSystemWorkflow() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (workflowId: string) => deleteSystemWorkflow(workflowId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['workflows'] })
    },
  })
}

// --- Project workflow hooks ---

export function useProjectWorkflows(projectKey: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'workflows'],
    queryFn: () => listProjectWorkflows(projectKey),
    enabled: !!projectKey,
  })
}

export function useProjectWorkflowDetail(projectKey: string, workflowId: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'workflows', workflowId],
    queryFn: () => getProjectWorkflow(projectKey, workflowId),
    enabled: !!projectKey && !!workflowId,
  })
}

export function useCreateProjectWorkflow(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateWorkflowInput) => createProjectWorkflow(projectKey, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'workflows'] })
      qc.invalidateQueries({ queryKey: ['workflows'] })
    },
  })
}

export function useUpdateProjectWorkflow(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ workflowId, input }: { workflowId: string; input: UpdateWorkflowInput }) =>
      updateProjectWorkflow(projectKey, workflowId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'workflows'] })
      qc.invalidateQueries({ queryKey: ['workflows'] })
    },
  })
}

export function useDeleteProjectWorkflow(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (workflowId: string) => deleteProjectWorkflow(projectKey, workflowId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'workflows'] })
      qc.invalidateQueries({ queryKey: ['workflows'] })
    },
  })
}

export function useAvailableStatuses(projectKey: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'workflows', 'statuses'],
    queryFn: () => listAvailableStatuses(projectKey),
    enabled: !!projectKey,
  })
}
