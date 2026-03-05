import { useQuery } from '@tanstack/react-query'
import { semanticSearch } from '@/api/search'
import { listWorkItems } from '@/api/workitems'
import { usePublicSettings } from '@/hooks/useSystemSettings'
import { useProjects } from '@/hooks/useProjects'
import type { SearchResult } from '@/api/search'

export type SearchMode = 'semantic' | 'fts'

interface UseSearchOptions {
  query: string
  limit?: number
}

interface UseSearchReturn {
  results: SearchResult[]
  total: number
  isLoading: boolean
  isError: boolean
  mode: SearchMode
}

export function useSearch({ query, limit = 20 }: UseSearchOptions): UseSearchReturn {
  const { data: publicSettings } = usePublicSettings()
  const { data: projectsData } = useProjects()
  const semanticEnabled = publicSettings?.feature_semantic_search === true

  // Semantic search query
  const semantic = useQuery({
    queryKey: ['search', 'semantic', query, limit],
    queryFn: () => semanticSearch(query, undefined, limit),
    enabled: semanticEnabled && query.length >= 2,
    retry: false,
  })

  // FTS fallback: search work items across all projects the user has access to
  const fts = useQuery({
    queryKey: ['search', 'fts', query, limit],
    queryFn: async () => {
      const projects = projectsData ?? []
      if (projects.length === 0) return []

      // Search across all accessible projects in parallel
      const results = await Promise.all(
        projects.map((p) =>
          listWorkItems(p.key, { q: query, limit }).then((res) =>
            res.data.map(
              (item): SearchResult => ({
                entity_type: 'work_item',
                entity_id: item.id,
                project_id: null,
                score: 0,
                snippet: `${item.display_id}: ${item.title}`,
                path: `/projects/${p.key}/items/${item.item_number}`,
              }),
            ),
          ),
        ),
      )
      return results.flat().slice(0, limit)
    },
    enabled:
      (!semanticEnabled || (semantic.isError && semanticEnabled)) &&
      query.length >= 2,
    retry: false,
  })

  // Determine active mode and results
  const useFts = !semanticEnabled || semantic.isError
  const activeQuery = useFts ? fts : semantic
  const mode: SearchMode = useFts ? 'fts' : 'semantic'

  const results: SearchResult[] = useFts
    ? (fts.data as SearchResult[] | undefined) ?? []
    : semantic.data?.results ?? []

  return {
    results,
    total: useFts ? results.length : (semantic.data?.total ?? 0),
    isLoading: activeQuery.isLoading,
    isError: activeQuery.isError,
    mode,
  }
}
