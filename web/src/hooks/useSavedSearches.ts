import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listSavedSearches,
  createSavedSearch,
  updateSavedSearch,
  deleteSavedSearch,
  type CreateSavedSearchInput,
  type UpdateSavedSearchInput,
} from '@/api/savedSearches'

export function useSavedSearches(projectKey: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'saved-searches'],
    queryFn: () => listSavedSearches(projectKey),
    enabled: !!projectKey,
  })
}

export function useCreateSavedSearch(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateSavedSearchInput) => createSavedSearch(projectKey, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'saved-searches'] })
    },
  })
}

export function useUpdateSavedSearch(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ searchId, input }: { searchId: string; input: UpdateSavedSearchInput }) =>
      updateSavedSearch(projectKey, searchId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'saved-searches'] })
    },
  })
}

export function useDeleteSavedSearch(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (searchId: string) => deleteSavedSearch(projectKey, searchId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'saved-searches'] })
    },
  })
}
