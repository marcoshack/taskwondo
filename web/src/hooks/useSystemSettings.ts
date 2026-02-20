import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getSystemSettings, getSystemSetting, setSystemSetting, deleteSystemSetting, getPublicSettings } from '@/api/systemSettings'

export function useSystemSettings() {
  return useQuery({
    queryKey: ['system-settings'],
    queryFn: getSystemSettings,
  })
}

export function useSystemSetting<T = unknown>(key: string) {
  return useQuery({
    queryKey: ['system-settings', key],
    queryFn: async () => {
      const setting = await getSystemSetting(key)
      return setting.value as T
    },
    enabled: !!key,
  })
}

export function useSetSystemSetting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ key, value }: { key: string; value: unknown }) =>
      setSystemSetting(key, value),
    onSuccess: (_data, { key }) => {
      qc.invalidateQueries({ queryKey: ['system-settings'] })
      qc.invalidateQueries({ queryKey: ['system-settings', key] })
      qc.invalidateQueries({ queryKey: ['public-settings'] })
    },
  })
}

export function useDeleteSystemSetting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (key: string) => deleteSystemSetting(key),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['system-settings'] })
      qc.invalidateQueries({ queryKey: ['public-settings'] })
    },
  })
}

export function usePublicSettings() {
  return useQuery({
    queryKey: ['public-settings'],
    queryFn: getPublicSettings,
    staleTime: 5 * 60 * 1000,
  })
}
