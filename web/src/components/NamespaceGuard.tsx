import { useEffect } from 'react'
import { Outlet, useParams } from 'react-router-dom'
import { useNamespaceContext } from '@/contexts/NamespaceContext'
import { fromUrlSegment } from '@/hooks/useNamespacePath'
import { setNamespaceSlug } from '@/api/client'

/**
 * Route-level component that syncs the :namespace URL param to NamespaceContext.
 * Sets the API client namespace synchronously so child components make requests
 * against the correct namespace even on first render.
 */
export function NamespaceGuard() {
  const { namespace } = useParams<{ namespace: string }>()
  const { syncFromUrl } = useNamespaceContext()

  const slug = fromUrlSegment(namespace ?? 'd')

  // Synchronously update the API client before children render.
  // This ensures API calls in child components use the correct namespace
  // even when navigating cross-namespace (e.g. from global inbox).
  setNamespaceSlug(slug === 'default' ? null : slug)

  // Async sync for React state (context, localStorage)
  useEffect(() => {
    syncFromUrl(slug)
  }, [slug, syncFromUrl])

  return <Outlet />
}
