import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listCategories,
  createCategory,
  updateCategory,
  deleteCategory,
  listQueueTeams,
  assignQueueTeam,
  unassignQueueTeam,
  type CreateCategoryInput,
  type UpdateCategoryInput,
} from '@/api/queueCategories'

export function useQueueCategories(projectKey: string, queueId: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'queues', queueId, 'categories'],
    queryFn: () => listCategories(projectKey, queueId),
    enabled: !!projectKey && !!queueId,
  })
}

export function useCreateCategory(projectKey: string, queueId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateCategoryInput) => createCategory(projectKey, queueId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'queues', queueId, 'categories'] })
    },
  })
}

export function useUpdateCategory(projectKey: string, queueId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ categoryId, input }: { categoryId: string; input: UpdateCategoryInput }) =>
      updateCategory(projectKey, queueId, categoryId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'queues', queueId, 'categories'] })
    },
  })
}

export function useDeleteCategory(projectKey: string, queueId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (categoryId: string) => deleteCategory(projectKey, queueId, categoryId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'queues', queueId, 'categories'] })
    },
  })
}

export function useQueueTeams(projectKey: string, queueId: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'queues', queueId, 'teams'],
    queryFn: () => listQueueTeams(projectKey, queueId),
    enabled: !!projectKey && !!queueId,
  })
}

export function useAssignQueueTeam(projectKey: string, queueId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (teamId: string) => assignQueueTeam(projectKey, queueId, teamId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'queues', queueId, 'teams'] })
    },
  })
}

export function useUnassignQueueTeam(projectKey: string, queueId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (teamId: string) => unassignQueueTeam(projectKey, queueId, teamId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'queues', queueId, 'teams'] })
    },
  })
}
