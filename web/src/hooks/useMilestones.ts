import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listMilestones,
  getMilestone,
  getMilestoneStats,
  createMilestone,
  updateMilestone,
  deleteMilestone,
  type CreateMilestoneInput,
  type UpdateMilestoneInput,
} from '@/api/milestones'

export function useMilestones(projectKey: string, namespaceSlug?: string) {
  return useQuery({
    // Scope the cache by namespace when provided so two projects that happen
    // to share the same key in different namespaces don't collide. Keep the
    // short key shape when no override is passed to preserve existing
    // invalidations.
    queryKey: namespaceSlug
      ? ['projects', namespaceSlug, projectKey, 'milestones']
      : ['projects', projectKey, 'milestones'],
    queryFn: () => listMilestones(projectKey, namespaceSlug),
    enabled: !!projectKey,
  })
}

export function useMilestone(projectKey: string, milestoneId: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'milestones', milestoneId],
    queryFn: () => getMilestone(projectKey, milestoneId),
    enabled: !!projectKey && !!milestoneId,
  })
}

export function useMilestoneStats(projectKey: string, milestoneId: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'milestones', milestoneId, 'stats'],
    queryFn: () => getMilestoneStats(projectKey, milestoneId),
    enabled: !!projectKey && !!milestoneId,
  })
}

export function useCreateMilestone(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateMilestoneInput) => createMilestone(projectKey, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'milestones'] })
    },
  })
}

export function useUpdateMilestone(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ milestoneId, input }: { milestoneId: string; input: UpdateMilestoneInput }) =>
      updateMilestone(projectKey, milestoneId, input),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'milestones'] })
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'milestones', vars.milestoneId] })
    },
  })
}

export function useDeleteMilestone(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (milestoneId: string) => deleteMilestone(projectKey, milestoneId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'milestones'] })
    },
  })
}
