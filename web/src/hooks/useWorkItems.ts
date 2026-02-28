import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listWorkItems,
  getWorkItem,
  createWorkItem,
  updateWorkItem,
  deleteWorkItem,
  listComments,
  createComment,
  updateComment,
  deleteComment,
  listRelations,
  createRelation,
  deleteRelation,
  listEvents,
  listAttachments,
  uploadAttachment,
  updateAttachmentComment,
  deleteAttachment,
  type WorkItemFilter,
  type CreateWorkItemInput,
  type UpdateWorkItemInput,
  listTimeEntries,
  createTimeEntry,
  updateTimeEntry,
  deleteTimeEntry,
  type CreateTimeEntryInput,
  type UpdateTimeEntryInput,
  listWatchers,
  addWatcher,
  removeWatcher,
  toggleWatch,
  listWatchedItemIDs,
  listWatchedItems,
  type WorkItemFilter as WIF,
} from '@/api/workitems'

export function useWorkItems(projectKey: string, filter: WorkItemFilter = {}) {
  return useQuery({
    queryKey: ['projects', projectKey, 'items', filter],
    queryFn: () => listWorkItems(projectKey, filter),
    enabled: !!projectKey,
  })
}

export function useWorkItem(projectKey: string, itemNumber: number) {
  return useQuery({
    queryKey: ['projects', projectKey, 'items', itemNumber],
    queryFn: () => getWorkItem(projectKey, itemNumber),
    enabled: !!projectKey && itemNumber > 0,
  })
}

export function useCreateWorkItem(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateWorkItemInput) => createWorkItem(projectKey, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items'] })
    },
  })
}

export function useUpdateWorkItem(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ itemNumber, input }: { itemNumber: number; input: UpdateWorkItemInput }) =>
      updateWorkItem(projectKey, itemNumber, input),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items'] })
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', vars.itemNumber] })
    },
    onError: (_err, vars) => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', vars.itemNumber] })
    },
  })
}

export function useDeleteWorkItem(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (itemNumber: number) => deleteWorkItem(projectKey, itemNumber),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items'] })
    },
  })
}

export interface BulkUpdateResult {
  succeeded: number
  failed: { itemNumber: number; message: string }[]
}

export function useBulkUpdateWorkItems(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (updates: { itemNumber: number; input: UpdateWorkItemInput }[]): Promise<BulkUpdateResult> => {
      const result: BulkUpdateResult = { succeeded: 0, failed: [] }
      for (const { itemNumber, input } of updates) {
        try {
          await updateWorkItem(projectKey, itemNumber, input)
          result.succeeded++
        } catch (err) {
          const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message ?? 'Unknown error'
          result.failed.push({ itemNumber, message: msg })
        }
      }
      return result
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items'] })
    },
  })
}

// --- Comment hooks ---

export function useComments(projectKey: string, itemNumber: number) {
  return useQuery({
    queryKey: ['projects', projectKey, 'items', itemNumber, 'comments'],
    queryFn: () => listComments(projectKey, itemNumber),
    enabled: !!projectKey && itemNumber > 0,
  })
}

export function useCreateComment(projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ body, visibility }: { body: string; visibility?: string }) =>
      createComment(projectKey, itemNumber, body, visibility),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'comments'] })
    },
  })
}

export function useUpdateComment(projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ commentId, body }: { commentId: string; body: string }) =>
      updateComment(projectKey, itemNumber, commentId, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'comments'] })
    },
  })
}

export function useDeleteComment(projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (commentId: string) => deleteComment(projectKey, itemNumber, commentId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'comments'] })
    },
  })
}

// --- Relation hooks ---

export function useRelations(projectKey: string, itemNumber: number) {
  return useQuery({
    queryKey: ['projects', projectKey, 'items', itemNumber, 'relations'],
    queryFn: () => listRelations(projectKey, itemNumber),
    enabled: !!projectKey && itemNumber > 0,
  })
}

export function useCreateRelation(projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ targetDisplayId, relationType }: { targetDisplayId: string; relationType: string }) =>
      createRelation(projectKey, itemNumber, targetDisplayId, relationType),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'relations'] })
    },
  })
}

export function useDeleteRelation(projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (relationId: string) => deleteRelation(projectKey, itemNumber, relationId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'relations'] })
    },
  })
}

// --- Event hooks ---

export function useEvents(projectKey: string, itemNumber: number) {
  return useQuery({
    queryKey: ['projects', projectKey, 'items', itemNumber, 'events'],
    queryFn: () => listEvents(projectKey, itemNumber),
    enabled: !!projectKey && itemNumber > 0,
  })
}

// --- Attachment hooks ---

export function useAttachments(projectKey: string, itemNumber: number) {
  return useQuery({
    queryKey: ['projects', projectKey, 'items', itemNumber, 'attachments'],
    queryFn: () => listAttachments(projectKey, itemNumber),
    enabled: !!projectKey && itemNumber > 0,
  })
}

export function useUploadAttachment(projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ file, comment }: { file: File; comment?: string }) =>
      uploadAttachment(projectKey, itemNumber, file, comment),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'attachments'] })
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'events'] })
    },
  })
}

export function useUpdateAttachmentComment(projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ attachmentId, comment }: { attachmentId: string; comment: string }) =>
      updateAttachmentComment(projectKey, itemNumber, attachmentId, comment),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'attachments'] })
    },
  })
}

export function useDeleteAttachment(projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (attachmentId: string) =>
      deleteAttachment(projectKey, itemNumber, attachmentId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'attachments'] })
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'events'] })
    },
  })
}

// --- Time entry hooks ---

export function useTimeEntries(projectKey: string, itemNumber: number) {
  return useQuery({
    queryKey: ['projects', projectKey, 'items', itemNumber, 'timeEntries'],
    queryFn: () => listTimeEntries(projectKey, itemNumber),
    enabled: !!projectKey && itemNumber > 0,
  })
}

export function useCreateTimeEntry(projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateTimeEntryInput) =>
      createTimeEntry(projectKey, itemNumber, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'timeEntries'] })
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'events'] })
    },
  })
}

export function useUpdateTimeEntry(projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ entryId, input }: { entryId: string; input: UpdateTimeEntryInput }) =>
      updateTimeEntry(projectKey, itemNumber, entryId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'timeEntries'] })
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'events'] })
    },
  })
}

export function useDeleteTimeEntry(projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (entryId: string) =>
      deleteTimeEntry(projectKey, itemNumber, entryId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'timeEntries'] })
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'events'] })
    },
  })
}

// --- Watcher hooks ---

export function useWatchers(projectKey: string, itemNumber: number) {
  return useQuery({
    queryKey: ['projects', projectKey, 'items', itemNumber, 'watchers'],
    queryFn: () => listWatchers(projectKey, itemNumber),
    enabled: !!projectKey && itemNumber > 0,
  })
}

export function useAddWatcher(projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) => addWatcher(projectKey, itemNumber, userId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'watchers'] })
      qc.invalidateQueries({ queryKey: ['watchedItems'] })
    },
  })
}

export function useRemoveWatcher(projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) => removeWatcher(projectKey, itemNumber, userId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'watchers'] })
      qc.invalidateQueries({ queryKey: ['watchedItems'] })
    },
  })
}

export function useToggleWatch(projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => toggleWatch(projectKey, itemNumber),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'watchers'] })
      qc.invalidateQueries({ queryKey: ['watchedItems'] })
    },
  })
}

export function useWatchedItemIDs(projectKey?: string) {
  return useQuery({
    queryKey: ['watchedItems', projectKey ?? ''],
    queryFn: () => listWatchedItemIDs(projectKey),
  })
}

export function useWatchedItems(projectKeys: string[], filter: WIF = {}) {
  return useQuery({
    queryKey: ['watchedItems', 'list', projectKeys, filter],
    queryFn: () => listWatchedItems(projectKeys, filter),
  })
}
