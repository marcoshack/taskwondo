import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listUsers, updateUser, listUserProjects, addUserToProject, removeUserFromProject } from '@/api/admin'

export function useAdminUsers() {
  return useQuery({
    queryKey: ['admin', 'users'],
    queryFn: listUsers,
  })
}

export function useUpdateUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, input }: { userId: string; input: { global_role?: string; is_active?: boolean } }) =>
      updateUser(userId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'users'] })
    },
  })
}

export function useUserProjects(userId: string) {
  return useQuery({
    queryKey: ['admin', 'users', userId, 'projects'],
    queryFn: () => listUserProjects(userId),
    enabled: !!userId,
  })
}

export function useAddUserToProject(userId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: { project_id: string; role: string }) => addUserToProject(userId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'users', userId, 'projects'] })
    },
  })
}

export function useRemoveUserFromProject(userId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (projectId: string) => removeUserFromProject(userId, projectId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'users', userId, 'projects'] })
    },
  })
}
