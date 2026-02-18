import { createContext, useContext, useState, useEffect, useCallback } from 'react'
import type { ReactNode } from 'react'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useAuth } from '@/contexts/AuthContext'

export type Theme = 'light' | 'dark'

interface ThemeContextValue {
  theme: Theme
  setTheme: (theme: Theme) => void
}

const STORAGE_KEY = 'trackforge_theme'
const ThemeContext = createContext<ThemeContextValue | null>(null)

function applyTheme(theme: Theme) {
  if (theme === 'dark') {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
}

function getStoredTheme(): Theme {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'dark' || stored === 'light') return stored
  return 'light'
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const [theme, setThemeState] = useState<Theme>(getStoredTheme)

  // Fetch theme preference from API when logged in
  const { data: apiTheme } = usePreference<string>(user ? 'theme' : '')
  const setPreferenceMutation = useSetPreference()

  // Sync API theme to local state on load
  useEffect(() => {
    if (apiTheme === 'dark' || apiTheme === 'light') {
      setThemeState(apiTheme)
      localStorage.setItem(STORAGE_KEY, apiTheme)
    }
  }, [apiTheme])

  // Apply theme class whenever it changes
  useEffect(() => {
    applyTheme(theme)
  }, [theme])

  const setTheme = useCallback(
    (newTheme: Theme) => {
      setThemeState(newTheme)
      localStorage.setItem(STORAGE_KEY, newTheme)
      applyTheme(newTheme)
      if (user) {
        setPreferenceMutation.mutate({ key: 'theme', value: newTheme })
      }
    },
    [user, setPreferenceMutation],
  )

  return (
    <ThemeContext.Provider value={{ theme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider')
  return ctx
}
