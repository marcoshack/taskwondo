import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getUserSettings, getUserSetting, setUserSetting, deleteUserSetting } from '@/api/userSettings'

export function useUserSettings(projectKey: string) {
  return useQuery({
    queryKey: ['user-settings', projectKey],
    queryFn: () => getUserSettings(projectKey),
    enabled: !!projectKey,
  })
}

export function useUserSetting<T = unknown>(projectKey: string, key: string) {
  return useQuery({
    queryKey: ['user-settings', projectKey, key],
    queryFn: async () => {
      const setting = await getUserSetting(projectKey, key)
      return setting.value as T
    },
    enabled: !!projectKey && !!key,
  })
}

export function useSetUserSetting(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ key, value }: { key: string; value: unknown }) =>
      setUserSetting(projectKey, key, value),
    onSuccess: (_data, { key }) => {
      qc.invalidateQueries({ queryKey: ['user-settings', projectKey] })
      qc.invalidateQueries({ queryKey: ['user-settings', projectKey, key] })
    },
  })
}

export function useDeleteUserSetting(projectKey: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (key: string) => deleteUserSetting(projectKey, key),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['user-settings', projectKey] })
    },
  })
}
