import { createContext, useContext, useState, useEffect, useCallback, useMemo } from 'react'
import { useLocation } from 'react-router-dom'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import type { ReactNode } from 'react'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useAuth } from '@/contexts/AuthContext'

export type SidebarType = 'project' | 'settings' | 'user'

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
  project: 'taskwondo_sidebar_project_collapsed',
  settings: 'taskwondo_sidebar_settings_collapsed',
  user: 'taskwondo_sidebar_user_collapsed',
}

const PREF_KEYS: Record<SidebarType, string> = {
  project: 'sidebarProjectCollapsed',
  settings: 'sidebarSettingsCollapsed',
  user: 'sidebarUserCollapsed',
}

const OLD_STORAGE_KEY = 'taskwondo_sidebar_collapsed'
const OLD_PREF_KEY = 'sidebarCollapsed'

const SidebarContext = createContext<SidebarContextValue | null>(null)

function migrateOldStorage(): void {
  const old = localStorage.getItem(OLD_STORAGE_KEY)
  if (old !== null) {
    // Migrate old value to project and settings keys (only if new keys don't exist yet)
    for (const key of [STORAGE_KEYS.project, STORAGE_KEYS.settings]) {
      if (localStorage.getItem(key) === null) {
        localStorage.setItem(key, old)
      }
    }
    localStorage.removeItem(OLD_STORAGE_KEY)
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

  const [projectCollapsed, setProjectCollapsed] = useState<boolean>(() => getStoredCollapsed('project'))
  const [settingsCollapsed, setSettingsCollapsed] = useState<boolean>(() => getStoredCollapsed('settings'))
  const [userCollapsed, setUserCollapsed] = useState<boolean>(() => getStoredCollapsed('user'))

  // API preference sync
  const { data: apiProjectCollapsed } = usePreference<string>(user ? PREF_KEYS.project : '')
  const { data: apiSettingsCollapsed } = usePreference<string>(user ? PREF_KEYS.settings : '')
  const { data: apiUserCollapsed } = usePreference<string>(user ? PREF_KEYS.user : '')
  const { data: apiOldCollapsed } = usePreference<string>(user ? OLD_PREF_KEY : '')
  const setPreferenceMutation = useSetPreference()

  // Migrate old API preference
  useEffect(() => {
    if (apiOldCollapsed === 'true' || apiOldCollapsed === 'false') {
      // Migrate old pref to new keys if they don't have values yet
      if (apiProjectCollapsed === undefined || apiProjectCollapsed === null) {
        setPreferenceMutation.mutate({ key: PREF_KEYS.project, value: apiOldCollapsed })
      }
      if (apiSettingsCollapsed === undefined || apiSettingsCollapsed === null) {
        setPreferenceMutation.mutate({ key: PREF_KEYS.settings, value: apiOldCollapsed })
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [apiOldCollapsed])

  // Sync API preferences to local state
  useEffect(() => {
    if (apiProjectCollapsed === 'true' || apiProjectCollapsed === 'false') {
      const val = apiProjectCollapsed === 'true'
      setProjectCollapsed(val)
      localStorage.setItem(STORAGE_KEYS.project, String(val))
    }
  }, [apiProjectCollapsed])

  useEffect(() => {
    if (apiSettingsCollapsed === 'true' || apiSettingsCollapsed === 'false') {
      const val = apiSettingsCollapsed === 'true'
      setSettingsCollapsed(val)
      localStorage.setItem(STORAGE_KEYS.settings, String(val))
    }
  }, [apiSettingsCollapsed])

  useEffect(() => {
    if (apiUserCollapsed === 'true' || apiUserCollapsed === 'false') {
      const val = apiUserCollapsed === 'true'
      setUserCollapsed(val)
      localStorage.setItem(STORAGE_KEYS.user, String(val))
    }
  }, [apiUserCollapsed])

  const setCollapsed = useCallback(
    (type: SidebarType, newCollapsed: boolean) => {
      const setter = type === 'project' ? setProjectCollapsed : type === 'settings' ? setSettingsCollapsed : setUserCollapsed
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
      const collapsed = type === 'project' ? projectCollapsed : type === 'settings' ? settingsCollapsed : userCollapsed
      return {
        collapsed,
        toggleCollapsed: () => setCollapsed(type, !collapsed),
      }
    },
    [projectCollapsed, settingsCollapsed, userCollapsed, setCollapsed],
  )

  const [mobileOpen, setMobileOpen] = useState(false)
  const toggleMobileOpen = useCallback(() => setMobileOpen((v) => !v), [])
  const closeMobile = useCallback(() => setMobileOpen(false), [])

  // Keyboard shortcut: [ to toggle the active sidebar based on current route
  const activeSidebarType: SidebarType = useMemo(() => {
    if (location.pathname.startsWith('/preferences') || location.pathname.startsWith('/admin')) {
      return 'settings'
    }
    if (location.pathname.startsWith('/user')) {
      return 'user'
    }
    return 'project'
  }, [location.pathname])

  useKeyboardShortcut({ key: '[' }, () => {
    const current = activeSidebarType === 'project' ? projectCollapsed : activeSidebarType === 'settings' ? settingsCollapsed : userCollapsed
    setCollapsed(activeSidebarType, !current)
  })

  return (
    <SidebarContext.Provider value={{ getSidebar, mobileOpen, toggleMobileOpen, closeMobile }}>
      {children}
    </SidebarContext.Provider>
  )
}

export function useSidebar(type: SidebarType = 'project') {
  const ctx = useContext(SidebarContext)
  if (!ctx) throw new Error('useSidebar must be used within SidebarProvider')
  const { collapsed, toggleCollapsed } = ctx.getSidebar(type)
  return { collapsed, toggleCollapsed, mobileOpen: ctx.mobileOpen, toggleMobileOpen: ctx.toggleMobileOpen, closeMobile: ctx.closeMobile }
}
