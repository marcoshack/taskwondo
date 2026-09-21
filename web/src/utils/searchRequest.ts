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

/**
 * Scripts whose characters carry meaning on their own (Chinese, Japanese,
 * Korean). Postgres tokenizes CJK runs as a single lexeme, so the backend
 * adds an ILIKE substring fallback for these queries.
 */
const CJK_SCRIPT = /[\p{Script=Han}\p{Script=Hiragana}\p{Script=Katakana}\p{Script=Hangul}]/u

/**
 * Whether a query is worth sending to the entity search endpoint.
 *
 * The historical floor is two characters, which suppresses noisy single-letter
 * latin queries — but a single CJK character is already a meaningful word, so
 * the floor drops to one character for scripts that contain one.
 */
export function meetsSearchFloor(query: string): boolean {
  const trimmed = query.trim()
  if (trimmed.length === 0) return false
  return trimmed.length >= 2 || CJK_SCRIPT.test(trimmed)
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
