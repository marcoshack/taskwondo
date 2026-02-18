import { createContext, useContext, useState, useEffect, useCallback } from 'react'
import type { ReactNode } from 'react'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useAuth } from '@/contexts/AuthContext'

interface SidebarContextValue {
  collapsed: boolean
  toggleCollapsed: () => void
}

const SIDEBAR_COLLAPSED_KEY = 'trackforge_sidebar_collapsed'
const SidebarContext = createContext<SidebarContextValue | null>(null)

function getStoredCollapsed(): boolean {
  return localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === 'true'
}

export function SidebarProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const [collapsed, setCollapsedState] = useState<boolean>(getStoredCollapsed)

  const { data: apiCollapsed } = usePreference<string>(user ? 'sidebarCollapsed' : '')
  const setPreferenceMutation = useSetPreference()

  // Sync API preference to local state on load
  useEffect(() => {
    if (apiCollapsed === 'true' || apiCollapsed === 'false') {
      const val = apiCollapsed === 'true'
      setCollapsedState(val)
      localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(val))
    }
  }, [apiCollapsed])

  const setCollapsed = useCallback(
    (newCollapsed: boolean) => {
      setCollapsedState(newCollapsed)
      localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(newCollapsed))
      if (user) {
        setPreferenceMutation.mutate({ key: 'sidebarCollapsed', value: String(newCollapsed) })
      }
    },
    [user, setPreferenceMutation],
  )

  const toggleCollapsed = useCallback(() => {
    setCollapsed(!collapsed)
  }, [collapsed, setCollapsed])

  // Keyboard shortcut: [ to toggle sidebar
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === '[' && !e.ctrlKey && !e.metaKey && !e.altKey) {
        const tag = (e.target as HTMLElement).tagName
        if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return
        if ((e.target as HTMLElement).isContentEditable) return
        e.preventDefault()
        toggleCollapsed()
      }
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [toggleCollapsed])

  return (
    <SidebarContext.Provider value={{ collapsed, toggleCollapsed }}>
      {children}
    </SidebarContext.Provider>
  )
}

export function useSidebar() {
  const ctx = useContext(SidebarContext)
  if (!ctx) throw new Error('useSidebar must be used within SidebarProvider')
  return ctx
}
