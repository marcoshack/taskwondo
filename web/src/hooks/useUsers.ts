import { useQuery } from '@tanstack/react-query'
import { searchUsers } from '@/api/users'
import { meetsSearchFloor } from '@/utils/searchRequest'

export function useSearchUsers(query: string) {
  return useQuery({
    queryKey: ['users', 'search', query],
    queryFn: () => searchUsers(query),
    enabled: meetsSearchFloor(query),
    staleTime: 30_000,
  })
}
