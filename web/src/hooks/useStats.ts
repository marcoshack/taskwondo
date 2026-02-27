import { useQuery } from '@tanstack/react-query'
import { getStatsTimeline, type StatsRange } from '@/api/stats'

export function useStatsTimeline(projectKey: string, range_: StatsRange) {
  return useQuery({
    queryKey: ['projects', projectKey, 'stats', 'timeline', range_],
    queryFn: () => getStatsTimeline(projectKey, range_),
    enabled: !!projectKey,
    refetchInterval: 5 * 60 * 1000, // refresh every 5 minutes
  })
}
