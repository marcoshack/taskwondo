import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listProjects, getProject, createProject, listMembers, type CreateProjectInput } from '@/api/projects'

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

export function useCreateProject() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateProjectInput) => createProject(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}
