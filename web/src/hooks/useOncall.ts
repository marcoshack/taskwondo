import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getOncallRotation,
  createOncallRotation,
  updateOncallRotation,
  deleteOncallRotation,
  getOncallHistory,
  listOncallOverrides,
  createOncallOverride,
  updateOncallOverride,
  deleteOncallOverride,
  type CreateOncallRotationInput,
  type UpdateOncallRotationInput,
  type CreateOncallOverrideInput,
  type UpdateOncallOverrideInput,
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

export function useOncallOverrides(projectKey: string, teamId: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'teams', teamId, 'oncall', 'overrides'],
    queryFn: () => listOncallOverrides(projectKey, teamId),
    enabled: !!projectKey && !!teamId,
  })
}

export function useCreateOncallOverride(projectKey: string, teamId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateOncallOverrideInput) => createOncallOverride(projectKey, teamId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'teams', teamId, 'oncall'] })
    },
  })
}

export function useUpdateOncallOverride(projectKey: string, teamId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ overrideId, input }: { overrideId: string; input: UpdateOncallOverrideInput }) =>
      updateOncallOverride(projectKey, teamId, overrideId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'teams', teamId, 'oncall'] })
    },
  })
}

export function useDeleteOncallOverride(projectKey: string, teamId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (overrideId: string) => deleteOncallOverride(projectKey, teamId, overrideId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'teams', teamId, 'oncall'] })
    },
  })
}
