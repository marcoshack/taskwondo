import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listProjects,
  getProject,
  createProject,
  listMembers,
  addMember,
  updateMemberRole,
  removeMember,
  updateProject,
  deleteProject,
  getTypeWorkflows,
  updateTypeWorkflow,
  listInvites,
  createInvite,
  deleteInvite,
  getInviteInfo,
  acceptInvite,
  type CreateProjectInput,
  type UpdateProjectInput,
  type AddMemberInput,
  type CreateInviteInput,
} from '@/api/projects'

export function useProjects() {
  return useQuery({
    queryKey: ['projects'],
    queryFn: listProjects,
    select: (data) => data.projects,
  })
}

export function useOwnedProjectCount() {
  return useQuery({
    queryKey: ['projects'],
    queryFn: listProjects,
    select: (data) => data.ownedProjectCount,
  })
}

export function useMaxProjects() {
  return useQuery({
    queryKey: ['projects'],
    queryFn: listProjects,
    select: (data) => data.maxProjects,
  })
}

export function useProject(projectKey: string) {
  return useQuery({
    queryKey: ['projects', projectKey],
    queryFn: () => getProject(projectKey),
    enabled: !!projectKey,
  })
}

export function useMembers(projectKey: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'members'],
    queryFn: () => listMembers(projectKey),
    enabled: !!projectKey,
  })
}

export function useAddMember(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: AddMemberInput) => addMember(projectKey, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'members'] })
    },
  })
}

export function useUpdateMemberRole(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: string }) =>
      updateMemberRole(projectKey, userId, role),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'members'] })
    },
  })
}

export function useRemoveMember(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) => removeMember(projectKey, userId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'members'] })
    },
  })
}

export function useCreateProject() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateProjectInput) => createProject(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}

export function useUpdateProject(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: UpdateProjectInput) => updateProject(projectKey, input),
    onSuccess: (updated) => {
      qc.setQueryData(['projects', projectKey], updated)
      qc.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}

export function useDeleteProject(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => deleteProject(projectKey),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}

export function useTypeWorkflows(projectKey: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'type-workflows'],
    queryFn: () => getTypeWorkflows(projectKey),
    enabled: !!projectKey,
  })
}

export function useUpdateTypeWorkflow(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ workItemType, workflowId }: { workItemType: string; workflowId: string }) =>
      updateTypeWorkflow(projectKey, workItemType, workflowId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'type-workflows'] })
    },
  })
}

// --- Invite Hooks ---

export function useInvites(projectKey: string) {
  return useQuery({
    queryKey: ['projects', projectKey, 'invites'],
    queryFn: () => listInvites(projectKey),
    enabled: !!projectKey,
  })
}

export function useCreateInvite(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateInviteInput) => createInvite(projectKey, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'invites'] })
    },
  })
}

export function useDeleteInvite(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (inviteId: string) => deleteInvite(projectKey, inviteId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects', projectKey, 'invites'] })
    },
  })
}

export function useInviteInfo(code: string) {
  return useQuery({
    queryKey: ['invites', code],
    queryFn: () => getInviteInfo(code),
    enabled: !!code,
  })
}

export function useAcceptInvite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (code: string) => acceptInvite(code),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}
