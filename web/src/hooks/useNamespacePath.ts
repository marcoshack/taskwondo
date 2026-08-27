import { useCallback } from 'react'
import { useNamespaceContext } from '@/contexts/NamespaceContext'
import { toUrlSegment, fromUrlSegment } from '@/utils/namespaceUrl'

// Re-exported for existing callers; the implementations live in a client-free
// module so pure, DOM-less units (e.g. the command palette catalog) can build
// namespace URLs without dragging in `@/api/client`.
export { toUrlSegment, fromUrlSegment }

/** Hook returning a path-prefix function `p(path)` for namespace-scoped URLs */
export function useNamespacePath() {
  const { activeNamespace } = useNamespaceContext()
  const segment = toUrlSegment(activeNamespace?.slug ?? 'default')

  /** Prefix a path with the namespace URL segment, e.g. p('/projects') → '/d/projects' */
  const p = useCallback(
    (path: string): string => `/${segment}${path.startsWith('/') ? path : `/${path}`}`,
    [segment],
  )

  return { p, segment }
}
