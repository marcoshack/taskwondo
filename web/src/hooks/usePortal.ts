import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listPortalQueues,
  createPortalTicket,
  updatePortalTicket,
  listPortalTickets,
  getPortalTicket,
  listPortalComments,
  addPortalComment,
  listPortalEvents,
  listPortalAttachments,
  uploadPortalAttachment,
  updatePortalAttachmentComment,
  deletePortalAttachment,
  type CreatePortalTicketInput,
} from '@/api/portal'

export function usePortalQueues(namespace: string, projectKey: string) {
  return useQuery({
    queryKey: ['portal', namespace, projectKey, 'queues'],
    queryFn: () => listPortalQueues(namespace, projectKey),
    enabled: !!namespace && !!projectKey,
  })
}

export function usePortalTickets(namespace: string, projectKey: string, params?: { status?: string; search?: string; hide_completed?: boolean }, refetchInterval?: number) {
  return useQuery({
    queryKey: ['portal', namespace, projectKey, 'tickets', params],
    queryFn: () => listPortalTickets(namespace, projectKey, params),
    enabled: !!namespace && !!projectKey,
    refetchInterval: refetchInterval || undefined,
  })
}

export function usePortalTicket(namespace: string, projectKey: string, itemNumber: number) {
  return useQuery({
    queryKey: ['portal', namespace, projectKey, 'tickets', itemNumber],
    queryFn: () => getPortalTicket(namespace, projectKey, itemNumber),
    enabled: !!namespace && !!projectKey && !!itemNumber,
  })
}

export function usePortalComments(namespace: string, projectKey: string, itemNumber: number) {
  return useQuery({
    queryKey: ['portal', namespace, projectKey, 'tickets', itemNumber, 'comments'],
    queryFn: () => listPortalComments(namespace, projectKey, itemNumber),
    enabled: !!namespace && !!projectKey && !!itemNumber,
  })
}

export function usePortalEvents(namespace: string, projectKey: string, itemNumber: number) {
  return useQuery({
    queryKey: ['portal', namespace, projectKey, 'tickets', itemNumber, 'events'],
    queryFn: () => listPortalEvents(namespace, projectKey, itemNumber),
    enabled: !!namespace && !!projectKey && !!itemNumber,
  })
}

export function useCreatePortalTicket(namespace: string, projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreatePortalTicketInput) => createPortalTicket(namespace, projectKey, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['portal', namespace, projectKey, 'tickets'] })
    },
  })
}

export function useUpdatePortalTicket(namespace: string, projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: { title?: string; description?: string | null }) => updatePortalTicket(namespace, projectKey, itemNumber, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['portal', namespace, projectKey, 'tickets', itemNumber] })
    },
  })
}

export function useAddPortalComment(namespace: string, projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: string) => addPortalComment(namespace, projectKey, itemNumber, body),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: ['portal', namespace, projectKey, 'tickets', itemNumber, 'comments'],
      })
    },
  })
}

export function usePortalAttachments(namespace: string, projectKey: string, itemNumber: number) {
  return useQuery({
    queryKey: ['portal', namespace, projectKey, 'tickets', itemNumber, 'attachments'],
    queryFn: () => listPortalAttachments(namespace, projectKey, itemNumber),
    enabled: !!namespace && !!projectKey && !!itemNumber,
  })
}

export function useUploadPortalAttachment(namespace: string, projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ file, comment }: { file: File; comment?: string }) =>
      uploadPortalAttachment(namespace, projectKey, itemNumber, file, comment),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: ['portal', namespace, projectKey, 'tickets', itemNumber, 'attachments'],
      })
    },
  })
}

export function useUpdatePortalAttachmentComment(namespace: string, projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ attachmentId, comment }: { attachmentId: string; comment: string }) =>
      updatePortalAttachmentComment(namespace, projectKey, itemNumber, attachmentId, comment),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: ['portal', namespace, projectKey, 'tickets', itemNumber, 'attachments'],
      })
    },
  })
}

export function useDeletePortalAttachment(namespace: string, projectKey: string, itemNumber: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (attachmentId: string) => deletePortalAttachment(namespace, projectKey, itemNumber, attachmentId),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: ['portal', namespace, projectKey, 'tickets', itemNumber, 'attachments'],
      })
    },
  })
}
