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
  type CreateProjectInput,
  type UpdateProjectInput,
  type AddMemberInput,
} from '@/api/projects'

export function useProjects() {
  return useQuery({
    queryKey: ['projects'],
    queryFn: listProjects,
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
    onSuccess: () => {
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
