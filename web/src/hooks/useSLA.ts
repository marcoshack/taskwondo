import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listSLATargets,
  bulkUpsertSLATargets,
  deleteSLATarget,
  type BulkUpsertSLAInput,
} from '@/api/sla'

export function useSLATargets(projectKey: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'sla-targets'],
    queryFn: () => listSLATargets(projectKey),
    enabled: !!projectKey,
  })
}

export function useBulkUpsertSLATargets(projectKey: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: BulkUpsertSLAInput) => bulkUpsertSLATargets(projectKey, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects', projectKey, 'sla-targets'] })
    },
  })
}

export function useDeleteSLATarget(projectKey: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (targetId: string) => deleteSLATarget(projectKey, targetId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects', projectKey, 'sla-targets'] })
    },
  })
}
