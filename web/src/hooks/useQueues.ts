import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listQueues,
  getQueue,
  createQueue,
  updateQueue,
  deleteQueue,
  type CreateQueueInput,
  type UpdateQueueInput,
} from '@/api/queues'

export function useQueues(projectKey: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'queues'],
    queryFn: () => listQueues(projectKey),
    enabled: !!projectKey,
  })
}

export function useQueue(projectKey: string, queueId: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'queues', queueId],
    queryFn: () => getQueue(projectKey, queueId),
    enabled: !!projectKey && !!queueId,
  })
}

export function useCreateQueue(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateQueueInput) => createQueue(projectKey, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'queues'] })
    },
  })
}

export function useUpdateQueue(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ queueId, input }: { queueId: string; input: UpdateQueueInput }) =>
      updateQueue(projectKey, queueId, input),
    onSuccess: (_d, vars) => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'queues'] })
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'queues', vars.queueId] })
    },
  })
}

export function useDeleteQueue(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (queueId: string) => deleteQueue(projectKey, queueId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'queues'] })
    },
  })
}
