import { createContext, useContext, useState, useEffect, useCallback } from 'react'
import type { ReactNode } from 'react'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useAuth } from '@/contexts/AuthContext'

export type Theme = 'light' | 'dark' | 'system'
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

const THEME_KEY = 'taskwondo_theme'
const FONT_SIZE_KEY = 'taskwondo_font_size'
const ThemeContext = createContext<ThemeContextValue | null>(null)

const darkMediaQuery = window.matchMedia('(prefers-color-scheme: dark)')

function resolveTheme(theme: Theme): 'light' | 'dark' {
  if (theme === 'system') return darkMediaQuery.matches ? 'dark' : 'light'
  return theme
}

function applyTheme(theme: Theme) {
  const resolved = resolveTheme(theme)
  if (resolved === 'dark') {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
}

function applyFontSize(size: FontSize) {
  document.documentElement.style.fontSize = fontSizePx[size]
}

function isValidTheme(v: unknown): v is Theme {
  return v === 'light' || v === 'dark' || v === 'system'
}

function getStoredTheme(): Theme {
  const stored = localStorage.getItem(THEME_KEY)
  if (isValidTheme(stored)) return stored
  return 'system'
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
    if (isValidTheme(apiTheme)) {
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

  // Listen for OS color scheme changes when theme is 'system'
  useEffect(() => {
    if (theme !== 'system') return
    const handler = () => applyTheme('system')
    darkMediaQuery.addEventListener('change', handler)
    return () => darkMediaQuery.removeEventListener('change', handler)
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
