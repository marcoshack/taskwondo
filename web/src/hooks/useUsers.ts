import { useQuery } from '@tanstack/react-query'
import { searchUsers } from '@/api/users'

export function useSearchUsers(query: string) {
  return useQuery({
    queryKey: ['users', 'search', query],
    queryFn: () => searchUsers(query),
    enabled: query.length >= 2,
    staleTime: 30_000,
  })
}
