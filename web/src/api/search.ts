import { api } from './client'

// --- Types ---

export interface SearchResult {
  entity_type: string
  entity_id: string
  project_id: string | null
  score: number
  snippet: string
  project_key?: string
  item_number?: number
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
  options?: { entityTypes?: string[]; limit?: number; signal?: AbortSignal },
): Promise<UnifiedSearchResponse> {
  const params = new URLSearchParams()
  params.set('q', query)
  if (options?.entityTypes?.length) params.set('entity_type', options.entityTypes.join(','))
  if (options?.limit) params.set('limit', String(options.limit))
  const res = await api.get<DataResponse<UnifiedSearchResponse>>(`/search?${params}`, {
    signal: options?.signal,
  })
  return res.data.data
}
