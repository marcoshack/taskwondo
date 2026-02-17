import { useQuery } from '@tanstack/react-query'
import { listProjects, getProject } from '@/api/projects'

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
