/**
 * Query-string assembly for `GET /api/v1/search`.
 *
 * Kept out of `@/api/search` (and therefore out of `@/api/client`, which reads
 * `localStorage` at import time) so the exact parameters the command palette
 * sends can be unit-tested without a DOM.
 */

export interface UnifiedSearchRequest {
  query: string
  entityTypes?: string[]
  limit?: number
  /**
   * Project **key** (e.g. `TF`), not a UUID. Scopes work_item, milestone,
   * queue, team, comment and attachment hits to that project; `project` hits
   * stay global so the palette is still how you jump to another project.
   * Omitted when absent or blank, which is the unscoped, global search.
   */
  project?: string | null
}

/** Build the query string for a unified search request. */
export function buildSearchParams(req: UnifiedSearchRequest): URLSearchParams {
  const params = new URLSearchParams()
  params.set('q', req.query)
  if (req.entityTypes?.length) params.set('entity_type', req.entityTypes.join(','))
  if (req.limit) params.set('limit', String(req.limit))
  const project = req.project?.trim()
  if (project) params.set('project', project)
  return params
}
