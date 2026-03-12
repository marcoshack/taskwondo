import { Outlet, useNavigate, useMatch } from 'react-router-dom'

import { useTranslation } from 'react-i18next'
import { useNamespacePath, toUrlSegment } from '@/hooks/useNamespacePath'
import { Settings, UserCog, Menu, HelpCircle, Inbox, LogOut, Search, Home, Plus } from 'lucide-react'
import { NamespaceIcon } from '@/components/NamespaceIcon'
import { useAuth } from '@/contexts/AuthContext'
import { useNamespaceContext } from '@/contexts/NamespaceContext'
import { useSidebar } from '@/contexts/SidebarContext'
import { useNavigationGuard } from '@/contexts/NavigationGuardContext'
import { useProject, useAllProjects } from '@/hooks/useProjects'
import { Avatar } from '@/components/ui/Avatar'
import { Modal } from '@/components/ui/Modal'
import { ProjectKeyBadge } from '@/components/ui/ProjectKeyBadge'
import { Spinner } from '@/components/ui/Spinner'
import { useState, useRef, useEffect, useLayoutEffect, useCallback } from 'react'
import { useKeyboardShortcutContext } from '@/contexts/KeyboardShortcutContext'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import { KeyboardShortcutsModal } from '@/components/KeyboardShortcutsModal'
import { SearchModal } from '@/components/SearchModal'
import { WelcomeModal } from '@/components/WelcomeModal'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useBrand } from '@/contexts/BrandContext'
import { useLayout } from '@/contexts/LayoutContext'
import { useInboxCount } from '@/hooks/useInbox'
import { PoweredByFooter } from '@/components/PoweredByFooter'
import { AppSidebar } from '@/components/AppSidebar'
import { CreateNamespaceModal } from '@/components/CreateNamespaceModal'

export function AppShell() {
  const { t } = useTranslation()
  const { brandName } = useBrand()
  const { containerClass } = useLayout()
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const { guardedNavigate } = useNavigationGuard()
  const { toggleMobileOpen } = useSidebar()
  const { namespaces, activeNamespace, setActiveNamespace, showSwitcher } = useNamespaceContext()
  const [menuOpen, setMenuOpen] = useState(false)
  const [switcherOpen, setSwitcherOpen] = useState(false)
  const [shortcutsOpen, setShortcutsOpen] = useState(false)
  const [searchOpen, setSearchOpen] = useState(false)
  const [welcomeOpen, setWelcomeOpen] = useState(false)
  const [nsDropdownOpen, setNsDropdownOpen] = useState(false)
  const [nsCreateOpen, setNsCreateOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const nsRef = useRef<HTMLDivElement>(null)

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
  const { p } = useNamespacePath()
  const projectMatch = useMatch('/:namespace/projects/:projectKey/*')
  const adminMatch = useMatch('/admin/*')
  const preferencesMatch = useMatch('/preferences/*')
  const routeProjectKey = projectMatch?.params.projectKey
  const lastProjectKey = localStorage.getItem('taskwondo_last_project_key') ?? undefined
  const activeProjectKey = routeProjectKey ?? lastProjectKey
  const { data: activeProject } = useProject(activeProjectKey ?? '')

  useEffect(() => {
    if (!menuOpen && !nsDropdownOpen) return
    const handler = (e: MouseEvent) => {
      if (menuOpen && menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
      if (nsDropdownOpen && nsRef.current && !nsRef.current.contains(e.target as Node)) {
        setNsDropdownOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [menuOpen, nsDropdownOpen])

  // Sequential combos: g-p (project switcher), g-i (inbox), g-o (project items)
  // useLayoutEffect ensures combos are registered before paint, so keyboard
  // shortcuts are available as soon as the UI is visible (prevents flaky E2E tests).
  const { registerSequentialCombo } = useKeyboardShortcutContext()
  useLayoutEffect(() => {
    return registerSequentialCombo({
      id: 'go-to-projects',
      keys: ['g', 'p'],
      callback: () => setSwitcherOpen(true),
    })
  }, [registerSequentialCombo])
  useLayoutEffect(() => {
    return registerSequentialCombo({
      id: 'go-to-inbox',
      keys: ['g', 'i'],
      callback: () => guardedNavigate('/user/inbox'),
    })
  }, [registerSequentialCombo])
  useLayoutEffect(() => {
    if (!activeProjectKey) return
    return registerSequentialCombo({
      id: 'go-to-items',
      keys: ['g', 'o'],
      callback: () => guardedNavigate(p(`/projects/${activeProjectKey}/items`)),
    })
  }, [activeProjectKey, navigate, registerSequentialCombo])

  useLayoutEffect(() => {
    return registerSequentialCombo({
      id: 'global-search',
      keys: ['g', 'k'],
      callback: () => setSearchOpen(true),
    })
  }, [registerSequentialCombo])

  useKeyboardShortcut({ key: '?' }, () => setShortcutsOpen(true))
  useKeyboardShortcut({ key: ',', ctrlKey: true }, () => guardedNavigate('/preferences'))

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return (
    <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
      <nav className="bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
        <div className={containerClass(true)}>
          <div className="flex justify-between h-14 relative">
            <div className="flex items-center gap-6 min-w-0">
              {/* Desktop: always show brand name */}
              <button onClick={() => guardedNavigate(p('/projects'))} className="hidden sm:block text-lg font-bold text-indigo-600 dark:text-indigo-400 shrink-0">
                {brandName}
              </button>
              {/* Mobile: home icon + project key when any project active, brand when none */}
              {activeProject ? (
                <div className="flex sm:hidden items-center gap-2 min-w-0">
                  <button
                    onClick={() => guardedNavigate(p('/projects'))}
                    className="p-1.5 rounded-md text-indigo-600 dark:text-indigo-400 hover:bg-gray-100 dark:hover:bg-gray-800 shrink-0"
                    aria-label={t('nav.home')}
                  >
                    <Home className="h-5 w-5" />
                  </button>
                  <button
                    onClick={() => setSwitcherOpen(true)}
                    className="hover:opacity-80 transition-opacity shrink-0"
                  >
                    <ProjectKeyBadge size="nav-mobile">{activeProject.key}</ProjectKeyBadge>
                  </button>
                </div>
              ) : (
                <button onClick={() => guardedNavigate(p('/projects'))} className="sm:hidden text-lg font-bold text-indigo-600 dark:text-indigo-400 shrink-0">
                  {brandName}
                </button>
              )}
              {adminMatch ? (
                <div className="hidden sm:flex items-center gap-2.5 min-w-0">
                  <Settings className="h-5 w-5 text-gray-500 dark:text-gray-400 shrink-0" />
                  <span className="text-base font-semibold text-gray-900 dark:text-gray-100">{t('admin.title')}</span>
                </div>
              ) : preferencesMatch ? (
                <div className="hidden sm:flex items-center gap-2.5 min-w-0">
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
            <div className="relative flex items-center gap-1.5 sm:gap-2 shrink-0" ref={menuRef}>
              <button
                onClick={() => setSearchOpen(true)}
                className="p-2 rounded-md text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800"
                aria-label={t('nav.search')}
              >
                <Search className="h-5 w-5" />
              </button>
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
              {/* Namespace switcher — icon-only dropdown */}
              {showSwitcher && (
                <div className="relative" ref={nsRef}>
                  <button
                    onClick={() => setNsDropdownOpen(!nsDropdownOpen)}
                    className="p-2 rounded-md hover:bg-gray-100 dark:hover:bg-gray-800"
                    aria-label={t('namespaces.switchNamespace')}
                    data-testid="namespace-switcher"
                  >
                    <NamespaceIcon
                      icon={activeNamespace?.icon ?? 'globe'}
                      color={activeNamespace?.color ?? 'slate'}
                      className="h-5 w-5"
                    />
                  </button>
                  {nsDropdownOpen && (
                    <div className="absolute right-0 top-full mt-1 w-64 bg-white/40 dark:bg-gray-800/40 backdrop-blur-sm rounded-md shadow-lg border border-gray-200 dark:border-gray-600 py-1 z-50">
                      <div className="px-3 py-1.5 text-xs font-medium text-gray-400 dark:text-gray-500 uppercase tracking-wider">
                        {t('namespaces.title')}
                      </div>
                      {namespaces.map((ns) => (
                        <button
                          key={ns.slug}
                          onClick={() => {
                            setNsDropdownOpen(false)
                            if (ns.slug !== activeNamespace?.slug) {
                              setActiveNamespace(ns.slug)
                            }
                          }}
                          className={`w-full text-left px-3 py-2 text-sm flex items-center gap-2.5 ${
                            ns.slug === activeNamespace?.slug
                              ? 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300'
                              : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
                          }`}
                        >
                          <NamespaceIcon icon={ns.icon} color={ns.color} className="h-4 w-4 shrink-0" />
                          <div className="min-w-0 flex-1">
                            <div className="font-medium truncate">{ns.display_name}</div>
                            {!ns.is_default && <div className="text-xs text-gray-400 dark:text-gray-500">{ns.slug}</div>}
                          </div>
                          {ns.slug === activeNamespace?.slug && (
                            <span className="text-xs text-indigo-600 dark:text-indigo-400 shrink-0">{t('common.current')}</span>
                          )}
                          {!ns.is_default && (
                            <button
                              onClick={(e) => {
                                e.stopPropagation()
                                setNsDropdownOpen(false)
                                guardedNavigate(`/${toUrlSegment(ns.slug)}/settings`)
                              }}
                              className="p-1 rounded text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 shrink-0"
                              aria-label={t('namespaces.settings')}
                            >
                              <Settings className="h-3.5 w-3.5" />
                            </button>
                          )}
                        </button>
                      ))}
                      <div className="border-t border-gray-100 dark:border-gray-700 mt-1 pt-1">
                        <button
                          onClick={() => {
                            setNsDropdownOpen(false)
                            setNsCreateOpen(true)
                          }}
                          className="w-full text-left px-3 py-2 text-sm text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2.5"
                        >
                          <Plus className="h-4 w-4" />
                          {t('namespaces.createNew')}
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              )}
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
        activeNamespaceSlug={activeNamespace?.slug}
        onSelect={(key, nsSlug) => {
          setSwitcherOpen(false)
          const segment = toUrlSegment(nsSlug || activeNamespace?.slug || 'default')
          guardedNavigate(`/${segment}/projects/${key}`)
        }}
      />
      <SearchModal open={searchOpen} onClose={() => setSearchOpen(false)} />
      <KeyboardShortcutsModal open={shortcutsOpen} onClose={() => setShortcutsOpen(false)} />
      <WelcomeModal
        open={welcomeOpen}
        onClose={() => setWelcomeOpen(false)}
        onDismiss={() => savePref({ key: 'welcome_dismissed', value: true })}
        alreadyDismissed={welcomeDismissed === true}
      />
      <CreateNamespaceModal
        open={nsCreateOpen}
        onClose={() => setNsCreateOpen(false)}
        onCreated={(ns) => {
          setNsCreateOpen(false)
          setActiveNamespace(ns.slug)
          navigate(`/${toUrlSegment(ns.slug)}/settings`)
        }}
      />
    </div>
  )
}

function ProjectSwitcherModal({
  open,
  onClose,
  activeProjectKey,
  activeNamespaceSlug,
  onSelect,
}: {
  open: boolean
  onClose: () => void
  activeProjectKey?: string
  activeNamespaceSlug?: string
  onSelect: (key: string, nsSlug?: string) => void
}) {
  const { t } = useTranslation()
  const { showSwitcher: showNamespaces } = useNamespaceContext()
  const { data: projects, isLoading } = useAllProjects()
  const { data: showAllNsPref, isSuccess: prefLoaded } = usePreference<boolean>('project_switcher_all_namespaces')
  const { mutate: savePref } = useSetPreference()
  const [showAllLocal, setShowAllLocal] = useState<boolean | null>(null)
  const showAllNamespaces = showAllLocal ?? (!prefLoaded || showAllNsPref !== false)
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
    // Filter by namespace when toggle is off
    if (showNamespaces && !showAllNamespaces && p.namespace_slug !== activeNamespaceSlug) return false
    if (!search) return true
    const q = search.toLowerCase()
    return p.key.toLowerCase().includes(q) || p.name.toLowerCase().includes(q) || (p.namespace_slug ?? '').toLowerCase().includes(q)
  })

  // Reset selection when search or namespace filter changes
  useEffect(() => {
    setSelectedIndex(0)
  }, [search, showAllNamespaces])

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
      const item = filtered[selectedIndex]
      onSelect(item.key, item.namespace_slug)
    }
  }, [selectedIndex, filtered, onSelect, scrollSelectedIntoView])

  const isCurrent = (p: { key: string; namespace_slug?: string }) =>
    p.key === activeProjectKey && (!showNamespaces || p.namespace_slug === activeNamespaceSlug)

  const handleToggleAllNamespaces = () => {
    const newValue = !showAllNamespaces
    setShowAllLocal(newValue)
    savePref({ key: 'project_switcher_all_namespaces', value: newValue })
  }

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
            <li key={p.id}>
              <button
                onClick={() => onSelect(p.key, p.namespace_slug)}
                onMouseEnter={() => setSelectedIndex(i)}
                className={`w-full text-left flex items-center gap-3 px-3 py-2.5 rounded-md text-sm ${
                  i === selectedIndex
                    ? 'bg-indigo-50 dark:bg-indigo-900/30'
                    : 'hover:bg-gray-100 dark:hover:bg-gray-700'
                }`}
              >
                <ProjectKeyBadge>{p.key}</ProjectKeyBadge>
                <span className="text-gray-900 dark:text-gray-100 font-medium truncate">{p.name}</span>
                <span className="ml-auto flex items-center gap-2 shrink-0">
                  {isCurrent(p) && (
                    <span className="text-xs text-indigo-600 dark:text-indigo-400">{t('common.current')}</span>
                  )}
                  {showNamespaces && p.namespace_slug && (
                    <span className="flex items-center gap-1 text-[0.7rem] text-gray-400 dark:text-gray-500">
                      <span>{p.namespace_slug}</span>
                      <NamespaceIcon icon={p.namespace_icon ?? 'building2'} color={p.namespace_color ?? 'slate'} className="h-3 w-3" />
                    </span>
                  )}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
      {showNamespaces && (
        <div className="flex justify-end mt-2 pt-2 border-t border-gray-100 dark:border-gray-700">
          <label className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400 cursor-pointer select-none" data-testid="all-namespaces-toggle">
            <input
              type="checkbox"
              checked={showAllNamespaces}
              onChange={handleToggleAllNamespaces}
              className="rounded border-gray-300 dark:border-gray-600 text-indigo-600 focus:ring-indigo-500 h-3.5 w-3.5"
            />
            {t('projects.switcher.showAllNamespaces')}
          </label>
        </div>
      )}
    </Modal>
  )
}
