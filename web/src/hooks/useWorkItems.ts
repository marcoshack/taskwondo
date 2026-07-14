import { useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listWorkItems,
  getWorkItem,
  createWorkItem,
  updateWorkItem,
  deleteWorkItem,
  listComments,
  createComment,
  createInlineComment,
  updateComment,
  deleteComment,
  listDescriptionRevisions,
  getDescriptionRevision,
  type CreateInlineCommentInput,
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
import {
  touchWorkItemDetailCache,
  WORK_ITEM_DETAIL_CACHE_LIMIT,
} from '@/hooks/workItemDetailCache'

export function useWorkItems(projectKey: string, filter: WorkItemFilter = {}, refetchInterval?: number) {
  return useQuery({
    queryKey: ['projects', projectKey, 'items', filter],
    queryFn: () => listWorkItems(projectKey, filter),
    enabled: !!projectKey,
    refetchInterval: refetchInterval || undefined,
  })
}

export interface UseWorkItemOptions {
  /** Keep the last N full detail responses in cache for instant pane reopen (TASK-79). */
  retainInCache?: boolean
  /** Override retained entry count (default {@link WORK_ITEM_DETAIL_CACHE_LIMIT}). */
  cacheLimit?: number
  staleTime?: number
}

export function useWorkItem(projectKey: string, itemNumber: number, options?: UseWorkItemOptions) {
  const queryClient = useQueryClient()
  const retainInCache = options?.retainInCache ?? false
  const cacheLimit = options?.cacheLimit ?? WORK_ITEM_DETAIL_CACHE_LIMIT

  const query = useQuery({
    queryKey: ['projects', projectKey, 'items', itemNumber],
    queryFn: () => getWorkItem(projectKey, itemNumber),
    enabled: !!projectKey && itemNumber > 0,
    staleTime: options?.staleTime,
    // Retain briefly so navigating back to a recently viewed item is instant.
    gcTime: retainInCache ? 15 * 60 * 1000 : undefined,
  })

  useEffect(() => {
    if (!retainInCache || !query.isSuccess || !projectKey || itemNumber <= 0) return
    touchWorkItemDetailCache(queryClient, projectKey, itemNumber, cacheLimit)
  }, [retainInCache, query.isSuccess, query.dataUpdatedAt, projectKey, itemNumber, queryClient, cacheLimit])

  return query
}

export function useCreateWorkItem(projectKey: string, namespaceSlug?: string) {
  const qc = useQueryClient()
  return useMutation({
    // `namespaceSlug` is required when the target project lives in a namespace
    // other than the currently active one — e.g. the cross-namespace picker in
    // the Inbox "New Item" modal.
    mutationFn: (input: CreateWorkItemInput) => createWorkItem(projectKey, input, namespaceSlug),
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
    mutationFn: ({ commentId, body, visibility }: { commentId: string; body: string; visibility?: string }) =>
      updateComment(projectKey, itemNumber, commentId, body, visibility),
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

export function useCreateInlineComment(projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateInlineCommentInput) =>
      createInlineComment(projectKey, itemNumber, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'comments'] })
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'description-revisions'] })
    },
  })
}

export function useDescriptionRevisions(projectKey: string, itemNumber: number, enabled = true) {
  return useQuery({
    queryKey: ['projects', projectKey, 'items', itemNumber, 'description-revisions'],
    queryFn: () => listDescriptionRevisions(projectKey, itemNumber),
    enabled: enabled && !!projectKey && itemNumber > 0,
  })
}

export function useDescriptionRevision(projectKey: string, itemNumber: number, revId: string | null) {
  return useQuery({
    queryKey: ['projects', projectKey, 'items', itemNumber, 'description-revisions', revId],
    queryFn: () => getDescriptionRevision(projectKey, itemNumber, revId!),
    enabled: !!projectKey && itemNumber > 0 && !!revId,
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

export function useWatchedItems(projectKeys: string[], filter: WIF = {}, refetchInterval?: number) {
  return useQuery({
    queryKey: ['watchedItems', 'list', projectKeys, filter],
    queryFn: () => listWatchedItems(projectKeys, filter),
    refetchInterval: refetchInterval || undefined,
  })
}
