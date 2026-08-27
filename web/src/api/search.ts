import { api } from './client'
import { buildSearchParams } from '@/utils/searchRequest'

// --- Types ---

export interface SearchResult {
  entity_type: string
  entity_id: string
  project_id: string | null
  score: number
  snippet: string
  project_key?: string
  item_number?: number
  namespace_slug?: string
  status?: string
  status_category?: string
}

export interface FTSSection {
  results: SearchResult[]
  total: number
}

export interface SemanticSection {
  results?: SearchResult[]
  total: number
  available: boolean
  status: string
}

export interface UnifiedSearchResponse {
  query: string
  fts: FTSSection
  semantic: SemanticSection
}

interface DataResponse<T> {
  data: T
}

// --- API Functions ---

/**
 * Unified search: calls GET /api/v1/search which runs FTS and semantic
 * searches concurrently on the backend and returns both result sets.
 */
export async function unifiedSearch(
  query: string,
  options?: {
    entityTypes?: string[]
    limit?: number
    /**
     * Project key (e.g. `TF`) to scope the search to. Everything but `project`
     * hits comes back limited to that project; omit it for a global search.
     * An unknown or inaccessible key is rejected with 403 by the backend — it
     * never silently falls back to the unscoped search.
     */
    project?: string | null
    signal?: AbortSignal
  },
): Promise<UnifiedSearchResponse> {
  const params = buildSearchParams({
    query,
    entityTypes: options?.entityTypes,
    limit: options?.limit,
    project: options?.project,
  })
  const res = await api.get<DataResponse<UnifiedSearchResponse>>(`/search?${params}`, {
    signal: options?.signal,
  })
  return res.data.data
}
