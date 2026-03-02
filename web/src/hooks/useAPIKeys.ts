import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listAPIKeys, createAPIKey, renameAPIKey, deleteAPIKey } from '@/api/auth'

export function useAPIKeys() {
  return useQuery({
    queryKey: ['apiKeys'],
    queryFn: listAPIKeys,
  })
}

export function useCreateAPIKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (params: { name: string; permissions: string[]; expiresAt?: string }) =>
      createAPIKey(params.name, params.permissions, params.expiresAt),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['apiKeys'] }),
  })
}

export function useRenameAPIKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (params: { id: string; name: string }) =>
      renameAPIKey(params.id, params.name),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['apiKeys'] }),
  })
}

export function useDeleteAPIKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteAPIKey(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['apiKeys'] }),
  })
}
