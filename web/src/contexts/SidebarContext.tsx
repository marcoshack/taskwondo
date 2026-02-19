import { createContext, useContext, useState, useEffect, useCallback } from 'react'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import type { ReactNode } from 'react'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useAuth } from '@/contexts/AuthContext'

interface SidebarContextValue {
  collapsed: boolean
  toggleCollapsed: () => void
}

const SIDEBAR_COLLAPSED_KEY = 'taskwondo_sidebar_collapsed'
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
  useKeyboardShortcut({ key: '[' }, () => toggleCollapsed())

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
