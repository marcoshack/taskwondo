import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listNamespaces,
  getNamespace,
  createNamespace,
  updateNamespace,
  deleteNamespace,
  listNamespaceMembers,
  addNamespaceMember,
  updateNamespaceMemberRole,
  removeNamespaceMember,
  migrateProject,
  type CreateNamespaceInput,
  type UpdateNamespaceInput,
  type AddNamespaceMemberInput,
} from '@/api/namespaces'

export function useNamespaces(enabled = true) {
  return useQuery({
    queryKey: ['namespaces'],
    queryFn: listNamespaces,
    enabled,
  })
}

export function useNamespace(slug: string) {
  return useQuery({
    queryKey: ['namespaces', slug],
    queryFn: () => getNamespace(slug),
    enabled: !!slug,
  })
}

export function useCreateNamespace() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateNamespaceInput) => createNamespace(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['namespaces'] })
    },
  })
}

export function useUpdateNamespace(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: UpdateNamespaceInput) => updateNamespace(slug, input),
    onSuccess: (updated) => {
      qc.setQueryData(['namespaces', updated.slug], updated)
      qc.invalidateQueries({ queryKey: ['namespaces'] })
    },
  })
}

export function useDeleteNamespace() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (slug: string) => deleteNamespace(slug),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['namespaces'] })
    },
  })
}

export function useNamespaceMembers(slug: string) {
  return useQuery({
    queryKey: ['namespaces', slug, 'members'],
    queryFn: () => listNamespaceMembers(slug),
    enabled: !!slug,
  })
}

export function useAddNamespaceMember(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: AddNamespaceMemberInput) => addNamespaceMember(slug, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['namespaces', slug, 'members'] })
    },
  })
}

export function useUpdateNamespaceMemberRole(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: string }) =>
      updateNamespaceMemberRole(slug, userId, role),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['namespaces', slug, 'members'] })
    },
  })
}

export function useRemoveNamespaceMember(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) => removeNamespaceMember(slug, userId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['namespaces', slug, 'members'] })
    },
  })
}

export function useMigrateProject() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ fromSlug, projectKey, targetSlug }: { fromSlug: string; projectKey: string; targetSlug: string }) =>
      migrateProject(fromSlug, projectKey, targetSlug),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects'] })
      qc.invalidateQueries({ queryKey: ['namespaces'] })
    },
  })
}
