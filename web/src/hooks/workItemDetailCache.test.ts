import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  touchWorkItemDetailCache,
  resetWorkItemDetailCacheOrder,
  WORK_ITEM_DETAIL_CACHE_LIMIT,
} from './workItemDetailCache'
import type { QueryClient } from '@tanstack/react-query'

function mockQueryClient() {
  return {
    removeQueries: vi.fn(),
  } as unknown as QueryClient
}

describe('touchWorkItemDetailCache', () => {
  beforeEach(() => {
    resetWorkItemDetailCacheOrder()
  })

  it('evicts the oldest detail query once the limit is exceeded', () => {
    const qc = mockQueryClient()
    const limit = 3
    touchWorkItemDetailCache(qc, 'TASK', 1, limit)
    touchWorkItemDetailCache(qc, 'TASK', 2, limit)
    touchWorkItemDetailCache(qc, 'TASK', 3, limit)
    expect((qc.removeQueries as ReturnType<typeof vi.fn>)).not.toHaveBeenCalled()

    touchWorkItemDetailCache(qc, 'TASK', 4, limit)
    expect(qc.removeQueries).toHaveBeenCalledWith({
      queryKey: ['projects', 'TASK', 'items', 1],
      exact: true,
    })
  })

  it('refreshes LRU order when revisiting an item', () => {
    const qc = mockQueryClient()
    touchWorkItemDetailCache(qc, 'TASK', 1, 2)
    touchWorkItemDetailCache(qc, 'TASK', 2, 2)
    // Revisiting 1 makes 2 the oldest
    touchWorkItemDetailCache(qc, 'TASK', 1, 2)
    touchWorkItemDetailCache(qc, 'TASK', 3, 2)
    expect(qc.removeQueries).toHaveBeenCalledWith({
      queryKey: ['projects', 'TASK', 'items', 2],
      exact: true,
    })
  })

  it('exports a positive default cache limit', () => {
    expect(WORK_ITEM_DETAIL_CACHE_LIMIT).toBeGreaterThan(0)
  })
})
