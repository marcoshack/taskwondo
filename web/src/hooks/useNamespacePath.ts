import { useCallback } from 'react'
import { useNamespaceContext } from '@/contexts/NamespaceContext'

/** Map namespace slug to URL segment: 'default' → 'd', anything else unchanged */
export function toUrlSegment(slug: string): string {
  return slug === 'default' ? 'd' : slug
}

/** Map URL segment back to namespace slug: 'd' → 'default', anything else unchanged */
export function fromUrlSegment(segment: string): string {
  return segment === 'd' ? 'default' : segment
}

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
