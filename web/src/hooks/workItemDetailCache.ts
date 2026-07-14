import type { QueryClient } from '@tanstack/react-query'

/** Max fully-fetched detail entries retained for instant pane re-open. */
export const WORK_ITEM_DETAIL_CACHE_LIMIT = 10

const orderByProject = new Map<string, number[]>()

/**
 * Track a successfully fetched work-item detail and drop the oldest cached
 * entries for that project once the limit is exceeded.
 */
export function touchWorkItemDetailCache(
  queryClient: QueryClient,
  projectKey: string,
  itemNumber: number,
  limit: number = WORK_ITEM_DETAIL_CACHE_LIMIT,
): void {
  if (!projectKey || itemNumber <= 0 || limit <= 0) return

  const prev = orderByProject.get(projectKey) ?? []
  const next = [itemNumber, ...prev.filter((n) => n !== itemNumber)]
  const evicted = next.slice(limit)
  orderByProject.set(projectKey, next.slice(0, limit))

  for (const n of evicted) {
    queryClient.removeQueries({
      queryKey: ['projects', projectKey, 'items', n],
      exact: true,
    })
  }
}

/** Test helper — clears in-memory LRU order. */
export function resetWorkItemDetailCacheOrder(): void {
  orderByProject.clear()
}
