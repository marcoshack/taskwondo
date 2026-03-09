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
  type NamespaceListResult,
  type CreateNamespaceInput,
  type UpdateNamespaceInput,
  type AddNamespaceMemberInput,
} from '@/api/namespaces'

export function useNamespaces(enabled = true) {
  return useQuery({
    queryKey: ['namespaces'],
    queryFn: listNamespaces,
    select: (data) => data.namespaces,
    enabled,
  })
}

export function useOwnedNamespaceCount() {
  return useQuery({
    queryKey: ['namespaces'],
    queryFn: listNamespaces,
    select: (data) => data.ownedNamespaceCount,
  })
}

export function useMaxNamespaces() {
  return useQuery({
    queryKey: ['namespaces'],
    queryFn: listNamespaces,
    select: (data) => data.maxNamespaces,
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
    onSuccess: (created) => {
      qc.setQueryData<NamespaceListResult>(['namespaces'], (old) =>
        old
          ? { ...old, namespaces: [...old.namespaces, created], ownedNamespaceCount: old.ownedNamespaceCount + 1 }
          : { namespaces: [created], ownedNamespaceCount: 1, maxNamespaces: 0 },
      )
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
