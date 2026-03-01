import { Outlet, useNavigate, useMatch } from 'react-router-dom'

import { useTranslation } from 'react-i18next'
import { Settings, UserCog, Menu, HelpCircle, Inbox, LogOut } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { useSidebar } from '@/contexts/SidebarContext'
import { useNavigationGuard } from '@/contexts/NavigationGuardContext'
import { useProject, useProjects } from '@/hooks/useProjects'
import { Avatar } from '@/components/ui/Avatar'
import { Modal } from '@/components/ui/Modal'
import { ProjectKeyBadge } from '@/components/ui/ProjectKeyBadge'
import { Spinner } from '@/components/ui/Spinner'
import { useState, useRef, useEffect, useCallback } from 'react'
import { useKeyboardShortcutContext } from '@/contexts/KeyboardShortcutContext'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import { KeyboardShortcutsModal } from '@/components/KeyboardShortcutsModal'
import { WelcomeModal } from '@/components/WelcomeModal'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useBrand } from '@/contexts/BrandContext'
import { useInboxCount } from '@/hooks/useInbox'
import { PoweredByFooter } from '@/components/PoweredByFooter'
import { AppSidebar } from '@/components/AppSidebar'

export function AppShell() {
  const { t } = useTranslation()
  const { brandName } = useBrand()
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const { guardedNavigate } = useNavigationGuard()
  const { toggleMobileOpen } = useSidebar()
  const [menuOpen, setMenuOpen] = useState(false)
  const [switcherOpen, setSwitcherOpen] = useState(false)
  const [shortcutsOpen, setShortcutsOpen] = useState(false)
  const [welcomeOpen, setWelcomeOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  const { data: welcomeDismissed, isSuccess: welcomeLoaded, isError: welcomeNotFound } = usePreference<boolean>('welcome_dismissed')
  const { mutate: savePref } = useSetPreference()

  // Show welcome modal on first load if not dismissed
  const [welcomeAutoShown, setWelcomeAutoShown] = useState(false)
  useEffect(() => {
    if (!welcomeAutoShown && (welcomeLoaded || welcomeNotFound)) {
      setWelcomeAutoShown(true)
      if (welcomeDismissed !== true) {
        setWelcomeOpen(true)
      }
    }
  }, [welcomeLoaded, welcomeNotFound, welcomeAutoShown, welcomeDismissed])

  const { data: inboxCount } = useInboxCount()
  const projectMatch = useMatch('/projects/:projectKey/*')
  const adminMatch = useMatch('/admin/*')
  const preferencesMatch = useMatch('/preferences/*')
  const routeProjectKey = projectMatch?.params.projectKey
  const lastProjectKey = localStorage.getItem('taskwondo_last_project_key') ?? undefined
  const activeProjectKey = routeProjectKey ?? lastProjectKey
  const { data: activeProject } = useProject(activeProjectKey ?? '')

  useEffect(() => {
    if (!menuOpen) return
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [menuOpen])

  // Sequential combos: g-p (project switcher), g-i (inbox), g-o (project items)
  const { registerSequentialCombo } = useKeyboardShortcutContext()
  useEffect(() => {
    return registerSequentialCombo({
      id: 'go-to-projects',
      keys: ['g', 'p'],
      callback: () => setSwitcherOpen(true),
    })
  }, [registerSequentialCombo])
  useEffect(() => {
    return registerSequentialCombo({
      id: 'go-to-inbox',
      keys: ['g', 'i'],
      callback: () => guardedNavigate('/user/inbox'),
    })
  }, [registerSequentialCombo])
  useEffect(() => {
    if (!activeProjectKey) return
    return registerSequentialCombo({
      id: 'go-to-items',
      keys: ['g', 'o'],
      callback: () => guardedNavigate(`/projects/${activeProjectKey}/items`),
    })
  }, [activeProjectKey, navigate, registerSequentialCombo])

  useKeyboardShortcut({ key: '?' }, () => setShortcutsOpen(true))
  useKeyboardShortcut({ key: ',', ctrlKey: true }, () => guardedNavigate('/preferences'))

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return (
    <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
      <nav className="bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-14 relative">
            <div className="flex items-center gap-6 min-w-0">
              <button onClick={() => guardedNavigate('/projects')} className="text-lg font-bold text-indigo-600 dark:text-indigo-400 shrink-0">
                {brandName}
              </button>
              {adminMatch ? (
                <div className="flex items-center gap-2.5 min-w-0">
                  <Settings className="h-5 w-5 text-gray-500 dark:text-gray-400 shrink-0" />
                  <span className="text-base font-semibold text-gray-900 dark:text-gray-100 sm:hidden">{t('admin.titleShort')}</span>
                  <span className="text-base font-semibold text-gray-900 dark:text-gray-100 hidden sm:inline">{t('admin.title')}</span>
                </div>
              ) : preferencesMatch ? (
                <div className="flex items-center gap-2.5 min-w-0">
                  <UserCog className="h-5 w-5 text-gray-500 dark:text-gray-400 shrink-0" />
                  <span className="text-base font-semibold text-gray-900 dark:text-gray-100 truncate">{t('preferences.navTitle')}</span>
                </div>
              ) : activeProject ? (
                <button
                  onClick={() => setSwitcherOpen(true)}
                  className="hidden sm:flex items-center gap-2.5 hover:opacity-80 transition-opacity min-w-0"
                >
                  <ProjectKeyBadge size="nav">{activeProject.key}</ProjectKeyBadge>
                  <span className="text-base font-semibold text-gray-900 dark:text-gray-100 truncate">
                    {activeProject.name}
                  </span>
                </button>
              ) : null}
            </div>
            {activeProject && !adminMatch && !preferencesMatch && (
              <button
                onClick={() => setSwitcherOpen(true)}
                className="sm:hidden absolute left-1/2 -translate-x-1/2 top-0 h-full flex items-center hover:opacity-80 transition-opacity"
              >
                <ProjectKeyBadge size="nav-mobile">{activeProject.key}</ProjectKeyBadge>
              </button>
            )}
            <div className="relative flex items-center gap-2 shrink-0" ref={menuRef}>
              <button
                onClick={() => guardedNavigate('/user/inbox')}
                className="relative p-2 rounded-md text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800"
                aria-label={t('inbox.title')}
              >
                <Inbox className="h-5 w-5" />
                {inboxCount != null && inboxCount > 0 && (
                  <span className="absolute -top-0.5 -right-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-indigo-600 px-1 text-[10px] font-bold text-white">
                    {inboxCount > 99 ? '99+' : inboxCount}
                  </span>
                )}
              </button>
              <button
                onClick={toggleMobileOpen}
                className="sm:hidden p-2 rounded-md text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800"
                aria-label={t('sidebar.menu')}
              >
                <Menu className="h-5 w-5" />
              </button>
              <div className="hidden sm:block w-px h-5 bg-gray-200 dark:bg-gray-700 mx-1" />
              <button
                onClick={() => setMenuOpen(!menuOpen)}
                className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100"
              >
                <Avatar name={user?.display_name ?? ''} avatarUrl={user?.avatar_url} size="sm" />
                <span className="hidden sm:block">{user?.display_name}</span>
              </button>
              {menuOpen && (
                <div className="absolute right-0 top-full mt-1 w-48 bg-white/40 dark:bg-gray-800/40 backdrop-blur-sm rounded-md shadow-lg border border-gray-200 dark:border-gray-600 py-1 z-50">
                  <div className="px-4 py-2 text-xs text-gray-500 dark:text-gray-400 border-b border-gray-100 dark:border-gray-700">
                    {user?.email}
                  </div>
                  <button
                    onClick={() => { setMenuOpen(false); guardedNavigate('/preferences') }}
                    className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
                  >
                    <UserCog className="h-4 w-4 text-gray-400 dark:text-gray-500" />
                    {t('nav.preferences')}
                  </button>
                  {user?.global_role === 'admin' && (
                    <button
                      onClick={() => { setMenuOpen(false); guardedNavigate('/admin') }}
                      className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
                    >
                      <Settings className="h-4 w-4 text-gray-400 dark:text-gray-500" />
                      {t('nav.systemSettings')}
                    </button>
                  )}
                  <button
                    onClick={() => { setMenuOpen(false); setWelcomeOpen(true) }}
                    className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
                  >
                    <HelpCircle className="h-4 w-4 text-gray-400 dark:text-gray-500" />
                    {t('nav.help')}
                  </button>
                  <div className="border-t border-gray-100 dark:border-gray-700" />
                  <button
                    onClick={handleLogout}
                    className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
                  >
                    <LogOut className="h-4 w-4 text-gray-400 dark:text-gray-500" />
                    {t('nav.signOut')}
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      </nav>
      <AppSidebar mobileOnly projectKey={activeProjectKey} />
      <main className="flex-1">
        <Outlet />
      </main>
      <PoweredByFooter />
      <ProjectSwitcherModal
        open={switcherOpen}
        onClose={() => setSwitcherOpen(false)}
        activeProjectKey={activeProjectKey}
        onSelect={(key) => {
          setSwitcherOpen(false)
          guardedNavigate(`/projects/${key}`)
        }}
      />
      <KeyboardShortcutsModal open={shortcutsOpen} onClose={() => setShortcutsOpen(false)} />
      <WelcomeModal
        open={welcomeOpen}
        onClose={() => setWelcomeOpen(false)}
        onDismiss={() => savePref({ key: 'welcome_dismissed', value: true })}
        alreadyDismissed={welcomeDismissed === true}
      />
    </div>
  )
}

function ProjectSwitcherModal({
  open,
  onClose,
  activeProjectKey,
  onSelect,
}: {
  open: boolean
  onClose: () => void
  activeProjectKey?: string
  onSelect: (key: string) => void
}) {
  const { t } = useTranslation()
  const { data: projects, isLoading } = useProjects()
  const [search, setSearch] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLUListElement>(null)

  useEffect(() => {
    if (open) {
      setSearch('')
      setSelectedIndex(0)
      setTimeout(() => inputRef.current?.focus(), 0)
    }
  }, [open])

  const filtered = (projects ?? []).filter((p) => {
    if (!search) return true
    const q = search.toLowerCase()
    return p.key.toLowerCase().includes(q) || p.name.toLowerCase().includes(q)
  })

  // Reset selection when search changes
  useEffect(() => {
    setSelectedIndex(0)
  }, [search])

  // Scroll selected item into view
  const scrollSelectedIntoView = useCallback((index: number) => {
    const list = listRef.current
    if (!list) return
    const item = list.children[index] as HTMLElement | undefined
    item?.scrollIntoView({ block: 'nearest' })
  }, [])

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      const next = Math.min(selectedIndex + 1, filtered.length - 1)
      setSelectedIndex(next)
      scrollSelectedIntoView(next)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      const prev = Math.max(selectedIndex - 1, 0)
      setSelectedIndex(prev)
      scrollSelectedIntoView(prev)
    } else if (e.key === 'Enter' && filtered.length > 0) {
      e.preventDefault()
      onSelect(filtered[selectedIndex].key)
    }
  }, [selectedIndex, filtered, onSelect, scrollSelectedIntoView])

  return (
    <Modal open={open} onClose={onClose} title={t('projects.switcher.title')} position="top">
      <input
        ref={inputRef}
        type="text"
        placeholder={t('projects.switcher.search')}
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        onKeyDown={handleKeyDown}
        className="block w-full rounded-md border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:text-gray-100 dark:placeholder-gray-400 mb-3"
      />
      {isLoading ? (
        <div className="flex justify-center py-6"><Spinner /></div>
      ) : filtered.length === 0 ? (
        <p className="text-sm text-gray-500 dark:text-gray-400 py-4 text-center">{t('projects.noProjectsFound')}</p>
      ) : (
        <ul ref={listRef} className="max-h-64 overflow-y-auto -mx-2">
          {filtered.map((p, i) => (
            <li key={p.key}>
              <button
                onClick={() => onSelect(p.key)}
                onMouseEnter={() => setSelectedIndex(i)}
                className={`w-full text-left flex items-center gap-3 px-3 py-2.5 rounded-md text-sm ${
                  i === selectedIndex
                    ? 'bg-indigo-50 dark:bg-indigo-900/30'
                    : 'hover:bg-gray-100 dark:hover:bg-gray-700'
                }`}
              >
                <ProjectKeyBadge>{p.key}</ProjectKeyBadge>
                <span className="text-gray-900 dark:text-gray-100 font-medium truncate">{p.name}</span>
                {p.key === activeProjectKey && (
                  <span className="ml-auto text-xs text-indigo-600 dark:text-indigo-400 shrink-0">{t('common.current')}</span>
                )}
              </button>
            </li>
          ))}
        </ul>
      )}
    </Modal>
  )
}
