import { api } from './client'

export interface StatsTimelinePoint {
  captured_at: string
  todo_count: number
  in_progress_count: number
  done_count: number
  cancelled_count: number
}

export type StatsRange = '24h' | '3d' | '7d'

export async function getStatsTimeline(
  projectKey: string,
  range_: StatsRange,
): Promise<StatsTimelinePoint[]> {
  const res = await api.get<{ data: StatsTimelinePoint[] }>(
    `/projects/${projectKey}/stats/timeline`,
    { params: { range: range_ } },
  )
  return res.data.data
}
