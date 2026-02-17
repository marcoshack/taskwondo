import { useQuery } from '@tanstack/react-query'
import { listWorkflows, getWorkflow, getTransitionsMap } from '@/api/workflows'
import { useProject } from './useProjects'

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
 * Resolves the effective workflow for a project.
 * Uses the project's default_workflow_id if set, otherwise falls back
 * to the first default workflow from the global list.
 */
export function useProjectWorkflow(projectKey: string) {
  const { data: project } = useProject(projectKey)
  const { data: allWorkflows } = useWorkflows()

  // Resolve the workflow ID: project-specific or first global default
  const workflowId = project?.default_workflow_id
    ?? allWorkflows?.find((w) => w.is_default)?.id
    ?? ''

  const workflowQuery = useWorkflow(workflowId)
  const transitionsQuery = useTransitionsMap(workflowId)

  return {
    workflowId,
    workflow: workflowQuery.data,
    statuses: workflowQuery.data?.statuses ?? [],
    transitionsMap: transitionsQuery.data,
  }
}
