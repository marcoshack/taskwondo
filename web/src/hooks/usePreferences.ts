import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getPreferences, setPreference } from '@/api/preferences'

export function usePreferences() {
  return useQuery({
    queryKey: ['user-preferences'],
    queryFn: getPreferences,
  })
}

// Reads a single preference by key from the shared bulk query cache.
// All callers subscribe to the same ['user-preferences'] query, so React Query
// dedupes them into one GET /user/preferences request regardless of how many
// components mount. A key that isn't set yet resolves to `null` (same as the
// previous per-key 404 behavior).
export function usePreference<T = unknown>(key: string) {
  return useQuery({
    queryKey: ['user-preferences'],
    queryFn: getPreferences,
    enabled: !!key,
    select: (prefs): T | null => {
      const found = prefs.find((p) => p.key === key)
      return found ? (found.value as T) : null
    },
  })
}

export function useSetPreference() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ key, value }: { key: string; value: unknown }) =>
      setPreference(key, value),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['user-preferences'] })
    },
  })
}
