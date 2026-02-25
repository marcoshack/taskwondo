import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listInboxItems,
  addToInbox,
  removeFromInbox,
  reorderInboxItem,
  clearCompletedInboxItems,
  getInboxCount,
  type InboxFilter,
} from '@/api/inbox'

export function useInboxItems(filter: InboxFilter = {}) {
  return useQuery({
    queryKey: ['inbox', 'items', filter],
    queryFn: () => listInboxItems(filter),
  })
}

export function useInboxCount() {
  return useQuery({
    queryKey: ['inbox', 'count'],
    queryFn: getInboxCount,
    refetchInterval: 60000, // refresh count every minute
  })
}

export function useAddToInbox() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (workItemId: string) => addToInbox(workItemId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['inbox'] })
    },
  })
}

export function useRemoveFromInbox() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (inboxItemId: string) => removeFromInbox(inboxItemId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['inbox'] })
    },
  })
}

export function useReorderInboxItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ inboxItemId, position }: { inboxItemId: string; position: number }) =>
      reorderInboxItem(inboxItemId, position),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['inbox', 'items'] })
    },
  })
}

export function useClearCompletedInbox() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => clearCompletedInboxItems(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['inbox'] })
    },
  })
}
