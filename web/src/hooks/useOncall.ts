import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getOncallRotation,
  createOncallRotation,
  updateOncallRotation,
  deleteOncallRotation,
  getOncallHistory,
  type CreateOncallRotationInput,
  type UpdateOncallRotationInput,
} from '@/api/oncall'

export function useOncallRotation(projectKey: string, teamId: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'teams', teamId, 'oncall'],
    queryFn: () => getOncallRotation(projectKey, teamId),
    enabled: !!projectKey && !!teamId,
    retry: (failureCount, error) => {
      // Don't retry on 404 (no rotation configured)
      if ((error as { response?: { status?: number } })?.response?.status === 404) return false
      return failureCount < 3
    },
  })
}

export function useOncallHistory(projectKey: string, teamId: string, limit?: number, offset?: number) {
  return useQuery({
    queryKey: ['projects', projectKey, 'teams', teamId, 'oncall', 'history', limit, offset],
    queryFn: () => getOncallHistory(projectKey, teamId, limit, offset),
    enabled: !!projectKey && !!teamId,
  })
}

export function useCreateOncallRotation(projectKey: string, teamId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateOncallRotationInput) => createOncallRotation(projectKey, teamId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'teams', teamId, 'oncall'] })
    },
  })
}

export function useUpdateOncallRotation(projectKey: string, teamId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: UpdateOncallRotationInput) => updateOncallRotation(projectKey, teamId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'teams', teamId, 'oncall'] })
    },
  })
}

export function useDeleteOncallRotation(projectKey: string, teamId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => deleteOncallRotation(projectKey, teamId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'teams', teamId, 'oncall'] })
    },
  })
}
