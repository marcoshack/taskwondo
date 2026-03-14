import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listSystemAPIKeys, createSystemAPIKey, renameSystemAPIKey, deleteSystemAPIKey } from '@/api/auth'

export function useSystemAPIKeys() {
  return useQuery({
    queryKey: ['systemAPIKeys'],
    queryFn: listSystemAPIKeys,
  })
}

export function useCreateSystemAPIKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (params: { name: string; permissions: string[]; expiresAt?: string }) =>
      createSystemAPIKey(params.name, params.permissions, params.expiresAt),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['systemAPIKeys'] }),
  })
}

export function useRenameSystemAPIKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (params: { id: string; name: string }) =>
      renameSystemAPIKey(params.id, params.name),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['systemAPIKeys'] }),
  })
}

export function useDeleteSystemAPIKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteSystemAPIKey(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['systemAPIKeys'] }),
  })
}
