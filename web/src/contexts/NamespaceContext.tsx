import { createContext, useContext, useState, useEffect, useCallback, useRef } from 'react'
import type { ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'
import { usePublicSettings } from '@/hooks/useSystemSettings'
import { useNamespaces } from '@/hooks/useNamespaces'
import { setNamespaceSlug } from '@/api/client'
import { toUrlSegment } from '@/hooks/useNamespacePath'
import type { Namespace } from '@/api/namespaces'

const NAMESPACE_KEY = 'taskwondo_namespace'

interface NamespaceContextValue {
  namespaces: Namespace[]
  activeNamespace: Namespace | null
  setActiveNamespace: (slug: string) => void
  /** Sync the active namespace from a URL param (no navigation) */
  syncFromUrl: (slug: string) => void
  showSwitcher: boolean
  isLoading: boolean
}

const NamespaceContext = createContext<NamespaceContextValue | null>(null)

function getStoredNamespace(): string | null {
  return localStorage.getItem(NAMESPACE_KEY)
}

export function NamespaceProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const { data: publicSettings } = usePublicSettings()
  const { data: namespaces, isLoading } = useNamespaces(!!user)
  const [activeSlug, setActiveSlug] = useState<string | null>(getStoredNamespace)

  // When namespaces load, set the active one
  useEffect(() => {
    if (!namespaces || namespaces.length === 0) return

    const stored = getStoredNamespace()
    const match = stored ? namespaces.find((ns) => ns.slug === stored) : null

    if (match) {
      setActiveSlug(match.slug)
      setNamespaceSlug(match.is_default ? null : match.slug)
    } else {
      // Default to the default namespace
      const defaultNs = namespaces.find((ns) => ns.is_default) ?? namespaces[0]
      setActiveSlug(defaultNs.slug)
      localStorage.setItem(NAMESPACE_KEY, defaultNs.slug)
      setNamespaceSlug(defaultNs.is_default ? null : defaultNs.slug)
    }
  }, [namespaces])

  // Clear namespace when user logs out (not on initial mount when user hasn't loaded yet)
  const hadUser = useRef(false)
  useEffect(() => {
    if (user) {
      hadUser.current = true
    } else if (hadUser.current) {
      // User was logged in and is now logged out
      setNamespaceSlug(null)
      localStorage.removeItem(NAMESPACE_KEY)
    }
  }, [user])

  const activeNamespace = namespaces?.find((ns) => ns.slug === activeSlug) ?? null
  const namespacesEnabled = publicSettings?.namespaces_enabled === true
  const showSwitcher = namespacesEnabled

  /** Switch active namespace and navigate to its projects page */
  const setActiveNamespace = useCallback(
    (slug: string) => {
      const ns = namespaces?.find((n) => n.slug === slug)

      setActiveSlug(slug)
      localStorage.setItem(NAMESPACE_KEY, slug)
      // If namespace isn't in the list yet (just created), set slug optimistically
      const isDefault = ns ? ns.is_default : slug === 'default'
      setNamespaceSlug(isDefault ? null : slug)

      // Clear last project so the nav doesn't show a stale project from the old namespace
      localStorage.removeItem('taskwondo_last_project_key')

      // Remove namespace-scoped query cache so the new namespace starts fresh
      // (loading state instead of stale data/errors from the old namespace)
      queryClient.removeQueries({ queryKey: ['projects'] })
      navigate(`/${toUrlSegment(slug)}/projects`)
    },
    [namespaces, queryClient, navigate],
  )

  /** Sync active namespace from URL param — no navigation, but clears stale cache */
  const syncFromUrl = useCallback(
    (slug: string) => {
      if (slug === activeSlug) return
      setActiveSlug(slug)
      localStorage.setItem(NAMESPACE_KEY, slug)
      const isDefault = slug === 'default'
      setNamespaceSlug(isDefault ? null : slug)

      // Reset namespace-scoped queries so active observers see a loading
      // state instead of stale data from the previous namespace, and
      // automatically refetch against the (already-updated) API client.
      // Using resetQueries instead of removeQueries because removeQueries
      // silently destroys queries without notifying active observers,
      // leaving mounted components stuck in a permanent loading state.
      queryClient.resetQueries({ queryKey: ['projects'] })
    },
    [activeSlug, queryClient],
  )

  return (
    <NamespaceContext.Provider value={{ namespaces: namespaces ?? [], activeNamespace, setActiveNamespace, syncFromUrl, showSwitcher, isLoading }}>
      {children}
    </NamespaceContext.Provider>
  )
}

export function useNamespaceContext() {
  const ctx = useContext(NamespaceContext)
  if (!ctx) throw new Error('useNamespaceContext must be used within NamespaceProvider')
  return ctx
}
