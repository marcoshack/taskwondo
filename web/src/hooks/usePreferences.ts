import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getPreferences, getPreference, setPreference } from '@/api/preferences'

export function usePreferences() {
  return useQuery({
    queryKey: ['user-preferences'],
    queryFn: getPreferences,
  })
}

export function usePreference<T = unknown>(key: string) {
  return useQuery({
    queryKey: ['user-preferences', key],
    queryFn: async () => {
      const pref = await getPreference(key)
      return pref.value as T
    },
    enabled: !!key,
  })
}

export function useSetPreference() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ key, value }: { key: string; value: unknown }) =>
      setPreference(key, value),
    onSuccess: (_data, { key }) => {
      qc.invalidateQueries({ queryKey: ['user-preferences'] })
      qc.invalidateQueries({ queryKey: ['user-preferences', key] })
    },
  })
}
