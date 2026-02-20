import { useQuery } from '@tanstack/react-query'
import { listWorkflows, getWorkflow, getTransitionsMap } from '@/api/workflows'
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
 * Priority: type-specific mapping > project default > first global default.
 */
export function useProjectWorkflow(projectKey: string, workItemType?: string) {
  const { data: project } = useProject(projectKey)
  const { data: allWorkflows } = useWorkflows()
  const { data: typeWorkflows } = useTypeWorkflows(projectKey)

  // Resolve workflow ID: type-specific > project default > first global default
  let workflowId = ''
  if (workItemType && typeWorkflows) {
    const mapping = typeWorkflows.find((tw) => tw.work_item_type === workItemType)
    if (mapping) workflowId = mapping.workflow_id
  }
  if (!workflowId) {
    workflowId = project?.default_workflow_id
      ?? allWorkflows?.find((w) => w.is_default)?.id
      ?? ''
  }

  const workflowQuery = useWorkflow(workflowId)
  const transitionsQuery = useTransitionsMap(workflowId)

  return {
    workflowId,
    workflow: workflowQuery.data,
    statuses: workflowQuery.data?.statuses ?? [],
    transitionsMap: transitionsQuery.data,
  }
}
