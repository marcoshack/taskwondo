import { createContext, useContext, useState, useEffect, useCallback, useMemo } from 'react'
import { useLocation } from 'react-router-dom'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import type { ReactNode } from 'react'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useAuth } from '@/contexts/AuthContext'

export type SidebarType = 'app' | 'settings'

interface SidebarState {
  collapsed: boolean
  toggleCollapsed: () => void
}

interface SidebarContextValue {
  getSidebar: (type: SidebarType) => SidebarState
  mobileOpen: boolean
  toggleMobileOpen: () => void
  closeMobile: () => void
}

const STORAGE_KEYS: Record<SidebarType, string> = {
  app: 'taskwondo_sidebar_app_collapsed',
  settings: 'taskwondo_sidebar_settings_collapsed',
}

const PREF_KEYS: Record<SidebarType, string> = {
  app: 'sidebarAppCollapsed',
  settings: 'sidebarSettingsCollapsed',
}

const OLD_STORAGE_KEY = 'taskwondo_sidebar_collapsed'
const OLD_PREF_KEY = 'sidebarCollapsed'

const SidebarContext = createContext<SidebarContextValue | null>(null)

function migrateOldStorage(): void {
  // Migrate legacy single-key storage
  const old = localStorage.getItem(OLD_STORAGE_KEY)
  if (old !== null) {
    for (const key of [STORAGE_KEYS.app, STORAGE_KEYS.settings]) {
      if (localStorage.getItem(key) === null) {
        localStorage.setItem(key, old)
      }
    }
    localStorage.removeItem(OLD_STORAGE_KEY)
  }

  // Migrate per-type keys (project/user) to unified app key
  const oldProject = localStorage.getItem('taskwondo_sidebar_project_collapsed')
  const oldUser = localStorage.getItem('taskwondo_sidebar_user_collapsed')
  if (oldProject !== null || oldUser !== null) {
    if (localStorage.getItem(STORAGE_KEYS.app) === null) {
      // Prefer project sidebar's state since it was more commonly used
      localStorage.setItem(STORAGE_KEYS.app, oldProject ?? oldUser ?? 'false')
    }
    // Migrate settings key if it used the old name
    const oldSettings = localStorage.getItem('taskwondo_sidebar_settings_collapsed')
    if (oldSettings !== null && localStorage.getItem(STORAGE_KEYS.settings) === null) {
      localStorage.setItem(STORAGE_KEYS.settings, oldSettings)
    }
    localStorage.removeItem('taskwondo_sidebar_project_collapsed')
    localStorage.removeItem('taskwondo_sidebar_user_collapsed')
  }
}

function getStoredCollapsed(type: SidebarType): boolean {
  return localStorage.getItem(STORAGE_KEYS[type]) === 'true'
}

export function SidebarProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const location = useLocation()

  // Run migration once on mount
  useEffect(() => {
    migrateOldStorage()
  }, [])

  const [appCollapsed, setAppCollapsed] = useState<boolean>(() => getStoredCollapsed('app'))
  const [settingsCollapsed, setSettingsCollapsed] = useState<boolean>(() => getStoredCollapsed('settings'))

  // API preference sync
  const { data: apiAppCollapsed } = usePreference<string>(user ? PREF_KEYS.app : '')
  const { data: apiSettingsCollapsed } = usePreference<string>(user ? PREF_KEYS.settings : '')
  const { data: apiOldCollapsed } = usePreference<string>(user ? OLD_PREF_KEY : '')
  // Read old per-type API prefs for migration
  const { data: apiOldProjectCollapsed } = usePreference<string>(user ? 'sidebarProjectCollapsed' : '')
  const { data: apiOldUserCollapsed } = usePreference<string>(user ? 'sidebarUserCollapsed' : '')
  const setPreferenceMutation = useSetPreference()

  // Migrate old API preference (single key)
  useEffect(() => {
    if (apiOldCollapsed === 'true' || apiOldCollapsed === 'false') {
      if (apiAppCollapsed === undefined || apiAppCollapsed === null) {
        setPreferenceMutation.mutate({ key: PREF_KEYS.app, value: apiOldCollapsed })
      }
      if (apiSettingsCollapsed === undefined || apiSettingsCollapsed === null) {
        setPreferenceMutation.mutate({ key: PREF_KEYS.settings, value: apiOldCollapsed })
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [apiOldCollapsed])

  // Migrate old per-type API prefs (project/user → app)
  useEffect(() => {
    const oldVal = apiOldProjectCollapsed ?? apiOldUserCollapsed
    if ((oldVal === 'true' || oldVal === 'false') && (apiAppCollapsed === undefined || apiAppCollapsed === null)) {
      setPreferenceMutation.mutate({ key: PREF_KEYS.app, value: oldVal })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [apiOldProjectCollapsed, apiOldUserCollapsed])

  // Sync API preferences to local state — only when localStorage is empty (cross-device sync).
  // localStorage is the source of truth to avoid race conditions where async API
  // values (including stale migration data) override the user's current preference.
  useEffect(() => {
    if (apiAppCollapsed === 'true' || apiAppCollapsed === 'false') {
      if (localStorage.getItem(STORAGE_KEYS.app) === null) {
        const val = apiAppCollapsed === 'true'
        setAppCollapsed(val)
        localStorage.setItem(STORAGE_KEYS.app, String(val))
      }
    }
  }, [apiAppCollapsed])

  useEffect(() => {
    if (apiSettingsCollapsed === 'true' || apiSettingsCollapsed === 'false') {
      if (localStorage.getItem(STORAGE_KEYS.settings) === null) {
        const val = apiSettingsCollapsed === 'true'
        setSettingsCollapsed(val)
        localStorage.setItem(STORAGE_KEYS.settings, String(val))
      }
    }
  }, [apiSettingsCollapsed])

  const setCollapsed = useCallback(
    (type: SidebarType, newCollapsed: boolean) => {
      const setter = type === 'app' ? setAppCollapsed : setSettingsCollapsed
      setter(newCollapsed)
      localStorage.setItem(STORAGE_KEYS[type], String(newCollapsed))
      if (user) {
        setPreferenceMutation.mutate({ key: PREF_KEYS[type], value: String(newCollapsed) })
      }
    },
    [user, setPreferenceMutation],
  )

  const getSidebar = useCallback(
    (type: SidebarType): SidebarState => {
      const collapsed = type === 'app' ? appCollapsed : settingsCollapsed
      return {
        collapsed,
        toggleCollapsed: () => setCollapsed(type, !collapsed),
      }
    },
    [appCollapsed, settingsCollapsed, setCollapsed],
  )

  const [mobileOpen, setMobileOpen] = useState(false)
  const toggleMobileOpen = useCallback(() => setMobileOpen((v) => !v), [])
  const closeMobile = useCallback(() => setMobileOpen(false), [])

  // Keyboard shortcut: [ to toggle the active sidebar based on current route
  const activeSidebarType: SidebarType = useMemo(() => {
    if (location.pathname.startsWith('/preferences') || location.pathname.startsWith('/admin')) {
      return 'settings'
    }
    return 'app'
  }, [location.pathname])

  useKeyboardShortcut({ key: '[' }, () => {
    const current = activeSidebarType === 'app' ? appCollapsed : settingsCollapsed
    setCollapsed(activeSidebarType, !current)
  })

  return (
    <SidebarContext.Provider value={{ getSidebar, mobileOpen, toggleMobileOpen, closeMobile }}>
      {children}
    </SidebarContext.Provider>
  )
}

export function useSidebar(type: SidebarType = 'app') {
  const ctx = useContext(SidebarContext)
  if (!ctx) throw new Error('useSidebar must be used within SidebarProvider')
  const { collapsed, toggleCollapsed } = ctx.getSidebar(type)
  return { collapsed, toggleCollapsed, mobileOpen: ctx.mobileOpen, toggleMobileOpen: ctx.toggleMobileOpen, closeMobile: ctx.closeMobile }
}
