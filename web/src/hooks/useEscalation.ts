import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listEscalationLists,
  getEscalationList,
  createEscalationList,
  updateEscalationList,
  deleteEscalationList,
  getEscalationMappings,
  updateEscalationMapping,
  deleteEscalationMapping,
  type EscalationListInput,
} from '@/api/escalation'

export function useEscalationLists(projectKey: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'escalation-lists'],
    queryFn: () => listEscalationLists(projectKey),
    enabled: !!projectKey,
  })
}

export function useEscalationListDetail(projectKey: string, id: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'escalation-lists', id],
    queryFn: () => getEscalationList(projectKey, id),
    enabled: !!projectKey && !!id,
  })
}

export function useCreateEscalationList(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: EscalationListInput) => createEscalationList(projectKey, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'escalation-lists'] })
    },
  })
}

export function useUpdateEscalationList(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ escalationListId, input }: { escalationListId: string; input: EscalationListInput }) =>
      updateEscalationList(projectKey, escalationListId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'escalation-lists'] })
    },
  })
}

export function useDeleteEscalationList(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteEscalationList(projectKey, id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'escalation-lists'] })
    },
  })
}

export function useEscalationMappings(projectKey: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'escalation-mappings'],
    queryFn: () => getEscalationMappings(projectKey),
    enabled: !!projectKey,
  })
}

export function useUpdateEscalationMapping(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ workItemType, escalationListId }: { workItemType: string; escalationListId: string }) =>
      updateEscalationMapping(projectKey, workItemType, escalationListId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'escalation-mappings'] })
    },
  })
}

export function useDeleteEscalationMapping(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (workItemType: string) => deleteEscalationMapping(projectKey, workItemType),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'escalation-mappings'] })
    },
  })
}
