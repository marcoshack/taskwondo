import { createContext, useContext, useState, useEffect, useCallback } from 'react'
import type { ReactNode } from 'react'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useAuth } from '@/contexts/AuthContext'

export type Theme = 'light' | 'dark'
export type FontSize = 'small' | 'normal' | 'large'

const fontSizePx: Record<FontSize, string> = {
  small: '16px',
  normal: '17.6px',
  large: '19.4px',
}

interface ThemeContextValue {
  theme: Theme
  setTheme: (theme: Theme) => void
  fontSize: FontSize
  setFontSize: (size: FontSize) => void
}

const THEME_KEY = 'trackforge_theme'
const FONT_SIZE_KEY = 'trackforge_font_size'
const ThemeContext = createContext<ThemeContextValue | null>(null)

function applyTheme(theme: Theme) {
  if (theme === 'dark') {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
}

function applyFontSize(size: FontSize) {
  document.documentElement.style.fontSize = fontSizePx[size]
}

function getStoredTheme(): Theme {
  const stored = localStorage.getItem(THEME_KEY)
  if (stored === 'dark' || stored === 'light') return stored
  return 'light'
}

function isValidFontSize(v: unknown): v is FontSize {
  return v === 'small' || v === 'normal' || v === 'large'
}

function getStoredFontSize(): FontSize {
  const stored = localStorage.getItem(FONT_SIZE_KEY)
  if (isValidFontSize(stored)) return stored
  return 'normal'
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const [theme, setThemeState] = useState<Theme>(getStoredTheme)
  const [fontSize, setFontSizeState] = useState<FontSize>(getStoredFontSize)

  // Fetch preferences from API when logged in
  const { data: apiTheme } = usePreference<string>(user ? 'theme' : '')
  const { data: apiFontSize } = usePreference<string>(user ? 'fontSize' : '')
  const setPreferenceMutation = useSetPreference()

  // Sync API theme to local state on load
  useEffect(() => {
    if (apiTheme === 'dark' || apiTheme === 'light') {
      setThemeState(apiTheme)
      localStorage.setItem(THEME_KEY, apiTheme)
    }
  }, [apiTheme])

  // Sync API font size to local state on load
  useEffect(() => {
    if (isValidFontSize(apiFontSize)) {
      setFontSizeState(apiFontSize)
      localStorage.setItem(FONT_SIZE_KEY, apiFontSize)
    }
  }, [apiFontSize])

  // Apply theme class whenever it changes
  useEffect(() => {
    applyTheme(theme)
  }, [theme])

  // Apply font size whenever it changes
  useEffect(() => {
    applyFontSize(fontSize)
  }, [fontSize])

  const setTheme = useCallback(
    (newTheme: Theme) => {
      setThemeState(newTheme)
      localStorage.setItem(THEME_KEY, newTheme)
      applyTheme(newTheme)
      if (user) {
        setPreferenceMutation.mutate({ key: 'theme', value: newTheme })
      }
    },
    [user, setPreferenceMutation],
  )

  const setFontSize = useCallback(
    (newSize: FontSize) => {
      setFontSizeState(newSize)
      localStorage.setItem(FONT_SIZE_KEY, newSize)
      applyFontSize(newSize)
      if (user) {
        setPreferenceMutation.mutate({ key: 'fontSize', value: newSize })
      }
    },
    [user, setPreferenceMutation],
  )

  return (
    <ThemeContext.Provider value={{ theme, setTheme, fontSize, setFontSize }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider')
  return ctx
}
