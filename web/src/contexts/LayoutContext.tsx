import { createContext, useContext, useState, useEffect, useCallback } from 'react'
import type { ReactNode } from 'react'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useAuth } from '@/contexts/AuthContext'

export type Layout = 'centered' | 'expanded'

interface LayoutContextValue {
  layout: Layout
  setLayout: (layout: Layout) => void
  containerClass: (expandable?: boolean) => string
}

const LAYOUT_KEY = 'taskwondo_layout'
const LayoutContext = createContext<LayoutContextValue | null>(null)

function isValidLayout(v: unknown): v is Layout {
  return v === 'centered' || v === 'expanded'
}

function getStoredLayout(): Layout {
  const stored = localStorage.getItem(LAYOUT_KEY)
  if (isValidLayout(stored)) return stored
  return 'centered'
}

export function LayoutProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const [layout, setLayoutState] = useState<Layout>(getStoredLayout)

  const { data: apiLayout } = usePreference<string>(user ? 'layout' : '')
  const setPreferenceMutation = useSetPreference()

  // Sync API layout to local state on load
  useEffect(() => {
    if (isValidLayout(apiLayout)) {
      setLayoutState(apiLayout)
      localStorage.setItem(LAYOUT_KEY, apiLayout)
    }
  }, [apiLayout])

  const setLayout = useCallback(
    (newLayout: Layout) => {
      setLayoutState(newLayout)
      localStorage.setItem(LAYOUT_KEY, newLayout)
      if (user) {
        setPreferenceMutation.mutate({ key: 'layout', value: newLayout })
      }
    },
    [user, setPreferenceMutation],
  )

  const containerClass = useCallback(
    (expandable = false) => {
      if (expandable && layout === 'expanded') {
        return 'px-4 sm:px-6 lg:px-8'
      }
      return 'max-w-7xl mx-auto px-4 sm:px-6 lg:px-8'
    },
    [layout],
  )

  return (
    <LayoutContext.Provider value={{ layout, setLayout, containerClass }}>
      {children}
    </LayoutContext.Provider>
  )
}

export function useLayout() {
  const ctx = useContext(LayoutContext)
  if (!ctx) throw new Error('useLayout must be used within LayoutProvider')
  return ctx
}
