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

export function useBulkUpdateWorkItems(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (updates: { itemNumber: number; input: UpdateWorkItemInput }[]) => {
      const results = []
      for (const { itemNumber, input } of updates) {
        results.push(await updateWorkItem(projectKey, itemNumber, input))
      }
      return results
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
