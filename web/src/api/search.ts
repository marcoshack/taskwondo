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

export interface SearchResponse {
  results: SearchResult[]
  query: string
  total: number
}

interface DataResponse<T> {
  data: T
}

// --- API Functions ---

export async function semanticSearch(
  query: string,
  entityTypes?: string[],
  limit?: number,
): Promise<SearchResponse> {
  const params = new URLSearchParams()
  params.set('q', query)
  if (entityTypes?.length) params.set('entity_type', entityTypes.join(','))
  if (limit) params.set('limit', String(limit))
  const res = await api.get<DataResponse<SearchResponse>>(`/search?${params}`)
  return res.data.data
}
